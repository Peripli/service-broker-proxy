package reconcile

import (
	"fmt"
	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-broker-proxy/pkg/platform/platformfakes"
	"github.com/Peripli/service-broker-proxy/pkg/sm"
	"github.com/Peripli/service-broker-proxy/pkg/sm/smfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	osbc "github.com/pmorie/go-open-service-broker-client/v2"
	"sync"
)

var _ = Describe("ReconcileBrokersTask", func() {
	const fakeSelfHost = "https://smproxy.com"

	var (
		fakeSMClient *smfakes.FakeClient

		fakePlatformCatalogFetcher *platformfakes.FakeCatalogFetcher
		fakePlatformServiceAccess  *platformfakes.FakeServiceAccess
		fakePlatformBrokerClient   *platformfakes.FakeClient

		fakeWG *sync.WaitGroup

		reconcilationTask *ReconcileBrokersTask

		smbroker1 sm.Broker
		smbroker2 sm.Broker

		platformbroker1        platform.ServiceBroker
		platformbroker2        platform.ServiceBroker
		platformbrokerNonProxy platform.ServiceBroker
	)

	stubCreateBrokerToSucceed := func(r *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {
		return &platform.ServiceBroker{
			GUID:      r.Name,
			Name:      r.Name,
			BrokerURL: r.BrokerURL,
		}, nil
	}

	stubCreateBrokerToReturnError := func(r *platform.CreateServiceBrokerRequest) (*platform.ServiceBroker, error) {
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

		fakeWG = &sync.WaitGroup{}

		reconcilationTask = NewTask(fakeWG, struct {
			*platformfakes.FakeCatalogFetcher
			*platformfakes.FakeServiceAccess
			*platformfakes.FakeClient
		}{
			FakeCatalogFetcher: fakePlatformCatalogFetcher,
			FakeServiceAccess:  fakePlatformServiceAccess,
			FakeClient:         fakePlatformBrokerClient,
		}, fakeSMClient, fakeSelfHost)

		smbroker1 = sm.Broker{
			ID:        "smBrokerID1",
			BrokerURL: "https://smBroker1.com",
			Catalog: &osbc.CatalogResponse{
				Services: []osbc.Service{
					{
						ID:                  "smBroker1ServiceID1",
						Name:                "smBroker1Service1",
						Description:         "description",
						Bindable:            true,
						BindingsRetrievable: true,
						Plans: []osbc.Plan{
							{
								ID:          "smBroker1ServiceID1PlanID1",
								Name:        "smBroker1Service1Plan1",
								Description: "description",
							},
							{
								ID:          "smBroker1ServiceID1PlanID2",
								Name:        "smBroker1Service1Plan2",
								Description: "description",
							},
						},
					},
				},
			},
		}

		smbroker2 = sm.Broker{
			ID:        "smBrokerID2",
			BrokerURL: "https://smBroker2.com",
			Catalog: &osbc.CatalogResponse{
				Services: []osbc.Service{
					{
						ID:                  "smBroker2ServiceID1",
						Name:                "smBroker2Service1",
						Description:         "description",
						Bindable:            true,
						BindingsRetrievable: true,
						Plans: []osbc.Plan{
							{
								ID:          "smBroker2ServiceID1PlanID1",
								Name:        "smBroker2Service1Plan1",
								Description: "description",
							},
							{
								ID:          "smBroker2ServiceID1PlanID2",
								Name:        "smBroker2Service1Plan2",
								Description: "description",
							},
						},
					},
				},
			},
		}

		platformbroker1 = platform.ServiceBroker{
			GUID:      "platformBrokerID1",
			Name:      ProxyBrokerPrefix + "smBrokerID1",
			BrokerURL: fakeSelfHost + "/" + smbroker1.ID,
		}

		platformbroker2 = platform.ServiceBroker{
			GUID:      "platformBrokerID2",
			Name:      ProxyBrokerPrefix + "smBrokerID2",
			BrokerURL: fakeSelfHost + "/" + smbroker2.ID,
		}

		platformbrokerNonProxy = platform.ServiceBroker{
			GUID:      "platformBrokerID3",
			Name:      "platformBroker3",
			BrokerURL: "https://platformBroker3.com",
		}
	})

	type reconcilationExceptations struct {
		reconcileCreate  []platform.ServiceBroker
		reconcileDelete  []platform.ServiceBroker
		reconcileCatalog []platform.ServiceBroker
		reconcileAccess  []osbc.CatalogResponse
	}

	type testCase struct {
		stubs           func()
		platformBrokers func() ([]platform.ServiceBroker, error)
		smBrokers       func() ([]sm.Broker, error)

		expectations func() reconcilationExceptations
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
			expectations: func() reconcilationExceptations {
				return reconcilationExceptations{
					reconcileCreate:  []platform.ServiceBroker{},
					reconcileDelete:  []platform.ServiceBroker{},
					reconcileCatalog: []platform.ServiceBroker{},
					reconcileAccess:  []osbc.CatalogResponse{},
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
			expectations: func() reconcilationExceptations {
				return reconcilationExceptations{
					reconcileCreate:  []platform.ServiceBroker{},
					reconcileDelete:  []platform.ServiceBroker{},
					reconcileCatalog: []platform.ServiceBroker{},
					reconcileAccess:  []osbc.CatalogResponse{},
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
			expectations: func() reconcilationExceptations {
				return reconcilationExceptations{
					reconcileCreate: []platform.ServiceBroker{
						platformbroker1,
					},
					reconcileDelete: []platform.ServiceBroker{
						platformbroker2,
					},
					reconcileCatalog: []platform.ServiceBroker{},
					reconcileAccess: []osbc.CatalogResponse{
						*smbroker1.Catalog,
					},
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
				smbroker1.Catalog = nil
				return []sm.Broker{
					smbroker1,
					smbroker2,
				}, nil
			},
			expectations: func() reconcilationExceptations {
				return reconcilationExceptations{
					reconcileCreate: []platform.ServiceBroker{},
					reconcileDelete: []platform.ServiceBroker{},
					reconcileCatalog: []platform.ServiceBroker{
						platformbroker1,
						platformbroker2,
					},
					reconcileAccess: []osbc.CatalogResponse{
						*smbroker2.Catalog,
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
			expectations: func() reconcilationExceptations {
				return reconcilationExceptations{
					reconcileCreate: []platform.ServiceBroker{
						platformbroker1,
						platformbroker2,
					},
					reconcileDelete:  []platform.ServiceBroker{},
					reconcileCatalog: []platform.ServiceBroker{},
					reconcileAccess: []osbc.CatalogResponse{
						*smbroker1.Catalog,
						*smbroker2.Catalog,
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
			expectations: func() reconcilationExceptations {
				return reconcilationExceptations{
					reconcileCreate: []platform.ServiceBroker{},
					reconcileDelete: []platform.ServiceBroker{},
					reconcileCatalog: []platform.ServiceBroker{
						platformbroker1,
					},
					reconcileAccess: []osbc.CatalogResponse{
						*smbroker1.Catalog,
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
			expectations: func() reconcilationExceptations {
				return reconcilationExceptations{
					reconcileCreate: []platform.ServiceBroker{},
					reconcileDelete: []platform.ServiceBroker{
						platformbroker1,
					},
					reconcileCatalog: []platform.ServiceBroker{},
					reconcileAccess:  []osbc.CatalogResponse{},
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
			expectations: func() reconcilationExceptations {
				return reconcilationExceptations{
					reconcileCreate:  []platform.ServiceBroker{},
					reconcileDelete:  []platform.ServiceBroker{},
					reconcileCatalog: []platform.ServiceBroker{},
					reconcileAccess:  []osbc.CatalogResponse{},
				}
			},
		}),
	}

	DescribeTable("Run", func(t testCase) {
		smBrokers, err1 := t.smBrokers()
		platformBrokers, err2 := t.platformBrokers()

		fakeSMClient.GetBrokersReturns(smBrokers, err1)
		fakePlatformBrokerClient.GetBrokersReturns(platformBrokers, err2)
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
		Expect(fakePlatformBrokerClient.CreateBrokerCallCount()).To(Equal(len(expected.reconcileCreate)))
		for index, broker := range expected.reconcileCreate {
			Expect(fakePlatformBrokerClient.CreateBrokerArgsForCall(index)).To(Equal(&platform.CreateServiceBrokerRequest{
				Name:      broker.Name,
				BrokerURL: broker.BrokerURL,
			}))
		}

		Expect(fakePlatformCatalogFetcher.FetchCallCount()).To(Equal(len(expected.reconcileCatalog)))
		for index, broker := range expected.reconcileCatalog {
			Expect(fakePlatformCatalogFetcher.FetchArgsForCall(index)).To(Equal(&broker))
		}

		servicesCount := 0
		index := 0
		for _, catalog := range expected.reconcileAccess {
			for _, service := range catalog.Services {
				_, serviceID := fakePlatformServiceAccess.EnableAccessForServiceArgsForCall(index)
				Expect(serviceID).To(Equal(service.ID))
				servicesCount++
				index++
			}
		}
		Expect(fakePlatformServiceAccess.EnableAccessForServiceCallCount()).To(Equal(servicesCount))

		Expect(fakePlatformBrokerClient.DeleteBrokerCallCount()).To(Equal(len(expected.reconcileDelete)))
		for index, broker := range expected.reconcileDelete {
			Expect(fakePlatformBrokerClient.DeleteBrokerArgsForCall(index)).To(Equal(&platform.DeleteServiceBrokerRequest{
				GUID: broker.GUID,
				Name: broker.Name,
			}))
		}
	}, entries...)
})
