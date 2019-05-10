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
	"strings"
	"sync"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
)

const (
	platformVisibilityCacheKey = "platform_visibilities"
	smPlansCacheKey            = "sm_plans"
)

func (r *resyncJob) stat(ctx context.Context, key string) interface{} {
	result, found := r.stats[key]
	if !found {
		log.C(ctx).Infof("No %s found in cache", key)
		return nil
	}

	log.C(ctx).Infof("Picked up %s from cache", key)

	return result
}

// processVisibilities handles the reconsilation of the service visibilities.
// It gets the service visibilities from Service Manager and the platform and runs the reconciliation
// nolint: gocyclo
func (r *resyncJob) processVisibilities(ctx context.Context) {
	logger := log.C(ctx)
	if r.platformClient.Visibility() == nil {
		logger.Debug("Platform client cannot handle visibilities. Visibility reconciliation will be skipped.")
		return
	}

	brokerFromStats := r.stat(ctx, smBrokersStats)
	smBrokers, ok := brokerFromStats.([]platform.ServiceBroker)
	logger.Infof("resyncJob SUCCESSFULLY retrieved %d Service Manager brokers from cache", len(smBrokers))

	if !ok {
		logger.Errorf("Expected %T to be %T", brokerFromStats, smBrokers)
		return
	}
	if len(smBrokers) == 0 {
		logger.Infof("No brokers from Service Manager found in cache. Skipping reconcile visibilities...")
		return
	}

	smOfferings, err := r.getSMServiceOfferingsByBrokers(ctx, smBrokers)
	if err != nil {
		logger.WithError(err).Error("An error occurred while obtaining service offerings from Service Manager")
		return
	}

	smPlans, err := r.getSMPlansByBrokersAndOfferings(ctx, smOfferings)
	if err != nil {
		logger.WithError(err).Error("An error occurred while obtaining plans from Service Manager")
		return
	}

	plansMap := smPlansToMap(smPlans)
	smVisibilities, err := r.getSMVisibilities(ctx, plansMap, smBrokers)
	if err != nil {
		logger.WithError(err).Error("An error occurred while obtaining Service Manager visibilities")
		return
	}

	var platformVisibilities []*platform.Visibility
	visibilityCacheUsed := false
	if r.options.VisibilityCache && r.areSMPlansSame(ctx, smPlans) {
		logger.Infof("Actual SM plans and cached SM plans are same. Attempting to pick up cached platform visibilities...")
		platformVisibilities = r.getPlatformVisibilitiesFromCache(ctx)
		visibilityCacheUsed = true
	}

	if platformVisibilities == nil {
		logger.Infof("Actual SM plans and cached SM plans are different. Invalidating cached platform visibilities and calling platform API to fetch actual platform visibilities...")
		platformVisibilities, err = r.getPlatformVisibilitiesByBrokersFromPlatform(ctx, smBrokers)
		if err != nil {
			logger.WithError(err).Error("An error occurred while loading visibilities from platform")
			return
		}
	}

	errorOccured := r.reconcileServiceVisibilities(ctx, platformVisibilities, smVisibilities)
	if errorOccured {
		logger.Error("Could not reconcile visibilities")
	}

	if r.options.VisibilityCache {
		if errorOccured {
			r.cache.Delete(platformVisibilityCacheKey)
			r.cache.Delete(smPlansCacheKey)
		} else {
			r.updateVisibilityCache(ctx, visibilityCacheUsed, plansMap, smVisibilities)
		}
	}
}

// updateVisibilityCache
func (r *resyncJob) updateVisibilityCache(ctx context.Context, visibilityCacheUsed bool, plansMap map[brokerPlanKey]*types.ServicePlan, visibilities []*platform.Visibility) {
	log.C(ctx).Infof("Updating cache with the %d newly fetched SM plans as cached-SM-plans and expiration duration %s", len(plansMap), r.options.CacheExpiration)
	r.cache.Set(smPlansCacheKey, plansMap, r.options.CacheExpiration)
	visibilitiesExpiration := r.options.CacheExpiration
	if visibilityCacheUsed {
		_, expiration, found := r.cache.GetWithExpiration(platformVisibilityCacheKey)
		if found {
			visibilitiesExpiration = time.Until(expiration)
		}
	}

	log.C(ctx).Infof("Updating cache with the %d newly fetched SM visibilities as cached-platform-visibilities and expiration duration %s", len(visibilities), visibilitiesExpiration)
	r.cache.Set(platformVisibilityCacheKey, visibilities, visibilitiesExpiration)
}

