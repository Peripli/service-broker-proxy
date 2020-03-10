package reconcile_test

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Peripli/service-broker-proxy/pkg/sbproxy/reconcile"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const maxParallelTasks = 5

var _ = Describe("TaskScheduler", func() {
	It("runs at max N tasks in parallel", func() {
		scheduler := reconcile.NewScheduler(context.Background(), maxParallelTasks)

		nTasks := maxParallelTasks * 3
		c := &counter{}
		for i := 0; i < nTasks; i++ {
			scheduler.Schedule(func(ctx context.Context) error {
				c.enter()
				defer c.leave()

				time.Sleep(100 * time.Millisecond)
				return nil
			})
		}
		Expect(scheduler.Await()).To(Succeed())
		Expect(c.maxCount).To(Equal(maxParallelTasks))
	})

	It("returns an error with the number of errors", func() {
		scheduler := reconcile.NewScheduler(context.Background(), maxParallelTasks)

		for i := 0; i < 5; i++ {
			i := i
			scheduler.Schedule(func(ctx context.Context) error {
				if i%2 == 0 {
					return fmt.Errorf("error %d", i)
				}
				return nil
			})
		}
		Expect(scheduler.Await()).To(MatchError("3 errors occurred"))
	})
})

type counter struct {
	mutex    sync.Mutex
	count    int
	maxCount int
}

func (c *counter) enter() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.count++
	if c.count > c.maxCount {
		c.maxCount = c.count
	}
}

func (c *counter) leave() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.count--
}
