package utils

import (
	"fmt"
	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-manager/pkg/types"
)

// LabelChangesToLabels transforms the specified label changes into two groups of labels - one for creation and one for deletion
func LabelChangesToLabels(changes types.LabelChanges) (types.Labels, types.Labels) {
	labelsToAdd, labelsToRemove := types.Labels{}, types.Labels{}
	for _, change := range changes {
		switch change.Operation {
		case types.AddLabelOperation:
			fallthrough
		case types.AddLabelValuesOperation:
			labelsToAdd[change.Key] = append(labelsToAdd[change.Key], change.Values...)
		case types.RemoveLabelOperation:
			fallthrough
		case types.RemoveLabelValuesOperation:
			labelsToRemove[change.Key] = append(labelsToRemove[change.Key], change.Values...)
		}
	}

	return labelsToAdd, labelsToRemove
}

// BrokerProxyName creates the service-manager name format for the broker.
// If the platform enforces any constraints via GetBrokerPlatformName, the name will first be altered accordingly.
func BrokerProxyName(client interface{}, brokerName string, brokerID string, proxyPrefix string) string {
	nameProvider, ok := client.(platform.BrokerPlatformNameProvider)
	if ok {
		brokerName = nameProvider.GetBrokerPlatformName(brokerName)
	}
	return fmt.Sprintf("%s%s-%s", proxyPrefix, brokerName, brokerID)
}
