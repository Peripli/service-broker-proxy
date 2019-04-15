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
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
)

const (
	platformVisibilityCacheKey = "platform_visibilities"
	smPlansCacheKey            = "sm_plans"
)

// processVisibilities handles the reconsilation of the service visibilities.
// It gets the service visibilities from Service Manager and the platform and runs the reconciliation
// nolint: gocyclo
func (r *ReconciliationTask) processVisibilities() {
	logger := log.C(r.runContext)
	if r.platformClient.Visibility() == nil {
		logger.Debug("Platform client cannot handle visibilities. Visibility reconciliation will be skipped.")
		return
	}

	brokerFromStats := r.stat(smBrokersStats)
	smBrokers, ok := brokerFromStats.([]platform.ServiceBroker)
	logger.Infof("ReconciliationTask SUCCESSFULLY retrieved %d Service Manager brokers from cache", len(smBrokers))

	if !ok {
		logger.Errorf("Expected %T to be %T", brokerFromStats, smBrokers)
		return
	}
	if len(smBrokers) == 0 {
		logger.Infof("No brokers from Service Manager found in cache. Skipping reconcile visibilities...")
		return
	}

	smOfferings, err := r.getSMServiceOfferingsByBrokers(smBrokers)
	if err != nil {
		logger.WithError(err).Error("An error occurred while obtaining service offerings from Service Manager")
		return
	}

	smPlans, err := r.getSMPlansByBrokersAndOfferings(smOfferings)
	if err != nil {
		logger.WithError(err).Error("An error occurred while obtaining plans from Service Manager")
		return
	}

	var platformVisibilities []platform.ServiceVisibilityEntity
	visibilityCacheUsed := false

	//TODO cache is always invalided? why
	if r.options.VisibilityCache && r.areSMPlansSame(smPlans) {
		logger.Infof("Actual SM plans and cached SM plans are same. Attempting to pick up cached platform visibilities...")
		platformVisibilities = r.getPlatformVisibilitiesFromCache()
		visibilityCacheUsed = true
	}

	if platformVisibilities == nil {
		logger.Infof("Actual SM plans and cached SM plans are different or cache has expired. Calling platform API to fetch actual platform visibilities...")
		platformVisibilities, err = r.getPlatformVisibilitiesByBrokersFromPlatform(smBrokers)
		if err != nil {
			logger.WithError(err).Error("An error occurred while loading visibilities from platform")
			return
		}
	}

	plansMap := smPlansToMap(smPlans)
	smVisibilities, err := r.getSMVisibilities(plansMap, smBrokers)
	if err != nil {
		logger.WithError(err).Error("An error occurred while obtaining Service Manager visibilities")
		return
	}

	errorOccured := r.reconcileServiceVisibilities(platformVisibilities, smVisibilities)
	if errorOccured {
		logger.Error("Could not reconcile visibilities")
	}

	if r.options.VisibilityCache {
		if errorOccured {
			r.cache.Delete(platformVisibilityCacheKey)
			r.cache.Delete(smPlansCacheKey)
		} else {
			r.updateVisibilityCache(visibilityCacheUsed, plansMap, smVisibilities)
		}
	}
}

// updateVisibilityCache
func (r *ReconciliationTask) updateVisibilityCache(visibilityCacheUsed bool, plansMap map[brokerPlanKey]types.ServicePlan, visibilities []platform.ServiceVisibilityEntity) {
	log.C(r.globalContext).Infof("Updating cache with the %d newly fetched SM plans as cached-SM-plans and expiration duration %s", len(plansMap), r.options.CacheExpiration)
	r.cache.Set(smPlansCacheKey, plansMap, r.options.CacheExpiration)
	visibilitiesExpiration := r.options.CacheExpiration
	if visibilityCacheUsed {
		_, expiration, found := r.cache.GetWithExpiration(platformVisibilityCacheKey)
		if found {
			visibilitiesExpiration = time.Until(expiration)
		}
	}

	log.C(r.globalContext).Infof("Updating cache with the %d newly fetched SM visibilities as cached-platform-visibilities and expiration duration %s", len(visibilities), visibilitiesExpiration)
	r.cache.Set(platformVisibilityCacheKey, visibilities, visibilitiesExpiration)
}

