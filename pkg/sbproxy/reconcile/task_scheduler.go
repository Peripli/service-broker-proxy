package reconcile

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/Peripli/service-manager/pkg/log"
)

//TaskScheduler schedules tasks to be executed in parallel
type TaskScheduler struct {
	ctx        context.Context
	errorCount uint32

	waitGroupLimit chan struct{}
	wg             sync.WaitGroup
}

//NewScheduler return a new task scheduler configured to execute a maximum of maxParallelTasks concurrently
func NewScheduler(ctx context.Context, maxParallelTasks int) *TaskScheduler {
	return &TaskScheduler{
		ctx:            ctx,
		waitGroupLimit: make(chan struct{}, maxParallelTasks),
	}
}

//Schedule schedules the specified task to be executed
func (state *TaskScheduler) Schedule(f func(context.Context) error) error {
	select {
	case <-state.ctx.Done():
		return state.ctx.Err()
	case state.waitGroupLimit <- struct{}{}:
	}
	state.wg.Add(1)
	go func() {
		defer func() {
			<-state.waitGroupLimit
			state.wg.Done()
		}()

		if err := f(state.ctx); err != nil {
			log.C(state.ctx).Error(err)
			atomic.AddUint32(&state.errorCount, 1)
		}
	}()

	return nil
}

//Await waits for the completion of all scheduled tasks
func (state *TaskScheduler) Await() error {
	state.wg.Wait()
	if state.errorCount > 0 {
		return fmt.Errorf("%d errors occurred", state.errorCount)
	}

	return nil
}
