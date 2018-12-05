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

const PlatformVisibilityCacheKey = "platform_visibilities"

func (r ReconcilationTask) getPlatformVisibilitiesWithCache(plans []*types.ServicePlan, updateCache bool) ([]*platform.ServiceVisibilityEntity, error) {
	logger := log.C(r.ctx)
	visibilities, found := r.cache.Get(PlatformVisibilityCacheKey)
	if !updateCache && found {
		result, ok := visibilities.([]*platform.ServiceVisibilityEntity)
		if !ok {
			return nil, errors.New("could not cast cached visibilities to core visibilities")
		}
		logger.Debugf("%d platform visibilities found in cache", len(result))

		return result, nil
	}

	platformVisibilities, err := r.getPlatformVisibilities(plans)
	if err != nil {
		return nil, err
	}
	// TODO: Extract expiration time
	r.cache.Set(PlatformVisibilityCacheKey, platformVisibilities, time.Minute*60)

	return platformVisibilities, nil
}

func (r ReconcilationTask) getPlatformVisibilities(plans []*types.ServicePlan) ([]*platform.ServiceVisibilityEntity, error) {
	logger := log.C(r.ctx)
	logger.Debug("ReconcilationTask getting visibilities from platform")

	visibilitiesClient, ok := r.platformClient.(platform.ServiceVisibility)
	if !ok {
		return nil, errors.New("not a visibilities client")
	}

	platformVisibilities, err := visibilitiesClient.GetVisibilitiesByPlans(r.ctx, plans)
	if err != nil {
		return nil, err
	}
	logger.Debugf("ReconcilationTask successfully retrieved %d visibilities from platform", len(platformVisibilities))

	return platformVisibilities, nil
}

func (r ReconcilationTask) getSMPlans() ([]*types.ServicePlan, error) {
	logger := log.C(r.ctx)
	logger.Debug("ReconcilationTask getting plans from Service Manager")

	plans, err := r.smClient.GetPlans(r.ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error gettings plans from SM")
	}
	logger.Debugf("ReconcilationTask successfully retrieved %d plans from Service Manager", len(plans))

	return plans, nil
}

func (r ReconcilationTask) getSMVisibilities(smPlansMap map[string]*types.ServicePlan) ([]*platform.ServiceVisibilityEntity, error) {
	logger := log.C(r.ctx)
	logger.Debug("ReconcilationTask getting visibilities from Service Manager")

	visibilities, err := r.smClient.GetVisibilities(r.ctx)
	if err != nil {
		return nil, err
	}
	logger.Debugf("ReconcilationTask successfully retrieved %d visibilities from Service Manager", len(visibilities))

	result := make([]*platform.ServiceVisibilityEntity, 0)
	converter, ok := r.platformClient.(platform.SMVisibilityConverter)
	if ok {
		for _, visibility := range visibilities {
			converted, err := converter.Convert(visibility, smPlansMap[visibility.ServicePlanID])
			if err != nil {
				logger.Error(err)
			}
			result = append(result, converted...)
		}
	}
	logger.Debugf("ReconcilationTask successfully converted %d SM visibilities to %d platform visibilities", len(visibilities), len(result))

	return result, nil
}

func (r ReconcilationTask) reconcileServiceVisibilities(platformVis, smVis []*platform.ServiceVisibilityEntity) {
	platformMap := r.convertVisListToMap(platformVis)
	newlyCreatedVisiblities := make([]*platform.ServiceVisibilityEntity, 0)
	for _, vis := range smVis {
		key := r.visibilityKeyMapper.Map(vis)
		existingVis := platformMap[key]
		delete(platformMap, key)
		if existingVis == nil {
			newlyCreatedVisiblities = append(newlyCreatedVisiblities, vis)
			r.createVisibility(vis)
		}
	}

	if len(newlyCreatedVisiblities) > 0 {
		newlyCreatedVisiblities = append(newlyCreatedVisiblities, platformVis...)
		_, expirationTime, found := r.cache.GetWithExpiration(PlatformVisibilityCacheKey)
		// TODO: Extract constant
		duration := time.Minute * 60
		if found {
			duration = expirationTime.Sub(time.Now())
		}
		r.cache.Set(PlatformVisibilityCacheKey, newlyCreatedVisiblities, duration)
	}

	for _, vis := range platformMap {
		r.deleteVisibility(vis)
	}
}

func (r ReconcilationTask) createVisibility(visibility *platform.ServiceVisibilityEntity) {
	log.C(r.ctx).Debugf("Creating visibility for %s with metadata %v", visibility.CatalogPlanID, visibility.Labels)
	r.enableServiceAccessVisibility(visibility)
}

func (r ReconcilationTask) deleteVisibility(visibility *platform.ServiceVisibilityEntity) {
	logger := log.C(r.ctx)
	logger.Debugf("Deleting visibility for %s with metadata %v", visibility.CatalogPlanID, visibility.Labels)

	if f, isEnabler := r.platformClient.(platform.ServiceAccess); isEnabler {
		json, err := json.Marshal(visibility.Labels)
		if err != nil {
			logger.WithError(err).Error("Could not marshal labels to json")
			return
		}

		if err = f.DisableAccessForPlan(r.ctx, json, visibility.CatalogPlanID); err != nil {
			logger.WithError(err).Errorf("Could not disable access for plan %s", visibility.CatalogPlanID)
		}
	}
}

func (r ReconcilationTask) enableServiceAccessVisibility(visibility *platform.ServiceVisibilityEntity) {
	logger := log.C(r.ctx)

	if f, isEnabler := r.platformClient.(platform.ServiceAccess); isEnabler {
		json, err := json.Marshal(visibility.Labels)
		if err != nil {
			logger.WithError(err).Error("Could not marshal labels to json")
			return
		}
		if err = f.EnableAccessForPlan(r.ctx, json, visibility.CatalogPlanID); err != nil {
			logger.WithError(err).Errorf("Could not enable access for plan %s", visibility.CatalogPlanID)
		}
	}
}

func (r ReconcilationTask) convertVisListToMap(list []*platform.ServiceVisibilityEntity) map[string]*platform.ServiceVisibilityEntity {
	result := make(map[string]*platform.ServiceVisibilityEntity, len(list))
	for _, vis := range list {
		key := r.visibilityKeyMapper.Map(vis)
		result[key] = vis
	}
	return result
}
