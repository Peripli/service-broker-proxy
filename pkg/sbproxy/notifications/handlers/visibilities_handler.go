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

type VisibilityPayload struct {
	New          VisibilityWithAdditionalDetails `json:"new"`
	Old          VisibilityWithAdditionalDetails `json:"old"`
	LabelChanges query.LabelChanges              `json:"label_changes"`
}

type VisibilityWithAdditionalDetails struct {
	Resource   *types.Visibility `json:"resource"`
	Additional VisibilityDetails `json:"additional"`
}

type VisibilityDetails struct {
	BrokerID      string `json:"broker_id"`
	CatalogPlanID string `json:"catalog_plan_id"`
}

func (vp VisibilityPayload) Validate(op notifications.OperationType) error {
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

func (vwad VisibilityWithAdditionalDetails) Validate() error {
	if vwad.Resource == nil {
		return fmt.Errorf("resource in notification payload cannot be nil")
	}

	return vwad.Additional.Validate()
}

func (vd VisibilityDetails) Validate() error {
	if vd.BrokerID == "" {
		return fmt.Errorf("broker id cannot be empty")
	}
	if vd.CatalogPlanID == "" {
		return fmt.Errorf("catalog plan id cannot be empty")
	}

	return nil
}

type VisibilityResourceNotificationsHandler struct {
	VisibilityClient platform.VisibilityClient

	ProxyPrefix string
}

func (vnh *VisibilityResourceNotificationsHandler) OnCreate(ctx context.Context, payload json.RawMessage) {
	visibilityPayload := VisibilityPayload{}
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

func (vnh *VisibilityResourceNotificationsHandler) OnUpdate(ctx context.Context, payload json.RawMessage) {
	visibilityPayload := VisibilityPayload{}
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

func (vnh *VisibilityResourceNotificationsHandler) OnDelete(ctx context.Context, payload json.RawMessage) {
	visibilityPayload := VisibilityPayload{}
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
