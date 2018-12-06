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

	"github.com/patrickmn/go-cache"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/platform/platformfakes"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-broker-proxy/pkg/sm/smfakes"
	"github.com/Peripli/service-manager/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("ReconcilationTask", func() {
	const fakeAppHost = "https://smproxy.com"

	var (
		fakeSMClient *smfakes.FakeClient

		fakePlatformCatalogFetcher *platformfakes.FakeCatalogFetcher
		fakePlatformServiceAccess  *platformfakes.FakeServiceAccess
		fakePlatformBrokerClient   *platformfakes.FakeClient
		fakeVisibilityKeyMapper    *platformfakes.FakeServiceVisibilityKeyMapper
		fakeServiceVisibility      *platformfakes.FakeServiceVisibility
		fakeSMVisibilityConverter  *platformfakes.FakeSMVisibilityConverter

		visibilityCache *cache.Cache
		running         bool

		waitGroup *sync.WaitGroup

		reconcilationTask *ReconcilationTask

		smbroker1 sm.Broker
		smbroker2 sm.Broker

		platformbroker1        platform.ServiceBroker
		platformbroker2        platform.ServiceBroker
		platformbrokerNonProxy platform.ServiceBroker

		smVisibility1 *sm.Visibility

		platformVisibility1 *platform.ServiceVisibilityEntity
	)

	stubCreateBrokerToSucceed := func(ctx context.Context, r *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {
		return &platform.ServiceBroker{
			GUID:      r.Name,
			Name:      r.Name,
			BrokerURL: r.BrokerURL,
		}, nil
	}

	stubCreateBrokerToReturnError := func(ctx context.Context, r *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {
		return nil, fmt.Errorf("error")
	}

	stubPlatformOpsToSucceed := func() {
		fakePlatformBrokerClient.CreateBrokerStub = stubCreateBrokerToSucceed
		fakePlatformBrokerClient.DeleteBrokerReturns(nil)
		fakePlatformServiceAccess.EnableAccessForServiceReturns(nil)
		fakePlatformCatalogFetcher.FetchReturns(nil)
	}

	BeforeEach(func() {
		fakeSMClient = &smfakes.FakeClient{}
		fakePlatformBrokerClient = &platformfakes.FakeClient{}
		fakePlatformCatalogFetcher = &platformfakes.FakeCatalogFetcher{}
		fakePlatformServiceAccess = &platformfakes.FakeServiceAccess{}
		fakeVisibilityKeyMapper = &platformfakes.FakeServiceVisibilityKeyMapper{}
		fakeServiceVisibility = &platformfakes.FakeServiceVisibility{}
		fakeSMVisibilityConverter = &platformfakes.FakeSMVisibilityConverter{}
		visibilityCache = cache.New(5*time.Minute, 10*time.Minute)

		waitGroup = &sync.WaitGroup{}

		reconcilationTask = NewTask(context.TODO(), DefaultSettings(), waitGroup, struct {
			*platformfakes.FakeCatalogFetcher
			*platformfakes.FakeServiceAccess
			*platformfakes.FakeClient
			*platformfakes.FakeServiceVisibility
			*platformfakes.FakeSMVisibilityConverter
		}{
			FakeCatalogFetcher:        fakePlatformCatalogFetcher,
			FakeServiceAccess:         fakePlatformServiceAccess,
			FakeClient:                fakePlatformBrokerClient,
			FakeServiceVisibility:     fakeServiceVisibility,
			FakeSMVisibilityConverter: fakeSMVisibilityConverter,
		}, fakeSMClient, fakeAppHost, visibilityCache, fakeVisibilityKeyMapper, &running)

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
							CatalogID:   "smBroker1ServiceID1CatalogPlanID1",
							Name:        "smBroker1Service1Plan1",
							Description: "description",
						},
						{
							ID:          "smBroker1ServiceID1PlanID2",
							CatalogID:   "smBroker1ServiceID1CatalogPlanID2",
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
							CatalogID:   "smBroker2ServiceID1CatalogPlanID1",
							Name:        "smBroker2Service1Plan1",
							Description: "description",
						},
						{
							ID:          "smBroker2ServiceID1PlanID2",
							CatalogID:   "smBroker2ServiceID1CatalogPlanID2",
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

		smVisibility1 = &sm.Visibility{
			Visibility: &types.Visibility{
				ID:            "smVisibilityID1",
				PlatformID:    "platformID1",
				ServicePlanID: "smBroker1ServiceID1PlanID1",
			},
			Labels: []*sm.Label{
				&sm.Label{
					Key:    "organization_guid",
					Values: []string{"org1guid"},
				},
			}}

		platformVisibility1 = &platform.ServiceVisibilityEntity{
			CatalogPlanID: "smBroker1ServiceID1CatalogPlanID1",
			Labels: map[string]string{
				"organization_guid": "org1guid",
			},
		}
	})

	type catalog struct {
		ServiceOfferings []types.ServiceOffering
	}
	type expectations struct {
		reconcileCreateCalledFor  []platform.ServiceBroker
		reconcileDeleteCalledFor  []platform.ServiceBroker
		reconcileCatalogCalledFor []platform.ServiceBroker
		reconcileAccessCalledFor  []catalog

		reconcileVisibilityCalledFor       []*platform.ServiceVisibilityEntity
		reconcileDeleteVisibilityCalledFor []*platform.ServiceVisibilityEntity
	}

	type testCase struct {
		stubs           func()
		platformBrokers func() ([]platform.ServiceBroker, error)
		smBrokers       func() ([]sm.Broker, error)

		platformVisibilities  func() ([]*platform.ServiceVisibilityEntity, error)
		smVisibilities        func() ([]*sm.Visibility, error)
		convertedVisibilities func() ([]*platform.ServiceVisibilityEntity, error)

		expectations func() expectations
	}

	platformVisibilitiesEmptyMethod := func() ([]*platform.ServiceVisibilityEntity, error) {
		return nil, nil
	}

	smVisibilitiesEmptyMethod := func() ([]*sm.Visibility, error) {
		return nil, nil
	}

	convertedVisibilitiesEmptyMethod := func() ([]*platform.ServiceVisibilityEntity, error) {
		return nil, nil
	}

	entries := []TableEntry{
		Entry("When fetching brokers from SM fails no reconcilation should be done", testCase{
			stubs: func() {

			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return nil, fmt.Errorf("error fetching brokers")
			},
			platformVisibilities:  platformVisibilitiesEmptyMethod,
			smVisibilities:        smVisibilitiesEmptyMethod,
			convertedVisibilities: convertedVisibilitiesEmptyMethod,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor:  []platform.ServiceBroker{},
					reconcileDeleteCalledFor:  []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{},
					reconcileAccessCalledFor:  []catalog{},
				}
			},
		}),

		Entry("When fetching brokers from platform fails no reconcilation should be done", testCase{
			stubs: func() {

			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return nil, fmt.Errorf("error fetching brokers")
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{}, nil
			},
			platformVisibilities:  platformVisibilitiesEmptyMethod,
			smVisibilities:        smVisibilitiesEmptyMethod,
			convertedVisibilities: convertedVisibilitiesEmptyMethod,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor:  []platform.ServiceBroker{},
					reconcileDeleteCalledFor:  []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{},
					reconcileAccessCalledFor:  []catalog{},
				}
			},
		}),

		Entry("When platform broker op fails reconcilation continues with the next broker", testCase{
			stubs: func() {
				fakePlatformBrokerClient.DeleteBrokerReturns(fmt.Errorf("error"))
				fakePlatformServiceAccess.EnableAccessForServiceReturns(fmt.Errorf("error"))
				fakePlatformCatalogFetcher.FetchReturns(fmt.Errorf("error"))
				fakePlatformBrokerClient.CreateBrokerStub = stubCreateBrokerToReturnError
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{
					platformbroker2,
				}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{
					smbroker1,
				}, nil
			},
			platformVisibilities:  platformVisibilitiesEmptyMethod,
			smVisibilities:        smVisibilitiesEmptyMethod,
			convertedVisibilities: convertedVisibilitiesEmptyMethod,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []platform.ServiceBroker{
						platformbroker1,
					},
					reconcileDeleteCalledFor: []platform.ServiceBroker{
						platformbroker2,
					},
					reconcileCatalogCalledFor: []platform.ServiceBroker{},
					reconcileAccessCalledFor:  []catalog{},
				}
			},
		}),

		Entry("When broker from SM has no catalog reconcilation continues with the next broker", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{
					platformbroker1,
					platformbroker2,
				}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				smbroker1.ServiceOfferings = nil
				return []sm.Broker{
					smbroker1,
					smbroker2,
				}, nil
			},
			platformVisibilities:  platformVisibilitiesEmptyMethod,
			smVisibilities:        smVisibilitiesEmptyMethod,
			convertedVisibilities: convertedVisibilitiesEmptyMethod,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []platform.ServiceBroker{},
					reconcileDeleteCalledFor: []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{
						platformbroker1,
						platformbroker2,
					},
					reconcileAccessCalledFor: []catalog{
						{
							ServiceOfferings: smbroker2.ServiceOfferings,
						},
					},
				}
			},
		}),

		Entry("When broker is in SM and is missing from platform it should be created and access enabled", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{
					smbroker1,
					smbroker2,
				}, nil
			},
			platformVisibilities:  platformVisibilitiesEmptyMethod,
			smVisibilities:        smVisibilitiesEmptyMethod,
			convertedVisibilities: convertedVisibilitiesEmptyMethod,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []platform.ServiceBroker{
						platformbroker1,
						platformbroker2,
					},
					reconcileDeleteCalledFor:  []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{},
					reconcileAccessCalledFor: []catalog{
						{
							ServiceOfferings: smbroker1.ServiceOfferings,
						},
						{
							ServiceOfferings: smbroker2.ServiceOfferings,
						},
					},
				}
			},
		}),

		Entry("When broker is in SM and is also in platform it should be catalog refetched and access enabled", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{
					platformbroker1,
				}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{
					smbroker1,
				}, nil
			},
			platformVisibilities:  platformVisibilitiesEmptyMethod,
			smVisibilities:        smVisibilitiesEmptyMethod,
			convertedVisibilities: convertedVisibilitiesEmptyMethod,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []platform.ServiceBroker{},
					reconcileDeleteCalledFor: []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{
						platformbroker1,
					},
					reconcileAccessCalledFor: []catalog{
						{
							ServiceOfferings: smbroker1.ServiceOfferings,
						},
					},
				}
			},
		}),

		Entry("When broker is missing from SM but is in platform it should be deleted", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{
					platformbroker1,
				}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{}, nil
			},
			platformVisibilities:  platformVisibilitiesEmptyMethod,
			smVisibilities:        smVisibilitiesEmptyMethod,
			convertedVisibilities: convertedVisibilitiesEmptyMethod,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []platform.ServiceBroker{},
					reconcileDeleteCalledFor: []platform.ServiceBroker{
						platformbroker1,
					},
					reconcileCatalogCalledFor: []platform.ServiceBroker{},
					reconcileAccessCalledFor:  []catalog{},
				}
			},
		}),

		Entry("When broker is missing from SM but is in platform that is not represented by the proxy should be ignored", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{
					platformbrokerNonProxy,
				}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{}, nil
			},
			platformVisibilities:  platformVisibilitiesEmptyMethod,
			smVisibilities:        smVisibilitiesEmptyMethod,
			convertedVisibilities: convertedVisibilitiesEmptyMethod,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor:  []platform.ServiceBroker{},
					reconcileDeleteCalledFor:  []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{},
					reconcileAccessCalledFor:  []catalog{},
				}
			},
		}),

		Entry("When visibility is missing in platform, should be reconciled", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{
					platformbroker1,
				}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{
					smbroker1,
				}, nil
			},
			platformVisibilities: platformVisibilitiesEmptyMethod,
			smVisibilities: func() ([]*sm.Visibility, error) {
				return []*sm.Visibility{
					smVisibility1,
				}, nil
			},
			convertedVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return []*platform.ServiceVisibilityEntity{
					platformVisibility1,
				}, nil
			},
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []platform.ServiceBroker{},
					reconcileDeleteCalledFor: []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{
						platformbroker1,
					},
					reconcileAccessCalledFor: []catalog{
						{
							ServiceOfferings: smbroker1.ServiceOfferings,
						},
					},
					reconcileVisibilityCalledFor: []*platform.ServiceVisibilityEntity{
						platformVisibility1,
					},
					reconcileDeleteVisibilityCalledFor: []*platform.ServiceVisibilityEntity{},
				}
			},
		}),

		Entry("When visibility is in platform and service manager, no reconcilation should be done", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{
					platformbroker1,
				}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{
					smbroker1,
				}, nil
			},
			platformVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return []*platform.ServiceVisibilityEntity{
					platformVisibility1,
				}, nil
			},
			smVisibilities: func() ([]*sm.Visibility, error) {
				return []*sm.Visibility{
					smVisibility1,
				}, nil
			},
			convertedVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return []*platform.ServiceVisibilityEntity{
					platformVisibility1,
				}, nil
			},
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []platform.ServiceBroker{},
					reconcileDeleteCalledFor: []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{
						platformbroker1,
					},
					reconcileAccessCalledFor: []catalog{
						{
							ServiceOfferings: smbroker1.ServiceOfferings,
						},
					},
					reconcileVisibilityCalledFor:       []*platform.ServiceVisibilityEntity{},
					reconcileDeleteVisibilityCalledFor: []*platform.ServiceVisibilityEntity{},
				}
			},
		}),

		Entry("When visibility is in platform, but no in Service manager, one should be deleted", testCase{
			stubs: func() {
				stubPlatformOpsToSucceed()
			},
			platformBrokers: func() ([]platform.ServiceBroker, error) {
				return []platform.ServiceBroker{
					platformbroker1,
				}, nil
			},
			smBrokers: func() ([]sm.Broker, error) {
				return []sm.Broker{
					smbroker1,
				}, nil
			},
			platformVisibilities: func() ([]*platform.ServiceVisibilityEntity, error) {
				return []*platform.ServiceVisibilityEntity{
					platformVisibility1,
				}, nil
			},
			smVisibilities:        smVisibilitiesEmptyMethod,
			convertedVisibilities: convertedVisibilitiesEmptyMethod,
			expectations: func() expectations {
				return expectations{
					reconcileCreateCalledFor: []platform.ServiceBroker{},
					reconcileDeleteCalledFor: []platform.ServiceBroker{},
					reconcileCatalogCalledFor: []platform.ServiceBroker{
						platformbroker1,
					},
					reconcileAccessCalledFor: []catalog{
						{
							ServiceOfferings: smbroker1.ServiceOfferings,
						},
					},
					reconcileVisibilityCalledFor: []*platform.ServiceVisibilityEntity{},
					reconcileDeleteVisibilityCalledFor: []*platform.ServiceVisibilityEntity{
						platformVisibility1,
					},
				}
			},
		}),
	}

	DescribeTable("Run", func(t testCase) {
		smBrokers, err1 := t.smBrokers()
		platformBrokers, err2 := t.platformBrokers()
		platformVisibilities, err3 := t.platformVisibilities()
		smVisibilities, err4 := t.smVisibilities()
		convertedVisibilities, err5 := t.convertedVisibilities()

		smPlans := make([]*types.ServicePlan, 0)
		for _, smBroker := range smBrokers {
			for _, offering := range smBroker.ServiceOfferings {
				for _, plan := range offering.Plans {
					smPlans = append(smPlans, plan)
				}
			}
		}

		fakeSMClient.GetBrokersReturns(smBrokers, err1)
		// TODO: Make error possible here
		fakeSMClient.GetPlansReturns(smPlans, nil)
		fakePlatformBrokerClient.GetBrokersReturns(platformBrokers, err2)
		fakeServiceVisibility.GetVisibilitiesByPlansReturns(platformVisibilities, err3)
		fakeSMClient.GetVisibilitiesReturns(smVisibilities, err4)
		fakeSMVisibilityConverter.ConvertReturnsOnCall(0, convertedVisibilities, err5)
		fakeSMVisibilityConverter.ConvertReturns(nil, nil)
		t.stubs()

		reconcilationTask.Run()

		if err1 != nil {
			Expect(len(fakePlatformBrokerClient.Invocations())).To(Equal(1))
			Expect(len(fakePlatformCatalogFetcher.Invocations())).To(Equal(0))
			Expect(len(fakePlatformServiceAccess.Invocations())).To(Equal(0))
			Expect(fakeSMClient.GetBrokersCallCount()).To(Equal(1))
			return
		}

		if err2 != nil {
			Expect(len(fakePlatformBrokerClient.Invocations())).To(Equal(1))
			Expect(len(fakePlatformCatalogFetcher.Invocations())).To(Equal(0))
			Expect(len(fakePlatformServiceAccess.Invocations())).To(Equal(0))
			Expect(fakeSMClient.GetBrokersCallCount()).To(Equal(0))
			return
		}

		Expect(fakeSMClient.GetBrokersCallCount()).To(Equal(1))
		Expect(fakePlatformBrokerClient.GetBrokersCallCount()).To(Equal(1))

		expected := t.expectations()
		Expect(fakePlatformBrokerClient.CreateBrokerCallCount()).To(Equal(len(expected.reconcileCreateCalledFor)))
		for index, broker := range expected.reconcileCreateCalledFor {
			_, request := fakePlatformBrokerClient.CreateBrokerArgsForCall(index)
			Expect(request).To(Equal(&platform.CreateServiceBrokerRequest{
				Name:      broker.Name,
				BrokerURL: broker.BrokerURL,
			}))
		}

		Expect(fakePlatformCatalogFetcher.FetchCallCount()).To(Equal(len(expected.reconcileCatalogCalledFor)))
		for index, broker := range expected.reconcileCatalogCalledFor {
			_, serviceBroker := fakePlatformCatalogFetcher.FetchArgsForCall(index)
			Expect(serviceBroker).To(Equal(&broker))
		}

		// TODO: Make this to take account only new service visibilities

		// servicesCount := 0
		// index := 0
		// for _, catalog := range expected.reconcileAccessCalledFor {
		// 	for _, service := range catalog.ServiceOfferings {
		// 		_, _, serviceID := fakePlatformServiceAccess.EnableAccessForServiceArgsForCall(index)
		// 		Expect(serviceID).To(Equal(service.CatalogID))
		// 		servicesCount++
		// 		index++
		// 	}
		// }
		// Expect(fakePlatformServiceAccess.EnableAccessForServiceCallCount()).To(Equal(servicesCount))

		Expect(fakePlatformServiceAccess.EnableAccessForPlanCallCount()).To(Equal(len(expected.reconcileVisibilityCalledFor)))
		for index, visibility := range expected.reconcileVisibilityCalledFor {
			_, _, planGUID := fakePlatformServiceAccess.EnableAccessForPlanArgsForCall(index)
			Expect(planGUID).To(Equal(visibility.CatalogPlanID))
		}

		Expect(fakePlatformServiceAccess.DisableAccessForPlanCallCount()).To(Equal(len(expected.reconcileDeleteVisibilityCalledFor)))
		for index, visibility := range expected.reconcileDeleteVisibilityCalledFor {
			_, _, planGUID := fakePlatformServiceAccess.DisableAccessForPlanArgsForCall(index)
			Expect(planGUID).To(Equal(visibility.CatalogPlanID))
		}

		Expect(fakePlatformBrokerClient.DeleteBrokerCallCount()).To(Equal(len(expected.reconcileDeleteCalledFor)))
		for index, broker := range expected.reconcileDeleteCalledFor {
			_, request := fakePlatformBrokerClient.DeleteBrokerArgsForCall(index)
			Expect(request).To(Equal(&platform.DeleteServiceBrokerRequest{
				GUID: broker.GUID,
				Name: broker.Name,
			}))
		}
	}, entries...)

	Describe("Settings", func() {
		Describe("Validate", func() {
			Context("when host is missing", func() {
				It("returns an error", func() {
					settings := DefaultSettings()
					err := settings.Validate()

					Expect(err).Should(HaveOccurred())
				})
			})
		})
	})
})
