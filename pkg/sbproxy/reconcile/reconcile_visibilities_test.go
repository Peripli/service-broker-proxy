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
	"encoding/json"
	"sync"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/platform/platformfakes"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-broker-proxy/pkg/sm/smfakes"
	"github.com/Peripli/service-manager/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	cache "github.com/patrickmn/go-cache"
)

var _ = Describe("Reconcile visibilities", func() {
	const fakeAppHost = "https://smproxy.com"

	var (
		fakeSMClient *smfakes.FakeClient

		fakePlatformClient         *platformfakes.FakeClient
		fakePlatformCatalogFetcher *platformfakes.FakeCatalogFetcher
		fakePlatformBrokerClient   *platformfakes.FakeBrokerClient
		fakeVisibilityClient       *platformfakes.FakeVisibilityClient

		visibilityCache *cache.Cache
		running         bool

		waitGroup *sync.WaitGroup

		reconciliationTask *ReconciliationTask

		smbroker1 sm.Broker
		smbroker2 sm.Broker

		platformbroker1        platform.ServiceBroker
		platformbroker2        platform.ServiceBroker
		platformbrokerNonProxy platform.ServiceBroker
	)

	stubGetSMPlans := func() ([]*types.ServicePlan, error) {
		return []*types.ServicePlan{
			smbroker1.ServiceOfferings[0].Plans[0],
			smbroker1.ServiceOfferings[0].Plans[1],
		}, nil
	}

	stubCreateBrokerToSucceed := func(ctx context.Context, r *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {
		return &platform.ServiceBroker{
			GUID:      r.Name,
			Name:      r.Name,
			BrokerURL: r.BrokerURL,
		}, nil
	}

	stubPlatformOpsToSucceed := func() {
		fakePlatformBrokerClient.CreateBrokerStub = stubCreateBrokerToSucceed
		fakePlatformBrokerClient.DeleteBrokerReturns(nil)
		fakePlatformCatalogFetcher.FetchReturns(nil)
	}

	checkAccessArguments := func(data json.RawMessage, servicePlanGUID string, visibilities []*platform.ServiceVisibilityEntity) {
		// Expect(servicePlanGUID).To(Equal(visibility.CatalogPlanID))
		var labels map[string]string
		err := json.Unmarshal(data, &labels)
		Expect(err).To(Not(HaveOccurred()))
		if labels == nil {
			labels = map[string]string{}
		}
		// Expect(labels).To(Equal(visibility.Labels))
		visibility := &platform.ServiceVisibilityEntity{
			Public:        len(labels) == 0,
			CatalogPlanID: servicePlanGUID,
			Labels:        labels,
		}
		Expect(visibilities).To(ContainElement(visibility))
	}

	BeforeEach(func() {
		fakeSMClient = &smfakes.FakeClient{}
		fakePlatformClient = &platformfakes.FakeClient{}
		fakePlatformBrokerClient = &platformfakes.FakeBrokerClient{}
		fakePlatformCatalogFetcher = &platformfakes.FakeCatalogFetcher{}
		fakeVisibilityClient = &platformfakes.FakeVisibilityClient{}

		visibilityCache = cache.New(5*time.Minute, 10*time.Minute)
		waitGroup = &sync.WaitGroup{}

		fakePlatformClient.BrokerReturns(fakePlatformBrokerClient)
		fakePlatformClient.VisibilityReturns(fakeVisibilityClient)
		fakePlatformClient.CatalogFetcherReturns(fakePlatformCatalogFetcher)

		reconciliationTask = NewTask(
			context.TODO(), DefaultSettings(), waitGroup, fakePlatformClient, fakeSMClient,
			fakeAppHost, visibilityCache, &running)

		smbroker1 = sm.Broker{
			ID:        "smBrokerID1",
			BrokerURL: "https://smBroker1.com",
			ServiceOfferings: []types.ServiceOffering{
				{
					ID:                  "smBroker1ServiceID1",
					Name:                "smBroker1Service1",
					Description:         "description",
					Bindable:            true,
					BindingsRetrievable: true,
					Plans: []*types.ServicePlan{
						{
							ID:          "smBroker1ServiceID1PlanID1",
							CatalogID:   "smBroker1ServiceID1PlanID1",
							Name:        "smBroker1Service1Plan1",
							Description: "description",
						},
						{
							ID:          "smBroker1ServiceID1PlanID2",
							CatalogID:   "smBroker1ServiceID1PlanID2",
							Name:        "smBroker1Service1Plan2",
							Description: "description",
						},
					},
				},
			},
		}

		smbroker2 = sm.Broker{
			ID:        "smBrokerID2",
			BrokerURL: "https://smBroker2.com",
			ServiceOfferings: []types.ServiceOffering{
				{
					ID:                  "smBroker2ServiceID1",
					Name:                "smBroker2Service1",
					Description:         "description",
					Bindable:            true,
					BindingsRetrievable: true,
					Plans: []*types.ServicePlan{
						{
							ID:          "smBroker2ServiceID1PlanID1",
							CatalogID:   "smBroker2ServiceID1PlanID1",
							Name:        "smBroker2Service1Plan1",
							Description: "description",
						},
						{
							ID:          "smBroker2ServiceID1PlanID2",
							CatalogID:   "smBroker2ServiceID1PlanID2",
							Name:        "smBroker2Service1Plan2",
							Description: "description",
						},
					},
				},
			},
		}

		platformbroker1 = platform.ServiceBroker{
			GUID:      "platformBrokerID1",
			Name:      ProxyBrokerPrefix + "smBrokerID1",
			BrokerURL: fakeAppHost + "/" + smbroker1.ID,
		}

		platformbroker2 = platform.ServiceBroker{
			GUID:      "platformBrokerID2",
			Name:      ProxyBrokerPrefix + "smBrokerID2",
			BrokerURL: fakeAppHost + "/" + smbroker2.ID,
		}

		platformbrokerNonProxy = platform.ServiceBroker{
			GUID:      "platformBrokerID3",
			Name:      "platformBroker3",
			BrokerURL: "https://platformBroker3.com",
		}
	})

	type expectations struct {
		enablePlanVisibilityCalledFor  []*platform.ServiceVisibilityEntity
		disablePlanVisibilityCalledFor []*platform.ServiceVisibilityEntity
	}

	type testCase struct {
		stubs func()

		platformVisibilities    func() ([]*platform.ServiceVisibilityEntity, error)
		smVisibilities          func() ([]*types.Visibility, error)
		smPlans                 func() ([]*types.ServicePlan, error)
		convertedSMVisibilities func() []*platform.ServiceVisibilityEntity

		expectations func() expectations
	}

	entries := []TableEntry{
		Entry("When no visibilities are present in platform and SM - should not enable access for plan", testCase{
			platformVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return []*platform.ServiceVisibilityEntity{}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{}, nil
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{}
			},
		}),

		Entry("When no visibilities are present in platform and there are some in SM - should reconcile", testCase{
			platformVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return []*platform.ServiceVisibilityEntity{}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					&types.Visibility{
						PlatformID:    "platformID",
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels: []*types.VisibilityLabel{
							&types.VisibilityLabel{
								Label: types.Label{
									Key:   "key",
									Value: []string{"value0", "value1"},
								},
							},
						},
					},
				}, nil
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{
					enablePlanVisibilityCalledFor: []*platform.ServiceVisibilityEntity{
						&platform.ServiceVisibilityEntity{
							CatalogPlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:        map[string]string{"key": "value0"},
						},
						&platform.ServiceVisibilityEntity{
							CatalogPlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:        map[string]string{"key": "value1"},
						},
					},
					disablePlanVisibilityCalledFor: []*platform.ServiceVisibilityEntity{},
				}
			},
		}),

		Entry("When visibilities in platform and in SM are the same - should do nothing", testCase{
			platformVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return []*platform.ServiceVisibilityEntity{
					&platform.ServiceVisibilityEntity{
						CatalogPlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels:        map[string]string{"key": "value0"},
					},
					&platform.ServiceVisibilityEntity{
						CatalogPlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels:        map[string]string{"key": "value1"},
					},
				}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					&types.Visibility{
						PlatformID:    "platformID",
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels: []*types.VisibilityLabel{
							&types.VisibilityLabel{
								Label: types.Label{
									Key:   "key",
									Value: []string{"value0", "value1"},
								},
							},
						},
					},
				}, nil
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{
					enablePlanVisibilityCalledFor:  []*platform.ServiceVisibilityEntity{},
					disablePlanVisibilityCalledFor: []*platform.ServiceVisibilityEntity{},
				}
			},
		}),

		Entry("When visibilities in platform and in SM are not the same - should reconcile", testCase{
			platformVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return []*platform.ServiceVisibilityEntity{
					&platform.ServiceVisibilityEntity{
						CatalogPlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels:        map[string]string{"key": "value2"},
					},
					&platform.ServiceVisibilityEntity{
						CatalogPlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels:        map[string]string{"key": "value3"},
					},
				}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					&types.Visibility{
						PlatformID:    "platformID",
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
						Labels: []*types.VisibilityLabel{
							&types.VisibilityLabel{
								Label: types.Label{
									Key:   "key",
									Value: []string{"value0", "value1"},
								},
							},
						},
					},
				}, nil
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{
					enablePlanVisibilityCalledFor: []*platform.ServiceVisibilityEntity{
						&platform.ServiceVisibilityEntity{
							CatalogPlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:        map[string]string{"key": "value0"},
						},
						&platform.ServiceVisibilityEntity{
							CatalogPlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:        map[string]string{"key": "value1"},
						},
					},
					disablePlanVisibilityCalledFor: []*platform.ServiceVisibilityEntity{
						&platform.ServiceVisibilityEntity{
							CatalogPlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:        map[string]string{"key": "value2"},
						},
						&platform.ServiceVisibilityEntity{
							CatalogPlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:        map[string]string{"key": "value3"},
						},
					},
				}
			},
		}),

		Entry("When visibilities in platform and in SM are both public - should reconcile", testCase{
			platformVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return []*platform.ServiceVisibilityEntity{
					&platform.ServiceVisibilityEntity{
						Public:        true,
						CatalogPlanID: smbroker1.ServiceOfferings[0].Plans[1].CatalogID,
					},
				}, nil
			},
			smVisibilities: func() ([]*types.Visibility, error) {
				return []*types.Visibility{
					&types.Visibility{
						ServicePlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
					},
				}, nil
			},
			smPlans: stubGetSMPlans,
			expectations: func() expectations {
				return expectations{
					enablePlanVisibilityCalledFor: []*platform.ServiceVisibilityEntity{
						&platform.ServiceVisibilityEntity{
							Public:        true,
							CatalogPlanID: smbroker1.ServiceOfferings[0].Plans[0].CatalogID,
							Labels:        map[string]string{},
						},
					},
					disablePlanVisibilityCalledFor: []*platform.ServiceVisibilityEntity{
						&platform.ServiceVisibilityEntity{
							Public:        true,
							CatalogPlanID: smbroker1.ServiceOfferings[0].Plans[1].CatalogID,
							Labels:        map[string]string{},
						},
					},
				}
			},
		}),
	}

	DescribeTable("Run", func(t testCase) {
		fakeSMClient.GetBrokersReturns([]sm.Broker{
			smbroker1,
			smbroker2,
		}, nil)
		fakePlatformBrokerClient.GetBrokersReturns([]platform.ServiceBroker{
			platformbroker1,
			platformbroker2,
			platformbrokerNonProxy,
		}, nil)

		fakeSMClient.GetVisibilitiesReturns(t.smVisibilities())
		fakeSMClient.GetPlansReturns(t.smPlans())

		fakeVisibilityClient.GetVisibilitiesByPlansReturns(t.platformVisibilities())

		fakeVisibilityClient.VisibilityScopeLabelKeyReturns("key")

		stubPlatformOpsToSucceed()

		if t.stubs != nil {
			t.stubs()
		}

		reconciliationTask.Run()

		Expect(fakeSMClient.GetBrokersCallCount()).To(Equal(1))
		Expect(fakePlatformBrokerClient.GetBrokersCallCount()).To(Equal(1))

		expected := t.expectations()

		Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(len(expected.enablePlanVisibilityCalledFor)))

		for index := range expected.enablePlanVisibilityCalledFor {
			_, data, servicePlanGUID := fakeVisibilityClient.EnableAccessForPlanArgsForCall(index)
			checkAccessArguments(data, servicePlanGUID, expected.enablePlanVisibilityCalledFor)
		}

		Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(len(expected.disablePlanVisibilityCalledFor)))

		for index := range expected.disablePlanVisibilityCalledFor {
			_, data, servicePlanGUID := fakeVisibilityClient.DisableAccessForPlanArgsForCall(index)
			checkAccessArguments(data, servicePlanGUID, expected.disablePlanVisibilityCalledFor)
		}

	}, entries...)

})