// areSMPlansSame checks if there are new or deleted plans in SM.
// Returns true if there are no new or deleted plans, false otherwise
func (r *resyncJob) areSMPlansSame(ctx context.Context, plans map[string][]*types.ServicePlan) bool {
	cachedPlans, isPresent := r.cache.Get(smPlansCacheKey)
	if !isPresent {
		return false
	}
	cachedPlansMap, ok := cachedPlans.(map[brokerPlanKey]*types.ServicePlan)
	if !ok {
		log.C(ctx).Error("Service Manager plans cache is in invalid state! Clearing...")
		r.cache.Delete(smPlansCacheKey)
		return false
	}

	for brokerID, brokerPlans := range plans {
		for _, plan := range brokerPlans {
			key := brokerPlanKey{
				brokerID: brokerID,
				planID:   plan.ID,
			}
			if cachedPlansMap[key] == nil {
				return false
			}
		}
	}
	return true
}

func (r *resyncJob) getPlatformVisibilitiesFromCache(ctx context.Context) []*platform.Visibility {
	platformVisibilities, found := r.cache.Get(platformVisibilityCacheKey)
	if !found {
		return nil
	}
	if result, ok := platformVisibilities.([]*platform.Visibility); ok {
		log.C(ctx).Infof("resyncJob fetched %d platform visibilities from cache", len(result))
		return result
	}
	log.C(ctx).Error("Platform visibilities cache is in invalid state! Clearing...")
	r.cache.Delete(platformVisibilityCacheKey)
	return nil
}

func (r *resyncJob) getPlatformVisibilitiesByBrokersFromPlatform(ctx context.Context, brokers []platform.ServiceBroker) ([]*platform.Visibility, error) {
	logger := log.C(ctx)
	logger.Debug("resyncJob getting visibilities from platform")

	names := r.brokerNames(brokers)
	visibilities, err := r.platformClient.Visibility().GetVisibilitiesByBrokers(ctx, names)
	if err != nil {
		return nil, err
	}
	logger.Debugf("resyncJob SUCCESSFULLY retrieved %d visibilities from platform", len(visibilities))

	return visibilities, nil
}

func (r *resyncJob) brokerNames(brokers []platform.ServiceBroker) []string {
	names := make([]string, 0, len(brokers))
	for _, broker := range brokers {
		names = append(names, r.options.BrokerPrefix+broker.Name)
	}
	return names
}

func (r *resyncJob) getSMPlansByBrokersAndOfferings(ctx context.Context, offerings map[string][]*types.ServiceOffering) (map[string][]*types.ServicePlan, error) {
	result := make(map[string][]*types.ServicePlan)
	count := 0
	for brokerID, sos := range offerings {
		if len(sos) == 0 {
			continue
		}
		brokerPlans, err := r.smClient.GetPlansByServiceOfferings(ctx, sos)
		if err != nil {
			return nil, err
		}
		result[brokerID] = brokerPlans
		count += len(brokerPlans)
	}
	log.C(ctx).Infof("resyncJob SUCCESSFULLY retrieved %d plans from Service Manager", count)

	return result, nil
}

func (r *resyncJob) getSMServiceOfferingsByBrokers(ctx context.Context, brokers []platform.ServiceBroker) (map[string][]*types.ServiceOffering, error) {
	result := make(map[string][]*types.ServiceOffering)
	brokerIDs := make([]string, 0, len(brokers))
	for _, broker := range brokers {
		brokerIDs = append(brokerIDs, broker.GUID)
	}

	offerings, err := r.smClient.GetServiceOfferingsByBrokerIDs(ctx, brokerIDs)
	if err != nil {
		return nil, err
	}
	log.C(ctx).Infof("resyncJob SUCCESSFULLY retrieved %d services from Service Manager", len(offerings))

	for _, offering := range offerings {
		if result[offering.BrokerID] == nil {
			result[offering.BrokerID] = make([]*types.ServiceOffering, 0)
		}
		result[offering.BrokerID] = append(result[offering.BrokerID], offering)
	}

	return result, nil
}

func (r *resyncJob) getSMVisibilities(ctx context.Context, smPlansMap map[brokerPlanKey]*types.ServicePlan, smBrokers []platform.ServiceBroker) ([]*platform.Visibility, error) {
	logger := log.C(ctx)
	logger.Info("resyncJob getting visibilities from Service Manager...")

	visibilities, err := r.smClient.GetVisibilities(ctx)
	if err != nil {
		return nil, err
	}
	logger.Infof("resyncJob SUCCESSFULLY retrieved %d visibilities from Service Manager", len(visibilities))

	result := make([]*platform.Visibility, 0)

	for _, visibility := range visibilities {
		for _, broker := range smBrokers {
			key := brokerPlanKey{
				brokerID: broker.GUID,
				planID:   visibility.ServicePlanID,
			}
			smPlan, found := smPlansMap[key]
			if !found {
				continue
			}
			converted := r.convertSMVisibility(visibility, smPlan, broker.Name)
			result = append(result, converted...)
		}
	}
	logger.Infof("resyncJob SUCCESSFULLY converted %d Service Manager visibilities to %d platform visibilities", len(visibilities), len(result))

	return result, nil
}

