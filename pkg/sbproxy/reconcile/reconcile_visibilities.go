/*
 * Copyright 2018 The Service Manager Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package reconcile

import (
	"context"
	"fmt"
	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/utils"
	"strings"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
)

func (r *resyncJob) reconcileVisibilities(ctx context.Context, smPlans map[string]brokerPlan, smBrokers []*platform.ServiceBroker) {
	logger := log.C(ctx)
	if r.options.VisibilityBrokerChunkSize == 0 {
		err := r.reconcilePlansVisibilities(ctx, false, smPlans, smBrokers)
		if err != nil {
			logger.WithError(err).Error("an error occurred while reconciling visibilities")
			return
		}
	} else {
		chunks := getBrokerChunks(smBrokers, r.options.VisibilityBrokerChunkSize)
		for _, brokersChunk := range chunks {
			plansChunk := getBrokersPlans(brokersChunk, smPlans)
			err := r.reconcilePlansVisibilities(ctx, true, plansChunk, brokersChunk)
			if err != nil {
				logger.WithError(err).Error("an error occurred while reconciling visibilities")
				return
			}
		}
	}
}

func (r *resyncJob) getPlatformVisibilitiesByBrokersFromPlatform(ctx context.Context, brokers []*platform.ServiceBroker) ([]*platform.Visibility, error) {
	logger := log.C(ctx)
	logger.Info("resyncJob getting visibilities from platform")

	names := r.brokerNames(brokers)
	visibilities, err := r.platformClient.Visibility().GetVisibilitiesByBrokers(ctx, names)
	if err != nil {
		return nil, err
	}
	logger.Infof("resyncJob successfully retrieved %d visibilities from platform", len(visibilities))

	return visibilities, nil
}

func (r *resyncJob) brokerNames(brokers []*platform.ServiceBroker) []string {
	names := make([]string, 0, len(brokers))
	for _, broker := range brokers {
		names = append(names, utils.BrokerProxyName(r.platformClient, broker.Name, broker.GUID, r.options.BrokerPrefix))
	}
	return names
}

func (r *resyncJob) getSMBrokerPlans(ctx context.Context, offerings map[string]*types.ServiceOffering, smBrokers []*platform.ServiceBroker) (map[string]brokerPlan, error) {
	log.C(ctx).Info("resyncJob getting service plans from Service Manager")
	plans, err := r.smClient.GetPlans(ctx)
	if err != nil {
		return nil, err
	}
	log.C(ctx).Infof("resyncJob successfully retrieved %d plans from Service Manager", len(plans))

	brokerMap := make(map[string]*platform.ServiceBroker, len(smBrokers))
	for _, broker := range smBrokers {
		brokerMap[broker.GUID] = broker
	}

	brokerPlans := make(map[string]brokerPlan, len(plans))
	for _, plan := range plans {
		service := offerings[plan.ServiceOfferingID]
		if service == nil {
			continue
		}
		broker := brokerMap[service.BrokerID]
		if broker == nil {
			continue
		}
		brokerPlans[plan.ID] = brokerPlan{
			ServicePlan: plan,
			broker:      broker,
		}
	}
	return brokerPlans, nil
}

func (r *resyncJob) getSMServiceOfferings(ctx context.Context) (map[string]*types.ServiceOffering, error) {
	log.C(ctx).Info("resyncJob getting service offerings from Service Manager...")
	offerings, err := r.smClient.GetServiceOfferings(ctx)
	if err != nil {
		return nil, err
	}
	log.C(ctx).Infof("resyncJob successfully retrieved %d service offerings from Service Manager", len(offerings))

	result := make(map[string]*types.ServiceOffering)
	for _, offering := range offerings {
		result[offering.ID] = offering
	}
	return result, nil
}

func (r *resyncJob) getVisibilitiesFromSM(ctx context.Context, useServicePlanFieldQuery bool, smPlansMap map[string]brokerPlan) ([]*platform.Visibility, error) {
	logger := log.C(ctx)
	logger.Info("resyncJob getting visibilities from Service Manager...")
	var planIDs []string
	if useServicePlanFieldQuery {
		//initialise the planIDs array. if the array is nil than r.smClient.GetVisibilities will fetch all the visibilities without service_plan_id fieldQuery
		planIDs = []string{}
		for _, plan := range smPlansMap {
			planIDs = append(planIDs, plan.ID)
		}
	}
	visibilities, err := r.smClient.GetVisibilities(ctx, planIDs)
	if err != nil {
		return nil, err
	}
	logger.Infof("resyncJob successfully retrieved %d visibilities from Service Manager", len(visibilities))

	result := make([]*platform.Visibility, 0)

	for _, visibility := range visibilities {
		smPlan, found := smPlansMap[visibility.ServicePlanID]
		if !found {
			continue
		}
		converted := r.convertSMVisibility(visibility, smPlan)
		result = append(result, converted...)
	}
	logger.Infof("resyncJob successfully converted %d Service Manager visibilities to %d platform visibilities", len(visibilities), len(result))

	return result, nil
}

func (r *resyncJob) convertSMVisibility(visibility *types.Visibility, smPlan brokerPlan) []*platform.Visibility {
	scopeLabelKey := r.platformClient.Visibility().VisibilityScopeLabelKey()
	shouldBePublic := visibility.PlatformID == "" || len(visibility.Labels[scopeLabelKey]) == 0

	if shouldBePublic {
		return []*platform.Visibility{
			{
				Public:             true,
				CatalogPlanID:      smPlan.CatalogID,
				PlatformBrokerName: utils.BrokerProxyName(r.platformClient, smPlan.broker.Name, smPlan.broker.GUID, r.options.BrokerPrefix),
				Labels:             map[string]string{},
			},
		}
	}

	scopes := visibility.Labels[scopeLabelKey]
	result := make([]*platform.Visibility, 0, len(scopes))
	for _, scope := range scopes {
		result = append(result, &platform.Visibility{
			Public:             false,
			CatalogPlanID:      smPlan.CatalogID,
			PlatformBrokerName: utils.BrokerProxyName(r.platformClient, smPlan.broker.Name, smPlan.broker.GUID, r.options.BrokerPrefix),
			Labels:             map[string]string{scopeLabelKey: scope},
		})
	}
	return result
}

func (r *resyncJob) reconcileServiceVisibilities(ctx context.Context, platformVis, smVis []*platform.Visibility) bool {
	logger := log.C(ctx)
	logger.Info("resyncJob reconciling platform and Service Manager visibilities...")

	platformMap := r.convertVisListToMap(platformVis)
	visibilitiesToCreate := make([]*platform.Visibility, 0)
	for _, visibility := range smVis {
		key := r.getVisibilityKey(visibility)
		existingVis := platformMap[key]
		delete(platformMap, key)
		if existingVis == nil {
			visibilitiesToCreate = append(visibilitiesToCreate, visibility)
		}
	}

	logger.Infof("resyncJob %d visibilities will be removed from the platform", len(platformMap))
	if errorOccured := r.deleteVisibilities(ctx, platformMap); errorOccured != nil {
		logger.WithError(errorOccured).Error("resyncJob - could not remove visibilities from platform")
		return true
	}

	logger.Infof("resyncJob %d visibilities will be created in the platform", len(visibilitiesToCreate))
	if errorOccured := r.createVisibilities(ctx, visibilitiesToCreate); errorOccured != nil {
		logger.WithError(errorOccured).Error("resyncJob - could not create visibilities in platform")
		return true
	}

	return false
}

// deleteVisibilities deletes visibilities from platform. Returns true if error has occurred
func (r *resyncJob) deleteVisibilities(ctx context.Context, visibilities map[string]*platform.Visibility) error {
	scheduler := NewScheduler(ctx, r.options.MaxParallelRequests)

	for _, visibility := range visibilities {
		visibility := visibility
		if err := scheduler.Schedule(func(ctx context.Context) error {
			return r.deleteVisibility(ctx, visibility)
		}); err != nil {
			return err
		}
	}
	return scheduler.Await()
}

// createVisibilities creates visibilities from platform. Returns true if error has occurred
func (r *resyncJob) createVisibilities(ctx context.Context, visibilities []*platform.Visibility) error {
	scheduler := NewScheduler(ctx, r.options.MaxParallelRequests)

	for _, visibility := range visibilities {
		visibility := visibility
		if err := scheduler.Schedule(func(ctx context.Context) error {
			return r.createVisibility(ctx, visibility)
		}); err != nil {
			return err
		}
	}
	return scheduler.Await()
}

// getVisibilityKey maps a generic visibility to a specific string. The string contains catalogID and scope for non-public plans
func (r *resyncJob) getVisibilityKey(visibility *platform.Visibility) string {
	scopeLabelKey := r.platformClient.Visibility().VisibilityScopeLabelKey()

	const idSeparator = "|"
	if visibility.Public {
		return strings.Join([]string{"public", "", visibility.PlatformBrokerName, visibility.CatalogPlanID}, idSeparator)
	}
	return strings.Join([]string{"!public", visibility.Labels[scopeLabelKey], visibility.PlatformBrokerName, visibility.CatalogPlanID}, idSeparator)
}

func (r *resyncJob) createVisibility(ctx context.Context, visibility *platform.Visibility) error {
	logger := log.C(ctx)
	logger.Infof("resyncJob creating visibility for catalog plan %s from broker %s ...",
		visibility.CatalogPlanID, visibility.PlatformBrokerName)

	if err := r.platformClient.Visibility().EnableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{
		BrokerName:    visibility.PlatformBrokerName,
		CatalogPlanID: visibility.CatalogPlanID,
		Labels:        mapToLabels(visibility.Labels),
	}); err != nil {
		return err
	}
	logger.Infof("resyncJob successfully created visibility for catalog plan %s from broker %s",
		visibility.CatalogPlanID, visibility.PlatformBrokerName)

	return nil
}

func (r *resyncJob) deleteVisibility(ctx context.Context, visibility *platform.Visibility) error {
	logger := log.C(ctx)
	logger.Infof("resyncJob deleting visibility for catalog plan %s ...", visibility.CatalogPlanID)

	if err := r.platformClient.Visibility().DisableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{
		BrokerName:    visibility.PlatformBrokerName,
		CatalogPlanID: visibility.CatalogPlanID,
		Labels:        mapToLabels(visibility.Labels),
	}); err != nil {
		return err
	}
	logger.Infof("resyncJob successfully deleted visibility for catalog plan %s", visibility.CatalogPlanID)

	return nil
}

func (r *resyncJob) convertVisListToMap(list []*platform.Visibility) map[string]*platform.Visibility {
	result := make(map[string]*platform.Visibility, len(list))
	for _, vis := range list {
		key := r.getVisibilityKey(vis)
		result[key] = vis
	}
	return result
}

//if planIDs is nil or empty all visibilities will be reconciled
func (r *resyncJob) reconcilePlansVisibilities(ctx context.Context, useServicePlanFieldQuery bool, smPlans map[string]brokerPlan, smBrokers []*platform.ServiceBroker) error {
	smVisibilities, err := r.getVisibilitiesFromSM(ctx, useServicePlanFieldQuery, smPlans)
	if err != nil {
		return fmt.Errorf("an error occurred while loading visibilities from SM")
	}
	log.C(ctx).Infof("Calling platform API to fetch actual platform visibilities")
	platformVisibilities, err := r.getPlatformVisibilitiesByBrokersFromPlatform(ctx, smBrokers)
	if err != nil {
		return fmt.Errorf("an error occurred while loading visibilities from platform")
	}

	errorOccurred := r.reconcileServiceVisibilities(ctx, platformVisibilities, smVisibilities)
	if errorOccurred {
		return fmt.Errorf("could not reconcile visibilities")
	}

	return nil
}

func mapToLabels(m map[string]string) types.Labels {
	labels := types.Labels{}
	for k, v := range m {
		labels[k] = []string{
			v,
		}
	}
	return labels
}

func getBrokersPlans(brokers []*platform.ServiceBroker, brokersPlans map[string]brokerPlan) map[string]brokerPlan {
	plans := make(map[string]brokerPlan)
	for _, broker := range brokers {
		for planID, plan := range brokersPlans {
			if plan.broker != nil && plan.broker.GUID == broker.GUID {
				plans[planID] = plan
			}
		}
	}
	return plans
}

func getBrokerChunks(brokers []*platform.ServiceBroker, chunkSize int) [][]*platform.ServiceBroker {
	var chunks [][]*platform.ServiceBroker
	for i := 0; i < len(brokers); i += chunkSize {
		end := i + chunkSize

		// check to avoid slicing beyond slice capacity
		if end > len(brokers) {
			end = len(brokers)
		}

		chunks = append(chunks, brokers[i:end])
	}

	return chunks
}

type brokerPlan struct {
	*types.ServicePlan
	broker *platform.ServiceBroker
}
