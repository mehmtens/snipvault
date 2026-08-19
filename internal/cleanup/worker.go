package cleanup

import (
	"context"
	"log/slog"
	"time"
)

type Cleaner interface {
	DeleteExpired(context.Context) (int64, error)
}

func Run(ctx context.Context, cleaner Cleaner, interval time.Duration, logger *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("cleanup worker stopped")
			return
		case <-ticker.C:
			deleted, err := cleaner.DeleteExpired(ctx)
			if err != nil {
				logger.Error("expired paste cleanup failed", "error", err)
				continue
			}
			if deleted > 0 {
				logger.Info("expired pastes deleted", "count", deleted)
			}
		}
	}
}
