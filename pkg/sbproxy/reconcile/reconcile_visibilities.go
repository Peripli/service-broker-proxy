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
	"encoding/json"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/platform"
	"github.com/Peripli/service-manager/pkg/log"
	"github.com/Peripli/service-manager/pkg/types"
	"github.com/pkg/errors"
)

const (
	platformVisibilityCacheKey = "platform_visibilities"
	smPlansCacheKey            = "sm_plans"
)

// processVisibilities handles the reconsilation of the service visibilities.
// it gets the service visibilities from SM and the platform and runs the reconciliation
func (r ReconciliationTask) processVisibilities() {
	logger := log.C(r.ctx)
	if r.getVisibilitiesClient() == nil {
		logger.Debug("Platform client cannot handle visibilities. Visibility reconciliation will be skipped.")
		return
	}

	plans, err := r.getSMPlans()
	if err != nil {
		logger.WithError(err).Error("An error occurred while obtaining plans from Service Manager")
		return
	}
	var platformVisibilities []*platform.ServiceVisibilityEntity
	if r.options.VisibilityCache {
		platformVisibilities = r.getPlatformVisibilitiesFromCache()
	}

	visibilityCacheUsed := true
	if platformVisibilities == nil || !r.areSMPlansSame(plans) {
		platformVisibilities, err = r.loadPlatformVisibilities(plans)
		if err != nil {
			logger.WithError(err).Error("An error occurred while loading visibilities from platform")
			return
		}
		visibilityCacheUsed = false
	}

	plansMap := smPlansToMap(plans)
	smVisibilities, err := r.getSMVisibilities(plansMap)
	if err != nil {
		logger.WithError(err).Error("An error occurred while obtaining SM visibilities")
	}

	errorOccured := r.reconcileServiceVisibilities(platformVisibilities, smVisibilities)

	if r.options.VisibilityCache {
		if errorOccured {
			r.cache.Delete(platformVisibilityCacheKey)
			r.cache.Delete(smPlansCacheKey)
		} else {
			r.updateVisibilityCache(visibilityCacheUsed, plansMap, smVisibilities)
		}
	}
}

func (r ReconciliationTask) getVisibilitiesClient() platform.ServiceVisibilityHandler {
	client, _ := r.platformClient.(platform.ServiceVisibilityHandler)
	return client
}

// updateVisibilityCache
func (r ReconciliationTask) updateVisibilityCache(visibilityCacheUsed bool, plansMap map[string]*types.ServicePlan, visibilities []*platform.ServiceVisibilityEntity) {
	r.cache.Set(smPlansCacheKey, plansMap, r.options.CacheExpiration)
	cacheExpiration := r.options.CacheExpiration
	if visibilityCacheUsed {
		_, expiration, found := r.cache.GetWithExpiration(platformVisibilityCacheKey)
		if found {
			cacheExpiration = time.Until(expiration)
		}
	}
	r.cache.Set(platformVisibilityCacheKey, visibilities, cacheExpiration)
}

// areSMPlansSame checks if there are new or deleted plans in SM
// returns true if there are no new or deleted plans, false otherwise
func (r ReconciliationTask) areSMPlansSame(plans []*types.ServicePlan) bool {
	cachedPlans, isPresent := r.cache.Get(smPlansCacheKey)
	if !isPresent {
		return false
	}
	cachedPlansMap, ok := cachedPlans.(map[string]*types.ServicePlan)
	if !ok {
		log.C(r.ctx).Error("SM plans cache is in invalid state! Clearing...")
		r.cache.Delete(smPlansCacheKey)
		return false
	}
	if len(cachedPlansMap) != len(plans) {
		return false
	}
	for _, plan := range plans {
		if cachedPlansMap[plan.ID] == nil {
			return false
		}
	}
	return true
}

func (r ReconciliationTask) getPlatformVisibilitiesFromCache() []*platform.ServiceVisibilityEntity {
	platformVisibilities, found := r.cache.Get(platformVisibilityCacheKey)
	if !found {
		return nil
	}
	if result, ok := platformVisibilities.([]*platform.ServiceVisibilityEntity); ok {
		log.C(r.ctx).Debugf("ReconciliationTask fetched %d platform visibilities from cache", len(result))
		return result
	}
	log.C(r.ctx).Error("Platform visibilities cache is in invalid state! Clearing...")
	r.cache.Delete(platformVisibilityCacheKey)
	return nil
}

func (r ReconciliationTask) loadPlatformVisibilities(plans []*types.ServicePlan) ([]*platform.ServiceVisibilityEntity, error) {
	logger := log.C(r.ctx)
	logger.Debug("ReconciliationTask getting visibilities from platform")

	platformVisibilities, err := r.getVisibilitiesClient().GetVisibilitiesByPlans(r.ctx, plans)
	if err != nil {
		return nil, err
	}
	logger.Debugf("ReconciliationTask successfully retrieved %d visibilities from platform", len(platformVisibilities))

	return platformVisibilities, nil
}

