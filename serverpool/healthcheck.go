package serverpool

import (
	"context"
	"load-balancer/backend"
	"load-balancer/utils"
	"time"

	"go.uber.org/zap"
)

// HealthCheck iterates over all backends in the server pool and updates their alive status.
// It runs asynchronously for each backend with a timeout defined in the load balancer config.
// The results are sent through a channel, and the backend status is updated accordingly.
// If the context is canceled, the health check exits gracefully.
func HealthCheck(ctx context.Context, s ServerPool, logger *zap.Logger) {
	config, err := utils.GetLBConfig()
	if err != nil {
		logger.Error("failed to get config for health check", zap.Error(err))
		return
	}

	// Channel for receiving health check results from goroutines
	ch := make(chan struct {
		b     backend.Backend
		alive bool
	}, len(s.GetBackends()))

	// Launch concurrent health checks for each backend
	for _, b := range s.GetBackends() {
		go func(b backend.Backend) {
			// Use a context with timeout for the backend check
			reqCtx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(config.BackendTimeout))
			defer cancel()

			alive := backend.CheckBackendHealth(reqCtx, b)

			// Send the result back to the main loop
			ch <- struct {
				b     backend.Backend
				alive bool
			}{b, alive}
		}(b)
	}
	// Collect results from all backend checks
	for i := 0; i < len(s.GetBackends()); i++ {
		select {
		case <-ctx.Done():
			// Stop the health check gracefully if the context is canceled
			logger.Info("gracefully shutting down health check")
			return
		case res := <-ch:
			// Update backend alive status
			res.b.SetAlive(res.alive)

			status := "up"

			if !res.alive {
				status = "down"
			}

			// Log the backend status
			logger.Debug(
				"url status",
				zap.String("url", res.b.GetURL().String()),
				zap.String("status", status),
			)
		}
	}
}

// LaunchHealthCheck repeatedly runs health checks at intervals defined in the config.
// It uses a ticker to schedule checks and exits when the provided context is canceled.
func LaunchHealthCheck(ctx context.Context, s ServerPool, logger *zap.Logger) {
	config, err := utils.GetLBConfig()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// Ticker to trigger health checks periodically
	t := time.NewTicker(time.Second * time.Duration(config.HealthCheckInterval))
	defer t.Stop()

	logger.Info("launching health check")

	for {
		select {
		case <-t.C:
			// Run health check asynchronously
			go HealthCheck(ctx, s, logger)
		case <-ctx.Done():
			// Stop launching health checks if context is canceled
			logger.Info("stopping health check")
			return
		}
	}
}
