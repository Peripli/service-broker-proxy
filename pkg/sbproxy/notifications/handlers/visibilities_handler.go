package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Peripli/service-manager/pkg/util/slice"

	"github.com/Peripli/service-manager/storage/interceptors"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
)

type visibilityPayload struct {
	New          visibilityWithAdditionalDetails `json:"new"`
	Old          visibilityWithAdditionalDetails `json:"old"`
	LabelChanges types.LabelChanges              `json:"label_changes"`
}

type visibilityWithAdditionalDetails struct {
	Resource   *types.Visibility                 `json:"resource"`
	Additional interceptors.VisibilityAdditional `json:"additional"`
}

// Validate validates the visibility payload
func (vp visibilityPayload) Validate(op types.NotificationOperation) error {
	switch op {
	case types.CREATED:
		if err := vp.New.Validate(); err != nil {
			return err
		}
	case types.MODIFIED:
		if err := vp.Old.Validate(); err != nil {
			return err
		}
		if err := vp.New.Validate(); err != nil {
			return err
		}
	case types.DELETED:
		if err := vp.Old.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates the visibility details
func (vwad visibilityWithAdditionalDetails) Validate() error {
	if vwad.Resource == nil {
		return fmt.Errorf("resource in notification payload cannot be nil")
	}

	if vwad.Resource.ID == "" {
		return fmt.Errorf("visibility id cannot be empty")
	}

	if vwad.Resource.ServicePlanID == "" {
		return fmt.Errorf("visibility service plan id cannot be empty")
	}

	return vwad.Additional.Validate()
}

// VisibilityResourceNotificationsHandler handles notifications for visibilities
type VisibilityResourceNotificationsHandler struct {
	VisibilityClient platform.VisibilityClient

	ProxyPrefix     string
	BrokerBlacklist []string
}

// OnCreate creates visibilities from the specified notification payload by invoking the proper platform clients
func (vnh *VisibilityResourceNotificationsHandler) OnCreate(ctx context.Context, notification *types.Notification) {
	logger := log.C(ctx)
	payload := notification.Payload
	if vnh.VisibilityClient == nil {
		logger.Warn("Platform client cannot handle visibilities. Visibility notification will be skipped")
		return
	}

	logger.Debugf("Processing visibility create notification with payload %s...", string(payload))

	visPayload := visibilityPayload{}
	if err := json.Unmarshal(payload, &visPayload); err != nil {
		logger.WithError(err).Error("error unmarshaling visibility create notification payload")
		return
	}

	if err := visPayload.Validate(types.CREATED); err != nil {
		logger.WithError(err).Error("error validating visibility payload")
		return
	}

	v := visPayload.New

	if slice.StringsAnyEquals(vnh.BrokerBlacklist, v.Additional.BrokerName) {
		logger.Infof("Broker name %s for the visibility create notification is part of broker blacklist. Skipping notification...", v.Additional.BrokerName)
		return
	}

	platformBrokerName := vnh.brokerProxyName(v.Additional.BrokerName, v.Additional.BrokerID)
	err := vnh.enableAccessForPlan(ctx, platformBrokerName, v.Additional.ServicePlan.CatalogID, v.Resource.GetLabels())
	if err != nil {
		logger.Error(err)
	}
}

// OnUpdate modifies visibilities from the specified notification payload by invoking the proper platform clients
func (vnh *VisibilityResourceNotificationsHandler) OnUpdate(ctx context.Context, notification *types.Notification) {
	logger := log.C(ctx)
	payload := notification.Payload
	if vnh.VisibilityClient == nil {
		logger.Warn("Platform client cannot handle visibilities. Visibility notification will be skipped.")
		return
	}

	logger.Debugf("Processing visibility update notification with payload %s...", string(payload))

	visibilityPayload := visibilityPayload{}
	if err := json.Unmarshal(payload, &visibilityPayload); err != nil {
		logger.WithError(err).Error("error unmarshaling visibility create notification payload")
		return
	}

	if err := visibilityPayload.Validate(types.MODIFIED); err != nil {
		logger.WithError(err).Error("error validating visibility payload")
		return
	}

	oldVisibilityPayload := visibilityPayload.Old
	newVisibilityPayload := visibilityPayload.New

	if slice.StringsAnyEquals(vnh.BrokerBlacklist, oldVisibilityPayload.Additional.BrokerName) {
		logger.Infof("Broker name %s for the visibility update notification is part of broker blacklist. Skipping notification...", oldVisibilityPayload.Additional.BrokerName)
		return
	}

	platformBrokerName := vnh.brokerProxyName(oldVisibilityPayload.Additional.BrokerName, oldVisibilityPayload.Additional.BrokerID)

	labelsToAdd, labelsToRemove := LabelChangesToLabels(visibilityPayload.LabelChanges)

	if oldVisibilityPayload.Additional.ServicePlan.CatalogID != newVisibilityPayload.Additional.ServicePlan.CatalogID {
		logger.Infof("The catalog plan ID has been modified")

		err := vnh.disableAccessForPlan(ctx, platformBrokerName,
			oldVisibilityPayload.Additional.ServicePlan.CatalogID,
			oldVisibilityPayload.Resource.GetLabels())
		if err != nil {
			logger.Error(err)
			return
		}

		err = vnh.enableAccessForPlan(ctx, platformBrokerName,
			newVisibilityPayload.Additional.ServicePlan.CatalogID,
			newVisibilityPayload.Resource.GetLabels())
		if err != nil {
			logger.Error(err)
			return
		}
	}

	if err := vnh.enableServiceAccess(ctx, labelsToAdd, newVisibilityPayload, platformBrokerName); err != nil {
		logger.Error(err)
		return
	}

	if err := vnh.disableServiceAccess(ctx, labelsToRemove, newVisibilityPayload, platformBrokerName); err != nil {
		logger.Error(err)
		return
	}
}

func (vnh *VisibilityResourceNotificationsHandler) disableServiceAccess(ctx context.Context, labelsToRemove types.Labels, newVisibilityPayload visibilityWithAdditionalDetails, platformBrokerName string) error {
	if (len(labelsToRemove) == 0 && newVisibilityPayload.Resource.PlatformID == "") || (len(labelsToRemove) != 0 && newVisibilityPayload.Resource.PlatformID != "") {
		return vnh.disableAccessForPlan(ctx, platformBrokerName, newVisibilityPayload.Additional.ServicePlan.CatalogID, labelsToRemove)
	}
	return nil
}

func (vnh *VisibilityResourceNotificationsHandler) enableServiceAccess(ctx context.Context, labelsToAdd types.Labels, newVisibilityPayload visibilityWithAdditionalDetails, platformBrokerName string) error {
	if (len(labelsToAdd) == 0 && newVisibilityPayload.Resource.PlatformID == "") || (len(labelsToAdd) != 0 && newVisibilityPayload.Resource.PlatformID != "") {
		return vnh.enableAccessForPlan(ctx, platformBrokerName, newVisibilityPayload.Additional.ServicePlan.CatalogID, labelsToAdd)
	}
	return nil
}

func (vnh *VisibilityResourceNotificationsHandler) enableAccessForPlan(
	ctx context.Context, platformBrokerName, catalogPlanID string, labels types.Labels,
) error {
	logger := log.C(ctx)
	logger.Infof("Attempting to enable access for plan with catalog ID %s for platform broker with name %s ...",
		catalogPlanID, platformBrokerName)

	err := vnh.VisibilityClient.EnableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{
		BrokerName:    platformBrokerName,
		CatalogPlanID: catalogPlanID,
		Labels:        labels,
	})
	if err != nil {
		return fmt.Errorf("error enabling access for plan %s in broker with name %s: %v", catalogPlanID, platformBrokerName, err)
	}

	logger.Infof("Successfully enabled access for plan with catalog ID %s for platform broker with name %s",
		catalogPlanID, platformBrokerName)
	return nil
}

func (vnh *VisibilityResourceNotificationsHandler) disableAccessForPlan(
	ctx context.Context, platformBrokerName, catalogPlanID string, labels types.Labels,
) error {
	logger := log.C(ctx)
	logger.Infof("Attempting to disable access for plan with catalog ID %s for platform broker with name %s ...",
		catalogPlanID, platformBrokerName)

	err := vnh.VisibilityClient.DisableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{
		BrokerName:    platformBrokerName,
		CatalogPlanID: catalogPlanID,
		Labels:        labels,
	})
	if err != nil {
		return fmt.Errorf("error disabling access for plan %s in broker with name %s: %v", catalogPlanID, platformBrokerName, err)
	}

	logger.Infof("Successfully disabled access for plan with catalog ID %s for platform broker with name %s",
		catalogPlanID, platformBrokerName)
	return nil
}

