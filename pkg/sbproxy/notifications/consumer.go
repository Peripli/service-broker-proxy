package notifications

import (
	"context"
	"encoding/json"

	"github.com/Peripli/service-manager/pkg/log"

	"github.com/Peripli/service-manager/pkg/types"
)

// ResourceNotificationHandler can handle notifications by processing the Payload
type ResourceNotificationHandler interface {
	// OnCreate is called when a notification for creating a resource arrives
	OnCreate(ctx context.Context, payload json.RawMessage) error

	// OnUpdate is called when a notification for modifying a resource arrives
	OnUpdate(ctx context.Context, payload json.RawMessage) error

	// OnDelete is called when a notification for deleting a resource arrives
	OnDelete(ctx context.Context, payload json.RawMessage) error
}

// Consumer allows consuming notifications by picking the correct handler to process it
type Consumer struct {
	Handlers map[types.ObjectType]ResourceNotificationHandler
}

// Consume consumes a notification and passes it to the correct handler for further processing
func (c *Consumer) Consume(ctx context.Context, n *types.Notification) *types.NotificationResult {
	notificationHandler, found := c.Handlers[n.Resource]

	if !found {
		log.C(ctx).Warnf("No notification handler found for notification for resource %s. Ignoring notification...", n.Resource)
		return nil
	}

	correlationID := n.CorrelationID
	if correlationID == "" {
		correlationID = n.ID
	}
	entry := log.C(ctx).WithField(log.FieldCorrelationID, correlationID)
	ctx = log.ContextWithLogger(ctx, entry)
	var err error
	switch n.Type {
	case types.CREATED:
		err = notificationHandler.OnCreate(ctx, n.Payload)
	case types.MODIFIED:
		err = notificationHandler.OnUpdate(ctx, n.Payload)
	case types.DELETED:
		err = notificationHandler.OnDelete(ctx, n.Payload)
	}
	errorString := ""
	if err != nil {
		errorString = err.Error()
	}

	return &types.NotificationResult{
		Successful:   err == nil,
		Error:        errorString,
		Notification: n,
	}
}
