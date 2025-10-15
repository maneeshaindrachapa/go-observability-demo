package observability

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	OrderCounter      metric.Int64Counter
	OrderDuration     metric.Float64Histogram
	PaymentAmount     metric.Float64Counter
	InventoryRequests metric.Int64Counter
	ErrorCounter      metric.Int64Counter
}

func NewMetrics() (*Metrics, error) {
	meter := otel.Meter("order-service")

	orderCounter, err := meter.Int64Counter(
		"orders.created",
		metric.WithDescription("Total number of orders created"),
		metric.WithUnit("{order}"),
	)
	if err != nil {
		return nil, err
	}

	orderDuration, err := meter.Float64Histogram(
		"orders.duration",
		metric.WithDescription("Order processing duration"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	paymentAmount, err := meter.Float64Counter(
		"payments.total_amount",
		metric.WithDescription("Total payment amount processed"),
		metric.WithUnit("USD"),
	)
	if err != nil {
		return nil, err
	}

	inventoryRequests, err := meter.Int64Counter(
		"inventory.requests",
		metric.WithDescription("Number of inventory check requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	errorCounter, err := meter.Int64Counter(
		"errors.total",
		metric.WithDescription("Total number of errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		OrderCounter:      orderCounter,
		OrderDuration:     orderDuration,
		PaymentAmount:     paymentAmount,
		InventoryRequests: inventoryRequests,
		ErrorCounter:      errorCounter,
	}, nil
}
