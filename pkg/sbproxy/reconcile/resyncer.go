package reconcile

import (
	"context"

	"github.com/patrickmn/go-cache"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sm"

	"github.com/Peripli/service-manager/pkg/log"
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
		stats:          make(map[string]interface{}),
	}
}

type resyncJob struct {
	options        *Settings
	platformClient platform.Client
	smClient       sm.Client
	proxyPath      string
	cache          *cache.Cache
	stats          map[string]interface{}
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
	r.processBrokers(resyncContext)
	r.processVisibilities(resyncContext)

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
