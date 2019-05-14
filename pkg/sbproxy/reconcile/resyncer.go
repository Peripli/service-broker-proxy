package reconcile

import (
	"context"

	"github.com/patrickmn/go-cache"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sm"

	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
)

// NewResyncer returns a resyncer that reconciles the state of the proxy brokers and visibilities
// in the platform to match the desired state provided by the Service Manager.
func NewResyncer(settings *Settings, platformClient platform.Client, smClient sm.Client, proxyPath string, cache *cache.Cache) Resyncer {
	return &resyncJob{
		options:        settings,
		platformClient: platformClient,
		smClient:       smClient,
		proxyPath:      proxyPath,
		cache:          cache,
	}
}

type resyncJob struct {
	options        *Settings
	platformClient platform.Client
	smClient       sm.Client
	proxyPath      string
	cache          *cache.Cache
}

// Resync reconciles the state of the proxy brokers and visibilities at the platform
// with the brokers provided by the Service Manager
func (r *resyncJob) Resync(ctx context.Context) {
	resyncContext, taskID, err := createResyncContext(ctx)
	if err != nil {
		log.C(ctx).WithError(err).Error("could not create resync job context")
		return
	}
	log.C(ctx).Infof("STARTING resync job %s...", taskID)
	r.process(resyncContext)
	log.C(ctx).Infof("FINISHED resync job %s", taskID)
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
	smBrokers, err := r.getSMBrokers(ctx)
	if err != nil {
		logger.Error(err)
		return
	}
	smVisibilities, mappedPlans, err := r.getSMVisibilitiesAndPlans(ctx, smBrokers) // fetch as soon as possible
	if err != nil {
		logger.Error(err)
		return
	}

	// get all the registered brokers from the platform
	brokersFromPlatform, err := r.getBrokersFromPlatform(ctx)
	if err != nil {
		logger.WithError(err).Error("an error occurred while obtaining brokers from Platform")
		return
	}

	r.reconcileBrokers(ctx, brokersFromPlatform, smBrokers)
	r.reconcileVisibilities(ctx, smVisibilities, smBrokers, mappedPlans)
}

func (r *resyncJob) getSMBrokers(ctx context.Context) ([]platform.ServiceBroker, error) {
	if r.platformClient.Broker() == nil {
		return nil, errors.New("platform client cannot handle brokers. Cannot perform reconciliation")
	}
	brokersFromSM, err := r.getBrokersFromSM(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "an error occurred while obtaining brokers from Service Manager")
	}
	return brokersFromSM, nil
}

func (r *resyncJob) getSMVisibilitiesAndPlans(ctx context.Context, smBrokers []platform.ServiceBroker) ([]*platform.Visibility, map[brokerPlanKey]*types.ServicePlan, error) {
	if r.platformClient.Visibility() == nil {
		return nil, nil, errors.New("platform client cannot handle visibilities. Cannot perform reconciliation")
	}

	smOfferings, err := r.getSMServiceOfferingsByBrokers(ctx, smBrokers)
	if err != nil {
		return nil, nil, errors.Wrap(err, "an error occurred while obtaining service offerings from Service Manager")
	}

	smPlans, err := r.getSMPlansByBrokersAndOfferings(ctx, smOfferings)
	if err != nil {
		return nil, nil, errors.Wrap(err, "an error occurred while obtaining plans from Service Manager")
	}

	mappedPlans := mapPlansByBrokerPlanID(smPlans)
	visibilitiesFromSM, err := r.getVisibilitiesFromSM(ctx, mappedPlans, smBrokers)
	if err != nil {
		return nil, nil, errors.Wrap(err, "an error occurred while obtaining visibilities from Service Manager")
	}
	return visibilitiesFromSM, mappedPlans, nil
}

func (r *resyncJob) getBrokersFromPlatform(ctx context.Context) ([]platform.ServiceBroker, error) {
	logger := log.C(ctx)
	logger.Info("resyncJob getting brokers from platform...")
	registeredBrokers, err := r.platformClient.Broker().GetBrokers(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error getting brokers from platform")
	}
	logger.Debugf("resyncJob SUCCESSFULLY retrieved %d brokers from platform", len(registeredBrokers))

	return registeredBrokers, nil
}
