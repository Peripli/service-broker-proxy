package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/notifications"

	"github.com/Peripli/service-manager/pkg/query"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
)

type visibilityPayload struct {
	New          visibilityWithAdditionalDetails `json:"new"`
	Old          visibilityWithAdditionalDetails `json:"old"`
	LabelChanges query.LabelChanges              `json:"label_changes"`
}

type visibilityWithAdditionalDetails struct {
	Resource   *types.Visibility `json:"resource"`
	Additional visibilityDetails `json:"additional"`
}

type visibilityDetails struct {
	BrokerID      string `json:"broker_id"`
	CatalogPlanID string `json:"catalog_plan_id"`
}

func (vp visibilityPayload) Validate(op notifications.OperationType) error {
	switch op {
	case notifications.CREATED:
		if err := vp.New.Validate(); err != nil {
			return err
		}
	case notifications.MODIFIED:
		if err := vp.Old.Validate(); err != nil {
			return err
		}
		if err := vp.New.Validate(); err != nil {
			return err
		}
	case notifications.DELETED:
		if err := vp.Old.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (vwad visibilityWithAdditionalDetails) Validate() error {
	if vwad.Resource == nil {
		return fmt.Errorf("resource in notification payload cannot be nil")
	}

	return vwad.Additional.Validate()
}

func (vd visibilityDetails) Validate() error {
	if vd.BrokerID == "" {
		return fmt.Errorf("broker id cannot be empty")
	}
	if vd.CatalogPlanID == "" {
		return fmt.Errorf("catalog plan id cannot be empty")
	}

	return nil
}

// VisibilityResourceNotificationsHandler handles notifications for visibilities
type VisibilityResourceNotificationsHandler struct {
	VisibilityClient platform.VisibilityClient

	ProxyPrefix string
}

// OnCreate creates visibilities from the specified notification payload by invoking the proper platform clients
func (vnh *VisibilityResourceNotificationsHandler) OnCreate(ctx context.Context, payload json.RawMessage) {
	if vnh.VisibilityClient == nil {
		log.C(ctx).Warn("Platform client cannot handle visibilities. Visibility notification will be skipped.")
		return
	}

	visibilityPayload := visibilityPayload{}
	if err := json.Unmarshal(payload, &visibilityPayload); err != nil {
		log.C(ctx).WithError(err).Error("error unmarshaling visibility create notification payload")
		return
	}

	if err := visibilityPayload.Validate(notifications.CREATED); err != nil {
		log.C(ctx).WithError(err).Error("error validating visibility payload")
		return
	}

	v := visibilityPayload.New

	platformBrokerName := vnh.ProxyPrefix + v.Additional.BrokerID
	if err := vnh.VisibilityClient.EnableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{
		BrokerName:    platformBrokerName,
		CatalogPlanID: v.Additional.CatalogPlanID,
		Labels:        v.Resource.GetLabels(),
	}); err != nil {
		log.C(ctx).WithError(err).Errorf("error enabling access for plan %s in broker with name %s", v.Additional.CatalogPlanID, platformBrokerName)
		return
	}
}

// OnUpdate modifies visibilities from the specified notification payload by invoking the proper platform clients
func (vnh *VisibilityResourceNotificationsHandler) OnUpdate(ctx context.Context, payload json.RawMessage) {
	if vnh.VisibilityClient == nil {
		log.C(ctx).Warn("Platform client cannot handle visibilities. Visibility notification will be skipped.")
		return
	}

	visibilityPayload := visibilityPayload{}
	if err := json.Unmarshal(payload, &visibilityPayload); err != nil {
		log.C(ctx).WithError(err).Error("error unmarshaling visibility create notification payload")
		return
	}

	if err := visibilityPayload.Validate(notifications.MODIFIED); err != nil {
		log.C(ctx).WithError(err).Error("error validating visibility payload")
		return
	}

	v := visibilityPayload.New

	labelsToAdd, labelsToRemove := LabelChangesToLabels(visibilityPayload.LabelChanges)

	platformBrokerName := vnh.ProxyPrefix + v.Additional.BrokerID
	if err := vnh.VisibilityClient.EnableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{
		BrokerName:    platformBrokerName,
		CatalogPlanID: v.Additional.CatalogPlanID,
		Labels:        labelsToAdd,
	}); err != nil {
		log.C(ctx).WithError(err).Errorf("error enabling access for plan %s in broker with name %s", v.Additional.CatalogPlanID, platformBrokerName)
		return
	}

	if err := vnh.VisibilityClient.DisableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{
		BrokerName:    platformBrokerName,
		CatalogPlanID: v.Additional.CatalogPlanID,
		Labels:        labelsToRemove,
	}); err != nil {
		log.C(ctx).WithError(err).Errorf("error disabling access for plan %s in broker with name %s", v.Additional.CatalogPlanID, platformBrokerName)
		return
	}
}

// OnDelete deletes visibilities from the provided notification payload by invoking the proper platform clients
func (vnh *VisibilityResourceNotificationsHandler) OnDelete(ctx context.Context, payload json.RawMessage) {
	if vnh.VisibilityClient == nil {
		log.C(ctx).Warn("Platform client cannot handle visibilities. Visibility notification will be skipped.")
		return
	}

	visibilityPayload := visibilityPayload{}
	if err := json.Unmarshal(payload, &visibilityPayload); err != nil {
		log.C(ctx).WithError(err).Error("error unmarshaling visibility delete notification payload")
		return
	}

	if err := visibilityPayload.Validate(notifications.DELETED); err != nil {
		log.C(ctx).WithError(err).Error("error validating visibility payload")
		return
	}

	v := visibilityPayload.Old

	platformBrokerName := vnh.ProxyPrefix + v.Additional.BrokerID
	if err := vnh.VisibilityClient.DisableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{
		BrokerName:    platformBrokerName,
		CatalogPlanID: v.Additional.CatalogPlanID,
		Labels:        v.Resource.GetLabels(),
	}); err != nil {
		log.C(ctx).WithError(err).Errorf("error disabling access for plan %s in broker with name %s", v.Additional.CatalogPlanID, platformBrokerName)
		return
	}
}