// areSMPlansSame checks if there are new or deleted plans in SM.
// Returns true if there are no new or deleted plans, false otherwise
func (r *ReconciliationTask) areSMPlansSame(plans map[string][]types.ServicePlan) bool {
	cachedPlans, isPresent := r.cache.Get(smPlansCacheKey)
	if !isPresent {
		return false
	}
	cachedPlansMap, ok := cachedPlans.(map[brokerPlanKey]types.ServicePlan)
	if !ok {
		log.C(r.runContext).Error("Service Manager plans cache is in invalid state! Clearing...")
		r.cache.Delete(smPlansCacheKey)
		return false
	}

	for brokerID, brokerPlans := range plans {
		for _, plan := range brokerPlans {
			key := brokerPlanKey{
				brokerID: brokerID,
				planID:   plan.ID,
			}
			if _, ok := cachedPlansMap[key]; !ok {
				return false
			}
		}
	}

	return true
}

func (r *ReconciliationTask) getPlatformVisibilitiesFromCache() []platform.ServiceVisibilityEntity {
	platformVisibilities, found := r.cache.Get(platformVisibilityCacheKey)
	if !found {
		return nil
	}
	if result, ok := platformVisibilities.([]platform.ServiceVisibilityEntity); ok {
		log.C(r.runContext).Infof("ReconciliationTask fetched %d platform visibilities from cache", len(result))
		return result
	}
	log.C(r.runContext).Error("Platform visibilities cache is in invalid state! Clearing...")
	r.cache.Delete(platformVisibilityCacheKey)
	return nil
}

func (r *ReconciliationTask) getPlatformVisibilitiesByBrokersFromPlatform(brokers []platform.ServiceBroker) ([]platform.ServiceVisibilityEntity, error) {
	logger := log.C(r.runContext)
	logger.Debug("ReconciliationTask getting visibilities from platform")

	names := r.brokerNames(brokers)
	visibilities, err := r.platformClient.Visibility().GetVisibilitiesByBrokers(r.runContext, names)
	if err != nil {
		return nil, err
	}
	logger.Debugf("ReconciliationTask SUCCESSFULLY retrieved %d visibilities from platform", len(visibilities))

	return visibilities, nil
}

func (r *ReconciliationTask) brokerNames(brokers []platform.ServiceBroker) []string {
	names := make([]string, 0, len(brokers))
	for _, broker := range brokers {
		names = append(names, r.options.BrokerPrefix+broker.GUID)
	}
	return names
}

func (r *ReconciliationTask) getSMPlansByBrokersAndOfferings(offerings map[string][]types.ServiceOffering) (map[string][]types.ServicePlan, error) {
	result := make(map[string][]types.ServicePlan)
	count := 0
	for brokerID, sos := range offerings {
		if len(sos) == 0 {
			continue
		}
		brokerPlans, err := r.smClient.GetPlansByServiceOfferings(r.runContext, sos)
		if err != nil {
			return nil, err
		}
		result[brokerID] = brokerPlans
		count += len(brokerPlans)
	}
	log.C(r.runContext).Infof("ReconciliationTask SUCCESSFULLY retrieved %d plans from Service Manager", count)

	return result, nil
}

func (r *ReconciliationTask) getSMServiceOfferingsByBrokers(brokers []platform.ServiceBroker) (map[string][]types.ServiceOffering, error) {
	result := make(map[string][]types.ServiceOffering)
	brokerIDs := make([]string, 0, len(brokers))
	for _, broker := range brokers {
		brokerIDs = append(brokerIDs, broker.GUID)
	}

	offerings, err := r.smClient.GetServiceOfferingsByBrokerIDs(r.runContext, brokerIDs)
	if err != nil {
		return nil, err
	}
	log.C(r.runContext).Infof("ReconciliationTask SUCCESSFULLY retrieved %d services from Service Manager", len(offerings))

	for _, offering := range offerings {
		if result[offering.BrokerID] == nil {
			result[offering.BrokerID] = make([]types.ServiceOffering, 0)
		}
		result[offering.BrokerID] = append(result[offering.BrokerID], offering)
	}

	return result, nil
}

