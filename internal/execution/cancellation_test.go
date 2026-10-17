package execution

import (
	"context"
	"errors"
	"testing"
)

func TestRunWithCancel(t *testing.T) {
	t.Run("returns runner error", func(t *testing.T) {
		want := errors.New("runner failed")
		cancelled, err := RunWithCancel(context.Background(), func(context.Context) error { return want }, nil)
		if cancelled || !errors.Is(err, want) {
			t.Fatalf("got cancelled=%v err=%v, want false and %v", cancelled, err, want)
		}
	})

	t.Run("cancels runner and notifies caller", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		notified := false
		cancelled, err := RunWithCancel(ctx, func(runCtx context.Context) error {
			cancel()
			<-runCtx.Done()

			return runCtx.Err()
		}, func() { notified = true })
		if !cancelled || !notified || !errors.Is(err, context.Canceled) {
			t.Fatalf("got cancelled=%v notified=%v err=%v", cancelled, notified, err)
		}
	})
}
