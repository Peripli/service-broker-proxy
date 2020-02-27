package platform

import "context"

// Caching provides cache control
type Caching interface {

	// ResetCache invalidates the whole cache
	ResetCache(ctx context.Context) error

	// ResetBroker resets the data for the given broker
	ResetBroker(ctx context.Context, broker *ServiceBroker, deleted bool) error
}
