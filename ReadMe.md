# Go Observability Demo

A production-ready example of implementing observability in Go microservices using OpenTelemetry, demonstrating distributed tracing, metrics, and structured logging.

**📖 Full Article**: [Stop Debugging Go Microservices in the Dark: An Observability Playbook That Actually Works](https://maneeshaindrachapa.medium.com/stop-debugging-go-microservices-in-the-dark-an-observability-playbook-that-actually-works-e5ef71cf3027)

## Features

- **Distributed Tracing** with OpenTelemetry
- **Metrics Collection** with Prometheus-compatible exporters
- **Structured Logging** with trace correlation
- **Context Propagation** across service boundaries
- **Production-Ready Patterns** (sampling, error handling, graceful shutdown)
- **Complete Observability Stack** (Jaeger, Prometheus, Grafana)

## Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       ▼
┌─────────────────────┐
│   Order Service     │
│  (Instrumented)     │
└─────────┬───────────┘
          │
          ▼
┌─────────────────────┐
│  OTel Collector     │
│  (Receives traces   │
│   and metrics)      │
└──────┬──────┬───────┘
       │      │
   ┌───▼──┐ ┌▼────────┐
   │Jaeger│ │Prometheus│
   └──────┘ └─────────┘
```

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.21+ (for local development)
- Make (optional, for convenience commands)

### Run the Complete Stack

```bash
# Start all services (application + observability stack)
make docker-up

# Or without Make:
docker-compose up -d
```

This starts:

- **Order Service** on http://localhost:8080
- **Jaeger UI** on http://localhost:16686 (for traces)
- **Prometheus** on http://localhost:9090 (for metrics)
- **Grafana** on http://localhost:3000 (for dashboards, login: admin/admin)

### Send Sample Requests

```bash
# Send a single order
make sample-request

# Or with curl:
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-123",
    "product_id": "prod-456",
    "quantity": 2,
    "amount": 99.99
  }'

# Run a varied load test (mix of successes, validation errors, VIP orders)
make load-test
```

### View Your Data

1. **Traces**: Open http://localhost:16686

   - Select "order-service" from the service dropdown
   - Click "Find Traces"
   - Click on any trace to see the full request flow

2. **Metrics**: Open http://localhost:9090

   - Try query: `rate(observability_orders_created_total[5m])`
   - Or: `histogram_quantile(0.95, rate(observability_orders_duration_bucket[5m]))`

3. **Logs**: View structured logs with trace correlation

   ```bash
   docker-compose logs -f order-service
   ```

4. **Grafana Dashboards & Alerts**: Open http://localhost:3000 (login: admin/admin)

   - The `Order Service` folder contains the pre-built **Order Service Observability** dashboard.
   - Built-in alerts live under **Alerting → Alert rules → Order Service Alerts**.

## Grafana Dashboards & Alerts

- Provisioning lives under `config/grafana/provisioning` and is baked into the Grafana image via `build/grafana/Dockerfile`. Starting the stack with `make docker-up` automatically installs datasources, dashboards, and alert rules—no extra scripting required.
- Dashboard panels track:
  - Order throughput: `rate(observability_orders_created_total[5m])`
  - Order latency (p95): `histogram_quantile(0.95, sum by (le) (rate(observability_orders_duration_bucket[5m])))`
  - Error rate by type: `sum by (error_type) (rate(observability_errors_total[5m]))`
  - Payment volume: `sum(rate(observability_payments_total_amount_total[5m]))`
- Alert rules ship with the image:
  - **Order Service High Error Rate** – fires when errors/orders > 10% for 5 minutes.
  - **Order Service High Latency (p95)** – fires when p95 stays above 2s for 5 minutes.
- If the dashboard or alerts don’t appear, rebuild Grafana to apply the provisioning bundle: `docker-compose up -d --build grafana`.
- Customize the dashboard or alerts by editing the JSON/YAML under `config/grafana/provisioning` (dashboard JSON lives at `config/grafana/provisioning/dashboards/order-service-observability.json`).
- Use `make load-test` to feed Grafana a mix of successful, invalid, and high-value orders so the panels and alert rules have representative data.

## Project Structure

```
.
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── observability/
│   │   ├── tracing.go          # OpenTelemetry initialization
│   │   ├── metrics.go          # Metrics definitions
│   │   └── logger.go           # Structured logger with trace correlation
│   └── service/
│       └── order_service.go    # Business logic with instrumentation
├── config/
│   ├── otel-collector-config.yaml
│   ├── prometheus.yml
│   └── grafana/
│       └── provisioning/
├── docker-compose.yml           # Complete observability stack
├── Dockerfile
├── Makefile
├── go.mod
└── README.md
```

## Key Code Patterns

### 1. Creating Spans for Operations

```go
ctx, span := tracer.Start(ctx, "OperationName")
defer span.End()

