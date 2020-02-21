package platform

import "context"

// Caching provides cache control
type Caching interface {

	// ResetCache invalidates the whole cache
	ResetCache(ctx context.Context) error
}
