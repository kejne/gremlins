package execution

import (
	"context"
	"sync"
)

// RunWithCancel runs runner with a child context and waits for it to finish.
// onCancel is called when the parent context is cancelled.
func RunWithCancel(ctx context.Context, runner func(context.Context) error, onCancel func()) (bool, error) {
	child, cancel := context.WithCancel(ctx)
	defer cancel()
	finished := make(chan struct{})
	monitorDone := make(chan struct{})
	var wasCancelled bool
	var notify sync.Once
	notifyCancellation := func() {
		notify.Do(func() {
			wasCancelled = true
			if onCancel != nil {
				onCancel()
			}
		})
	}
	go func() {
		defer close(monitorDone)
		select {
		case <-ctx.Done():
			cancel()
			notifyCancellation()
		case <-finished:
		}
	}()
	err := runner(child)
	close(finished)
	if ctx.Err() != nil {
		cancel()
		notifyCancellation()
	}
	cancel()
	<-monitorDone

	return wasCancelled, err
}
