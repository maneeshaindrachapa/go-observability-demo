// File: internal/service/order_service_test.go
package service

import (
	"bytes"
	"encoding/json"
	"go-observability-demo/internal/observability"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func setupTestService(t *testing.T) (*OrderService, *tracetest.InMemoryExporter) {
	// Create in-memory exporter for testing
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)

	logger := observability.NewLogger()
	metrics, err := observability.NewMetrics()
	if err != nil {
		t.Fatalf("Failed to create metrics: %v", err)
	}

	service := NewOrderService(logger, metrics)
	return service, exporter
}

func TestCreateOrderHandler_Success(t *testing.T) {
	service, exporter := setupTestService(t)

	reqBody := CreateOrderRequest{
		UserID:    "test-user",
		ProductID: "test-product",
		Quantity:  2,
		Amount:    99.99,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	service.CreateOrderHandler(rec, req)

	// Check response
	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rec.Code)
	}

	var resp CreateOrderResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("Expected status 'success', got %s", resp.Status)
	}

	// Verify spans were created
	spans := exporter.GetSpans()
	if len(spans) < 4 {
		t.Errorf("Expected at least 4 spans (CreateOrder, CheckInventory, ProcessPayment, ReserveInventory), got %d", len(spans))
	}

	// Verify main span has correct attributes
	var createOrderSpan *tracetest.SpanStub
	for i := range spans {
		if spans[i].Name == "CreateOrder" {
			createOrderSpan = &spans[i]
			break
		}
	}

	if createOrderSpan == nil {
		t.Fatal("CreateOrder span not found")
	}

	attrs := createOrderSpan.Attributes
	foundUserID := false
	foundProductID := false

	for _, attr := range attrs {
		if string(attr.Key) == "user.id" && attr.Value.AsString() == "test-user" {
			foundUserID = true
		}
		if string(attr.Key) == "product.id" && attr.Value.AsString() == "test-product" {
			foundProductID = true
		}
	}

	if !foundUserID {
		t.Error("user.id attribute not found in span")
	}
	if !foundProductID {
		t.Error("product.id attribute not found in span")
	}
}

func TestCreateOrderHandler_ValidationError(t *testing.T) {
	service, _ := setupTestService(t)

	tests := []struct {
		name    string
		request CreateOrderRequest
	}{
		{
			name: "missing user_id",
			request: CreateOrderRequest{
				ProductID: "test-product",
				Quantity:  2,
				Amount:    99.99,
			},
		},
		{
			name: "missing product_id",
			request: CreateOrderRequest{
				UserID:   "test-user",
				Quantity: 2,
				Amount:   99.99,
			},
		},
		{
			name: "invalid quantity",
			request: CreateOrderRequest{
				UserID:    "test-user",
				ProductID: "test-product",
				Quantity:  0,
				Amount:    99.99,
			},
		},
		{
			name: "invalid amount",
			request: CreateOrderRequest{
				UserID:    "test-user",
				ProductID: "test-product",
				Quantity:  2,
				Amount:    -10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			service.CreateOrderHandler(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", rec.Code)
			}
		})
	}
}

func TestValidateRequest(t *testing.T) {
	service, _ := setupTestService(t)

	validReq := CreateOrderRequest{
		UserID:    "test-user",
		ProductID: "test-product",
		Quantity:  2,
		Amount:    99.99,
	}

	if err := service.validateRequest(validReq); err != nil {
		t.Errorf("Valid request failed validation: %v", err)
	}
}

func BenchmarkCreateOrderHandler(b *testing.B) {
	service, _ := setupTestService(&testing.T{})

	reqBody := CreateOrderRequest{
		UserID:    "test-user",
		ProductID: "test-product",
		Quantity:  2,
		Amount:    99.99,
	}

	body, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		service.CreateOrderHandler(rec, req)
	}
}