func (r *ReconciliationTask) getSMVisibilities(smPlansMap map[brokerPlanKey]types.ServicePlan, smBrokers []platform.ServiceBroker) ([]platform.ServiceVisibilityEntity, error) {
	logger := log.C(r.runContext)
	logger.Info("ReconciliationTask getting visibilities from Service Manager...")

	visibilities, err := r.smClient.GetVisibilities(r.runContext)
	if err != nil {
		return nil, err
	}
	logger.Infof("ReconciliationTask SUCCESSFULLY retrieved %d visibilities from Service Manager", len(visibilities))

	result := make([]platform.ServiceVisibilityEntity, 0)

	for visIndex := range visibilities {
		for _, broker := range smBrokers {
			key := brokerPlanKey{
				brokerID: broker.GUID,
				planID:   visibilities[visIndex].ServicePlanID,
			}
			smPlan, found := smPlansMap[key]
			if !found {
				continue
			}
			converted := r.convertSMVisibility(visibilities[visIndex], smPlan, broker.GUID)
			result = append(result, converted...)
		}
	}
	logger.Infof("ReconciliationTask SUCCESSFULLY converted %d Service Manager visibilities to %d platform visibilities", len(visibilities), len(result))

	return result, nil
}

func (r *ReconciliationTask) convertSMVisibility(visibility types.Visibility, smPlan types.ServicePlan, brokerGUID string) []platform.ServiceVisibilityEntity {
	scopeLabelKey := r.platformClient.Visibility().VisibilityScopeLabelKey()

	if visibility.PlatformID == "" || scopeLabelKey == "" {
		return []platform.ServiceVisibilityEntity{
			{
				Public:             true,
				CatalogPlanID:      smPlan.CatalogID,
				PlatformBrokerName: r.options.BrokerPrefix + brokerGUID,
				Labels:             map[string]string{},
			},
		}
	}

	scopes := visibility.Labels[scopeLabelKey]
	result := make([]platform.ServiceVisibilityEntity, 0, len(scopes))
	for _, scope := range scopes {
		result = append(result, platform.ServiceVisibilityEntity{
			Public:             false,
			CatalogPlanID:      smPlan.CatalogID,
			PlatformBrokerName: r.options.BrokerPrefix + brokerGUID,
			Labels:             map[string]string{scopeLabelKey: scope},
		})
	}

	return result
}

func (r *ReconciliationTask) reconcileServiceVisibilities(platformVis, smVis []platform.ServiceVisibilityEntity) bool {
	logger := log.C(r.runContext)
	logger.Info("ReconciliationTask reconciling platform and Service Manager visibilities...")

	platformMap := r.convertVisListToMap(platformVis)
	visibilitiesToCreate := make([]platform.ServiceVisibilityEntity, 0)

	for index := range smVis {
		key := r.getVisibilityKey(smVis[index])
		_, ok := platformMap[key]
		delete(platformMap, key)
		if !ok {
			visibilitiesToCreate = append(visibilitiesToCreate, smVis[index])
		}
	}

	logger.Infof("ReconciliationTask %d visibilities will be removed from the platform", len(platformMap))
	if errorOccured := r.deleteVisibilities(platformMap); errorOccured != nil {
		logger.WithError(errorOccured).Error("ReconciliationTask - could not remove visibilities from platform")
		return true
	}

	logger.Infof("ReconciliationTask %d visibilities will be created in the platform", len(visibilitiesToCreate))
	if errorOccured := r.createVisibilities(visibilitiesToCreate); errorOccured != nil {
		logger.WithError(errorOccured).Error("ReconciliationTask - could not create visibilities in platform")
		return true
	}

	return false
}

type visibilityProcessingState struct {
	Mutex          sync.Mutex
	Ctx            context.Context
	StopProcessing context.CancelFunc
	WaitGroup      sync.WaitGroup
	ErrorOccured   error
}

func (r *ReconciliationTask) newVisibilityProcessingState() *visibilityProcessingState {
	visibilitiesContext, cancel := context.WithCancel(r.runContext)
	return &visibilityProcessingState{
		Ctx:            visibilitiesContext,
		StopProcessing: cancel,
	}
}

// deleteVisibilities deletes visibilities from platform. Returns true if error has occurred
func (r *ReconciliationTask) deleteVisibilities(visibilities map[string]platform.ServiceVisibilityEntity) error {
	state := r.newVisibilityProcessingState()
	defer state.StopProcessing()

	for _, visibility := range visibilities {
		visibility := visibility

		execAsync(state, visibility, r.deleteVisibility)
	}
	return await(state)
}

// createVisibilities creates visibilities from platform. Returns true if error has occurred
func (r *ReconciliationTask) createVisibilities(visibilities []platform.ServiceVisibilityEntity) error {
	state := r.newVisibilityProcessingState()
	defer state.StopProcessing()

	for _, visibility := range visibilities {
		visibility := visibility

		execAsync(state, visibility, r.createVisibility)
	}

	return await(state)
}

