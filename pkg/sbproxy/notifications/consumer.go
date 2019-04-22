package notifications

import (
	"context"
	"encoding/json"

	"github.com/Peripli/service-manager/pkg/types"
)

type OperationType string

const (
	CREATED  OperationType = "CREATED"
	MODIFIED OperationType = "MODIFIED"
	DELETED  OperationType = "DELETED"
)

type ResourceNotificationHandler interface {
	OnCreate(ctx context.Context, payload json.RawMessage)
	OnUpdate(ctx context.Context, payload json.RawMessage)
	OnDelete(ctx context.Context, payload json.RawMessage)
}

type Consumer struct {
	Handlers map[string]ResourceNotificationHandler
}

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
