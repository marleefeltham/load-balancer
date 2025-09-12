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
	"load-balancer/serverpool"
	"load-balancer/utils"

	"go.uber.org/zap"
)

func main() {
	// Initialize the logger
	logger := utils.InitLogger()
	defer logger.Sync()

	// Get the load balancer configuration
	config, err := utils.GetLBConfig()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// Setup context to handle SIGINT/SIGTERM for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create a server pool with the configured strategy
	serverPool, err := serverpool.NewServerPool(utils.GetLBStrategy(config.Strategy))
	if err != nil {
		logger.Fatal(err.Error())
	}
	loadBalancer := lb.NewLoadBalancer(serverPool)

	// Initialize backend servers
	for _, u := range config.Backends {
		endpoint, err := url.Parse(u)
		if err != nil {
			logger.Fatal(err.Error())
		}

		backendServer := backend.NewBackend(endpoint)

		// Configure the error handler for backend failures
		backendServer.SetErrorHandler(func(w http.ResponseWriter, r *http.Request, e error) {
			logger.Error("error handling the request", zap.String("host", endpoint.Host), zap.Error(e))
			backendServer.SetAlive(false)

			if !lb.AllowRetry(r) {
				http.Error(w, "service not available", http.StatusServiceUnavailable)
				return
			}

			// Retry request with load balancer
			loadBalancer.ServeHTTP(
				w,
				r.WithContext(context.WithValue(r.Context(), lb.RetryAttemptedKey, true)),
			)
		})

		serverPool.AddBackend(backendServer)
	}

	// Create HTTP server for the load balancer
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: http.HandlerFunc(loadBalancer.ServeHTTP),
	}

	// Start periodic health checks in the background
	go serverpool.LaunchHealthCheck(ctx, serverPool, logger)

	// Handle graceful shutdown
	go func() {
		<-ctx.Done() // Wait for terminatino signal(SIGINT/SIGTERM)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(config.ShutdownTimeout))
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Fatal("failed to shutdown", zap.Error(err))
		}
	}()

	// Start the load balancer
	logger.Info("load Balancer started", zap.Int("port", config.Port))
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logger.Fatal("ListenAndServe() error", zap.Error(err))
	}
}
