package handlers

import (
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
