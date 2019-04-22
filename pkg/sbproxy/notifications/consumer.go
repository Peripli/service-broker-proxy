package notifications

import (
	"context"
	"encoding/json"

	"github.com/Peripli/service-manager/pkg/types"
)

// OperationType is the notification type
type OperationType string

const (
	// CREATED represents a notification type for creating a resource
	CREATED OperationType = "CREATED"

	// MODIFIED represents a notification type for modifying a resource
	MODIFIED OperationType = "MODIFIED"

	// DELETED represents a notification type for deleting a resource
	DELETED OperationType = "DELETED"
)

// ResourceNotificationHandler can handle notifications by processing the payload
type ResourceNotificationHandler interface {
	// OnCreate is called when a notification for creating a resource arrives
	OnCreate(ctx context.Context, payload json.RawMessage)

	// OnUpdate is called when a notification for modifying a resource arrives
	OnUpdate(ctx context.Context, payload json.RawMessage)

	// OnDelete is called when a notification for deleting a resource arrives
	OnDelete(ctx context.Context, payload json.RawMessage)
}

// Consumer allows consuming notifications by picking the correct handler to process it
type Consumer struct {
	Handlers map[string]ResourceNotificationHandler
}

// Consume consumes a notification and passes it to the correct handler for further processing
func (c *Consumer) Consume(ctx context.Context, n *types.Notification) {
	notificationHandler := c.Handlers[n.Resource]

	t := OperationType(n.Type)
	switch t {
	case CREATED:
		notificationHandler.OnCreate(ctx, n.Payload)
	case MODIFIED:
		notificationHandler.OnUpdate(ctx, n.Payload)
	case DELETED:
		notificationHandler.OnDelete(ctx, n.Payload)
	}
}
