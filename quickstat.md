# Quick Start Guide

## 1. Initial Setup (5 minutes)

### Clone and Start the Stack

```bash
# Clone the repository
git clone https://github.com/yourusername/go-observability-demo
cd go-observability-demo

# Start all services (app + observability stack)
make docker-up

# Optional: wait a moment for services to settle
sleep 30
```

> Prefer raw Docker Compose? Use `docker-compose up -d` instead of `make docker-up`.

### Verify Services

```bash
docker-compose ps
```

You should see:

- `order-service` (HTTP API on port 8080)
- `otel-collector` (OTLP on 4317/4318)
- `jaeger` (UI on port 16686)
- `prometheus` (UI on port 9090)
- `grafana` (UI on port 3000)

## 2. Generate Some Data

```bash
# Send 100 test requests with randomized payloads
for i in {1..100}; do
  curl -X POST http://localhost:8080/orders \
    -H "Content-Type: application/json" \
    -d "{\"user_id\":\"user-$i\",\"product_id\":\"prod-$((RANDOM % 10))\",\"quantity\":$((RANDOM % 5 + 1)),\"amount\":$((RANDOM % 100 + 10)).99}" \
    -s -o /dev/null -w "Request $i: %{http_code}\n"
  sleep 0.1
done
```

Or lean on the Makefile:

```bash
# Mix of successes, validation failures, and VIP orders
make load-test
```

## 3. Explore Your Data

### A. Jaeger (Distributed Tracing)

- **URL**: http://localhost:16686
- **Workflow**:
  1. Select **order-service** from the *Service* dropdown.
  2. Click **Find Traces**.
  3. Inspect any trace to see the waterfall view and span attributes.

**Tips**

- Filter slow requests: set *Duration* ≥ `1000ms`.
- Filter by tags: e.g. `user.id=user-123`, `error=true`.
- Copy a `trace_id` from API responses to jump straight to a trace.

```
# Example search filters
Duration: min=1000ms
Tags: user.id=user-123
Tags: error=true
```

### B. Prometheus (Metrics)

- **URL**: http://localhost:9090
- **Workflow**:
  1. Click the **Graph** tab.
  2. Enter a PromQL query.
  3. Click **Execute** and toggle between **Table** and **Graph** views.

**Essential Queries**

```promql
# Request rate (orders per second)
rate(observability_orders_created_total[5m])

# P95 latency (milliseconds)
histogram_quantile(0.95, sum by (le) (rate(observability_orders_duration_bucket[5m])))

# Error rate percentage
(rate(observability_errors_total[5m]) / rate(observability_orders_created_total[5m])) * 100

# Total orders created
observability_orders_created_total

# Total revenue processed
observability_payments_total_amount_total

# Inventory check rate
rate(observability_inventory_requests_total[5m])

# Average latency
rate(observability_orders_duration_sum[5m]) / rate(observability_orders_duration_count[5m])

# Success rate
(1 - (rate(observability_errors_total[5m]) / rate(observability_orders_created_total[5m]))) * 100
```

**Pro Tips**

- Adjust time range (top-right dropdown) to explore different windows.
- Use the legend to drill into label dimensions.
- Switch to Table view for precise numbers.

### C. Grafana (Dashboards & Alerts)

- **URL**: http://localhost:3000  
- **Login**: `admin` / `admin`

Provisioning is baked into the Docker image, so dashboards and alert rules appear automatically.

1. **Login** with the default credentials.
2. **Verify Prometheus datasource** (optional):  
   `≡ menu` → **Connections** → **Data sources** → **Prometheus** → **Save & Test**.
3. **Open the dashboard**:  
   `≡ menu` → **Dashboards** → **Order Service** folder → **Order Service Observability**.
4. **Inspect alert rules**:  
   `≡ menu` → **Alerting** → **Alert rules** → **Order Service Alerts** folder.

**Dashboard Panels (default)**

1. **Order Throughput** – `rate(observability_orders_created_total[5m])`
2. **Order Duration p95** – `histogram_quantile(0.95, sum by (le) (rate(observability_orders_duration_bucket[5m])))`
3. **Error Rate by Type** – `sum by (error_type) (rate(observability_errors_total[5m]))`
4. **Payment Volume** – `sum(rate(observability_payments_total_amount_total[5m]))`

**Customizing**

- Edit `config/grafana/provisioning/dashboards/order-service-observability.json` then rebuild Grafana:
  ```bash
  docker-compose up -d --build grafana
  ```
- Grafana allows in-UI edits; saving writes back to the mounted JSON so you can commit changes.
- To add new alerts, modify `config/grafana/provisioning/alerting/order-service-alerts.yml` and rebuild Grafana.

## 4. Practical Workflows

### Workflow 1: Debug a Slow Request

1. Grafana → Check if **Order Duration p95** panel spikes.
2. Jaeger → Filter *Duration* ≥ `3000ms`, inspect a slow trace.  
   Example waterfall:
   ```
   CreateOrder (5.2s)
   ├─ CheckInventory (45ms)
   ├─ ProcessPayment (5.1s)  ← culprit
   └─ ReserveInventory (50ms)
   ```
