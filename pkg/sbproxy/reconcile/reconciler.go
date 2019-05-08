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

	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/notifications"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/notifications/handlers"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/gofrs/uuid"
	cache "github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
)

const running int32 = 1
const notRunning int32 = 0

const smBrokersStats = "sm_brokers"

// for stubbing in unit tests
type executor interface {
	apply(*types.Notification)
	resync()
}

// Reconciler takes care of propagating broker and visibility changes to the platform.
// It reconciles the state of the proxy brokers and visibilities
// in the platform to match the desired state provided by the Service Manager.
// TODO if the reg credentials are changed (the ones under cf.reg) we need to update the already registered brokers
type Reconciler struct {
	Options        *Settings
	PlatformClient platform.Client
	SMClient       sm.Client
	ProxyPath      string
	Cache          *cache.Cache
	GlobalContext  context.Context

	notificationConsumer *notifications.Consumer
	exec                 executor
}

type resyncJob struct {
	*Reconciler

	resyncContext context.Context
	stats         map[string]interface{}
}

// Process resync and notification messages sequentially in one goroutine
// to avoid concurrent changes in the platform
func (r *Reconciler) Process(messages <-chan *notifications.Message) {
	exec := r.exec
	if exec == nil {
		exec = r
	}

	for {
		select {
		case <-r.GlobalContext.Done():
			log.C(r.GlobalContext).Info("Context cancelled. Terminating reconciler.")
			return
		case m, ok := <-messages:
			if !ok {
				log.C(r.GlobalContext).Info("Messages channel closed. Terminating reconciler.")
				return
			}
			log.C(r.GlobalContext).Debugf("Reconciler received message %+v", m)
			if m.Resync {
				// discard any pending change notifications as we will do a full resync
				drain(messages)
				exec.resync()
			} else {
				exec.apply(m.Notification)
			}
		}
	}
}

func drain(messages <-chan *notifications.Message) {
	for {
		select {
		case _, ok := <-messages:
			if !ok {
				return
			}
		default:
			return
		}
	}
}

func (r *Reconciler) apply(n *types.Notification) {
	if r.notificationConsumer == nil {
		r.notificationConsumer = r.createConsumer()
	}
	r.notificationConsumer.Consume(r.GlobalContext, n)
}

func (r *Reconciler) createConsumer() *notifications.Consumer {
	return &notifications.Consumer{
		Handlers: map[string]notifications.ResourceNotificationHandler{
			"/v1/service_brokers": &handlers.BrokerResourceNotificationsHandler{
				BrokerClient: r.PlatformClient.Broker(),
				ProxyPrefix:  r.Options.BrokerPrefix,
				ProxyPath:    r.ProxyPath,
			},
			"/v1/visibilities": &handlers.VisibilityResourceNotificationsHandler{
				VisibilityClient: r.PlatformClient.Visibility(),
				ProxyPrefix:      r.Options.BrokerPrefix,
			},
		},
	}
}

// resync reconciles the state of the proxy brokers and visibilities at the platform
// with the brokers provided by the Service Manager
func (r *Reconciler) resync() {
	resyncContext, taskID, err := r.createResyncContext()
	if err != nil {
		log.C(r.GlobalContext).WithError(err).Error("could not create resync job context")
		return
	}
	log.C(r.GlobalContext).Infof("STARTING resync job %s...", taskID)

	job := &resyncJob{
		Reconciler:    r,
		resyncContext: resyncContext,
		stats:         make(map[string]interface{}),
	}
	job.processBrokers()
	job.processVisibilities()

	log.C(r.GlobalContext).Infof("FINISHED resync job %s", taskID)
}

func (r *Reconciler) createResyncContext() (context.Context, string, error) {
	correlationID, err := uuid.NewV4()
	if err != nil {
		return nil, "", errors.Wrap(err, "could not generate correlationID")
	}
	entry := log.C(r.GlobalContext).WithField(log.FieldCorrelationID, correlationID.String())
	return log.ContextWithLogger(r.GlobalContext, entry), correlationID.String(), nil
}

func (r *resyncJob) stat(key string) interface{} {
	result, found := r.stats[key]
	if !found {
		log.C(r.resyncContext).Infof("No %s found in cache", key)
		return nil
	}

	log.C(r.resyncContext).Infof("Picked up %s from cache", key)

	return result
}
