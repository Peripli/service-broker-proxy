package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	. "github.com/onsi/gomega"

	"github.com/Peripli/service-broker-proxy/pkg/platform/platformfakes"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/notifications/handlers"
	. "github.com/onsi/ginkgo"
)

var _ = Describe("Visibility Handler", func() {
	var ctx context.Context

	var fakeVisibilityClient *platformfakes.FakeVisibilityClient

	var visibilityHandler *handlers.VisibilityResourceNotificationsHandler

	var visibilityNotificationPayload string

	var labels string
	var catalogPlanID string
	var smBrokerID string

	unmarshalLabels := func(labelsJSON string) types.Labels {
		labels := types.Labels{}
		err := json.Unmarshal([]byte(labelsJSON), &labels)
		Expect(err).ShouldNot(HaveOccurred())

		return labels
	}

	unmarshalLabelChanges := func(labelChangeJSON string) query.LabelChanges {
		labelChanges := query.LabelChanges{}
		err := json.Unmarshal([]byte(labelChangeJSON), &labelChanges)
		Expect(err).ShouldNot(HaveOccurred())

		return labelChanges
	}

	BeforeEach(func() {
		ctx = context.TODO()

		smBrokerID = "brokerID"
		labels = `
		{
			"key1": ["value1", "value2"],
			"key2": ["value3", "value4"]
		}`
		catalogPlanID = "catalogPlanID"

		fakeVisibilityClient = &platformfakes.FakeVisibilityClient{}

		visibilityHandler = &handlers.VisibilityResourceNotificationsHandler{
			VisibilityClient: fakeVisibilityClient,
		}
	})

	Describe("OnCreate", func() {
		BeforeEach(func() {
			visibilityNotificationPayload = fmt.Sprintf(`
			{
				"new": {
					"resource": {
						"id": "visID",
						"platform_id": "smPlatformID",
						"service_plan_id": "smServicePlanID",
						"labels": %s
					},
					"additional": {
						"broker_id": "%s",
						"catalog_plan_id": "%s"
					}
				}
			}`, labels, smBrokerID, catalogPlanID)
		})

		Context("when unmarshaling notification payload fails", func() {
			BeforeEach(func() {
				visibilityNotificationPayload = `randomString`
			})

			It("does not try to enable or disable access", func() {
				visibilityHandler.OnCreate(ctx, json.RawMessage(visibilityNotificationPayload))

				Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
				Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
			})
		})

		Context("when notification payload is invalid", func() {
			Context("when new resource is missing", func() {
				BeforeEach(func() {
					visibilityNotificationPayload = `{"randomKey":"randomValue"}`
				})

				It("does not try to enable or disable access", func() {
					visibilityHandler.OnCreate(ctx, json.RawMessage(visibilityNotificationPayload))

					Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
					Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
				})
			})

			Context("when the broker ID is missing", func() {
				BeforeEach(func() {
					visibilityNotificationPayload = fmt.Sprintf(`
					{
						"new": {
							"resource": {
								"id": "visID",
								"platform_id": "smPlatformID",
								"service_plan_id": "smServicePlanID",
								"labels": %s
							},
							"additional": {
								"catalog_plan_id": "%s"
							}
						}
					}`, labels, catalogPlanID)
				})

				It("does not try to enable or disable access", func() {
					visibilityHandler.OnCreate(ctx, json.RawMessage(visibilityNotificationPayload))

					Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
					Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
				})
			})

			Context("when the catalog plan ID is missing", func() {
				BeforeEach(func() {
					visibilityNotificationPayload = fmt.Sprintf(`
					{
						"new": {
							"resource": {
								"id": "visID",
								"platform_id": "smPlatformID",
								"service_plan_id": "smServicePlanID",
								"labels": %s
							},
							"additional": {
								"broker_id": "%s"
							}
						}
					}`, labels, smBrokerID)
				})

				It("does not try to enable or disable access", func() {
					visibilityHandler.OnCreate(ctx, json.RawMessage(visibilityNotificationPayload))

					Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
					Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
				})
			})
		})

		Context("when the notification payload is valid", func() {
			Context("when an error occurs while enabling access", func() {
				BeforeEach(func() {
					fakeVisibilityClient.EnableAccessForPlanReturns(fmt.Errorf("error"))
				})

				It("logs an error", func() {
					VerifyErrorLogged(func() {
						visibilityHandler.OnCreate(ctx, json.RawMessage(visibilityNotificationPayload))
					})
				})
			})

			Context("when no error occurs", func() {
				var expectedRequest *platform.ModifyPlanAccessRequest

				BeforeEach(func() {
					fakeVisibilityClient.EnableAccessForPlanReturns(nil)

					expectedRequest = &platform.ModifyPlanAccessRequest{
						BrokerName:    visibilityHandler.ProxyPrefix + smBrokerID,
						CatalogPlanID: catalogPlanID,
						Labels:        unmarshalLabels(labels),
					}
				})

				It("invokes enable access for plan", func() {
					Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
					visibilityHandler.OnCreate(ctx, json.RawMessage(visibilityNotificationPayload))

					Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(1))

					callCtx, callRequest := fakeVisibilityClient.EnableAccessForPlanArgsForCall(0)

					Expect(callCtx).To(Equal(ctx))
					Expect(callRequest).To(Equal(expectedRequest))
				})
			})
		})
	})

	Describe("OnUpdate", func() {
		var addLabelChanges string
		var removeLabelChanges string
		var labelChanges string

		BeforeEach(func() {
			addLabelChanges = `
			{
				"op": "add",
				"key": "key3",
				"values": ["value5", "value6"]
			}`

			removeLabelChanges = `
			{
				"op": "remove",
				"key": "key2",
				"values": ["value3", "value4"]
			}`

			labelChanges = fmt.Sprintf(`
			[%s, %s]`, addLabelChanges, removeLabelChanges)

			visibilityNotificationPayload = fmt.Sprintf(`
			{
				"old": {
					"resource": {
						"id": "visID",
						"platform_id": "smPlatformID",
						"service_plan_id": "smServicePlanID",
						"labels": %s
					},
					"additional": {
						"broker_id": "%s",
						"catalog_plan_id": "%s"
					}
				},
				"new": {
					"resource": {
						"id": "visID",
						"platform_id": "smPlatformID",
						"service_plan_id": "smServicePlanID",
						"labels": %s
					},
					"additional": {
						"broker_id": "%s",
						"catalog_plan_id": "%s"
					}
				},
				"label_changes": %s
			}`, labels, smBrokerID, catalogPlanID, labels, smBrokerID, catalogPlanID, labelChanges)
		})

		Context("when unmarshaling notification payload fails", func() {
			BeforeEach(func() {
				visibilityNotificationPayload = `randomString`
			})

			It("does not try to enable or disable access", func() {
				visibilityHandler.OnUpdate(ctx, json.RawMessage(visibilityNotificationPayload))

				Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
				Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
			})
		})

		Context("when old resource is missing", func() {
			BeforeEach(func() {
				visibilityNotificationPayload = fmt.Sprintf(`
				{
					"new": {
						"resource": {
							"id": "visID",
							"platform_id": "smPlatformID",
							"service_plan_id": "smServicePlanID",
							"labels": %s
						},
						"additional": {
							"broker_id": "%s",
							"catalog_plan_id": "%s"
						}
					},
					"label_changes": %s
				}`, labels, smBrokerID, catalogPlanID, labelChanges)
			})

			It("does not try to enable or disable access", func() {
				visibilityHandler.OnUpdate(ctx, json.RawMessage(visibilityNotificationPayload))

				Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
				Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
			})
		})

		Context("when new resource is missing", func() {
			BeforeEach(func() {
				visibilityNotificationPayload = fmt.Sprintf(`
				{
					"old": {
						"resource": {
							"id": "visID",
							"platform_id": "smPlatformID",
							"service_plan_id": "smServicePlanID",
							"labels": %s
						},
						"additional": {
							"broker_id": "%s",
							"catalog_plan_id": "%s"
						}
					},
					"label_changes": %s
				}`, labels, smBrokerID, catalogPlanID, labelChanges)
			})

			It("does not try to enable or disable access", func() {
				visibilityHandler.OnUpdate(ctx, json.RawMessage(visibilityNotificationPayload))

				Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
				Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
			})
		})

		Context("when the notification payload is valid", func() {
			Context("when an error occurs while enabling access", func() {
				BeforeEach(func() {
					fakeVisibilityClient.EnableAccessForPlanReturns(fmt.Errorf("error"))
					fakeVisibilityClient.DisableAccessForPlanReturns(nil)

				})

				It("logs an error", func() {
					VerifyErrorLogged(func() {
						visibilityHandler.OnUpdate(ctx, json.RawMessage(visibilityNotificationPayload))
					})
				})
			})

			Context("when an error occurs while disabling access", func() {
				BeforeEach(func() {
					fakeVisibilityClient.EnableAccessForPlanReturns(nil)
					fakeVisibilityClient.DisableAccessForPlanReturns(fmt.Errorf("error"))
				})

				It("logs an error", func() {
					VerifyErrorLogged(func() {
						visibilityHandler.OnUpdate(ctx, json.RawMessage(visibilityNotificationPayload))
					})
				})
			})

			Context("when no error occurs", func() {
				var expectedRequests []*platform.ModifyPlanAccessRequest

				BeforeEach(func() {
					fakeVisibilityClient.EnableAccessForPlanReturns(nil)
					fakeVisibilityClient.DisableAccessForPlanReturns(nil)

					labelsToAdd, labelsToRemove := handlers.LabelChangesToLabels(unmarshalLabelChanges(labelChanges))
					expectedRequests = []*platform.ModifyPlanAccessRequest{
						{
							BrokerName:    visibilityHandler.ProxyPrefix + smBrokerID,
							CatalogPlanID: catalogPlanID,
							Labels:        labelsToAdd,
						},
						{
							BrokerName:    visibilityHandler.ProxyPrefix + smBrokerID,
							CatalogPlanID: catalogPlanID,
							Labels:        labelsToRemove,
						},
					}
				})

				It("invokes enable and disable access for the plan", func() {
					Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
					Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))

					visibilityHandler.OnUpdate(ctx, json.RawMessage(visibilityNotificationPayload))

					Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(1))
					Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(1))

					callCtx, enableAccessRequest := fakeVisibilityClient.EnableAccessForPlanArgsForCall(0)
					Expect(callCtx).To(Equal(ctx))
					Expect(enableAccessRequest).To(Equal(expectedRequests[0]))

					callCtx, disableAccessRequest := fakeVisibilityClient.DisableAccessForPlanArgsForCall(0)
					Expect(callCtx).To(Equal(ctx))
					Expect(disableAccessRequest).To(Equal(expectedRequests[1]))
				})
			})
		})
	})

	Describe("OnDelete", func() {
		BeforeEach(func() {
			visibilityNotificationPayload = fmt.Sprintf(`
					{
						"old": {
							"resource": {
								"id": "visID",
								"platform_id": "smPlatformID",
								"service_plan_id": "smServicePlanID",
								"labels": %s
							},
							"additional": {
								"broker_id": "%s",
								"catalog_plan_id": "%s"
							}
						}
					}`, labels, smBrokerID, catalogPlanID)
		})

		Context("when unmarshaling notification payload fails", func() {
			BeforeEach(func() {
				visibilityNotificationPayload = `randomString`
			})

			It("does not try to enable or disable access", func() {
				visibilityHandler.OnDelete(ctx, json.RawMessage(visibilityNotificationPayload))

				Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
				Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
			})
		})

		Context("when notification payload is invalid", func() {
			Context("when old resource is missing", func() {
				BeforeEach(func() {
					visibilityNotificationPayload = `{"randomKey":"randomValue"}`
				})

				It("does not try to enable or disable access", func() {
					visibilityHandler.OnDelete(ctx, json.RawMessage(visibilityNotificationPayload))

					Expect(fakeVisibilityClient.EnableAccessForPlanCallCount()).To(Equal(0))
					Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
				})
			})
		})

		Context("when the notification payload is valid", func() {
			Context("when an error occurs while disabling access", func() {
				BeforeEach(func() {
					fakeVisibilityClient.DisableAccessForPlanReturns(fmt.Errorf("error"))
				})

				It("logs an error", func() {
					VerifyErrorLogged(func() {
						visibilityHandler.OnDelete(ctx, json.RawMessage(visibilityNotificationPayload))
					})
				})
			})

			Context("when no error occurs", func() {
				var expectedRequest *platform.ModifyPlanAccessRequest

				BeforeEach(func() {
					fakeVisibilityClient.DisableAccessForPlanReturns(nil)

					expectedRequest = &platform.ModifyPlanAccessRequest{
						BrokerName:    visibilityHandler.ProxyPrefix + smBrokerID,
						CatalogPlanID: catalogPlanID,
						Labels:        unmarshalLabels(labels),
					}
				})

				It("invokes disable access for plan", func() {
					Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(0))
					visibilityHandler.OnDelete(ctx, json.RawMessage(visibilityNotificationPayload))

					Expect(fakeVisibilityClient.DisableAccessForPlanCallCount()).To(Equal(1))

					callCtx, callRequest := fakeVisibilityClient.DisableAccessForPlanArgsForCall(0)

					Expect(callCtx).To(Equal(ctx))
					Expect(callRequest).To(Equal(expectedRequest))
				})
			})

		})
	})
})
