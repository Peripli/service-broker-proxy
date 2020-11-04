package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Peripli/service-broker-proxy/pkg/sm/smfakes"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/tidwall/sjson"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	. "github.com/onsi/gomega"

	"github.com/Peripli/service-broker-proxy/pkg/platform/platformfakes"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/notifications/handlers"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("Broker Handler", func() {

	const testNotificationID = "test-notification-id"

	var ctx context.Context

	var fakeCatalogFetcher *platformfakes.FakeCatalogFetcher
	var fakeBrokerClient *platformfakes.FakeBrokerClient
	var fakeSMClient *smfakes.FakeClient
	var fakeBrokerPlatformNameProvider *platformfakes.FakeBrokerPlatformNameProvider

	var brokerHandler *handlers.BrokerResourceNotificationsHandler

	var brokerNotification *types.Notification
	var brokerPlatformCredential *types.BrokerPlatformCredential

	var brokerNotificationPayload string
	var brokerName string
	var brokerURL string
	var expectedBrokerName string
	var brokerNameInNextFuncCall string

	var smBrokerID string
	var catalog string

	assertCreateBrokerRequest := func(actualReq, expectedReq *platform.CreateServiceBrokerRequest) {
		Expect(actualReq.ID).To(Equal(expectedReq.ID))
		Expect(actualReq.Name).To(Equal(expectedReq.Name))
		Expect(actualReq.BrokerURL).To(Equal(expectedReq.BrokerURL))
		Expect(actualReq.Username).ToNot(BeEmpty())
		Expect(actualReq.Password).ToNot(BeEmpty())
	}

	assertUpdateBrokerRequest := func(actualReq, expectedReq *platform.UpdateServiceBrokerRequest) {
		Expect(actualReq.ID).To(Equal(expectedReq.ID))
		Expect(actualReq.GUID).To(Equal(expectedReq.GUID))
		Expect(actualReq.Name).To(Equal(expectedReq.Name))
		Expect(actualReq.BrokerURL).To(Equal(expectedReq.BrokerURL))
		Expect(actualReq.Username).ToNot(BeEmpty())
		Expect(actualReq.Password).ToNot(BeEmpty())
	}

	assertPutCredentialsRequest := func() {
		Expect(fakeSMClient.PutCredentialsCallCount()).To(Equal(1))
		_, credentials := fakeSMClient.PutCredentialsArgsForCall(0)
		Expect(credentials.Username).ToNot(BeEmpty())
		Expect(credentials.PasswordHash).ToNot(BeEmpty())
		Expect(credentials.BrokerID).To(Equal(smBrokerID))
		Expect(credentials.NotificationID).To(Equal(testNotificationID))
		Expect(credentials.Active).To(Equal(false))
	}

	assertActivateCredentialsRequest := func() {
		Expect(fakeSMClient.ActivateCredentialsCallCount()).To(Equal(1))
		_, credentialsID := fakeSMClient.ActivateCredentialsArgsForCall(0)
		Expect(credentialsID).ToNot(BeEmpty())
	}

	setupBrokerNameProvider := func() {
		brokerNameInNextFuncCall = ""
		expectedBrokerName = brokerProxyName(brokerHandler.ProxyPrefix, "broker-name", smBrokerID)

		fakeBrokerPlatformNameProvider.GetBrokerPlatformNameStub = mockGetBrokerPlatformNameFunc

		fakeBrokerClient.GetBrokerByNameStub = func(ctx context.Context, name string) (*platform.ServiceBroker, error) {
			brokerNameInNextFuncCall = name
			return &platform.ServiceBroker{
				GUID:      smBrokerID,
				Name:      name,
				BrokerURL: "some-url.com",
			}, nil
		}
		fakeSMClient.PutCredentialsReturns(brokerPlatformCredential, nil)

		brokerClientWithNameProvider := struct {
			platform.BrokerClient
			platform.BrokerPlatformNameProvider
		}{
			fakeBrokerClient,
			fakeBrokerPlatformNameProvider,
		}

		brokerHandler.BrokerClient = brokerClientWithNameProvider
	}

	BeforeEach(func() {
		ctx = context.TODO()

		smBrokerID = "brokerID"
		brokerName = "brokerName"
		brokerURL = "brokerURL"

		catalog = `{
			"services": [
				{
					"name": "another-fake-service",
					"id": "another-fake-service-id",
					"description": "test-description",
					"requires": ["another-route_forwarding"],
					"tags": ["another-no-sql", "another-relational"],
					"bindable": true,
					"instances_retrievable": true,
					"bindings_retrievable": true,
					"metadata": {
					"provider": {
					"name": "another name"
				},
					"listing": {
					"imageUrl": "http://example.com/cat.gif",
					"blurb": "another blurb here",
					"longDescription": "A long time ago, in a another galaxy far far away..."
				},
					"displayName": "another Fake Service Broker"
				},
					"plan_updateable": true,
					"plans": []
				}
			]
		}`

		fakeSMClient = &smfakes.FakeClient{}
		fakeCatalogFetcher = &platformfakes.FakeCatalogFetcher{}
		fakeBrokerClient = &platformfakes.FakeBrokerClient{}
		fakeBrokerPlatformNameProvider = &platformfakes.FakeBrokerPlatformNameProvider{}

		brokerHandler = &handlers.BrokerResourceNotificationsHandler{
			SMClient:        fakeSMClient,
			BrokerClient:    fakeBrokerClient,
			CatalogFetcher:  fakeCatalogFetcher,
			ProxyPrefix:     "proxyPrefix",
			SMPath:          "proxyPath",
			BrokerBlacklist: []string{},
			TakeoverEnabled: true,
		}

		brokerNotification = &types.Notification{
			Base: types.Base{
				ID: testNotificationID,
			},
			Payload: json.RawMessage(brokerNotificationPayload),
		}

		brokerPlatformCredential = &types.BrokerPlatformCredential{
			Base: types.Base{
				ID: "213456",
			},
		}
	})

	Describe("OnCreate", func() {
		BeforeEach(func() {
			brokerNotification.Payload = json.RawMessage(fmt.Sprintf(`
			{
				"new": {
					"resource": {
						"id": "%s",
						"name": "%s",
						"broker_url": "%s",
						"description": "brokerDescription",
						"labels": {
							"key1": ["value1", "value2"],
							"key2": ["value3", "value4"]
						}
					},
					"additional": %s
				}
			}`, smBrokerID, brokerName, brokerURL, catalog))
		})

		Context("when unmarshaling notification payload fails", func() {
			BeforeEach(func() {
				brokerNotification.Payload = json.RawMessage(`randomString`)
			})

			It("does not try to create, update or delete broker", func() {
				brokerHandler.OnCreate(ctx, brokerNotification)

				Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
				Expect(fakeSMClient.PutCredentialsCallCount()).To(Equal(0))
				Expect(fakeSMClient.ActivateCredentialsCallCount()).To(Equal(0))
			})
		})

		Context("when notification payload is invalid", func() {
			Context("when new resource is missing", func() {
				BeforeEach(func() {
					brokerNotification.Payload = json.RawMessage(`{"randomKey":"randomValue"}`)
				})

				It("does not try to create, update or delete broker", func() {
					brokerHandler.OnCreate(ctx, brokerNotification)

					Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
					Expect(fakeSMClient.PutCredentialsCallCount()).To(Equal(0))
					Expect(fakeSMClient.ActivateCredentialsCallCount()).To(Equal(0))
				})
			})

			Context("when the resource ID is missing", func() {
				BeforeEach(func() {
					payload, err := sjson.DeleteBytes(brokerNotification.Payload, "new.resource.id")
					Expect(err).ShouldNot(HaveOccurred())
					brokerNotification.Payload = payload
				})

				It("does not try to create, update or delete broker", func() {
					brokerHandler.OnCreate(ctx, brokerNotification)

					Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
					Expect(fakeSMClient.PutCredentialsCallCount()).To(Equal(0))
					Expect(fakeSMClient.ActivateCredentialsCallCount()).To(Equal(0))
				})
			})

			Context("when the resource name is missing", func() {
				BeforeEach(func() {
					payload, err := sjson.DeleteBytes(brokerNotification.Payload, "new.resource.name")
					Expect(err).ShouldNot(HaveOccurred())
					brokerNotification.Payload = payload
				})

				It("does not try to create, update or delete broker", func() {
					brokerHandler.OnCreate(ctx, brokerNotification)

					Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
					Expect(fakeSMClient.PutCredentialsCallCount()).To(Equal(0))
					Expect(fakeSMClient.ActivateCredentialsCallCount()).To(Equal(0))
				})
			})

			Context("when the resource URL is missing", func() {
				BeforeEach(func() {
					payload, err := sjson.DeleteBytes(brokerNotification.Payload, "new.resource.broker_url")
					Expect(err).ShouldNot(HaveOccurred())
					brokerNotification.Payload = payload
				})

				It("does not try to create, update or delete broker", func() {
					brokerHandler.OnCreate(ctx, brokerNotification)

					Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
					Expect(fakeSMClient.PutCredentialsCallCount()).To(Equal(0))
					Expect(fakeSMClient.ActivateCredentialsCallCount()).To(Equal(0))
				})
			})

			Context("when additional services are empty", func() {
				BeforeEach(func() {
					payload, err := sjson.DeleteBytes(brokerNotification.Payload, "new.additional.services")
					Expect(err).ShouldNot(HaveOccurred())
					brokerNotification.Payload = payload
				})

				It("does not try to create, update or delete broker", func() {
					brokerHandler.OnCreate(ctx, brokerNotification)

					Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
					Expect(fakeSMClient.PutCredentialsCallCount()).To(Equal(0))
					Expect(fakeSMClient.ActivateCredentialsCallCount()).To(Equal(0))
				})
			})
		})

		Context("when getting broker by name from the platform returns an error", func() {
			BeforeEach(func() {
				fakeBrokerClient.GetBrokerByNameReturns(nil, fmt.Errorf("error"))
				fakeSMClient.PutCredentialsReturns(brokerPlatformCredential, nil)
			})

			It("does try to create and not update or delete broker", func() {
				brokerHandler.OnCreate(ctx, brokerNotification)

				Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(1))
				Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
				assertPutCredentialsRequest()
				assertActivateCredentialsRequest()
			})
		})

		Context("when a broker with the same name and URL exists in the platform", func() {
			var expectedUpdateBrokerRequest *platform.UpdateServiceBrokerRequest

			BeforeEach(func() {
				fakeBrokerClient.GetBrokerByNameReturns(&platform.ServiceBroker{
					GUID:      smBrokerID,
					Name:      brokerName,
					BrokerURL: brokerURL,
				}, nil)

				expectedUpdateBrokerRequest = &platform.UpdateServiceBrokerRequest{
					ID:        smBrokerID,
					GUID:      smBrokerID,
					Name:      brokerProxyName(brokerHandler.ProxyPrefix, brokerName, smBrokerID),
					BrokerURL: brokerHandler.SMPath + "/" + smBrokerID,
				}

				fakeBrokerClient.UpdateBrokerReturns(nil, fmt.Errorf("error"))
				fakeSMClient.PutCredentialsReturns(brokerPlatformCredential, nil)
			})

			When("broker is not in broker blacklist", func() {
				It("invokes update broker with the correct arguments", func() {
					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
					brokerHandler.OnCreate(ctx, brokerNotification)

					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(1))

					callCtx, callRequest := fakeBrokerClient.UpdateBrokerArgsForCall(0)

					Expect(callCtx).To(Equal(ctx))
					assertUpdateBrokerRequest(callRequest, expectedUpdateBrokerRequest)

					assertPutCredentialsRequest()
					Expect(fakeSMClient.ActivateCredentialsCallCount()).To(Equal(0))
				})
			})

			When("broker is in broker blacklist", func() {
				It("doesn't invoke update broker", func() {
					brokerHandler.BrokerBlacklist = []string{brokerName}

					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
					brokerHandler.OnCreate(ctx, brokerNotification)

					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
					Expect(fakeSMClient.PutCredentialsCallCount()).To(Equal(0))
					Expect(fakeSMClient.ActivateCredentialsCallCount()).To(Equal(0))
				})
			})

			When("broker takeover is disabled", func() {
				It("doesn't invoke update broker", func() {
					brokerHandler.TakeoverEnabled = false

					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
					brokerHandler.OnCreate(ctx, brokerNotification)

					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
					Expect(fakeSMClient.PutCredentialsCallCount()).To(Equal(0))
					Expect(fakeSMClient.ActivateCredentialsCallCount()).To(Equal(0))
				})
			})
		})

		Context("when a broker with the same name and URL does not exist in the platform", func() {
			Context("when an error occurs", func() {
				BeforeEach(func() {
					fakeBrokerClient.CreateBrokerReturns(nil, fmt.Errorf("error"))
				})

				It("logs an error", func() {
					VerifyErrorLogged(func() {
						brokerHandler.OnCreate(ctx, brokerNotification)
					})
				})
			})

			Context("when no error occurs", func() {
				var expectedCreateBrokerRequest *platform.CreateServiceBrokerRequest

				BeforeEach(func() {
					fakeBrokerClient.GetBrokerByNameReturns(nil, nil)

					expectedCreateBrokerRequest = &platform.CreateServiceBrokerRequest{
						ID:        smBrokerID,
						Name:      brokerProxyName(brokerHandler.ProxyPrefix, brokerName, smBrokerID),
						BrokerURL: brokerHandler.SMPath + "/" + smBrokerID,
					}

					fakeBrokerClient.CreateBrokerReturns(nil, nil)
					fakeSMClient.PutCredentialsReturns(brokerPlatformCredential, nil)
				})

				It("invokes create broker with the correct arguments", func() {
					Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
					brokerHandler.OnCreate(ctx, brokerNotification)

					Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(1))

					callCtx, callRequest := fakeBrokerClient.CreateBrokerArgsForCall(0)

					Expect(callCtx).To(Equal(ctx))
					assertCreateBrokerRequest(callRequest, expectedCreateBrokerRequest)
					assertPutCredentialsRequest()
					assertActivateCredentialsRequest()
				})
			})
		})

		Context("when a broker with the same name and different URL exists in the platform", func() {
			BeforeEach(func() {
				fakeBrokerClient.GetBrokerByNameReturns(&platform.ServiceBroker{
					GUID:      smBrokerID,
					Name:      brokerName,
					BrokerURL: "randomURL",
				}, nil)

			})

			It("does not try to create, update or delete broker", func() {
				brokerHandler.OnCreate(ctx, brokerNotification)

				Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
				Expect(fakeSMClient.PutCredentialsCallCount()).To(Equal(0))
				Expect(fakeSMClient.ActivateCredentialsCallCount()).To(Equal(0))
			})

			It("logs an error", func() {
				VerifyErrorLogged(func() {
					brokerHandler.OnCreate(ctx, brokerNotification)
				})
			})
		})

		Context("with platform name provider", func() {
			BeforeEach(func() {
				brokerNotification.Payload = json.RawMessage(fmt.Sprintf(`
			{
				"new": {
					"resource": {
						"id": "%s",
						"name": "%s",
						"broker_url": "%s",
						"description": "brokerDescription",
						"labels": {
							"key1": ["value1", "value2"],
							"key2": ["value3", "value4"]
						}
					},
					"additional": %s
				}
			}`, smBrokerID, brokerNameForNameProvider, brokerURL, catalog))
				setupBrokerNameProvider()
			})

			It("changes the broker name according to the name provider function", func() {
				brokerHandler.OnCreate(ctx, brokerNotification)
				Expect(brokerNameInNextFuncCall).To(Equal(expectedBrokerName))
			})
		})
	})

	Describe("OnUpdate", func() {
		BeforeEach(func() {
			brokerNotification.Payload = json.RawMessage(fmt.Sprintf(`
		{
			"old": {
				"resource": {
					"id": "%s",
					"name": "%s",
					"broker_url": "%s",
					"description": "brokerDescription",
					"labels": {
						"key1": ["value1", "value2"],
						"key2": ["value3", "value4"]
					}
				},
				"additional": %s
			},
			"new": {
				"resource": {
					"id": "%s",
					"name": "%s",
					"broker_url": "%s",
					"description": "brokerDescription",
					"labels": {
						"key1": ["value1", "value2"],
						"key2": ["value3", "value4"],
						"key3": ["value5", "value6"]
					}
				},
				"additional": %s
			},
			"label_changes": {
				"op": "add",
				"key": "key3",
				"values": ["value5", "value6"]
			}
		}`, smBrokerID, brokerName, brokerURL, catalog, smBrokerID, brokerName, brokerURL, catalog))
		})

		Context("when unmarshaling notification payload fails", func() {
			BeforeEach(func() {
				brokerNotification.Payload = json.RawMessage(`randomString`)
			})

			It("does not try to create, update or delete broker", func() {
				brokerHandler.OnUpdate(ctx, brokerNotification)

				Expect(len(fakeBrokerClient.Invocations())).To(Equal(0))
				Expect(fakeSMClient.PutCredentialsCallCount()).To(Equal(0))
				Expect(fakeSMClient.ActivateCredentialsCallCount()).To(Equal(0))
			})
		})

		Context("when old resource is missing", func() {
			BeforeEach(func() {
				payload, err := sjson.DeleteBytes(brokerNotification.Payload, "old")
				Expect(err).ShouldNot(HaveOccurred())
				brokerNotification.Payload = payload
			})

			It("does not try to create, update or delete broker", func() {
				brokerHandler.OnUpdate(ctx, brokerNotification)

				Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
				Expect(fakeSMClient.PutCredentialsCallCount()).To(Equal(0))
				Expect(fakeSMClient.ActivateCredentialsCallCount()).To(Equal(0))
			})
		})

		Context("when new resource is missing", func() {
			BeforeEach(func() {
				payload, err := sjson.DeleteBytes(brokerNotification.Payload, "new")
				Expect(err).ShouldNot(HaveOccurred())
				brokerNotification.Payload = payload
			})

			It("does not try to create, update or delete broker", func() {
				brokerHandler.OnUpdate(ctx, brokerNotification)

				Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
				Expect(fakeSMClient.PutCredentialsCallCount()).To(Equal(0))
				Expect(fakeSMClient.ActivateCredentialsCallCount()).To(Equal(0))
			})
		})

		Context("when getting broker by name from the platform returns an error", func() {
			BeforeEach(func() {
				fakeBrokerClient.GetBrokerByNameReturns(nil, fmt.Errorf("error"))
			})

			It("does not try to create, update or delete broker", func() {
				brokerHandler.OnUpdate(ctx, brokerNotification)

				Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
				Expect(fakeSMClient.PutCredentialsCallCount()).To(Equal(0))
				Expect(fakeSMClient.ActivateCredentialsCallCount()).To(Equal(0))
			})
		})

		Context("when a broker with the same name and URL exists in the platform", func() {
			BeforeEach(func() {
				fakeBrokerClient.GetBrokerByNameReturns(&platform.ServiceBroker{
					GUID:      smBrokerID,
					Name:      brokerName,
					BrokerURL: brokerURL,
				}, nil)
			})

			When("broker is not in broker blacklist", func() {
				It("does not try to create, update or delete broker", func() {
					brokerHandler.OnUpdate(ctx, brokerNotification)

					Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
					Expect(fakeSMClient.PutCredentialsCallCount()).To(Equal(0))
					Expect(fakeSMClient.ActivateCredentialsCallCount()).To(Equal(0))
				})
			})

			When("broker is in broker blacklist", func() {
				It("does not try to create, update or delete broker", func() {
					brokerHandler.BrokerBlacklist = []string{brokerName}
					brokerHandler.OnUpdate(ctx, brokerNotification)

					Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
					Expect(fakeSMClient.PutCredentialsCallCount()).To(Equal(0))
					Expect(fakeSMClient.ActivateCredentialsCallCount()).To(Equal(0))
				})
			})
		})

		Context("when the broker name is updated", func() {
			oldBrokerName, newBrokerName := "old-broker", "new-broker"
			BeforeEach(func() {
				brokerNotification.Payload = json.RawMessage(fmt.Sprintf(`
		{
			"old": {
				"resource": {
					"id": "%s",
					"name": "%s",
					"broker_url": "%s",
					"description": "brokerDescription",
					"labels": {
						"key1": ["value1", "value2"],
						"key2": ["value3", "value4"]
					}
				},
				"additional": %s
			},
			"new": {
				"resource": {
					"id": "%s",
					"name": "%s",
					"broker_url": "%s",
					"description": "brokerDescription",
					"labels": {
						"key1": ["value1", "value2"],
						"key2": ["value3", "value4"],
						"key3": ["value5", "value6"]
					}
				},
				"additional": %s
			},
			"label_changes": {
				"op": "add",
				"key": "key3",
				"values": ["value5", "value6"]
			}
		}`, smBrokerID, oldBrokerName, brokerURL, catalog, smBrokerID, newBrokerName, brokerURL, catalog))

				fakeBrokerClient.GetBrokerByNameStub = func(_ context.Context, name string) (*platform.ServiceBroker, error) {
					if name != brokerProxyName(brokerHandler.ProxyPrefix, oldBrokerName, smBrokerID) {
						return nil, fmt.Errorf("could not find broker with name %s", name)
					}
					return &platform.ServiceBroker{
						GUID:      smBrokerID,
						Name:      brokerProxyName(brokerHandler.ProxyPrefix, name, smBrokerID),
						BrokerURL: brokerHandler.SMPath + "/" + smBrokerID,
					}, nil
				}
				fakeBrokerClient.UpdateBrokerReturns(nil, nil)
				fakeSMClient.PutCredentialsReturns(brokerPlatformCredential, nil)
			})

			It("Should update the broker name in the platform", func() {
				var updateRequest *platform.UpdateServiceBrokerRequest
				fakeBrokerClient.UpdateBrokerStub = func(_ context.Context, request *platform.UpdateServiceBrokerRequest) (*platform.ServiceBroker, error) {
					updateRequest = request
					return &platform.ServiceBroker{
						Name:      request.Name,
						BrokerURL: request.BrokerURL,
						GUID:      request.GUID,
					}, nil
				}
				brokerHandler.OnUpdate(ctx, brokerNotification)
				expectedReq := &platform.UpdateServiceBrokerRequest{
					ID:        smBrokerID,
					GUID:      smBrokerID,
					Name:      brokerProxyName(brokerHandler.ProxyPrefix, newBrokerName, smBrokerID),
					BrokerURL: brokerHandler.SMPath + "/" + smBrokerID,
				}
				assertUpdateBrokerRequest(updateRequest, expectedReq)
				assertPutCredentialsRequest()
				assertActivateCredentialsRequest()
			})
		})

		Context("when a proxy registration for the SM broker exists in the platform", func() {
			BeforeEach(func() {
				fakeBrokerClient.GetBrokerByNameReturns(&platform.ServiceBroker{
					GUID:      smBrokerID,
					Name:      brokerProxyName(brokerHandler.ProxyPrefix, smBrokerID, smBrokerID),
					BrokerURL: brokerHandler.SMPath + "/" + smBrokerID,
				}, nil)

			})

			Context("when an error occurs", func() {
				BeforeEach(func() {
					fakeCatalogFetcher.FetchReturns(fmt.Errorf("error"))
				})

				It("logs an error", func() {
					VerifyErrorLogged(func() {
						brokerHandler.OnUpdate(ctx, brokerNotification)
					})
				})
			})

			Context("when no error occurs", func() {
				var expectedUpdateBrokerRequest *platform.UpdateServiceBrokerRequest

				BeforeEach(func() {
					expectedUpdateBrokerRequest = &platform.UpdateServiceBrokerRequest{
						ID:        smBrokerID,
						GUID:      smBrokerID,
						Name:      brokerProxyName(brokerHandler.ProxyPrefix, brokerName, smBrokerID),
						BrokerURL: brokerHandler.SMPath + "/" + smBrokerID,
					}

					fakeCatalogFetcher.FetchReturns(nil)
					fakeSMClient.PutCredentialsReturns(brokerPlatformCredential, nil)
				})

				It("fetches the catalog and does not try to update/overtake the platform broker", func() {
					Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
					brokerHandler.OnUpdate(ctx, brokerNotification)

					Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(1))
					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))

					callCtx, callRequest := fakeCatalogFetcher.FetchArgsForCall(0)

					Expect(callCtx).To(Equal(ctx))
					assertUpdateBrokerRequest(callRequest, expectedUpdateBrokerRequest)
					assertPutCredentialsRequest()
					assertActivateCredentialsRequest()
				})
			})
		})

		Context("with platform name provider", func() {
			BeforeEach(func() {
				brokerNotification.Payload = json.RawMessage(fmt.Sprintf(`
		{
			"old": {
				"resource": {
					"id": "%s",
					"name": "%s",
					"broker_url": "%s",
					"description": "brokerDescription",
					"labels": {
						"key1": ["value1", "value2"],
						"key2": ["value3", "value4"]
					}
				},
				"additional": %s
			},
			"new": {
				"resource": {
					"id": "%s",
					"name": "%s",
					"broker_url": "%s",
					"description": "brokerDescription",
					"labels": {
						"key1": ["value1", "value2"],
						"key2": ["value3", "value4"],
						"key3": ["value5", "value6"]
					}
				},
				"additional": %s
			},
			"label_changes": {
				"op": "add",
				"key": "key3",
				"values": ["value5", "value6"]
			}
		}`, smBrokerID, brokerNameForNameProvider, brokerURL, catalog, smBrokerID, brokerNameForNameProvider, brokerURL, catalog))
				setupBrokerNameProvider()
			})

			It("changes the broker name according to the name provider function", func() {
				brokerHandler.OnUpdate(ctx, brokerNotification)
				Expect(brokerNameInNextFuncCall).To(Equal(expectedBrokerName))
			})
		})

	})

	Describe("OnDelete", func() {
		BeforeEach(func() {
			brokerNotification.Payload = json.RawMessage(fmt.Sprintf(`
		{
			"old": {
				"resource": {
					"id": "%s",
					"name": "%s",
					"broker_url": "%s",
					"description": "brokerDescription",
					"labels": {
						"key1": ["value1", "value2"],
						"key2": ["value3", "value4"]
					}
				},
				"additional": %s
			}
		}`, smBrokerID, brokerName, brokerURL, catalog))
		})

		Context("when unmarshaling notification payload fails", func() {
			BeforeEach(func() {
				brokerNotification.Payload = json.RawMessage(`randomString`)
			})

			It("does not try to create or update broker", func() {
				brokerHandler.OnDelete(ctx, brokerNotification)

				Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
			})
		})

		Context("when notification payload is invalid", func() {
			Context("when old resource is missing", func() {
				BeforeEach(func() {
					payload, err := sjson.DeleteBytes(brokerNotification.Payload, "old")
					Expect(err).ShouldNot(HaveOccurred())
					brokerNotification.Payload = payload
				})

				It("does not try to create, update or delete broker", func() {
					brokerHandler.OnDelete(ctx, brokerNotification)

					Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
				})
			})
		})

		Context("when getting broker by name from the platform returns an error", func() {
			BeforeEach(func() {
				fakeBrokerClient.GetBrokerByNameReturns(nil, fmt.Errorf("error"))
			})

			It("does not try to create, update or delete broker", func() {
				brokerHandler.OnDelete(ctx, brokerNotification)

				Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
			})
		})

		Context("when a broker with the same name and URL does not exist in the platform", func() {
			BeforeEach(func() {
				fakeBrokerClient.GetBrokerByNameReturns(&platform.ServiceBroker{
					GUID:      smBrokerID,
					Name:      "randomName",
					BrokerURL: "randomURL",
				}, nil)
			})

			When("broker is not in broker blacklist", func() {
				It("does not try to create, update or delete broker", func() {
					brokerHandler.OnDelete(ctx, brokerNotification)

					Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
				})
			})

			When("broker is in broker blacklist", func() {
				It("does not try to create, update or delete broker", func() {
					brokerHandler.BrokerBlacklist = []string{brokerName}
					brokerHandler.OnDelete(ctx, brokerNotification)

					Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
				})
			})
		})

		Context("when a broker with the same name and URL exists in the platform", func() {
			BeforeEach(func() {
				fakeBrokerClient.GetBrokerByNameReturns(&platform.ServiceBroker{
					GUID:      smBrokerID,
					Name:      brokerHandler.ProxyPrefix + brokerName,
					BrokerURL: brokerHandler.SMPath + "/" + smBrokerID,
				}, nil)
			})

			Context("when an error occurs", func() {
				BeforeEach(func() {
					fakeBrokerClient.DeleteBrokerReturns(fmt.Errorf("error"))
				})

				It("logs an error", func() {
					VerifyErrorLogged(func() {
						brokerHandler.OnDelete(ctx, brokerNotification)
					})
				})
			})

			Context("when no error occurs", func() {
				var expectedDeleteBrokerRequest *platform.DeleteServiceBrokerRequest

				BeforeEach(func() {
					expectedDeleteBrokerRequest = &platform.DeleteServiceBrokerRequest{
						ID:   smBrokerID,
						GUID: smBrokerID,
						Name: brokerProxyName(brokerHandler.ProxyPrefix, brokerName, smBrokerID),
					}

					fakeBrokerClient.DeleteBrokerReturns(nil)
				})

				It("invokes delete broker with the correct arguments", func() {
					Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
					brokerHandler.OnDelete(ctx, brokerNotification)

					Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(1))

					callCtx, callRequest := fakeBrokerClient.DeleteBrokerArgsForCall(0)

					Expect(callCtx).To(Equal(ctx))
					Expect(callRequest).To(Equal(expectedDeleteBrokerRequest))
				})
			})
		})

		Context("with platform name provider", func() {
			BeforeEach(func() {
				brokerNotification.Payload = json.RawMessage(fmt.Sprintf(`
		{
			"old": {
				"resource": {
					"id": "%s",
					"name": "%s",
					"broker_url": "%s",
					"description": "brokerDescription",
					"labels": {
						"key1": ["value1", "value2"],
						"key2": ["value3", "value4"]
					}
				},
				"additional": %s
			}
		}`, smBrokerID, brokerNameForNameProvider, brokerURL, catalog))
				setupBrokerNameProvider()
			})

			It("changes the broker name according to the name provider function", func() {
				brokerHandler.OnDelete(ctx, brokerNotification)
				Expect(brokerNameInNextFuncCall).To(Equal(expectedBrokerName))
			})
		})
	})
})
