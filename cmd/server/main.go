// File: cmd/server/main.go
package main

import (
	"context"
	"go-observability-demo/internal/observability"
	"go-observability-demo/internal/service"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	ctx := context.Background()

	// Initialize observability
	serviceName := getEnv("SERVICE_NAME", "order-service")
	otelEndpoint := getEnv("OTEL_ENDPOINT", "localhost:4318")

	shutdown, err := observability.InitObservability(ctx, serviceName, otelEndpoint)
	if err != nil {
		log.Fatalf("Failed to initialize observability: %v", err)
	}
	defer shutdown(ctx)

	// Initialize logger
	logger := observability.NewLogger()

	// Initialize metrics
	metrics, err := observability.NewMetrics()
	if err != nil {
		log.Fatalf("Failed to initialize metrics: %v", err)
	}

	// Create order service
	orderService := service.NewOrderService(logger, metrics)

	// Setup HTTP routes with otelhttp middleware
	mux := http.NewServeMux()

	mux.Handle("/orders", otelhttp.NewHandler(
		http.HandlerFunc(orderService.CreateOrderHandler),
		"POST /orders",
	))

	mux.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// Create server
	port := getEnv("PORT", "8080")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Server starting", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Server shutting down")

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
