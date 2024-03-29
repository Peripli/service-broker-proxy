package utils

import (
	"github.com/Peripli/service-broker-proxy/pkg/platform/platformfakes"
	"github.com/Peripli/service-manager/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Handlers Helpers", func() {
	Describe("LabelChangesToLabels", func() {
		Context("for changes with add and remove operations", func() {
			var labelChanges types.LabelChanges
			var expectedLabelsToAdd types.Labels
			var expectedLabelsToRemove types.Labels

			BeforeEach(func() {
				labelChanges = types.LabelChanges{
					&types.LabelChange{
						Operation: types.AddLabelOperation,
						Key:       "organization_guid",
						Values: []string{
							"org1",
							"org2",
						},
					},
					&types.LabelChange{
						Operation: types.AddLabelValuesOperation,
						Key:       "organization_guid",
						Values: []string{
							"org3",
							"org4",
						},
					},
					&types.LabelChange{
						Operation: types.RemoveLabelValuesOperation,
						Key:       "organization_guid",
						Values: []string{
							"org5",
							"org6",
						},
					},
					&types.LabelChange{
						Operation: types.RemoveLabelOperation,
						Key:       "organization_guid",
						Values: []string{
							"org7",
							"org8",
						},
					},
				}

				expectedLabelsToAdd = types.Labels{
					"organization_guid": {
						"org1",
						"org2",
						"org3",
						"org4",
					},
				}

				expectedLabelsToRemove = types.Labels{
					"organization_guid": {
						"org5",
						"org6",
						"org7",
						"org8",
					},
				}
			})

			It("generates correct labels", func() {
				labelsToAdd, labelsToRemove := LabelChangesToLabels(labelChanges)

				Expect(labelsToAdd).To(Equal(expectedLabelsToAdd))
				Expect(labelsToRemove).To(Equal(expectedLabelsToRemove))
			})
		})
	})
	Describe("BrokerProxyName", func() {
		It("changes name according to platform name provider function", func() {
			returnName := "some-broker-name"
			brokerID := "1234"
			fakeBrokerPlatformNameProvider := &platformfakes.FakeBrokerPlatformNameProvider{}
			fakeBrokerPlatformNameProvider.GetBrokerPlatformNameReturns(returnName)
			newName := BrokerProxyName(fakeBrokerPlatformNameProvider, "name", brokerID, "")
			Expect(newName).To(Equal(returnName + "-" + brokerID))
		})
	})

})
