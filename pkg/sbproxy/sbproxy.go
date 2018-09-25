package sbproxy

import (
	"sync"

	"fmt"

	"context"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/logging"
	"github.com/Peripli/service-broker-proxy/pkg/osb"
	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/api/filters"
	smosb "github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/server"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/reconcile"
)

const (
	// BrokerPathParam for the broker id
	BrokerPathParam = "brokerID"

	// APIPrefix for the Reconcile OSB API
	APIPrefix = "/v1/osb"

	// Path for the Reconcile OSB API
	Path = APIPrefix + "/{" + BrokerPathParam + "}"
)

// SMProxyBuilder type is an extension point that allows adding additional filters, plugins and
// controllers before running SMProxy.
type SMProxyBuilder struct {
	*web.API

	ctx context.Context
	cfg *Settings

	*cron.Cron
	group *sync.WaitGroup
}

// SMProxy  struct
type SMProxy struct {
	*server.Server

	ctx context.Context

	scheduler *cron.Cron
	group     *sync.WaitGroup
}

// DefaultEnv creates a default environment that can be used to boot up a Service Broker proxy
func DefaultEnv(additionalPFlags ...func(set *pflag.FlagSet)) env.Environment {
	set := pflag.NewFlagSet("Configuration Flags", pflag.ExitOnError)

	AddPFlags(set)
	for _, addFlags := range additionalPFlags {
		addFlags(set)
	}
	environment, err := env.New(set)
	if err != nil {
		panic(fmt.Errorf("error loading environment: %s", err))
	}
	return environment
}

// New creates service broker proxy that is configured from the provided environment and platform client.
func New(ctx context.Context, env env.Environment, platformClient platform.Client) *SMProxyBuilder {
	cronScheduler := cron.New()
	var group sync.WaitGroup

	cfg, err := NewSettings(env)
	if err != nil {
		panic(err)
	}

	if err := cfg.Validate(); err != nil {
		panic(err)
	}

	logging.Setup(cfg.Log)

	api := &web.API{
		Controllers: []web.Controller{
			smosb.NewController(&osb.BrokerDetails{
				URL:      cfg.Sm.Host + cfg.Sm.OsbAPI,
				Username: cfg.Sm.User,
				Password: cfg.Sm.Password,
			}, &sm.SkipSSLTransport{
				SkipSslValidation: cfg.Sm.SkipSSLValidation,
			}),
		},
	}

	sbProxy := &SMProxyBuilder{
		API:   api,
		Cron:  cronScheduler,
		ctx:   ctx,
		cfg:   cfg,
		group: &group,
	}

	smClient, err := sm.NewClient(cfg.Sm)
	if err != nil {
		panic(err)
	}
	regJob := reconcile.NewTask(&group, platformClient, smClient, cfg.Reconcile.Host+APIPrefix)
	
	resyncSchedule := "@every " + cfg.Sm.ResyncPeriod.String()
	logrus.Info("Brokers and Access resync schedule: ", resyncSchedule)

	if err := cronScheduler.AddJob(resyncSchedule, regJob); err != nil {
		panic(err)
	}

	return sbProxy
}

// Build builds the Service Manager
func (smb *SMProxyBuilder) Build() *SMProxy {
	srv := server.New(smb.cfg.Server, smb.API)
	srv.Use(filters.NewRecoveryMiddleware())

	return &SMProxy{
		Server:    srv,
		scheduler: smb.Cron,
		ctx:       smb.ctx,
		group:     smb.group,
	}
}

// Run starts the proxy
func (p *SMProxy) Run() {
	p.scheduler.Start()
	defer p.scheduler.Stop()
	defer waitWithTimeout(p.group, p.Server.Config.ShutdownTimeout)

	logrus.Info("Running SBProxy...")

	p.Server.Run(p.ctx)
}


// waitWithTimeout waits for a WaitGroup to finish for a certain duration and times out afterwards
// WaitGroup parameter should be pointer or else the copy won't get notified about .Done() calls
func waitWithTimeout(group *sync.WaitGroup, timeout time.Duration) {
	c := make(chan struct{})
	go func() {
		defer close(c)
		group.Wait()
	}()
	select {
	case <-c:
		logrus.Debug(fmt.Sprintf("Timeout WaitGroup %+v finished successfully", group))
	case <-time.After(timeout):
		logrus.Fatal("Shutdown took more than ", timeout)
		close(c)
	}
}
