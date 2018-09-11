package platform

import (
	"context"
	"encoding/json"
)

// ServiceAccess provides a way to add a hook for a platform specific way of enabling and disabling
// service and plan access.
//go:generate counterfeiter . ServiceAccess
type ServiceAccess interface {

	// EnableAccessForService enables the access to all plans of the service with the specified GUID
	// for the entities in the context
	EnableAccessForService(ctx context.Context, context json.RawMessage, serviceGUID string) error

	// EnableAccessForPlan enables the access to the plan with the specified GUID for
	// the entities in the context
	EnableAccessForPlan(ctx context.Context, context json.RawMessage, servicePlanGUID string) error

	// DisableAccessForService disables the access to all plans of the service with the specified GUID
	// for the entities in the context
	DisableAccessForService(ctx context.Context, context json.RawMessage, serviceGUID string) error

	// DisableAccessForPlan disables the access to the plan with the specified GUID for
	// the entities in the context
	DisableAccessForPlan(ctx context.Context, context json.RawMessage, servicePlanGUID string) error
}