func (r ReconciliationTask) getSMPlans() ([]*types.ServicePlan, error) {
	logger := log.C(r.ctx)
	logger.Debug("ReconciliationTask getting plans from Service Manager")

	plans, err := r.smClient.GetPlans(r.ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error gettings plans from SM")
	}
	logger.Debugf("ReconciliationTask successfully retrieved %d plans from Service Manager", len(plans))

	return plans, nil
}

func (r ReconciliationTask) getSMVisibilities(smPlansMap map[string]*types.ServicePlan) ([]*platform.ServiceVisibilityEntity, error) {
	logger := log.C(r.ctx)
	logger.Debug("ReconciliationTask getting visibilities from Service Manager")

	visibilities, err := r.smClient.GetVisibilities(r.ctx)
	if err != nil {
		return nil, err
	}
	logger.Debugf("ReconciliationTask successfully retrieved %d visibilities from Service Manager", len(visibilities))

	visibilitiesClient := r.getVisibilitiesClient()

	result := make([]*platform.ServiceVisibilityEntity, 0)
	for _, visibility := range visibilities {
		converted := visibilitiesClient.Convert(visibility, smPlansMap[visibility.ServicePlanID])
		result = append(result, converted...)
	}
	logger.Debugf("ReconciliationTask successfully converted %d SM visibilities to %d platform visibilities", len(visibilities), len(result))

	return result, nil
}

func (r ReconciliationTask) reconcileServiceVisibilities(platformVis, smVis []*platform.ServiceVisibilityEntity) bool {
	logger := log.C(r.ctx)
	logger.Debug("ReconciliationTask reconsiling platform and SM visibilities...")

	visibilitiesClient := r.getVisibilitiesClient()
	platformMap := r.convertVisListToMap(platformVis)
	visibilitiesToCreate := make([]*platform.ServiceVisibilityEntity, 0)
	for _, visibility := range smVis {
		key := visibilitiesClient.Map(visibility)
		existingVis := platformMap[key]
		delete(platformMap, key)
		if existingVis == nil {
			visibilitiesToCreate = append(visibilitiesToCreate, visibility)
		}
	}

	errorOccured := false
	logger.Debugf("ReconciliationTask %d visibilities will be removed from the platform", len(platformMap))
	for _, visibility := range platformMap {
		if err := r.deleteVisibility(visibility); err != nil {
			errorOccured = true
		}
	}

	logger.Debugf("ReconciliationTask %d visibilities will be created in the platform", len(visibilitiesToCreate))
	for _, visibility := range visibilitiesToCreate {
		if err := r.createVisibility(visibility); err != nil {
			errorOccured = true
		}
	}
	return errorOccured
}

func (r ReconciliationTask) createVisibility(visibility *platform.ServiceVisibilityEntity) error {
	logger := log.C(r.ctx)
	logger.Debugf("Creating visibility for %s with labels %v", visibility.CatalogPlanID, visibility.Labels)

	json, err := json.Marshal(visibility.Labels)
	if err != nil {
		logger.WithError(err).Error("Could not marshal labels to json")
		return err
	}
	if err = r.getVisibilitiesClient().EnableAccessForPlan(r.ctx, json, visibility.CatalogPlanID); err != nil {
		logger.WithError(err).Errorf("Could not enable access for plan %s", visibility.CatalogPlanID)
		return err
	}
	return nil
}

func (r ReconciliationTask) deleteVisibility(visibility *platform.ServiceVisibilityEntity) error {
	logger := log.C(r.ctx)
	logger.Debugf("Deleting visibility for %s with labels %v", visibility.CatalogPlanID, visibility.Labels)

	json, err := json.Marshal(visibility.Labels)
	if err != nil {
		logger.WithError(err).Error("Could not marshal labels to json")
		return err
	}
	if err = r.getVisibilitiesClient().DisableAccessForPlan(r.ctx, json, visibility.CatalogPlanID); err != nil {
		logger.WithError(err).Errorf("Could not disable access for plan %s", visibility.CatalogPlanID)
		return err
	}
	return nil
}

func (r ReconciliationTask) convertVisListToMap(list []*platform.ServiceVisibilityEntity) map[string]*platform.ServiceVisibilityEntity {
	visibilitiesClient := r.getVisibilitiesClient()
	result := make(map[string]*platform.ServiceVisibilityEntity, len(list))
	for _, vis := range list {
		key := visibilitiesClient.Map(vis)
		result[key] = vis
	}
	return result
}

func smPlansToMap(plans []*types.ServicePlan) map[string]*types.ServicePlan {
	plansMap := make(map[string]*types.ServicePlan, len(plans))
	for _, plan := range plans {
		plansMap[plan.ID] = plan
	}
	return plansMap
}
