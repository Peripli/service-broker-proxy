package sbproxy

import (
	"sync"

	"fmt"

	"github.com/Peripli/service-broker-proxy/pkg/config"
	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-manager/pkg/server"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/robfig/cron"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"time"
	"github.com/Peripli/service-broker-proxy/pkg/logging"
	"github.com/Peripli/service-manager/pkg/web"
	smOSB "github.com/Peripli/service-manager/api/osb"
	"context"
	"github.com/Peripli/service-manager/api/filters"
)

const (
	// BrokerPathParam for the broker id
	BrokerPathParam = "brokerID"

	// APIPrefix for the Proxy OSB API
	APIPrefix = "/v1/osb"

	// Path for the Proxy OSB API
	Path = APIPrefix + "/{" + BrokerPathParam + "}"
)

// SMProxyBuilder type is an extension point that allows adding additional filters, plugins and
// controllers before running SMProxy.
type SMProxyBuilder struct {
	*web.API
	*cron.Cron

	ctx context.Context
	cfg *server.Settings
}

// SMProxy  struct
type SMProxy struct {
	Server *server.Server

	scheduler *cron.Cron
	ctx    context.Context
	group         *sync.WaitGroup
}


// DefaultEnv creates a default environment that can be used to boot up a Service Manager
func DefaultEnv(additionalPFlags ...func(set *pflag.FlagSet)) env.Environment {
	set := env.EmptyFlagSet()

	config.AddPFlags(set)
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
func New(ctx context.Context, env env.Environment, client platform.Client) *SMProxyBuilder {
	cronScheduler := cron.New()
	var group sync.WaitGroup

	cfg, err := config.New(env)
	panic(err)

	if err := cfg.Validate(); err != nil {
		panic(err)
	}

	logging.Setup(cfg.Log)

	api := &web.API{
		Controllers: []web.Controller{
			&smOSB.Controller{
				//TODO
				Tr: sm.Transport{
					Username: cfg.Sm.User,
					Password: cfg.Sm.Password,
					URL:      cfg.Sm.Host,
					Rt:       sm.SkipSSLTransport{
						SkipSslValidation: cfg.Sm.SkipSslValidation,
					},
				},
			},
		},
	}

	sbProxy := &SMProxyBuilder{
		API:           api,
		ctx:           ctx,
		cfg:           cfg.Server,
	}

	regJob, err := defaultRegJob(&group, client, cfg.Sm, cfg.Server.Host)
	if err != nil {
		panic(err)
	}

	resyncSchedule := "@every " + cfg.Sm.ResyncPeriod.String()
	logrus.Info("Brokers and Access resync schedule: ", resyncSchedule)

	if err := cronScheduler.AddJob(resyncSchedule, regJob); err != nil {
		panic(err)
	}

	return sbProxy
}

// Build builds the Service Manager
func (smb *SMProxyBuilder) Build() *SMProxy {
	srv := server.New(smb.cfg, smb.API)
	srv.Use(filters.NewRecoveryMiddleware())

	return &SMProxy{
		ctx:    smb.ctx,
		Server: srv,
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

func defaultRegJob(group *sync.WaitGroup, platformClient platform.Client, smConfig *sm.Settings, proxyHost string) (cron.Job, error) {
	smClient, err := smConfig.CreateFunc(smConfig)
	if err != nil {
		return nil, err
	}
	regTask := NewTask(group, platformClient, smClient, proxyHost+APIPrefix)

	return regTask, nil
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
		logrus.Debug("Timeout WaitGroup ", group, " finished successfully")
	case <-time.After(timeout):
		logrus.Fatal("Shutdown took more than ", timeout)
		close(c)
	}
}