3. Copy `trace_id`, correlate with logs:
   ```bash
   docker-compose logs order-service | grep TRACE_ID_HERE
   ```
4. Prometheus → Check if latency spike is ongoing:
   ```promql
   histogram_quantile(0.95, sum by (le) (rate(observability_orders_duration_bucket[5m])))
   ```

### Workflow 2: Monitor Error Rate

1. Prometheus → Current rate:
   ```promql
   (rate(observability_errors_total[5m]) / rate(observability_orders_created_total[5m])) * 100
   ```
2. Jaeger → Filter `error=true` to inspect failing traces.
3. Grafana → Alert rule **Order Service High Error Rate** (fires above 10% for 5 minutes).

### Workflow 3: Capacity Planning

1. Prometheus → Throughput:
   ```promql
   rate(observability_orders_created_total[5m])
   ```
2. Grafana → Watch p95 latency to ensure it remains within SLO.
3. Run a heavier load:
   ```bash
   for i in {1..1000}; do
     curl -X POST http://localhost:8080/orders \
       -H "Content-Type: application/json" \
       -d "{\"user_id\":\"user-$i\",\"product_id\":\"prod-123\",\"quantity\":2,\"amount\":99.99}" \
       -s -o /dev/null &
   done
   wait
   ```
4. Re-check metrics and traces for regressions.

## 5. Common Queries Cheatsheet

### Prometheus

```promql
# Rates
rate(observability_orders_created_total[5m])
rate(observability_errors_total[5m])
rate(observability_inventory_requests_total[5m])

# Latency percentiles
histogram_quantile(0.50, sum by (le) (rate(observability_orders_duration_bucket[5m])))
histogram_quantile(0.95, sum by (le) (rate(observability_orders_duration_bucket[5m])))
histogram_quantile(0.99, sum by (le) (rate(observability_orders_duration_bucket[5m])))

# Error & success rates
(rate(observability_errors_total[5m]) / rate(observability_orders_created_total[5m])) * 100
1 - (rate(observability_errors_total[5m]) / rate(observability_orders_created_total[5m]))

# Business KPIs
observability_orders_created_total
observability_payments_total_amount_total
increase(observability_orders_created_total[1h])

# Aggregations
sum(rate(observability_orders_created_total[5m])) by (status)
avg(rate(observability_orders_duration_sum[5m]) / rate(observability_orders_duration_count[5m]))
```

### Jaeger Filters

```
Trace ID: 4bf92f3577b34da6a3ce929d0e0e4736
Duration: min=1000ms max=5000ms
Tags: user.id=user-123
Tags: product.id=prod-456
Tags: error=true
Operation: CreateOrder
Operation: ProcessPayment
Lookback: Last 1 hour
Limit Results: 100
```

## 6. Troubleshooting

### No Metrics in Prometheus

1. Check the collector exposes metrics:
   ```bash
   curl http://localhost:8889/metrics | grep observability | head
   ```
2. Prometheus targets: http://localhost:9090/targets → `otel-collector` should be **UP**.
3. Collector logs:
   ```bash
   docker-compose logs otel-collector | grep -i metric
   ```
4. Still blank? Restart:
   ```bash
   docker-compose restart
   ```

### No Data in Grafana

1. Datasource test (Grafana UI) → **Save & Test** should be green.
2. Adjust the time range (top-right).
3. Ensure you generated traffic (`make sample-request` or `make load-test`).
4. Rebuild Grafana to reapply provisioning:
   ```bash
   docker-compose up -d --build grafana
   ```

### Dashboard Missing

1. Confirm provisioning file exists:
   ```bash
   ls config/grafana/provisioning/dashboards/order-service-observability.json
   ```
2. Check Grafana logs for provisioning errors:
   ```bash
   docker-compose logs grafana | grep -i provision
   ```
3. As a fallback, import the JSON manually via Grafana’s **Dashboards → Import**.

### Services Fail to Start

1. Check for port conflicts:
   ```bash
   lsof -i :8080
   lsof -i :3000
   ```
2. Confirm Docker is running:
   ```bash
   docker ps
   ```
3. Stop conflicting services or tweak ports in `docker-compose.yml`.

## 7. Next Steps

1. **Customize dashboards** – add panels, tweak thresholds, capture new metrics.
2. **Tune alerting** – extend `order-service-alerts.yml`, connect notification channels.
3. **Instrument more services** – propagate context to downstream dependencies.
4. **Production hardening** – persistent Prometheus storage, secrets, TLS, RBAC.

## 8. Handy Commands

```bash
make docker-up        # Start everything
make docker-down      # Stop everything
make docker-logs      # Tail logs for all services
make docker-rebuild   # Rebuild containers (use after provisioning changes)
make sample-request   # Send one order
make load-test        # Send varied traffic
make clean            # Tear down containers and volumes

docker-compose logs -f order-service  # Live app logs
docker-compose logs -f grafana        # Grafana provisioning logs
curl http://localhost:8080/health     # Health check
```

## Need Help?

- `docker-compose logs [service-name]`
- `docker-compose restart [service-name]`
- `make clean && make docker-up`
- Open an issue: https://github.com/yourusername/go-observability-demo/issues

Happy observing! 🎉
