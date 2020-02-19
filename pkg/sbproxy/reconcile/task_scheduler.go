package reconcile

import (
	"context"
	"sync"
)

//TaskScheduler schedules tasks to be executed in parallel
type TaskScheduler struct {
	ctx            context.Context
	mutex          sync.Mutex
	errorsCocurred *CompositeError

	waitGroupLimit chan struct{}
	wg             sync.WaitGroup
}

//NewScheduler return a new task scheduler configured to execute a maximum of maxParallelTasks concurrently
func NewScheduler(ctx context.Context, maxParallelTasks int) *TaskScheduler {
	return &TaskScheduler{
		ctx:            ctx,
		errorsCocurred: &CompositeError{},
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
			state.mutex.Lock()
			defer state.mutex.Unlock()
			state.errorsCocurred.Add(err)
		}
	}()

	return nil
}

//Await waits for the completion of all scheduled tasks
func (state *TaskScheduler) Await() error {
	state.wg.Wait()
	if state.errorsCocurred.Len() != 0 {
		return state.errorsCocurred
	}

	return nil
}
