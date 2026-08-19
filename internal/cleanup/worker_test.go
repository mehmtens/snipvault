package cleanup

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

type fakeCleaner struct{ calls atomic.Int32 }

func (f *fakeCleaner) DeleteExpired(context.Context) (int64, error) {
	f.calls.Add(1)
	return 1, nil
}

func TestWorkerRunsAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cleaner := &fakeCleaner{}
	done := make(chan struct{})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	go func() {
		Run(ctx, cleaner, 5*time.Millisecond, logger)
		close(done)
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
	if cleaner.calls.Load() == 0 {
		t.Fatal("worker did not run cleanup")
	}
}
