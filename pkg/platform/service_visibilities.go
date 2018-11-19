package platform

import (
	"context"

	"github.com/Peripli/service-broker-proxy/pkg/platform/paging"
)

type ServiceVisibility interface {
	GetAllVisibilities(context.Context) (paging.Pager, error)
}
