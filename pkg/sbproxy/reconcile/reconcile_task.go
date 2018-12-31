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
	"sync"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/pkg/log"
	cache "github.com/patrickmn/go-cache"
)

// ReconciliationTask type represents a registration task that takes care of propagating broker creations
// and deletions to the platform. It reconciles the state of the proxy brokers in the platform to match
// the desired state provided by the Service Manager.
// TODO if the reg credentials are changed (the ones under cf.reg) we need to update the already registered brokers
type ReconciliationTask struct {
	options        *Settings
	group          *sync.WaitGroup
	platformClient platform.Client
	smClient       sm.Client
	proxyPath      string
	ctx            context.Context
	cache          *cache.Cache
}

// NewTask builds a new ReconciliationTask
func NewTask(ctx context.Context,
	options *Settings,
	group *sync.WaitGroup,
	platformClient platform.Client,
	smClient sm.Client,
	proxyPath string,
	c *cache.Cache) *ReconciliationTask {
	return &ReconciliationTask{
		options:        options,
		group:          group,
		platformClient: platformClient,
		smClient:       smClient,
		proxyPath:      proxyPath,
		ctx:            ctx,
		cache:          c,
	}
}

// Run executes the registration task that is responsible for reconciling the state of the proxy brokers at the
// platform with the brokers provided by the Service Manager
func (r ReconciliationTask) Run() {
	logger := log.C(r.ctx)

	logger.Debug("STARTING scheduled reconciliation task...")

	r.group.Add(1)
	defer r.group.Done()
	r.run()

	logger.Debug("FINISHED scheduled reconciliation task...")
}

func (r ReconciliationTask) run() {
	r.processBrokers()
	r.processVisibilities()
}
