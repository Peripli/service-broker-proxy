/*
 * Copyright 2019 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sbproxy

import (
	"sync"

	"github.com/Peripli/service-broker-proxy/pkg/filter"
	"github.com/Peripli/service-broker-proxy/pkg/logging"
	"github.com/Peripli/service-manager/api/healthcheck"
	"github.com/Peripli/service-manager/pkg/health"
	"github.com/Peripli/service-manager/pkg/log"
	secfilters "github.com/Peripli/service-manager/pkg/security/filters"
	"github.com/Peripli/service-manager/pkg/util"
	"github.com/patrickmn/go-cache"

	"fmt"

	"context"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/osb"
	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/notifications"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/reconcile"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/api/filters"
	smosb "github.com/Peripli/service-manager/api/osb"
	"github.com/Peripli/service-manager/pkg/env"
	"github.com/Peripli/service-manager/pkg/server"
	"github.com/Peripli/service-manager/pkg/web"
	"github.com/robfig/cron"
	"github.com/spf13/pflag"
)

const (
	// BrokerPathParam for the broker id
	BrokerPathParam = "brokerID"

	// APIPrefix for the Proxy OSB API
	APIPrefix = "/v1/osb"

	// Path for the Proxy OSB API
	Path = APIPrefix + "/{" + BrokerPathParam + "}"

	cacheCleanupInterval = 1 * time.Minute
)

// SMProxyBuilder type is an extension point that allows adding additional filters, plugins and
// controllers before running SMProxy.
type SMProxyBuilder struct {
	*web.API
	*cron.Cron

	ctx                   context.Context
	cfg                   *Settings
	group                 *sync.WaitGroup
	notificationsProducer *notifications.Producer
}

// SMProxy  struct
type SMProxy struct {
	*server.Server

	scheduler             *cron.Cron
	ctx                   context.Context
	group                 *sync.WaitGroup
	notificationsProducer *notifications.Producer
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
func New(ctx context.Context, cancel context.CancelFunc, env env.Environment, platformClient platform.Client) *SMProxyBuilder {
	cfg, err := NewSettings(env)
	if err != nil {
		panic(err)
	}

	if err := cfg.Validate(); err != nil {
		panic(err)
	}

	ctx = log.Configure(ctx, cfg.Log)
	log.AddHook(&logging.ErrorLocationHook{})

	util.HandleInterrupts(ctx, cancel)

	api := &web.API{
		Controllers: []web.Controller{
			smosb.NewController(&osb.BrokerDetailsFetcher{
				URL:      cfg.Sm.URL + cfg.Sm.OSBAPIPath,
				Username: cfg.Sm.User,
				Password: cfg.Sm.Password,
			}, nil, &sm.SkipSSLTransport{
				SkipSslValidation: cfg.Sm.SkipSSLValidation,
			}),
		},
		Filters: []web.Filter{
			&filters.Logging{},
			filter.NewBasicAuthFilter(cfg.Reconcile.Username, cfg.Reconcile.Password),
			secfilters.NewRequiredAuthnFilter(),
		},
		Registry: health.NewDefaultRegistry(),
	}

	smClient, err := sm.NewClient(cfg.Sm)
	if err != nil {
		panic(err)
	}

	var group sync.WaitGroup

	c := cache.New(cfg.Reconcile.CacheExpiration, cacheCleanupInterval)
	resyncJob := reconcile.NewTask(ctx, cfg.Reconcile, &group, platformClient, smClient, cfg.Reconcile.URL+APIPrefix, c)
	go resyncJob.Run() //TODO: this should be removed and should be triggered in the processor

	notificationsProducer, err := notifications.NewProducer(cfg.Producer, cfg.Sm)
	if err != nil {
		panic(err)
	}

	return &SMProxyBuilder{
		API:                   api,
		ctx:                   ctx,
		cfg:                   cfg,
		group:                 &group,
		notificationsProducer: notificationsProducer,
	}
}

// Build builds the Service Manager
func (smb *SMProxyBuilder) Build() *SMProxy {
	smb.installHealth()

	srv := server.New(smb.cfg.Server, smb.API)
	srv.Use(filters.NewRecoveryMiddleware())

	return &SMProxy{
		Server:                srv,
		scheduler:             smb.Cron,
		ctx:                   smb.ctx,
		group:                 smb.group,
		notificationsProducer: smb.notificationsProducer,
	}
}

func (smb *SMProxyBuilder) installHealth() {
	if len(smb.HealthIndicators()) > 0 {
		smb.RegisterControllers(healthcheck.NewController(smb.HealthIndicators(), smb.HealthAggregationPolicy()))
	}
}

// Run starts the proxy
func (p *SMProxy) Run() {
	defer waitWithTimeout(p.ctx, p.group, p.Server.Config.ShutdownTimeout)
	p.scheduler.Start()
	defer p.scheduler.Stop()

	messages := make(chan *notifications.Message, 1024)
	p.notificationsProducer.Start(p.ctx, messages)

	log.C(p.ctx).Info("Running SBProxy...")

	p.Server.Run(p.ctx)
}

// waitWithTimeout waits for a WaitGroup to finish for a certain duration and times out afterwards
// WaitGroup parameter should be pointer or else the copy won't get notified about .Done() calls
func waitWithTimeout(ctx context.Context, group *sync.WaitGroup, timeout time.Duration) {
	c := make(chan struct{})
	go func() {
		defer close(c)
		group.Wait()
	}()
	select {
	case <-c:
		log.C(ctx).Debug(fmt.Sprintf("Timeout WaitGroup %+v finished successfully", group))
	case <-time.After(timeout):
		log.C(ctx).Fatal("Shutdown took more than ", timeout)
		close(c)
	}
}
