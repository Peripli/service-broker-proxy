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

package reconcile_test

import (
	"context"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/reconcile"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/utils"
	"sync"

	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/notifications"
	"github.com/Peripli/service-manager/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reconciler", func() {
	Describe("Process", func() {
		const timeout = 1
		var (
			wg         *sync.WaitGroup
			reconciler *reconcile.Reconciler
			messages   chan *notifications.Message
			ctx        context.Context
			cancel     context.CancelFunc
			resyncer   *utils.FakeResyncer
			consumer   *utils.FakeConsumer
		)

		BeforeEach(func() {
			ctx, cancel = context.WithCancel(context.Background())
			wg = &sync.WaitGroup{}
			resyncer = &utils.FakeResyncer{}
			consumer = &utils.FakeConsumer{}

			reconciler = &reconcile.Reconciler{
				Resyncer: resyncer,
				Consumer: consumer,
			}
			messages = make(chan *notifications.Message, 10)
		})

		Context("when the context is canceled", func() {
			It("quits", func(done Done) {
				reconciler.Reconcile(ctx, messages, wg)
				cancel()
				wg.Wait()
				close(done)
			}, timeout)
		})

		Context("when the messages channel is closed", func() {
			It("quits", func(done Done) {
				reconciler.Reconcile(ctx, messages, wg)
				close(messages)
				wg.Wait()
				close(done)
			}, timeout)
		})

		Context("when the messages channel is closed after resync", func() {
			It("quits", func(done Done) {
				messages <- &notifications.Message{Resync: true}
				close(messages)
				reconciler.Reconcile(ctx, messages, wg)
				wg.Wait()
				close(done)
			}, timeout)
		})

		Context("when notifications are sent", func() {
			It("applies them in the same order", func(done Done) {
				ns := []*types.Notification{
					{
						Resource: "/v1/service_brokers",
						Type:     "CREATED",
					},
					{
						Resource: "/v1/service_brokers",
						Type:     "DELETED",
					},
				}
				for _, n := range ns {
					messages <- &notifications.Message{Notification: n}
				}
				close(messages)
				reconciler.Reconcile(ctx, messages, wg)
				wg.Wait()
				Expect(consumer.GetConsumedNotifications()).To(Equal(ns))
				close(done)
			}, timeout)
		})

		Context("when resync is sent", func() {
			It("drops all remaining messages in the queue and processes all new messages", func(done Done) {
				nCreated := &types.Notification{
					Resource: "/v1/service_brokers",
					Type:     "CREATED",
				}
				nDeleted := &types.Notification{
					Resource: "/v1/service_brokers",
					Type:     "DELETED",
				}
				nModified := &types.Notification{
					Resource: "/v1/service_brokers",
					Type:     "MODIFIED",
				}
				messages <- &notifications.Message{Notification: nCreated}
				messages <- &notifications.Message{Resync: true}
				messages <- &notifications.Message{Notification: nDeleted}
				messages <- &notifications.Message{Resync: true}

				Expect(messages).To(HaveLen(4))
				reconciler.Reconcile(ctx, messages, wg)
				Eventually(messages).Should(HaveLen(0)) // drain was called
				Expect(consumer.GetConsumedNotifications()).Should(Equal([]*types.Notification{nCreated}))

				messages <- &notifications.Message{Notification: nModified}
				close(messages)
				wg.Wait()

				Expect(resyncer.GetResyncCount()).Should(Equal(1))
				Expect(consumer.GetConsumedNotifications()).To(Equal([]*types.Notification{nCreated, nModified}))
				close(done)
			}, timeout)
		})
	})
})
