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

package platform

import (
	"context"
	"encoding/json"

	"github.com/Peripli/service-manager/pkg/types"
)

// ServiceVisibilityHandler interface for platform clients to implement if they support
// platform specific service and plan visibilities
//go:generate counterfeiter . ServiceVisibilityHandler
type ServiceVisibilityHandler interface {
	// GetVisibilitiesByPlans get currently available visibilities in the platform for a specific plans
	GetVisibilitiesByPlans(context.Context, []*types.ServicePlan) ([]*ServiceVisibilityEntity, error)

	// Convert translates visibility from Service Manager and the referenced plan to array of generic visibilities.
	// These generic visibilities will then be reconciled with the visibilities taken from the platform
	Convert(*types.Visibility, *types.ServicePlan) []*ServiceVisibilityEntity

	// Map maps a generic visibility to a specific string. These strings will be compared when reconciling
	// the visibilities taken from the platform and Service Manager
	Map(*ServiceVisibilityEntity) string

	// EnableAccessForPlan enables the access to the plan with the specified GUID for
	// the entities in the data
	EnableAccessForPlan(ctx context.Context, data json.RawMessage, servicePlanGUID string) error

	// DisableAccessForPlan disables the access to the plan with the specified GUID for
	// the entities in the data
	DisableAccessForPlan(ctx context.Context, data json.RawMessage, servicePlanGUID string) error
}
