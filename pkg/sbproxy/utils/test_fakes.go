package utils

import (
	"context"
	"github.com/Peripli/service-manager/pkg/types"
	"sync"
)

type FakeResyncer struct {
	mutex       sync.Mutex
	resyncCount int
}

func (r *FakeResyncer) Resync(ctx context.Context, resyncVisibilities bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.resyncCount++
}

func (r *FakeResyncer) GetResyncCount() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.resyncCount
}

type FakeConsumer struct {
	mutex                 sync.Mutex
	consumedNotifications []*types.Notification
}

func (c *FakeConsumer) Consume(ctx context.Context, notification *types.Notification) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.consumedNotifications = append(c.consumedNotifications, notification)
}

func (c *FakeConsumer) GetConsumedNotifications() []*types.Notification {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.consumedNotifications
}
