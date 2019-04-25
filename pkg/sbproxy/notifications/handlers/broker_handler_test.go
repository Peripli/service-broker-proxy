package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	. "github.com/onsi/gomega"

	"github.com/Peripli/service-broker-proxy/pkg/platform/platformfakes"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/notifications/handlers"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("Broker Handler", func() {
	var ctx context.Context

	var fakeCatalogFetcher *platformfakes.FakeCatalogFetcher
	var fakeBrokerClient *platformfakes.FakeBrokerClient

	var brokerHandler *handlers.BrokerResourceNotificationsHandler

	var brokerNotificationPayload string
	var brokerName string
	var brokerURL string

	var smBrokerID string

	BeforeEach(func() {
		ctx = context.TODO()

		smBrokerID = "brokerID"
		brokerName = "brokerName"
		brokerURL = "brokerURL"

		fakeCatalogFetcher = &platformfakes.FakeCatalogFetcher{}
		fakeBrokerClient = &platformfakes.FakeBrokerClient{}

		brokerHandler = &handlers.BrokerResourceNotificationsHandler{
			BrokerClient:   fakeBrokerClient,
			CatalogFetcher: fakeCatalogFetcher,
			ProxyPrefix:    "proxyPrefix",
			ProxyPath:      "proxyPath",
		}
	})

	Describe("OnCreate", func() {
		BeforeEach(func() {
			brokerNotificationPayload = fmt.Sprintf(`
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
					"additional": {

					}
				}
			}`, smBrokerID, brokerName, brokerURL)
		})

		Context("when unmarshaling notification payload fails", func() {
			BeforeEach(func() {
				brokerNotificationPayload = `randomString`
			})

			It("does not try to create, update or delete broker", func() {
				brokerHandler.OnCreate(ctx, json.RawMessage(brokerNotificationPayload))

				Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
			})
		})

		Context("when notification payload is invalid", func() {
			Context("when new resource is missing", func() {
				BeforeEach(func() {
					brokerNotificationPayload = `{"randomKey":"randomValue"}`
				})

				It("does not try to create, update or delete broker", func() {
					brokerHandler.OnCreate(ctx, json.RawMessage(brokerNotificationPayload))

					Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
				})
			})

			Context("when the resource ID is missing", func() {
				BeforeEach(func() {
					brokerNotificationPayload = fmt.Sprintf(`
					{
						"new": {
							"resource": {
								"name": "%s",
								"broker_url": "%s",
								"description": "brokerDescription",
								"labels": {
									"key1": ["value1", "value2"],
									"key2": ["value3", "value4"]
								}
							},
							"additional": {

							}
						}
					}`, brokerName, brokerURL)
				})

				It("does not try to create, update or delete broker", func() {
					brokerHandler.OnCreate(ctx, json.RawMessage(brokerNotificationPayload))

					Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
				})
			})

			Context("when the resource name is missing", func() {
				BeforeEach(func() {
					brokerNotificationPayload = fmt.Sprintf(`
					{
						"new": {
							"resource": {
								"id": "%s",
								"broker_url": "%s",
								"description": "brokerDescription",
								"labels": {
									"key1": ["value1", "value2"],
									"key2": ["value3", "value4"]
								}
							},
							"additional": {

							}
						}
					}`, smBrokerID, brokerURL)
				})

				It("does not try to create, update or delete broker", func() {
					brokerHandler.OnCreate(ctx, json.RawMessage(brokerNotificationPayload))

					Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
					Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
				})
			})

			Context("when the resource URL is missing", func() {
				BeforeEach(func() {
					brokerNotificationPayload = fmt.Sprintf(`
					{
						"new": {
							"resource": {
								"id": "%s",
								"name": "%s",
								"description": "brokerDescription",
								"labels": {
									"key1": ["value1", "value2"],
									"key2": ["value3", "value4"]
								}
							},
							"additional": {

							}
						}
					}`, smBrokerID, brokerName)
				})

				It("does not try to create, update or delete broker", func() {
					brokerHandler.OnCreate(ctx, json.RawMessage(brokerNotificationPayload))

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
				brokerHandler.OnCreate(ctx, json.RawMessage(brokerNotificationPayload))

				Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
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
					GUID:      smBrokerID,
					Name:      brokerHandler.ProxyPrefix + smBrokerID,
					BrokerURL: brokerHandler.ProxyPath + "/" + smBrokerID,
				}

				fakeBrokerClient.UpdateBrokerReturns(nil, fmt.Errorf("error"))
			})

			It("invokes update broker with the correct arguments", func() {
				Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
				brokerHandler.OnCreate(ctx, json.RawMessage(brokerNotificationPayload))

				Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(1))

				callCtx, callRequest := fakeBrokerClient.UpdateBrokerArgsForCall(0)

				Expect(callCtx).To(Equal(ctx))
				Expect(callRequest).To(Equal(expectedUpdateBrokerRequest))
			})
		})

		Context("when a broker with the same name and URL does not exist in the platform", func() {
			Context("when an error occurs", func() {
				BeforeEach(func() {
					fakeBrokerClient.CreateBrokerReturns(nil, fmt.Errorf("error"))
				})

				It("logs an error", func() {
					VerifyErrorLogged(func() {
						brokerHandler.OnCreate(ctx, json.RawMessage(brokerNotificationPayload))
					})
				})
			})

			Context("when no error occurs", func() {
				var expectedCreateBrokerRequest *platform.CreateServiceBrokerRequest

				BeforeEach(func() {
					fakeBrokerClient.GetBrokerByNameReturns(&platform.ServiceBroker{
						GUID:      smBrokerID,
						Name:      brokerName,
						BrokerURL: "randomURL",
					}, nil)

					expectedCreateBrokerRequest = &platform.CreateServiceBrokerRequest{
						Name:      brokerHandler.ProxyPrefix + smBrokerID,
						BrokerURL: brokerHandler.ProxyPath + "/" + smBrokerID,
					}

					fakeBrokerClient.CreateBrokerReturns(nil, nil)

				})

				It("invokes create broker with the correct arguments", func() {
					Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
					brokerHandler.OnCreate(ctx, json.RawMessage(brokerNotificationPayload))

					Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(1))

					callCtx, callRequest := fakeBrokerClient.CreateBrokerArgsForCall(0)

					Expect(callCtx).To(Equal(ctx))
					Expect(callRequest).To(Equal(expectedCreateBrokerRequest))
				})
			})
		})
	})

	Describe("OnUpdate", func() {
		BeforeEach(func() {
			brokerNotificationPayload = fmt.Sprintf(`
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
				"additional": {

				}
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
				"additional": {

				}
			},
			"label_changes": {
				"op": "add",
				"key": "key3",
				"values": ["value5", "value6"]
			}
		}`, smBrokerID, brokerName, brokerURL, smBrokerID, brokerName, brokerURL)
		})

		Context("when unmarshaling notification payload fails", func() {
			BeforeEach(func() {
				brokerNotificationPayload = `randomString`
			})

			It("does not try to create, update or delete broker", func() {
				brokerHandler.OnUpdate(ctx, json.RawMessage(brokerNotificationPayload))

				Expect(len(fakeBrokerClient.Invocations())).To(Equal(0))
			})
		})

		Context("when old resource is missing", func() {
			BeforeEach(func() {
				brokerNotificationPayload = fmt.Sprintf(`
				{
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
						"additional": {

						}
					},
					"label_changes": {
						"op": "add",
						"key": "key3",
						"values": ["value5", "value6"]
					}
				}`, smBrokerID, brokerName, brokerURL)
			})

			It("does not try to create, update or delete broker", func() {
				brokerHandler.OnUpdate(ctx, json.RawMessage(brokerNotificationPayload))

				Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
			})
		})

		Context("when new resource is missing", func() {
			BeforeEach(func() {
				brokerNotificationPayload = fmt.Sprintf(`
				{
					"old": {
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
						"additional": {

						}
					},
					"label_changes": {
						"op": "add",
						"key": "key3",
						"values": ["value5", "value6"]
					}
				}`, smBrokerID, brokerName, brokerURL)
			})

			It("does not try to create, update or delete broker", func() {
				brokerHandler.OnUpdate(ctx, json.RawMessage(brokerNotificationPayload))

				Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
			})
		})

		Context("when getting broker by name from the platform returns an error", func() {
			BeforeEach(func() {
				fakeBrokerClient.GetBrokerByNameReturns(nil, fmt.Errorf("error"))
			})

			It("does not try to create, update or delete broker", func() {
				brokerHandler.OnUpdate(ctx, json.RawMessage(brokerNotificationPayload))

				Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
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

			It("does not try to create, update or delete broker", func() {
				brokerHandler.OnUpdate(ctx, json.RawMessage(brokerNotificationPayload))

				Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
			})
		})

		Context("when a proxy registration for the SM broker exists in the platform", func() {
			BeforeEach(func() {
				fakeBrokerClient.GetBrokerByNameReturns(&platform.ServiceBroker{
					GUID:      smBrokerID,
					Name:      brokerHandler.ProxyPrefix + smBrokerID,
					BrokerURL: brokerHandler.ProxyPath + "/" + smBrokerID,
				}, nil)

			})

			Context("when an error occurs", func() {
				BeforeEach(func() {
					fakeCatalogFetcher.FetchReturns(fmt.Errorf("error"))
				})

				It("logs an error", func() {
					VerifyErrorLogged(func() {
						brokerHandler.OnUpdate(ctx, json.RawMessage(brokerNotificationPayload))
					})
				})
			})

			Context("when no error occurs", func() {
				var expectedUpdateBrokerRequest *platform.ServiceBroker

				BeforeEach(func() {
					expectedUpdateBrokerRequest = &platform.ServiceBroker{
						GUID:      smBrokerID,
						Name:      brokerHandler.ProxyPrefix + smBrokerID,
						BrokerURL: brokerHandler.ProxyPath + "/" + smBrokerID,
					}

					fakeCatalogFetcher.FetchReturns(nil)
				})

				It("fetches the catalog and does not try to update/overtake the platform broker", func() {
					Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
					brokerHandler.OnUpdate(ctx, json.RawMessage(brokerNotificationPayload))

					Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(1))
					Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))

					callCtx, callRequest := fakeCatalogFetcher.FetchArgsForCall(0)

					Expect(callCtx).To(Equal(ctx))
					Expect(callRequest).To(Equal(expectedUpdateBrokerRequest))
				})
			})
		})
	})

	Describe("OnDelete", func() {
		BeforeEach(func() {
			brokerNotificationPayload = fmt.Sprintf(`
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
				"additional": {

				}
			}
		}`, smBrokerID, brokerName, brokerURL)
		})

		Context("when unmarshaling notification payload fails", func() {
			BeforeEach(func() {
				brokerNotificationPayload = `randomString`
			})

			It("does not try to create or update broker", func() {
				brokerHandler.OnDelete(ctx, json.RawMessage(brokerNotificationPayload))

				Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
			})
		})

		Context("when notification payload is invalid", func() {
			Context("when old resource is missing", func() {
				BeforeEach(func() {
					brokerNotificationPayload = `{"randomKey":"randomValue"}`
				})

				It("does not try to create, update or delete broker", func() {
					brokerHandler.OnDelete(ctx, json.RawMessage(brokerNotificationPayload))

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
				brokerHandler.OnDelete(ctx, json.RawMessage(brokerNotificationPayload))

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

			It("does not try to create, update or delete broker", func() {
				brokerHandler.OnDelete(ctx, json.RawMessage(brokerNotificationPayload))

				Expect(fakeCatalogFetcher.FetchCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.CreateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.UpdateBrokerCallCount()).To(Equal(0))
				Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
			})
		})

		Context("when a broker with the same name and URL exists in the platform", func() {
			BeforeEach(func() {
				fakeBrokerClient.GetBrokerByNameReturns(&platform.ServiceBroker{
					GUID:      smBrokerID,
					Name:      brokerHandler.ProxyPrefix + brokerName,
					BrokerURL: brokerHandler.ProxyPath + "/" + smBrokerID,
				}, nil)
			})

			Context("when an error occurs", func() {
				BeforeEach(func() {
					fakeBrokerClient.DeleteBrokerReturns(fmt.Errorf("error"))
				})

				It("logs an error", func() {
					VerifyErrorLogged(func() {
						brokerHandler.OnDelete(ctx, json.RawMessage(brokerNotificationPayload))
					})
				})
			})

			Context("when no error occurs", func() {
				var expectedDeleteBrokerRequest *platform.DeleteServiceBrokerRequest

				BeforeEach(func() {
					expectedDeleteBrokerRequest = &platform.DeleteServiceBrokerRequest{
						GUID: smBrokerID,
						Name: brokerHandler.ProxyPrefix + smBrokerID,
					}

					fakeBrokerClient.DeleteBrokerReturns(nil)
				})

				It("invokes delete broker with the correct arguments", func() {
					Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(0))
					brokerHandler.OnDelete(ctx, json.RawMessage(brokerNotificationPayload))

					Expect(fakeBrokerClient.DeleteBrokerCallCount()).To(Equal(1))

					callCtx, callRequest := fakeBrokerClient.DeleteBrokerArgsForCall(0)

					Expect(callCtx).To(Equal(ctx))
					Expect(callRequest).To(Equal(expectedDeleteBrokerRequest))
				})
			})
		})
	})
})
