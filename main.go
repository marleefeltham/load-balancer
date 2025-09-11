package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"load-balancer/backend"
	"load-balancer/lb"

	// "load-balancer/lb"
	"load-balancer/serverpool"
	"load-balancer/utils"

	"go.uber.org/zap"
)

func main() {
	logger := utils.InitLogger()
	defer logger.Sync()

	config, err := utils.GetLBConfig()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverPool, err := serverpool.NewServerPool(utils.GetLBStrategy(config.Strategy))
	if err != nil {
		logger.Fatal(err.Error())
	}
	loadBalancer := lb.NewLoadBalancer(serverPool)

	for _, u := range config.Backends {
		endpoint, err := url.Parse(u)
		if err != nil {
			logger.Fatal(err.Error())
		}

		backendServer := backend.NewBackend(endpoint)

		backendServer.SetErrorHandler(func(w http.ResponseWriter, r *http.Request, e error) {
			logger.Error("error handling the request", zap.String("host", endpoint.Host), zap.Error(e))
			backendServer.SetAlive(false)

			if !lb.AllowRetry(r) {
				http.Error(w, "Service not available", http.StatusServiceUnavailable)
				return
			}

			loadBalancer.ServeHTTP(
				w,
				r.WithContext(context.WithValue(r.Context(), lb.RetryAttemptedKey, true)),
			)
		})

		serverPool.AddBackend(backendServer)
	}

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: http.HandlerFunc(loadBalancer.ServeHTTP),
	}

	go serverpool.LaunchHealthCheck(ctx, serverPool)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(config.ShutdownTimeout))
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Fatal("failed to shutdown", zap.Error(err))
		}
	}()

	logger.Info("Load Balancer started", zap.Int("port", config.Port))
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logger.Fatal("ListenAndServe() error", zap.Error(err))
	}
}