// Add attributes for context
span.SetAttributes(
    attribute.String("user.id", userID),
    attribute.Int("quantity", quantity),
)

// Record errors
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
}
```

### 2. Logging with Trace Correlation

```go
observability.InfoWithTrace(ctx, logger, "operation started",
    slog.String("user_id", userID),
    slog.Float64("amount", amount),
)
// Output includes trace_id and span_id automatically
```

### 3. Recording Metrics

```go
metrics.OrderCounter.Add(ctx, 1, metric.WithAttributes(
    attribute.String("status", "success"),
))

metrics.OrderDuration.Record(ctx, float64(duration))
```

### 4. Context Propagation

```go
// HTTP clients automatically propagate context
client := &http.Client{
    Transport: otelhttp.NewTransport(http.DefaultTransport),
}

// Spans automatically inherit from parent context
ctx, span := tracer.Start(ctx, "ChildOperation")
```

## Configuration

Environment variables:

| Variable        | Default          | Description                           |
| --------------- | ---------------- | ------------------------------------- |
| `SERVICE_NAME`  | `order-service`  | Service identifier in traces          |
| `OTEL_ENDPOINT` | `localhost:4318` | OpenTelemetry collector endpoint      |
| `ENVIRONMENT`   | `development`    | Environment (affects sampling rate)   |
| `LOG_LEVEL`     | `info`           | Logging level (debug/info/warn/error) |
| `PORT`          | `8080`           | HTTP server port                      |

### Sampling Configuration

- **Development**: 100% sampling (see all traces)
- **Production**: 10% sampling (configurable in `internal/observability/tracing.go`)

## Common Use Cases

### Debugging a Slow Request

1. Go to Jaeger UI
2. Filter by duration > 1s
3. Click on a slow trace
4. See which operation took the most time
5. Copy the trace_id
6. Search logs by trace_id to see detailed messages

### Tuning Alerts

Grafana starts with two managed rules defined in `config/grafana/provisioning/alerting/order-service-alerts.yml`. Adjust the thresholds or queries there, then rebuild Grafana. The underlying PromQL expressions are:

```promql
# Error ratio (used by the high error rate alert)
sum(rate(observability_errors_total[5m])) /
clamp_min(sum(rate(observability_orders_created_total[5m])), 0.01)

# P95 latency (used by the high latency alert)
histogram_quantile(0.95, sum by (le) (rate(observability_orders_duration_bucket[5m])))
```

### Adding New Instrumentation

1. Create a span: `ctx, span := tracer.Start(ctx, "NewOperation")`
2. Add relevant attributes
3. Record errors if they occur
4. Update metrics if needed
5. Add structured logs for important events

## Testing

```bash
# Run all tests
make test

# Run with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Production Considerations

#### Sampling Strategy

The default 10% sampling in production is a starting point. Adjust based on:

- Traffic volume
- Observability budget
- Debugging needs

Consider implementing smart sampling (always sample errors and slow requests).

#### Resource Usage

OpenTelemetry adds overhead:

- ~5-10% CPU increase
- ~3-6% latency increase
- Memory proportional to span cardinality

#### Security

- Sanitize sensitive data before adding to spans
- Use RBAC for trace access
- Encrypt data in transit (TLS for OTLP)
- Review what PII is being captured

## Troubleshooting

#### No traces appearing in Jaeger

1. Check collector logs: `docker-compose logs otel-collector`
2. Verify endpoint: `curl http://localhost:4318/v1/traces`
3. Check sampling rate (development should be 1.0)

#### Metrics not in Prometheus

1. Check Prometheus targets: http://localhost:9090/targets
2. Verify collector is exposing metrics: `curl http://localhost:8889/metrics`
3. Check collector config exporters

#### High memory usage

1. Reduce batch size in collector config
2. Lower sampling rate
3. Limit span attributes cardinality

## Resources

- [Article on Medium](https://maneeshaindrachapa.medium.com/stop-debugging-go-microservices-in-the-dark-an-observability-playbook-that-actually-works-e5ef71cf3027)
- [OpenTelemetry Go Docs](https://opentelemetry.io/docs/instrumentation/go/)
- [CNCF Observability Whitepaper](https://www.cncf.io/blog/2021/05/18/observability-a-3-year-retrospective/)
