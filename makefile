# File: Makefile
SHELL := /bin/bash
.PHONY: help build run test docker-up docker-down docker-logs clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the Go application
	go build -o bin/server ./cmd/server

run: ## Run the application locally
	go run ./cmd/server

test: ## Run tests
	go test -v -race -cover ./...

docker-up: ## Start all services with Docker Compose
	docker-compose up -d
	@echo ""
	@echo "Services started! Access:"
	@echo "  - Application:    http://localhost:8080"
	@echo "  - Jaeger UI:      http://localhost:16686"
	@echo "  - Prometheus:     http://localhost:9090"
	@echo "  - Grafana:        http://localhost:3000 (admin/admin)"

docker-down: ## Stop all services
	docker-compose down

docker-logs: ## Show logs from all services
	docker-compose logs -f

docker-rebuild: ## Rebuild and restart services
	docker-compose down
	docker-compose up -d --build

clean: ## Clean up build artifacts and volumes
	rm -rf bin/
	docker-compose down -v

load-test: ## Run a varied load test (successes, validation errors, high-value orders)
	@echo "Sending 80 successful orders with varied payloads..."
	@PRODUCTS=("prod-123" "prod-456" "prod-789" "prod-321"); \
	AMOUNTS=(29.99 49.50 79.95 129.00 249.99); \
	for i in $$(seq 1 80); do \
		product=$${PRODUCTS[$$RANDOM % $${#PRODUCTS[@]}]}; \
		amount=$${AMOUNTS[$$RANDOM % $${#AMOUNTS[@]}]}; \
		quantity=$$(( ($$RANDOM % 4) + 1 )); \
		curl -s -o /dev/null -w "Success $$i: %{http_code}\n" \
		  -X POST http://localhost:8080/orders \
		  -H "Content-Type: application/json" \
		  -d "{\"user_id\":\"user-$${i}\",\"product_id\":\"$${product}\",\"quantity\":$${quantity},\"amount\":$${amount}}"; \
	done
	@echo ""
	@echo "⚠️  Triggering 10 validation errors..."
	@for i in $$(seq 1 10); do \
		curl -s -o /dev/null -w "Validation $$i: %{http_code}\n" \
		  -X POST http://localhost:8080/orders \
		  -H "Content-Type: application/json" \
		  -d "{\"user_id\":\"\",\"product_id\":\"prod-invalid\",\"quantity\":0,\"amount\":-42.0}"; \
	done
	@echo ""
	@echo "💰 Sending 10 high-value orders to exercise payment metrics..."
	@for i in $$(seq 1 10); do \
		curl -s -o /dev/null -w "HighValue $$i: %{http_code}\n" \
		  -X POST http://localhost:8080/orders \
		  -H "Content-Type: application/json" \
		  -d "{\"user_id\":\"vip-$${i}\",\"product_id\":\"prod-vip\",\"quantity\":1,\"amount\":999.99}"; \
	done
	@echo ""
	@echo "Done! Check Grafana at http://localhost:3000 and Jaeger at http://localhost:16686"

sample-request: ## Send a sample order request
	curl -X POST http://localhost:8080/orders \
	  -H "Content-Type: application/json" \
	  -d '{"user_id":"user-123","product_id":"prod-456","quantity":2,"amount":99.99}' | jq
