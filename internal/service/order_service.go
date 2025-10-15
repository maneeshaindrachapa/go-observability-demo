package service

import (
	"context"
	"encoding/json"
	"fmt"
	"go-observability-demo/internal/observability"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type OrderService struct {
	tracer          trace.Tracer
	logger          *slog.Logger
	metrics         *observability.Metrics
	paymentClient   *http.Client
	inventoryClient *http.Client
}

type CreateOrderRequest struct {
	UserID    string  `json:"user_id"`
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Amount    float64 `json:"amount"`
}

type CreateOrderResponse struct {
	Status  string `json:"status"`
	OrderID string `json:"order_id"`
	TraceID string `json:"trace_id"`
}

func NewOrderService(logger *slog.Logger, metrics *observability.Metrics) *OrderService {
	return &OrderService{
		tracer:  otel.Tracer("order-service"),
		logger:  logger,
		metrics: metrics,
		paymentClient: &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
			Timeout:   5 * time.Second,
		},
		inventoryClient: &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
			Timeout:   5 * time.Second,
		},
	}
}

func (s *OrderService) CreateOrderHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	// Create main span
	ctx, span := s.tracer.Start(ctx, "CreateOrder",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	defer span.End()

	observability.InfoWithTrace(ctx, s.logger, "order creation started")

	// Parse request
	var req CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request body")
		observability.ErrorWithTrace(ctx, s.logger, "failed to parse request", slog.String("error", err.Error()))
		http.Error(w, "invalid request", http.StatusBadRequest)
		s.metrics.ErrorCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("error.type", "invalid_request"),
		))
		return
	}

	// Validate request
	if err := s.validateRequest(req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "validation failed")
		observability.ErrorWithTrace(ctx, s.logger, "request validation failed", slog.String("error", err.Error()))
		http.Error(w, err.Error(), http.StatusBadRequest)
		s.metrics.ErrorCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("error.type", "validation_error"),
		))
		return
	}

	// Add request attributes to span
	span.SetAttributes(
		attribute.String("user.id", req.UserID),
		attribute.String("product.id", req.ProductID),
		attribute.Int("order.quantity", req.Quantity),
		attribute.Float64("order.amount", req.Amount),
	)

	// Process order
	orderID, err := s.processOrder(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		observability.ErrorWithTrace(ctx, s.logger, "order processing failed",
			slog.String("error", err.Error()),
			slog.String("user_id", req.UserID),
		)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		s.metrics.ErrorCounter.Add(ctx, 1, metric.WithAttributes(
			attribute.String("error.type", "processing_error"),
		))
		return
	}

	// Record metrics
	duration := time.Since(start).Milliseconds()
	s.metrics.OrderDuration.Record(ctx, float64(duration), metric.WithAttributes(
		attribute.String("status", "success"),
	))
	s.metrics.OrderCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", "success"),
	))
	s.metrics.PaymentAmount.Add(ctx, req.Amount)

	span.SetStatus(codes.Ok, "order created successfully")
	observability.InfoWithTrace(ctx, s.logger, "order created successfully",
		slog.String("order_id", orderID),
		slog.Int64("duration_ms", duration),
	)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreateOrderResponse{
		Status:  "success",
		OrderID: orderID,
		TraceID: span.SpanContext().TraceID().String(),
	})
}

func (s *OrderService) validateRequest(req CreateOrderRequest) error {
	if req.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if req.ProductID == "" {
		return fmt.Errorf("product_id is required")
	}
	if req.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	if req.Amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	return nil
}

func (s *OrderService) processOrder(ctx context.Context, req CreateOrderRequest) (string, error) {
	// Step 1: Check inventory
	if err := s.checkInventory(ctx, req.ProductID, req.Quantity); err != nil {
		return "", fmt.Errorf("inventory check failed: %w", err)
	}

	// Step 2: Process payment
	if err := s.processPayment(ctx, req.UserID, req.Amount); err != nil {
		return "", fmt.Errorf("payment failed: %w", err)
	}

	// Step 3: Reserve inventory
	if err := s.reserveInventory(ctx, req.ProductID, req.Quantity); err != nil {
		return "", fmt.Errorf("inventory reservation failed: %w", err)
	}

	// Generate order ID
	orderID := fmt.Sprintf("order-%d", time.Now().UnixNano())
	return orderID, nil
}

func (s *OrderService) checkInventory(ctx context.Context, productID string, quantity int) error {
	ctx, span := s.tracer.Start(ctx, "CheckInventory")
	defer span.End()

	span.SetAttributes(
		attribute.String("product.id", productID),
		attribute.Int("requested.quantity", quantity),
	)

	observability.DebugWithTrace(ctx, s.logger, "checking inventory",
		slog.String("product_id", productID),
		slog.Int("quantity", quantity),
	)

	// Simulate inventory check (in real app, this would be an HTTP call)
	time.Sleep(time.Duration(30+rand.Intn(50)) * time.Millisecond)

	s.metrics.InventoryRequests.Add(ctx, 1)

	// Simulate occasional inventory issues
	if rand.Float64() < 0.1 {
		err := fmt.Errorf("insufficient inventory")
		span.RecordError(err)
		span.SetStatus(codes.Error, "insufficient inventory")
		return err
	}

	span.AddEvent("inventory_available")
	return nil
}

func (s *OrderService) processPayment(ctx context.Context, userID string, amount float64) error {
	ctx, span := s.tracer.Start(ctx, "ProcessPayment")
	defer span.End()

	span.SetAttributes(
		attribute.String("user.id", userID),
		attribute.Float64("payment.amount", amount),
	)

	observability.DebugWithTrace(ctx, s.logger, "processing payment",
		slog.String("user_id", userID),
		slog.Float64("amount", amount),
	)

	// Simulate payment processing
	time.Sleep(time.Duration(80+rand.Intn(100)) * time.Millisecond)

	span.AddEvent("payment_gateway_called", trace.WithAttributes(
		attribute.String("gateway", "stripe"),
		attribute.String("payment.method", "credit_card"),
	))

	// Simulate occasional slow payments (10% of time)
	if rand.Intn(10) == 0 {
		span.AddEvent("payment_slow_path")
		observability.WarnWithTrace(ctx, s.logger, "payment processing slow")
		time.Sleep(3 * time.Second)
	}

	// Simulate occasional payment failures
	if rand.Float64() < 0.05 {
		err := fmt.Errorf("payment declined")
		span.RecordError(err)
		span.SetStatus(codes.Error, "payment declined")
		return err
	}

	span.AddEvent("payment_completed")
	span.SetStatus(codes.Ok, "payment successful")
	return nil
}

func (s *OrderService) reserveInventory(ctx context.Context, productID string, quantity int) error {
	ctx, span := s.tracer.Start(ctx, "ReserveInventory")
	defer span.End()

	span.SetAttributes(
		attribute.String("product.id", productID),
		attribute.Int("quantity", quantity),
	)

	observability.DebugWithTrace(ctx, s.logger, "reserving inventory",
		slog.String("product_id", productID),
		slog.Int("quantity", quantity),
	)

	// Simulate database operation
	start := time.Now()
	time.Sleep(time.Duration(40+rand.Intn(60)) * time.Millisecond)
	duration := time.Since(start)

	span.SetAttributes(
		attribute.Int64("db.duration_ms", duration.Milliseconds()),
		attribute.String("db.operation", "UPDATE"),
		attribute.String("db.table", "inventory"),
	)

	span.AddEvent("inventory_reserved")
	return nil
}
