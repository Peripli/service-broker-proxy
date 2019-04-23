package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/notifications"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/types"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
)

type brokerPayload struct {
	New brokerWithAdditionalDetails `json:"new"`
	Old brokerWithAdditionalDetails `json:"old"`
}

type brokerWithAdditionalDetails struct {
	Resource *types.ServiceBroker `json:"resource"`
}

func (bp brokerPayload) Validate(op notifications.OperationType) error {
	switch op {
	case notifications.CREATED:
		if err := bp.New.Validate(); err != nil {
			return err
		}
	case notifications.MODIFIED:
		if err := bp.Old.Validate(); err != nil {
			return err
		}
		if err := bp.New.Validate(); err != nil {
			return err
		}
	case notifications.DELETED:
		if err := bp.Old.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (bad brokerWithAdditionalDetails) Validate() error {
	if bad.Resource == nil {
		return fmt.Errorf("resource in notification payload cannot be nil")
	}
	if bad.Resource.ID == "" {
		return fmt.Errorf("broker ID cannot be empty")
	}
	if bad.Resource.BrokerURL == "" {
		return fmt.Errorf("broker URL cannot be empty")
	}
	if bad.Resource.Name == "" {
		return fmt.Errorf("broker name cannot be empty")
	}

	return nil
}

// BrokerResourceNotificationsHandler handles notifications for brokers
type BrokerResourceNotificationsHandler struct {
	BrokerClient   platform.BrokerClient
	CatalogFetcher platform.CatalogFetcher

	ProxyPrefix string
	ProxyPath   string
}

// OnCreate creates brokers from the specified notification payload by invoking the proper platform clients
func (bnh *BrokerResourceNotificationsHandler) OnCreate(ctx context.Context, payload json.RawMessage) {
	brokerPayload := brokerPayload{}
	if err := json.Unmarshal(payload, &brokerPayload); err != nil {
		log.C(ctx).WithError(err).Error("error unmarshaling broker create notification payload")
		return
	}

	if err := brokerPayload.Validate(notifications.CREATED); err != nil {
		log.C(ctx).WithError(err).Error("error validating broker payload")
		return
	}

	brokerToCreate := brokerPayload.New
	brokerProxyPath := bnh.brokerProxyPath(brokerToCreate.Resource)
	brokerProxyName := bnh.brokerProxyName(brokerToCreate.Resource)

	existingBroker, err := bnh.BrokerClient.GetBrokerByName(ctx, brokerProxyPath)
	if err != nil {
		log.C(ctx).WithError(err).Errorf("error finding broker with name %s in the platform", brokerToCreate.Resource.GetID())
		return
	}

	if existingBroker != nil && shouldBeProxified(existingBroker, brokerToCreate.Resource) {
		updateRequest := &platform.UpdateServiceBrokerRequest{
			GUID:      existingBroker.GUID,
			Name:      brokerProxyName,
			BrokerURL: brokerProxyPath,
		}
		if _, err := bnh.BrokerClient.UpdateBroker(ctx, updateRequest); err != nil {
			log.C(ctx).WithError(err).Errorf("error proxifying platform broker with GUID %s with SM broker with id %s", existingBroker.GUID, brokerToCreate.Resource.GetID())
			return
		}
	} else {
		createRequest := &platform.CreateServiceBrokerRequest{
			Name:      brokerProxyName,
			BrokerURL: brokerProxyPath,
		}
		if _, err := bnh.BrokerClient.CreateBroker(ctx, createRequest); err != nil {
			log.C(ctx).WithError(err).Errorf("error creating broker with name %s url %s", createRequest.Name, createRequest.BrokerURL)
			return
		}
	}
}

// OnUpdate modifies brokers from the specified notification payload by invoking the proper platform clients
func (bnh *BrokerResourceNotificationsHandler) OnUpdate(ctx context.Context, payload json.RawMessage) {
	brokerPayload := brokerPayload{}

	if err := json.Unmarshal(payload, &brokerPayload); err != nil {
		log.C(ctx).WithError(err).Error("error unmarshaling broker create notification payload")
		return
	}

	if err := brokerPayload.Validate(notifications.MODIFIED); err != nil {
		log.C(ctx).WithError(err).Error("error validating broker payload")
		return
	}

	brokerAfterUpdate := brokerPayload.New
	brokerProxyName := bnh.brokerProxyName(brokerAfterUpdate.Resource)
	brokerProxyPath := bnh.brokerProxyPath(brokerAfterUpdate.Resource)

	existingBroker, err := bnh.BrokerClient.GetBrokerByName(ctx, brokerProxyName)
	if err != nil {
		log.C(ctx).WithError(err).Errorf("error finding broker with name %s in the platform", brokerAfterUpdate.Resource.GetID())
		return
	}
	if existingBroker.BrokerURL != brokerProxyPath {
		log.C(ctx).Infof("No broker in platform found for sm broker ID %s. Nothing to update", brokerAfterUpdate.Resource.ID)
		return
	}

	fetchCatalogRequest := &platform.ServiceBroker{
		GUID:      existingBroker.GUID,
		Name:      brokerProxyName,
		BrokerURL: brokerProxyPath,
	}
	if bnh.CatalogFetcher != nil {
		if err := bnh.CatalogFetcher.Fetch(ctx, fetchCatalogRequest); err != nil {
			log.C(ctx).WithError(err).Errorf("error during fetching catalog for platform guid %s and sm id %s", fetchCatalogRequest.GUID, brokerAfterUpdate.Resource.GetID())
			return
		}
	}
}

// OnDelete deletes brokers from the provided notification payload by invoking the proper platform clients
func (bnh *BrokerResourceNotificationsHandler) OnDelete(ctx context.Context, payload json.RawMessage) {
	brokerPayload := brokerPayload{}

	if err := json.Unmarshal(payload, &brokerPayload); err != nil {
		log.C(ctx).WithError(err).Error("error unmarshaling broker create notification payload")
		return
	}

	if err := brokerPayload.Validate(notifications.DELETED); err != nil {
		log.C(ctx).WithError(err).Error("error validating broker payload")
		return
	}

	brokerToDelete := brokerPayload.Old
	brokerProxyName := bnh.brokerProxyName(brokerToDelete.Resource)
	brokerProxyPath := bnh.brokerProxyPath(brokerToDelete.Resource)

	existingBroker, err := bnh.BrokerClient.GetBrokerByName(ctx, brokerProxyName)
	if err != nil {
		log.C(ctx).WithError(err).Errorf("error finding broker with name %s in the platform", brokerToDelete.Resource.GetID())
		return
	}
	if existingBroker.BrokerURL != brokerProxyPath {
		log.C(ctx).Infof("No broker in platform found for sm broker ID %s. Nothing to delete", brokerToDelete.Resource.ID)
		return
	}

	deleteRequest := &platform.DeleteServiceBrokerRequest{
		GUID: existingBroker.GUID,
		Name: brokerProxyName,
	}

	if err := bnh.BrokerClient.DeleteBroker(ctx, deleteRequest); err != nil {
		log.C(ctx).WithError(err).Errorf("error deleting broker with id %s name %s", deleteRequest.GUID, deleteRequest.Name)
		return
	}
}

func (bnh *BrokerResourceNotificationsHandler) brokerProxyPath(broker *types.ServiceBroker) string {
	return bnh.ProxyPath + "/" + broker.GetID()
}

func (bnh *BrokerResourceNotificationsHandler) brokerProxyName(broker *types.ServiceBroker) string {
	return bnh.ProxyPrefix + broker.GetID()
}

func shouldBeProxified(brokerFromPlatform *platform.ServiceBroker, brokerFromSM *types.ServiceBroker) bool {
	return brokerFromPlatform.BrokerURL == brokerFromSM.BrokerURL &&
		brokerFromPlatform.Name == brokerFromSM.Name
}