func (r *ReconciliationTask) convertSMVisibility(visibility *types.Visibility, smPlan *types.ServicePlan, brokerName string) []*platform.ServiceVisibilityEntity {
	scopeLabelKey := r.platformClient.Visibility().VisibilityScopeLabelKey()

	if visibility.PlatformID == "" || scopeLabelKey == "" {
		return []*platform.Visibility{
			{
				Public:             true,
				CatalogPlanID:      smPlan.CatalogID,
				PlatformBrokerName: r.options.BrokerPrefix + brokerName,
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
			PlatformBrokerName: r.options.BrokerPrefix + brokerName,
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

type visibilityProcessingState struct {
	Mutex          sync.Mutex
	Ctx            context.Context
	StopProcessing context.CancelFunc
	WaitGroup      sync.WaitGroup
	ErrorOccured   error
}

func (r *resyncJob) newVisibilityProcessingState(ctx context.Context) *visibilityProcessingState {
	visibilitiesContext, cancel := context.WithCancel(ctx)
	return &visibilityProcessingState{
		Ctx:            visibilitiesContext,
		StopProcessing: cancel,
	}
}

// deleteVisibilities deletes visibilities from platform. Returns true if error has occurred
func (r *resyncJob) deleteVisibilities(ctx context.Context, visibilities map[string]*platform.Visibility) error {
	state := r.newVisibilityProcessingState(ctx)
	defer state.StopProcessing()

	for _, visibility := range visibilities {
		execAsync(state, visibility, r.deleteVisibility)
	}
	return await(state)
}

// createVisibilities creates visibilities from platform. Returns true if error has occurred
func (r *resyncJob) createVisibilities(ctx context.Context, visibilities []*platform.Visibility) error {
	state := r.newVisibilityProcessingState(ctx)
	defer state.StopProcessing()

	for _, visibility := range visibilities {
		execAsync(state, visibility, r.createVisibility)
	}
	return await(state)
}

func execAsync(state *visibilityProcessingState, visibility *platform.Visibility, f func(context.Context, *platform.Visibility) error) {
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
	logger.Infof("resyncJob creating visibility for catalog plan %s with labels %v...", visibility.CatalogPlanID, visibility.Labels)

	if err := r.platformClient.Visibility().EnableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{
		BrokerName:    visibility.PlatformBrokerName,
		CatalogPlanID: visibility.CatalogPlanID,
		Labels:        mapToLabels(visibility.Labels),
	}); err != nil {
		return err
	}
	logger.Infof("resyncJob SUCCESSFULLY created visibility for catalog plan %s with labels %v", visibility.CatalogPlanID, visibility.Labels)

	return nil
}

func (r *resyncJob) deleteVisibility(ctx context.Context, visibility *platform.Visibility) error {
	logger := log.C(ctx)
	logger.Infof("resyncJob deleting visibility for catalog plan %s with labels %v...", visibility.CatalogPlanID, visibility.Labels)

	if err := r.platformClient.Visibility().DisableAccessForPlan(ctx, &platform.ModifyPlanAccessRequest{
		BrokerName:    visibility.PlatformBrokerName,
		CatalogPlanID: visibility.CatalogPlanID,
		Labels:        mapToLabels(visibility.Labels),
	}); err != nil {
		return err
	}
	logger.Infof("resyncJob SUCCESSFULLY deleted visibility for catalog plan %s with labels %v", visibility.CatalogPlanID, visibility.Labels)

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

func smPlansToMap(plansByBroker map[string][]*types.ServicePlan) map[brokerPlanKey]*types.ServicePlan {
	plansMap := make(map[brokerPlanKey]*types.ServicePlan, len(plansByBroker))
	for brokerID, brokerPlans := range plansByBroker {
		for _, plan := range brokerPlans {
			key := brokerPlanKey{
				brokerID: brokerID,
				planID:   plan.ID,
			}
			plansMap[key] = plan
		}
	}
	return plansMap
}

type brokerPlanKey struct {
	brokerID string
	planID   string
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
