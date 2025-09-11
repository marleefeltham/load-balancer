package serverpool

import (
	"context"
	"load-balancer/backend"
	"load-balancer/utils"
	"time"

	"go.uber.org/zap"
)

func HealthCheck(ctx context.Context, s ServerPool) {
	config, err := utils.GetLBConfig()
	if err != nil {
		utils.Logger.Error("failed to get config for health check", zap.Error(err))
		return
	}

	ch := make(chan struct {
		b     backend.Backend
		alive bool
	}, len(s.GetBackends()))

	for _, b := range s.GetBackends() {
		go func(b backend.Backend) {
			reqCtx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(config.BackendTimeout))
			defer cancel()

			alive := backend.CheckBackendHealth(reqCtx, b)
			ch <- struct {
				b     backend.Backend
				alive bool
			}{b, alive}
		}(b)
	}

	for i := 0; i < len(s.GetBackends()); i++ {
		select {
		case <-ctx.Done():
			utils.Logger.Info("gracefully shutting down health check")
			return
		case res := <-ch:
			res.b.SetAlive(res.alive)

			status := "up"

			if !res.alive {
				status = "down"
			}

			utils.Logger.Debug(
				"URL Status",
				zap.String("URL", res.b.GetURL().String()),
				zap.String("status", status),
			)
		}
	}
}

func LaunchHealthCheck(ctx context.Context, s ServerPool) {
	config, err := utils.GetLBConfig()
	if err != nil {
		utils.Logger.Fatal("failed to load config", zap.Error(err))
	}

	t := time.NewTicker(time.Second * time.Duration(config.HealthCheckInterval))
	defer t.Stop()

	utils.Logger.Info("launching health check")

	for {
		select {
		case <-t.C:
			go HealthCheck(ctx, s)
		case <-ctx.Done():
			utils.Logger.Info("stopping health check")
			return
		}
	}
}
