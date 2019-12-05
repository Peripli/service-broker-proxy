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
	"strings"
	"sync"
	"time"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/notifications"
	"github.com/Peripli/service-manager/pkg/log"
)

// Consumer provides functionality for consuming notifications
type Consumer interface {
	Consume(ctx context.Context, notification *types.Notification)
}

// Resyncer provides functionality for triggering a resync on the platform
type Resyncer interface {
	Resync(ctx context.Context)
}

// TimestamppedError contains an error and a timestamp in time.RFC3339Nano format
type TimestamppedError string

func (te TimestamppedError) Error() string {
	return fmt.Sprintf("%v:%s", time.Now().Format(time.RFC3339Nano), string(te))
}

// CompositeError consists of multiple errors and attaches timestamps to them
type CompositeError []error

// Error implements the error interface and returns a string representation of the composite error
func (ce *CompositeError) Error() string {
	errs := make([]string, 0, len(*ce))
	for i := range *ce {
		if (*ce)[i] != nil {
			errs = append(errs, (*ce)[i].Error())
		}
	}

	return fmt.Sprintf("composite error: %v", strings.Join(errs, "; "))
}

// Add allows appending errors
func (ce *CompositeError) Add(e error) {
	*ce = append(*ce, TimestamppedError(e.Error()))
}

// Len returns the number of errors present in the composite error. If the composite error is nil, Len returns 0.
func (ce *CompositeError) Len() int {
	if ce == nil {
		return 0
	}
	return len(*ce)
}

// Reconciler takes care of propagating broker and visibility changes to the platform.
// TODO if the reg credentials are changed (the ones under cf.reg) we need to update the already registered brokers
type Reconciler struct {
	Consumer Consumer
	Resyncer Resyncer
}

// Reconcile listens for notification messages and either consumes the notification or triggers a resync
func (r *Reconciler) Reconcile(ctx context.Context, messages <-chan *notifications.Message, group *sync.WaitGroup) {
	group.Add(1)
	go r.process(ctx, messages, group)
}

// Process resync and notification messages sequentially in one goroutine
// to avoid concurrent changes in the platform
func (r *Reconciler) process(ctx context.Context, messages <-chan *notifications.Message, group *sync.WaitGroup) {
	defer group.Done()
	for {
		select {
		case <-ctx.Done():
			log.C(ctx).Info("Context cancelled. Terminating reconciler.")
			return
		case m, ok := <-messages:
			if !ok {
				log.C(ctx).Info("Messages channel closed. Terminating reconciler.")
				return
			}
			log.C(ctx).Debugf("Reconciler received message %+v", m)
			if m.Resync {
				// discard any pending change notifications as we will do a full resync
				drain(messages)
				r.Resyncer.Resync(ctx)
			} else {
				r.Consumer.Consume(ctx, m.Notification)
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