func execAsync(state *visibilityProcessingState, visibility platform.ServiceVisibilityEntity, f func(context.Context, platform.ServiceVisibilityEntity) error) {
	state.WaitGroup.Add(1)

	go func() {
		defer state.WaitGroup.Done()
		err := f(state.Ctx, visibility)
		if err == nil {
			return
		}
		state.Mutex.Lock()
		defer state.Mutex.Unlock()
		if state.ErrorOccured == nil {
			state.ErrorOccured = err
		}
	}()
}

func await(state *visibilityProcessingState) error {
	state.WaitGroup.Wait()
	return state.ErrorOccured
}

// getVisibilityKey maps a generic visibility to a specific string. The string contains catalogID and scope for non-public plans
func (r *ReconciliationTask) getVisibilityKey(visibility platform.ServiceVisibilityEntity) string {
	scopeLabelKey := r.platformClient.Visibility().VisibilityScopeLabelKey()

	const idSeparator = "|"
	if visibility.Public {
		return strings.Join([]string{"public", "", visibility.PlatformBrokerName, visibility.CatalogPlanID}, idSeparator)
	}
	return strings.Join([]string{"!public", visibility.Labels[scopeLabelKey], visibility.PlatformBrokerName, visibility.CatalogPlanID}, idSeparator)
}

func (r *ReconciliationTask) createVisibility(ctx context.Context, visibility platform.ServiceVisibilityEntity) error {
	logger := log.C(r.runContext)
	logger.Infof("Reconciliation task attempting to create visibility for catalog plan %s with labels %v", visibility.CatalogPlanID, visibility.Labels)

	json, err := marshalVisibilityLabels(logger, visibility.Labels)
	if err != nil {
		return err
	}
	if err = r.platformClient.Visibility().EnableAccessForPlan(ctx, json, visibility.CatalogPlanID, visibility.PlatformBrokerName); err != nil {
		logger.WithError(err).Errorf("Could not enable access for plan %s", visibility.CatalogPlanID)
		return err
	}
	logger.Infof("Reconciliation task SUCCESSFULLY created visibility for catalog plan %s with labels %v", visibility.CatalogPlanID, visibility.Labels)

	return nil
}

func (r *ReconciliationTask) deleteVisibility(ctx context.Context, visibility platform.ServiceVisibilityEntity) error {
	logger := log.C(r.runContext)
	logger.Infof("Reconciliation task attempting to delete visibility for catalog plan %s with labels %v", visibility.CatalogPlanID, visibility.Labels)

	json, err := marshalVisibilityLabels(logger, visibility.Labels)
	if err != nil {
		return err
	}
	if err = r.platformClient.Visibility().DisableAccessForPlan(ctx, json, visibility.CatalogPlanID, visibility.PlatformBrokerName); err != nil {
		logger.WithError(err).Errorf("Could not disable access for plan %s", visibility.CatalogPlanID)
		return err
	}
	logger.Infof("Reconciliation task SUCCESSFULLY deleted visibility for catalog plan %s with labels %v", visibility.CatalogPlanID, visibility.Labels)

	return nil
}

func marshalVisibilityLabels(logger *logrus.Entry, labels map[string]string) ([]byte, error) {
	json, err := json.Marshal(labels)
	if err != nil {
		logger.WithError(err).Error("Could not marshal labels to json")
	}
	return json, err
}

func (r *ReconciliationTask) convertVisListToMap(list []platform.ServiceVisibilityEntity) map[string]platform.ServiceVisibilityEntity {
	result := make(map[string]platform.ServiceVisibilityEntity, len(list))
	for index := range list {
		key := r.getVisibilityKey(list[index])
		result[key] = list[index]
	}

	return result
}

func smPlansToMap(plansByBroker map[string][]types.ServicePlan) map[brokerPlanKey]types.ServicePlan {
	plansMap := make(map[brokerPlanKey]types.ServicePlan, len(plansByBroker))

	for brokerID, brokerPlans := range plansByBroker {
		for index := range brokerPlans {
			key := brokerPlanKey{
				brokerID: brokerID,
				planID:   brokerPlans[index].ID,
			}
			plansMap[key] = brokerPlans[index]
		}
	}

	return plansMap
}

type brokerPlanKey struct {
	brokerID string
	planID   string
}
