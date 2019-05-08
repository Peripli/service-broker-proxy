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
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/notifications"
	"github.com/Peripli/service-manager/pkg/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reconciler", func() {
	Describe("Process", func() {
		var (
			wg         *sync.WaitGroup
			reconciler *Reconciler
			messages   chan *notifications.Message
			ctx        context.Context
			cancel     context.CancelFunc
			fakeExec   *fakeExecutor
		)

		startProcess := func() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer GinkgoRecover()
				reconciler.Process(messages)
			}()
		}

		BeforeEach(func() {
			ctx, cancel = context.WithCancel(context.Background())
			wg = &sync.WaitGroup{}
			fakeExec = &fakeExecutor{}
			reconciler = &Reconciler{
				GlobalContext: ctx,
				exec:          fakeExec,
			}
			messages = make(chan *notifications.Message, 10)
		})

		Context("when the context is canceled", func() {
			It("quits", func(done Done) {
				startProcess()
				cancel()
				wg.Wait()
				close(done)
			}, 0.1)
		})

		Context("when the messages channel is closed", func() {
			It("quits", func(done Done) {
				startProcess()
				close(messages)
				wg.Wait()
				close(done)
			}, 0.1)
		})

		Context("when the messages channel is closed after resync", func() {
			It("quits", func(done Done) {
				messages <- &notifications.Message{Resync: true}
				close(messages)
				startProcess()
				wg.Wait()
				close(done)
			}, 0.1)
		})

		Context("when notifications are sent", func() {
			It("applies them in the same order", func(done Done) {
				ns := []*types.Notification{
					&types.Notification{
						Resource: "/v1/service_brokers",
						Type:     "CREATED",
					},
					&types.Notification{
						Resource: "/v1/service_brokers",
						Type:     "DELETED",
					},
				}
				for _, n := range ns {
					messages <- &notifications.Message{Notification: n}
				}
				close(messages)
				startProcess()
				wg.Wait()
				Expect(fakeExec.notifications).To(Equal(ns))
				close(done)
			})
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
				startProcess()

				time.Sleep(50 * time.Millisecond)
				Expect(fakeExec.getNotifications()).To(Equal([]*types.Notification{nCreated}))
				Expect(fakeExec.getResyncCount()).To(Equal(1))

				messages <- &notifications.Message{Notification: nModified}
				close(messages)
				wg.Wait()
				Expect(fakeExec.getNotifications()).To(Equal([]*types.Notification{nCreated, nModified}))
				close(done)
			})
		})
	})
})

type fakeExecutor struct {
	mutex         sync.Mutex
	notifications []*types.Notification
	resyncCount   int
}

func (f *fakeExecutor) apply(n *types.Notification) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.notifications = append(f.notifications, n)
}

func (f *fakeExecutor) resync() {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.resyncCount++
}

func (f *fakeExecutor) getResyncCount() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return f.resyncCount
}

func (f *fakeExecutor) getNotifications() []*types.Notification {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	notifications := make([]*types.Notification, len(f.notifications))
	copy(notifications, f.notifications)
	return notifications
}
