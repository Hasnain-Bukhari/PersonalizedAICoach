package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// The worker process is the deployment boundary for outbox consumption,
// ingestion, scheduled generation, and notification delivery. Adapters are
// deliberately injected in the same way as the API; this bootstrap keeps a
// healthy process in the dependency-free local profile.
func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	log.Info("worker ready")
	for {
		select {
		case <-ctx.Done():
			log.Info("worker stopped")
			return
		case <-ticker.C:
			log.Debug("worker heartbeat")
		}
	}
}
