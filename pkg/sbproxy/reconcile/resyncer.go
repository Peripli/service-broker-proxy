package reconcile

import (
	"context"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sm"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// NewResyncer returns a resyncer that reconciles the state of the proxy brokers and visibilities
// in the platform to match the desired state provided by the Service Manager.
func NewResyncer(settings *Settings, platformClient platform.Client, smClient sm.Client, smPath, proxyPathPattern string) Resyncer {
	return &resyncJob{
		options:          settings,
		platformClient:   platformClient,
		smClient:         smClient,
		smPath:           smPath,
		proxyPathPattern: proxyPathPattern,
	}
}

type resyncJob struct {
	options          *Settings
	platformClient   platform.Client
	smClient         sm.Client
	smPath           string
	proxyPathPattern string
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

	smBrokers, err := r.getBrokersFromSM(ctx)
	if err != nil {
		logger.WithError(err).Error("an error occurred while obtaining brokers from Service Manager")
		return
	}

	plansFromSM, err := r.getSMPlans(ctx, smBrokers)
	if err != nil {
		logger.Error(err)
		return
	}

	smVisibilities, err := r.getVisibilitiesFromSM(ctx, plansFromSM) // fetch as soon as possible
	if err != nil {
		logger.WithError(err).Error("an error occurred while obtaining visibilities from Service Manager")
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

	r.reconcileBrokers(ctx, platformBrokers, smBrokers)
	r.resetPlatformCache(ctx)
	r.reconcileVisibilities(ctx, smVisibilities, smBrokers)
}

func (r *resyncJob) getSMPlans(ctx context.Context, smBrokers []*platform.ServiceBroker) (map[string]brokerPlan, error) {
	smOfferings, err := r.getSMServiceOfferings(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "an error occurred while obtaining service offerings from Service Manager")
	}
	smPlans, err := r.getSMBrokerPlans(ctx, smOfferings, smBrokers)
	if err != nil {
		return nil, errors.Wrap(err, "an error occurred while obtaining plans from Service Manager")
	}
	return smPlans, nil
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
