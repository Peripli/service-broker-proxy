package reconcile

import (
	"context"
	"fmt"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sm"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// NewResyncer returns a resyncer that reconciles the state of the proxy brokers and visibilities
// in the platform to match the desired state provided by the Service Manager.
func NewResyncer(settings *Settings, platformClient platform.Client, smClient sm.Client, smSettings *sm.Settings, smPath, proxyPathPattern string) Resyncer {
	return &resyncJob{
		options:               settings,
		platformClient:        platformClient,
		smClient:              smClient,
		defaultBrokerUsername: smSettings.User,
		defaultBrokerPassword: smSettings.Password,
		smPath:                smPath,
		proxyPathPattern:      proxyPathPattern,
	}
}

type resyncJob struct {
	options               *Settings
	platformClient        platform.Client
	smClient              sm.Client
	defaultBrokerUsername string
	defaultBrokerPassword string
	smPath                string
	proxyPathPattern      string
}

// Resync reconciles the state of the proxy brokers and visibilities at the platform
// with the brokers provided by the Service Manager
func (r *resyncJob) Resync(ctx context.Context) {
	resyncContext, taskID, err := createResyncContext(ctx)
	if err != nil {
		log.C(ctx).WithError(err).Error("could not create resync job context")
		return
	}
	log.C(ctx).Infof("Starting resync job %s...", taskID)
	start := time.Now()
	r.process(resyncContext)
	log.C(ctx).Infof("Finished resync job %s in %v", taskID, time.Since(start))
}

func createResyncContext(ctx context.Context) (context.Context, string, error) {
	correlationID, err := uuid.NewV4()
	if err != nil {
		return nil, "", errors.Wrap(err, "could not generate correlationID")
	}
	entry := log.C(ctx).WithField(log.FieldCorrelationID, correlationID.String())
	return log.ContextWithLogger(ctx, entry), correlationID.String(), nil
}

func (r *resyncJob) process(ctx context.Context) {
	logger := log.C(ctx)
	if r.platformClient.Broker() == nil || r.platformClient.Visibility() == nil {
		logger.Error("platform must be able to handle both brokers and visibilities. Cannot perform reconciliation")
		return
	}

	allBrokers, err := r.getBrokersFromSM(ctx)
	if err != nil {
		logger.WithError(err).Error("an error occurred while obtaining brokers from Service Manager")
		return
	}

	allBrokersPlans, err := r.getSMPlans(ctx, allBrokers)
	if err != nil {
		logger.Error(err)
		return
	}
	// get all the registered brokers from the platform
	logger.Info("resyncJob getting brokers from platform...")
	platformBrokers, err := r.platformClient.Broker().GetBrokers(ctx)
	if err != nil {
		logger.WithError(err).Error("an error occurred while obtaining brokers from Platform")
		return
	}
	logger.Infof("resyncJob successfully retrieved %d brokers from platform", len(platformBrokers))

	r.reconcileBrokers(ctx, platformBrokers, allBrokers)
	r.resetPlatformCache(ctx)

	if r.options.VisibilityBrokerChunkSize > 0 {
		brokerChunks := chunkSlice(allBrokers, r.options.VisibilityBrokerChunkSize)
		for _, brokers := range brokerChunks {
			visibilities, err := getBrokersVisibilities(ctx, r, brokers, allBrokersPlans)
			if err != nil {
				logger.WithError(err).Error("an error occurred while obtaining visibilities from Service Manager")
				return
			}
			r.reconcileVisibilities(ctx, visibilities, brokers)
		}
	} else {
		visibilities, err := getBrokersVisibilities(ctx, r, allBrokers, allBrokersPlans)
		if err != nil {
			logger.WithError(err).Error("an error occurred while obtaining visibilities from Service Manager")
			return
		}
		r.reconcileVisibilities(ctx, visibilities, allBrokers)
	}
}
func (r *resyncJob) getSMPlans(ctx context.Context, smBrokers []*platform.ServiceBroker) (map[string]map[string]brokerPlan, error) {
	smOfferings, err := r.getSMServiceOfferings(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "an error occurred while obtaining service offerings from Service Manager")
	}
	brokerPlans, err := r.getSMBrokerPlans(ctx, smOfferings, smBrokers)
	if err != nil {
		return nil, errors.Wrap(err, "an error occurred while obtaining plans from Service Manager")
	}
	return brokerPlans, nil
}

func (r *resyncJob) resetPlatformCache(ctx context.Context) {
	logger := log.C(ctx)

	if cache, ok := r.platformClient.(platform.Caching); ok {
		if err := cache.ResetCache(ctx); err != nil {
			logger.WithError(err).Error("an error occurred while loading platform data")
			return
		}
	}
}

func getBrokersVisibilities(ctx context.Context, r *resyncJob, brokers []*platform.ServiceBroker, brokersPlans map[string]map[string]brokerPlan) ([]*platform.Visibility, error) {
	plans := make(map[string]brokerPlan)
	for _, broker := range brokers {
		for planID, plan := range brokersPlans[broker.GUID] {
			plans[planID] = plan
		}
	}
	smVisibilities, err := r.getVisibilitiesFromSM(ctx, plans)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while obtaining visibilities from Service Manager")
	}
	return smVisibilities, nil
}

func chunkSlice(slice []*platform.ServiceBroker, chunkSize int) [][]*platform.ServiceBroker {
	var chunks [][]*platform.ServiceBroker
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize

		// necessary check to avoid slicing beyond slice capacity
		if end > len(slice) {
			end = len(slice)
		}

		chunks = append(chunks, slice[i:end])
	}

	return chunks
}
