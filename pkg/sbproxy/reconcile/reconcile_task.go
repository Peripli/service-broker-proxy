/*
 * Copyright 2018 The Service Manager Authors
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

package reconcile

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	cache "github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
)

// ReconcilationTask type represents a registration task that takes care of propagating broker creations
// and deletions to the platform. It reconciles the state of the proxy brokers in the platform to match
// the desired state provided by the Service Manager.
// TODO if the reg credentials are changed (the ones under cf.reg) we need to update the already registered brokers
type ReconcilationTask struct {
	options             *Settings
	group               *sync.WaitGroup
	platformClient      platform.Client
	smClient            sm.Client
	visibilityKeyMapper platform.ServiceVisibilityKeyMapper
	proxyPath           string
	ctx                 context.Context
	cache               *cache.Cache

	running *bool
}

// Settings type represents the sbproxy settings
type Settings struct {
	URL      string
	Username string
	Password string

	VisibilityCache bool          `mapstructure:"visibility_cache"`
	CacheExpiration time.Duration `mapstructure:"cache_expiration"`
}

// DefaultSettings creates default proxy settings
func DefaultSettings() *Settings {
	return &Settings{
		URL:             "",
		Username:        "",
		Password:        "",
		VisibilityCache: true,
		CacheExpiration: time.Hour,
	}
}

// NewTask builds a new ReconcilationTask
func NewTask(ctx context.Context,
	options *Settings,
	group *sync.WaitGroup,
	platformClient platform.Client,
	smClient sm.Client,
	proxyPath string,
	c *cache.Cache,
	visibilityKeyMapper platform.ServiceVisibilityKeyMapper,
	running *bool) *ReconcilationTask {
	return &ReconcilationTask{
		options:             options,
		group:               group,
		platformClient:      platformClient,
		smClient:            smClient,
		proxyPath:           proxyPath,
		ctx:                 ctx,
		cache:               c,
		running:             running,
		visibilityKeyMapper: visibilityKeyMapper,
	}
}

// Validate validates that the configuration contains all mandatory properties
func (c *Settings) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("validate settings: missing host")
	}
	if len(c.Username) == 0 {
		return errors.New("validate settings: missing username")
	}
	if len(c.Password) == 0 {
		return errors.New("validate settings: missing password")
	}
	if c.VisibilityCache {
		if time.Minute > c.CacheExpiration {
			return errors.New("validate settings: if cache is enabled, cache_expiration should be at least 1 minute")
		}
	}
	return nil
}

// Run executes the registration task that is responsible for reconciling the state of the proxy brokers at the
// platform with the brokers provided by the Service Manager
func (r ReconcilationTask) Run() {
	logger := log.C(r.ctx)
	if *r.running {
		logger.Info("Another reconcile job is in process... I will skip this one.")
		return
	}

	logger.Debug("STARTING scheduled reconciliation task...")

	r.group.Add(1)
	*r.running = true
	defer func() {
		r.group.Done()
		*r.running = false
	}()
	r.run()

	logger.Debug("FINISHED scheduled reconciliation task...")
}

func (r ReconcilationTask) run() {
	// get all the registered proxy brokers from the platform
	brokersFromPlatform, err := r.getBrokersFromPlatform()
	if err != nil {
		log.C(r.ctx).WithError(err).Error("An error occurred while obtaining already registered brokers")
		return
	}

	// get all the brokers that are in SM and for which a proxy broker should be present in the platform
	brokersFromSM, err := r.getBrokersFromSM()
	if err != nil {
		log.C(r.ctx).WithError(err).Error("An error occurred while obtaining brokers from Service Manager")
		return
	}

	// control logic - make sure current state matches desired state
	changesOnBrokers := r.reconcileBrokers(brokersFromPlatform, brokersFromSM)

	plans, err := r.getSMPlans()
	if err != nil {
		log.C(r.ctx).WithError(err).Error("An error occurred while obtaining plans from Service Manager")
		return
	}

	platformVisibilities, err := r.getPlatformVisibilitiesWithCache(plans, changesOnBrokers)
	if err != nil {
		log.C(r.ctx).WithError(err).Error("An error occurred while obtaining existing visibilities from platform")
		return
	}

	smPlansMap := smPlansToMap(plans)
	smVisibilities, err := r.getSMVisibilities(smPlansMap)
	if err != nil {
		fmt.Println(err)
	}

	r.reconcileServiceVisibilities(platformVisibilities, smVisibilities)
}

func smPlansToMap(plans []*types.ServicePlan) map[string]*types.ServicePlan {
	plansMap := make(map[string]*types.ServicePlan, len(plans))
	for _, plan := range plans {
		plansMap[plan.ID] = plan
	}
	return plansMap
}