// OnDelete deletes visibilities from the provided notification payload by invoking the proper platform clients
func (vnh *VisibilityResourceNotificationsHandler) OnDelete(ctx context.Context, notification *types.Notification) {
	logger := log.C(ctx)
	payload := notification.Payload
	if vnh.VisibilityClient == nil {
		logger.Warn("Platform client cannot handle visibilities. Visibility notification will be skipped")
		return
	}

	logger.Debugf("Processing visibility delete notification with payload %s...", string(payload))

	visibilityPayload := visibilityPayload{}
	if err := json.Unmarshal(payload, &visibilityPayload); err != nil {
		logger.WithError(err).Error("error unmarshaling visibility delete notification payload")
		return
	}

	if err := visibilityPayload.Validate(types.DELETED); err != nil {
		logger.WithError(err).Error("error validating visibility payload")
		return
	}

	v := visibilityPayload.Old

	if slice.StringsAnyEquals(vnh.BrokerBlacklist, v.Additional.BrokerName) {
		logger.Infof("Broker name %s for the visibility create notification is part of broker blacklist. Skipping notification...", v.Additional.BrokerName)
		return
	}

	platformBrokerName := vnh.brokerProxyName(v.Additional.BrokerName, v.Additional.BrokerID)
	err := vnh.disableAccessForPlan(ctx, platformBrokerName, v.Additional.ServicePlan.CatalogID, v.Resource.GetLabels())
	if err != nil {
		logger.Error(err)
	}
}

func (vnh *VisibilityResourceNotificationsHandler) brokerProxyName(brokerName, brokerID string) string {
	return fmt.Sprintf("%s%s-%s", vnh.ProxyPrefix, brokerName, brokerID)
}
