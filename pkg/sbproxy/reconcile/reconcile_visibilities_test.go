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
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/platform/platformfakes"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/reconcile"
	"github.com/Peripli/service-broker-proxy/pkg/sm/smfakes"
	"github.com/Peripli/service-manager/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("Reconcile visibilities", func() {
	const (
		maxParallelRequests = 10
		scopeKey            = "key"
	)

	var (
		fakeSMClient                   *smfakes.FakeClient
		fakePlatformClient             *platformfakes.FakeClient
		fakePlatformCatalogFetcher     *platformfakes.FakeCatalogFetcher
		fakePlatformBrokerClient       *platformfakes.FakeBrokerClient
		fakeVisibilityClient           *platformfakes.FakeVisibilityClient
		fakeBrokerPlatformNameProvider *platformfakes.FakeBrokerPlatformNameProvider

		reconciler *reconcile.Reconciler

		smBrokers []*types.ServiceBroker

		platformBrokers        []*platform.ServiceBroker
		platformBrokerNonProxy *platform.ServiceBroker

		parallelRequestsCounter      int
		maxParallelRequestsCounter   int
		parallelRequestsCounterMutex sync.Mutex

		brokerNameInNextFunc string
	)

	generatePlansFor := func(service *types.ServiceOffering, count int) {
		for i := 0; i < count; i++ {
			plan := &types.ServicePlan{
				Base: types.Base{
					ID: fmt.Sprintf("%splanID%d", service.ID, i),
				},
				CatalogID:         fmt.Sprintf("%splanCatalogID%d", service.ID, i),
				Name:              fmt.Sprintf("%splanName%d", service.ID, i),
				Description:       "description",
				ServiceOfferingID: service.ID,
			}

			service.Plans = append(service.Plans, plan)
		}
	}

	generateServicesFor := func(broker *types.ServiceBroker, servicesPerBroker, plansPerService int) {
		for i := 0; i < servicesPerBroker; i++ {
			service := &types.ServiceOffering{
				Base: types.Base{
					ID: fmt.Sprintf("%sserviceID%d", broker.ID, i),
				},
				Name:                fmt.Sprintf("%sserviceName%d", broker.ID, i),
				CatalogID:           fmt.Sprintf("%sserviceCatalogID%d", broker.ID, i),
				Description:         "description",
				Bindable:            true,
				BindingsRetrievable: true,
				BrokerID:            broker.ID,
			}

			generatePlansFor(service, plansPerService)

			broker.Services = append(broker.Services, service)
		}
	}

	generateSMBrokers := func(brokers, servicesPerBroker, plansPerService int) []*types.ServiceBroker {
		result := make([]*types.ServiceBroker, 0, brokers)
		for i := 0; i < brokers; i++ {
			broker := &types.ServiceBroker{
				Base: types.Base{
					ID: fmt.Sprintf("smBrokerID%d", i),
				},
				Name:      fmt.Sprintf("smBrokerName%d", i),
				BrokerURL: fmt.Sprintf("http://smbroker%d", i),
			}

			generateServicesFor(broker, servicesPerBroker, plansPerService)
			result = append(result, broker)
		}

		return result
	}

	generatePlatformBrokersFor := func(brokers []*types.ServiceBroker, platformBrokerNameFunc func(name string) string) []*platform.ServiceBroker {
		result := make([]*platform.ServiceBroker, 0, len(brokers))
		var name string
		for _, broker := range brokers {
			name = broker.Name
			if platformBrokerNameFunc != nil {
				name = platformBrokerNameFunc(name)
			}
			result = append(result, &platform.ServiceBroker{
				GUID:      "platformBrokerID1",
				Name:      brokerProxyName(name, broker.ID),
				BrokerURL: fakeSMAppHost + "/" + broker.ID,
			})
		}

		return result
	}

	generatePlatformVisibilitiesFor := func(broker *types.ServiceBroker, plan *types.ServicePlan, count int) []*platform.Visibility {
		result := make([]*platform.Visibility, 0, count)
		for i := 0; i < count; i++ {
			result = append(result, &platform.Visibility{
				Public:             false,
				CatalogPlanID:      plan.CatalogID,
				PlatformBrokerName: brokerProxyName(broker.Name, broker.ID),
				Labels: map[string]string{
					scopeKey: fmt.Sprintf("value%d", i),
				},
			})
		}
		return result
	}

	generateSMVisibilityFor := func(plan *types.ServicePlan, count int) *types.Visibility {
		labelValues := make([]string, 0, count)
		for i := 0; i < count; i++ {
			labelValues = append(labelValues, fmt.Sprintf("value%d", i))
		}
		return &types.Visibility{
			Base: types.Base{
				Labels: types.Labels{
					scopeKey: labelValues,
				},
			},
			PlatformID:    "platformID",
			ServicePlanID: smBrokers[0].Services[0].Plans[0].ID,
		}
	}

	flattenLabelsMap := func(labels types.Labels) []map[string]string {
		m := make([]map[string]string, len(labels), len(labels))
		for k, values := range labels {
			for i, value := range values {
				if m[i] == nil {
					m[i] = make(map[string]string)
				}
				m[i][k] = value
			}
		}

		return m
	}

	checkAccessArguments := func(data types.Labels, servicePlanGUID, platformBrokerName string, visibilities []*platform.Visibility) {
		maps := flattenLabelsMap(data)
		if len(maps) == 0 {
			visibility := &platform.Visibility{
				Public:             true,
				CatalogPlanID:      servicePlanGUID,
				Labels:             map[string]string{},
				PlatformBrokerName: platformBrokerName,
			}
			Expect(visibilities).To(ContainElement(visibility))
		} else {
			for _, m := range maps {
				visibility := &platform.Visibility{
					Public:             false,
					CatalogPlanID:      servicePlanGUID,
					Labels:             m,
					PlatformBrokerName: platformBrokerName,
				}
				Expect(visibilities).To(ContainElement(visibility))
			}
		}
	}

	countMaxParallelRequests := func(workPeriod time.Duration) {
		parallelRequestsCounterMutex.Lock()
		parallelRequestsCounter++
		if parallelRequestsCounter > maxParallelRequestsCounter {
			maxParallelRequestsCounter = parallelRequestsCounter
		}
		parallelRequestsCounterMutex.Unlock()

		<-time.After(workPeriod)

		parallelRequestsCounterMutex.Lock()
		parallelRequestsCounter--
		parallelRequestsCounterMutex.Unlock()
	}

	mockGetPlatformBrokerName := func(name string) string {
		return strings.ToLower(name)
	}

	BeforeEach(func() {
		fakeSMClient = &smfakes.FakeClient{}
		fakePlatformClient = &platformfakes.FakeClient{}
		fakePlatformBrokerClient = &platformfakes.FakeBrokerClient{}
		fakePlatformCatalogFetcher = &platformfakes.FakeCatalogFetcher{}
		fakeVisibilityClient = &platformfakes.FakeVisibilityClient{}
		fakeBrokerPlatformNameProvider = &platformfakes.FakeBrokerPlatformNameProvider{}

		fakePlatformClient.BrokerReturns(fakePlatformBrokerClient)
		fakePlatformClient.VisibilityReturns(fakeVisibilityClient)
		fakePlatformClient.CatalogFetcherReturns(fakePlatformCatalogFetcher)

		settings := reconcile.DefaultSettings()
		settings.MaxParallelRequests = maxParallelRequests
		reconciler = &reconcile.Reconciler{
			Resyncer: reconcile.NewResyncer(settings, fakePlatformClient, fakeSMClient, defaultSMSettings(), fakeSMAppHost, fakeProxyPathPattern),
		}

		smBrokers = generateSMBrokers(2, 4, 100)
		platformBrokers = generatePlatformBrokersFor(smBrokers, nil)

		platformBrokerNonProxy = &platform.ServiceBroker{
			GUID:      "platformBrokerID3",
			Name:      "platformBroker3",
			BrokerURL: "https://platformBroker3.com",
		}

		fakePlatformBrokerClient.CreateBrokerCalls(func(ctx context.Context, r *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {
			return &platform.ServiceBroker{
				GUID:      r.Name,
				Name:      r.Name,
				BrokerURL: r.BrokerURL,
			}, nil
		})

		maxParallelRequestsCounter = 0
		parallelRequestsCounter = 0
		brokerNameInNextFunc = ""
	})

	type testCase struct {
		stubs func()

		prepareClientWithPlatformNameProvider func()

		platformVisibilities func() []*platform.Visibility
		smVisibilities       func() []*types.Visibility

		enablePlanVisibilityCalledFor  func() []*platform.Visibility
		disablePlanVisibilityCalledFor func() []*platform.Visibility
		expectations                   func()
	}

	entries := []TableEntry{
		Entry("When no visibilities are present in platform and SM - should not enable access for plan", testCase{
			platformVisibilities: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
			smVisibilities: func() []*types.Visibility {
				return []*types.Visibility{}
			},
			enablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
			disablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
		}),

		Entry("When no visibilities are present in platform and there are some in SM - should reconcile", testCase{
			platformVisibilities: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
			smVisibilities: func() []*types.Visibility {
				return []*types.Visibility{
					{
						Base: types.Base{
							Labels: types.Labels{
								scopeKey: []string{"value0", "value1"},
							},
						},
						PlatformID:    "platformID",
						ServicePlanID: smBrokers[0].Services[0].Plans[0].ID,
					},
				}
			},
			enablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{
					{
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
						CatalogPlanID:      smBrokers[0].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{scopeKey: "value0"},
					},
					{
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
						CatalogPlanID:      smBrokers[0].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{scopeKey: "value1"},
					},
				}
			},
			disablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
		}),

		Entry("When visibilities in platform and in SM are the same - should do nothing", testCase{
			platformVisibilities: func() []*platform.Visibility {
				return []*platform.Visibility{
					{
						CatalogPlanID:      smBrokers[0].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{scopeKey: "value0"},
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
					{
						CatalogPlanID:      smBrokers[0].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{scopeKey: "value1"},
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
				}
			},
			smVisibilities: func() []*types.Visibility {
				return []*types.Visibility{
					{
						Base: types.Base{
							Labels: types.Labels{
								scopeKey: []string{"value0", "value1"},
							},
						},
						PlatformID:    "platformID",
						ServicePlanID: smBrokers[0].Services[0].Plans[0].ID,
					},
				}
			},
			enablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
			disablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
		}),

		Entry("When VisibilityBrokerChunkSize>0 && NumberOfBrokers<VisibilityBrokerChunkSize && visibilities in platform and in SM are not the same - should reconcile all brokers visibilities", testCase{
			platformVisibilities: func() []*platform.Visibility {
				return []*platform.Visibility{
					{
						CatalogPlanID:      smBrokers[0].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{scopeKey: "value2"},
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
				}
			},
			smVisibilities: func() []*types.Visibility {
				return []*types.Visibility{
					{
						Base: types.Base{
							Labels: types.Labels{
								scopeKey: []string{"value0", "value1"},
							},
						},
						PlatformID:    "platformID",
						ServicePlanID: smBrokers[0].Services[0].Plans[0].ID,
					},
					{
						Base: types.Base{
							Labels: types.Labels{
								scopeKey: []string{"value2"},
							},
						},
						PlatformID:    "platformID",
						ServicePlanID: smBrokers[1].Services[0].Plans[0].ID,
					},
					{
						Base: types.Base{
							Labels: types.Labels{
								scopeKey: []string{"value3"},
							},
						},
						PlatformID:    "platformID",
						ServicePlanID: smBrokers[2].Services[0].Plans[0].ID,
					},
				}
			},
			enablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{
					{
						CatalogPlanID:      smBrokers[0].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{scopeKey: "value0"},
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
					{
						CatalogPlanID:      smBrokers[0].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{scopeKey: "value1"},
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
					{
						CatalogPlanID:      smBrokers[1].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{scopeKey: "value2"},
						PlatformBrokerName: brokerProxyName(smBrokers[1].Name, smBrokers[1].ID),
					},
					{
						CatalogPlanID:      smBrokers[2].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{scopeKey: "value3"},
						PlatformBrokerName: brokerProxyName(smBrokers[2].Name, smBrokers[2].ID),
					},
				}
			},
			disablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{
					{
						CatalogPlanID:      smBrokers[0].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{scopeKey: "value2"},
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
				}
			},
			prepareClientWithPlatformNameProvider: func() {
				smBrokers = generateSMBrokers(3, 4, 100)
				platformBrokers = generatePlatformBrokersFor(smBrokers, nil)
				settings := reconcile.DefaultSettings()
				settings.MaxParallelRequests = maxParallelRequests
				settings.VisibilityBrokerChunkSize = 3
				reconciler = &reconcile.Reconciler{
					Resyncer: reconcile.NewResyncer(settings, fakePlatformClient, fakeSMClient, defaultSMSettings(), fakeSMAppHost, fakeProxyPathPattern),
				}
			},
		}),

		Entry("When VisibilityBrokerChunkSize>0 && NumberOfBrokers>VisibilityBrokerChunkSize && visibilities in platform and in SM are not the same - should reconcile all brokers visibilities", testCase{
			platformVisibilities: func() []*platform.Visibility {
				return []*platform.Visibility{
					{
						CatalogPlanID:      smBrokers[0].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{scopeKey: "value2"},
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
				}
			},
			smVisibilities: func() []*types.Visibility {
				return []*types.Visibility{
					{
						Base: types.Base{
							Labels: types.Labels{
								scopeKey: []string{"value0", "value1"},
							},
						},
						PlatformID:    "platformID",
						ServicePlanID: smBrokers[0].Services[0].Plans[0].ID,
					},
					{
						Base: types.Base{
							Labels: types.Labels{
								scopeKey: []string{"value2"},
							},
						},
						PlatformID:    "platformID",
						ServicePlanID: smBrokers[1].Services[0].Plans[0].ID,
					},
					{
						Base: types.Base{
							Labels: types.Labels{
								scopeKey: []string{"value3"},
							},
						},
						PlatformID:    "platformID",
						ServicePlanID: smBrokers[2].Services[0].Plans[0].ID,
					},
				}
			},
			enablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{
					{
						CatalogPlanID:      smBrokers[0].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{scopeKey: "value0"},
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
					{
						CatalogPlanID:      smBrokers[0].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{scopeKey: "value1"},
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
					{
						CatalogPlanID:      smBrokers[1].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{scopeKey: "value2"},
						PlatformBrokerName: brokerProxyName(smBrokers[1].Name, smBrokers[1].ID),
					},
					{
						CatalogPlanID:      smBrokers[2].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{scopeKey: "value3"},
						PlatformBrokerName: brokerProxyName(smBrokers[2].Name, smBrokers[2].ID),
					},
				}
			},
			disablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{
					{
						CatalogPlanID:      smBrokers[0].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{scopeKey: "value2"},
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
				}
			},
			prepareClientWithPlatformNameProvider: func() {
				smBrokers = generateSMBrokers(3, 4, 100)
				platformBrokers = generatePlatformBrokersFor(smBrokers, nil)
				settings := reconcile.DefaultSettings()
				settings.MaxParallelRequests = maxParallelRequests
				settings.VisibilityBrokerChunkSize = 2
				reconciler = &reconcile.Reconciler{
					Resyncer: reconcile.NewResyncer(settings, fakePlatformClient, fakeSMClient, defaultSMSettings(), fakeSMAppHost, fakeProxyPathPattern),
				}
			},
		}),

		Entry("When enable visibility returns error - should continue with reconciliation", testCase{
			stubs: func() {
				fakeVisibilityClient.EnableAccessForPlanReturnsOnCall(0, errors.New("Expected"))
			},
			platformVisibilities: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
			smVisibilities: func() []*types.Visibility {
				return []*types.Visibility{
					{
						Base: types.Base{
							Labels: types.Labels{
								scopeKey: []string{"value0"},
							},
						},
						PlatformID:    "platformID",
						ServicePlanID: smBrokers[0].Services[0].Plans[0].ID,
					},
					{
						Base: types.Base{
							Labels: types.Labels{
								scopeKey: []string{"value1"},
							},
						},
						PlatformID:    "platformID",
						ServicePlanID: smBrokers[0].Services[0].Plans[1].ID,
					},
				}
			},
			enablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{
					{
						CatalogPlanID:      smBrokers[0].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{scopeKey: "value0"},
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
					{
						CatalogPlanID:      smBrokers[0].Services[0].Plans[1].CatalogID,
						Labels:             map[string]string{scopeKey: "value1"},
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
				}
			},
		}),

		Entry("When disable visibility returns error - should continue with reconciliation", testCase{
			stubs: func() {
				fakeVisibilityClient.DisableAccessForPlanReturnsOnCall(0, errors.New("Expected"))
			},
			platformVisibilities: func() []*platform.Visibility {
				return []*platform.Visibility{
					{
						CatalogPlanID:      smBrokers[0].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{scopeKey: "value0"},
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
					{
						CatalogPlanID:      smBrokers[0].Services[0].Plans[1].CatalogID,
						Labels:             map[string]string{scopeKey: "value1"},
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
				}
			},
			smVisibilities: func() []*types.Visibility {
				return []*types.Visibility{}
			},
			disablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{
					{
						CatalogPlanID:      smBrokers[0].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{scopeKey: "value0"},
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
					{
						CatalogPlanID:      smBrokers[0].Services[0].Plans[1].CatalogID,
						Labels:             map[string]string{scopeKey: "value1"},
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
				}
			},
		}),

		Entry("When visibility from SM doesn't have scope label and scope is enabled - should enable visibility", testCase{
			platformVisibilities: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
			smVisibilities: func() []*types.Visibility {
				return []*types.Visibility{
					{
						Base: types.Base{
							Labels: types.Labels{
								"some other key": []string{"some value"},
							},
						},
						PlatformID:    "platformID",
						ServicePlanID: smBrokers[0].Services[0].Plans[0].ID,
					},
				}
			},
			enablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{
					{
						Public:             true,
						CatalogPlanID:      smBrokers[0].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{},
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
				}
			},
			disablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
		}),

		Entry("When visibility from SM doesn't have scope label and scope is disabled - should enable visibility", testCase{
			stubs: func() {
				fakeVisibilityClient.VisibilityScopeLabelKeyReturns("")
			},
			platformVisibilities: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
			smVisibilities: func() []*types.Visibility {
				return []*types.Visibility{
					{
						Base: types.Base{
							Labels: types.Labels{
								"some key": []string{"some value"},
							},
						},
						PlatformID:    "platformID",
						ServicePlanID: smBrokers[0].Services[0].Plans[0].ID,
					},
				}
			},
			enablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{
					{
						Public:             true,
						CatalogPlanID:      smBrokers[0].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{},
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
				}
			},
		}),

		Entry("When visibilities in platform and in SM are both public - should reconcile", testCase{
			platformVisibilities: func() []*platform.Visibility {
				return []*platform.Visibility{
					{
						Public:             true,
						CatalogPlanID:      smBrokers[0].Services[0].Plans[1].CatalogID,
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
				}
			},
			smVisibilities: func() []*types.Visibility {
				return []*types.Visibility{
					{
						ServicePlanID: smBrokers[0].Services[0].Plans[0].ID,
					},
				}
			},
			enablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{
					{
						Public:             true,
						CatalogPlanID:      smBrokers[0].Services[0].Plans[0].CatalogID,
						Labels:             map[string]string{},
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
				}
			},
			disablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{
					{
						Public:             true,
						CatalogPlanID:      smBrokers[0].Services[0].Plans[1].CatalogID,
						Labels:             map[string]string{},
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
				}
			},
		}),

		Entry("When services from SM could not be fetched - should not reconcile", testCase{
			stubs: func() {
				fakeSMClient.GetServiceOfferingsReturns(nil, fmt.Errorf("error"))
			},
			platformVisibilities: func() []*platform.Visibility {
				return []*platform.Visibility{
					{
						Public:             true,
						CatalogPlanID:      smBrokers[0].Services[0].Plans[1].CatalogID,
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
				}
			},
			smVisibilities: func() []*types.Visibility {
				return []*types.Visibility{
					{
						ServicePlanID: smBrokers[0].Services[0].Plans[0].ID,
					},
				}
			},
			enablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
			disablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
		}),

		Entry("When plans from SM could not be fetched - should not reconcile", testCase{
			stubs: func() {
				fakeSMClient.GetPlansReturns(nil, fmt.Errorf("error"))
			},
			platformVisibilities: func() []*platform.Visibility {
				return []*platform.Visibility{
					{
						Public:             true,
						CatalogPlanID:      smBrokers[0].Services[0].Plans[1].CatalogID,
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
				}
			},
			smVisibilities: func() []*types.Visibility {
				return []*types.Visibility{
					{
						ServicePlanID: smBrokers[0].Services[0].Plans[0].ID,
					},
				}
			},
			enablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
			disablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
		}),

		Entry("When visibilities from platform cannot be fetched - should not reconcile", testCase{
			stubs: func() {
				fakeVisibilityClient.GetVisibilitiesByBrokersCalls(func(ctx context.Context, brokerIDs []string) ([]*platform.Visibility, error) {
					return nil, fmt.Errorf("error")
				})
			},
			platformVisibilities: func() []*platform.Visibility {
				return []*platform.Visibility{
					{
						Public:             true,
						CatalogPlanID:      smBrokers[0].Services[0].Plans[1].CatalogID,
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
				}
			},
			smVisibilities: func() []*types.Visibility {
				return []*types.Visibility{
					{
						ServicePlanID: smBrokers[0].Services[0].Plans[0].ID,
					},
				}
			},
			enablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
			disablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
		}),

		Entry("When visibilities from SM cannot be fetched - no reconcilation is done", testCase{
			stubs: func() {
				fakeSMClient.GetVisibilitiesCalls(func(ctx context.Context) ([]*types.Visibility, error) {
					return nil, fmt.Errorf("error")
				})
			},
			platformVisibilities: func() []*platform.Visibility {
				return []*platform.Visibility{
					{
						Public:             true,
						CatalogPlanID:      smBrokers[0].Services[0].Plans[1].CatalogID,
						PlatformBrokerName: brokerProxyName(smBrokers[0].Name, smBrokers[0].ID),
					},
				}
			},
			smVisibilities: func() []*types.Visibility {
				return []*types.Visibility{}
			},
			enablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
			disablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
		}),

		Entry("When visibilities are reconciled, no more than max parallel request count visibilities are enabled simultaneously", testCase{
			stubs: func() {
				fakeVisibilityClient.EnableAccessForPlanCalls(func(ctx context.Context, request *platform.ModifyPlanAccessRequest) error {
					countMaxParallelRequests(100 * time.Millisecond)
					return nil
				})
			},
			platformVisibilities: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
			smVisibilities: func() []*types.Visibility {
				return append([]*types.Visibility{}, generateSMVisibilityFor(smBrokers[0].Services[0].Plans[0], 100),
					generateSMVisibilityFor(smBrokers[1].Services[0].Plans[0], 100))
			},
			enablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return append(generatePlatformVisibilitiesFor(smBrokers[0], smBrokers[0].Services[0].Plans[0], 100),
					generatePlatformVisibilitiesFor(smBrokers[1], smBrokers[1].Services[0].Plans[0], 100)...)
			},
			disablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
			expectations: func() {
				Expect(maxParallelRequestsCounter).To(Equal(maxParallelRequests))
			},
		}),

		Entry("When visibilities are reconciled, no more than max parallel request count visibilities are disabled simultaneously", testCase{
			stubs: func() {
				fakeVisibilityClient.DisableAccessForPlanCalls(func(ctx context.Context, request *platform.ModifyPlanAccessRequest) error {
					countMaxParallelRequests(100 * time.Millisecond)
					return nil
				})
			},
			platformVisibilities: func() []*platform.Visibility {
				return append(generatePlatformVisibilitiesFor(smBrokers[0], smBrokers[0].Services[0].Plans[0], 100),
					generatePlatformVisibilitiesFor(smBrokers[1], smBrokers[1].Services[0].Plans[0], 100)...)
			},
			smVisibilities: func() []*types.Visibility {
				return []*types.Visibility{}
			},
			enablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return []*platform.Visibility{}
			},
			disablePlanVisibilityCalledFor: func() []*platform.Visibility {
				return append(generatePlatformVisibilitiesFor(smBrokers[0], smBrokers[0].Services[0].Plans[0], 100),
					generatePlatformVisibilitiesFor(smBrokers[1], smBrokers[1].Services[0].Plans[0], 100)...)
			},
			expectations: func() {
				Expect(maxParallelRequestsCounter).To(Equal(maxParallelRequests))
			},
		}),

		Entry("When client contains platform name provider, it changes the broker name accordingly", testCase{
			stubs: func() {
				fakeBrokerPlatformNameProvider.GetBrokerPlatformNameStub = mockGetPlatformBrokerName
				fakeVisibilityClient.GetVisibilitiesByBrokersStub = func(ctx context.Context, names []string) ([]*platform.Visibility, error) {
					brokerNameInNextFunc = names[0]
					return []*platform.Visibility{
						{
							Public:        true,
							CatalogPlanID: smBrokers[0].Services[0].Plans[1].CatalogID,
							// current PlatformBrokerName is assumed to be with proper name according to function
							PlatformBrokerName: brokerProxyName(mockGetPlatformBrokerName(smBrokers[0].Name), smBrokers[0].ID),
						},
					}, nil
				}
			},
			smVisibilities: func() []*types.Visibility {
				return []*types.Visibility{
					{
						ServicePlanID: smBrokers[0].Services[0].Plans[0].ID,
					},
				}
			},
			expectations: func() {
				// brokerNameInNextFunc should match the current broker name in platform
				Expect(brokerNameInNextFunc).To(Equal(brokerProxyName(mockGetPlatformBrokerName(smBrokers[0].Name), smBrokers[0].ID)))
			},

			prepareClientWithPlatformNameProvider: func() {
				smBrokers = generateSMBrokers(1, 1, 2)
				platformBrokers = generatePlatformBrokersFor(smBrokers, mockGetPlatformBrokerName)
				platformClient := struct {
					*platformfakes.FakeClient
					*platformfakes.FakeBrokerPlatformNameProvider
				}{
					FakeClient:                     fakePlatformClient,
					FakeBrokerPlatformNameProvider: fakeBrokerPlatformNameProvider,
				}

				reconciler = &reconcile.Reconciler{
					Resyncer: reconcile.NewResyncer(reconcile.DefaultSettings(), platformClient, fakeSMClient, defaultSMSettings(), fakeSMAppHost, fakeProxyPathPattern),
				}
			},
		}),
	}

	DescribeTable("Resync", func(t testCase) {
		if t.prepareClientWithPlatformNameProvider != nil {
			t.prepareClientWithPlatformNameProvider()
		}

		fakeSMClient.GetVisibilitiesReturns(t.smVisibilities(), nil)
		fakeSMClient.GetBrokersReturns(smBrokers, nil)

		fakeSMClient.GetServiceOfferingsCalls(func(ctx context.Context) ([]*types.ServiceOffering, error) {
			var result []*types.ServiceOffering
			for _, broker := range smBrokers {
				result = append(result, broker.Services...)
			}
			return result, nil
		})

		fakeSMClient.GetPlansCalls(func(ctx context.Context) ([]*types.ServicePlan, error) {
			var result []*types.ServicePlan
			for _, broker := range smBrokers {
				for _, brokerServiceOffering := range broker.Services {
					result = append(result, brokerServiceOffering.Plans...)
				}
			}
			return result, nil
		})

		fakePlatformBrokerClient.GetBrokersReturns(append(platformBrokers, platformBrokerNonProxy), nil)

		fakeVisibilityClient.VisibilityScopeLabelKeyReturns(scopeKey)

		if t.platformVisibilities != nil {
			fakeVisibilityClient.GetVisibilitiesByBrokersReturns(t.platformVisibilities(), nil)
		}

		if t.stubs != nil {
			t.stubs()
		}

		reconciler.Resyncer.Resync(context.TODO())

		invocations := append([]map[string][][]interface{}{}, fakeSMClient.Invocations(), fakePlatformClient.Invocations(), fakePlatformCatalogFetcher.Invocations(), fakePlatformBrokerClient.Invocations(), fakeVisibilityClient.Invocations())
		verifyInvocationsUseSameCorrelationID(invocations)

		Expect(fakeSMClient.GetBrokersCallCount()).To(Equal(1))

		if t.enablePlanVisibilityCalledFor != nil {
			visibilities := t.enablePlanVisibilityCalledFor()
			Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(len(visibilities)))
			for index := range visibilities {
				_, request := fakeVisibilityClient.EnableAccessForPlanArgsForCall(index)
				checkAccessArguments(request.Labels, request.CatalogPlanID, request.BrokerName, visibilities)
			}
		}

		if t.disablePlanVisibilityCalledFor != nil {
			visibilities := t.disablePlanVisibilityCalledFor()
			Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(len(visibilities)))

			for index := range visibilities {
				_, request := fakeVisibilityClient.DisableAccessForPlanArgsForCall(index)
				checkAccessArguments(request.Labels, request.CatalogPlanID, request.BrokerName, visibilities)
			}
		}

		if t.expectations != nil {
			t.expectations()
		}
	}, entries...)
})
