# File: Makefile
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

load-test: ## Run a simple load test
	@echo "Sending 100 requests to the order service..."
	@for i in $$(seq 1 100); do \
		curl -X POST http://localhost:8080/orders \
		  -H "Content-Type: application/json" \
		  -d '{"user_id":"user-'$$i'","product_id":"prod-123","quantity":2,"amount":99.99}' \
		  -s -o /dev/null -w "Request $$i: %{http_code}\n"; \
	done
	@echo "Done! Check Jaeger UI at http://localhost:16686"

sample-request: ## Send a sample order request
	curl -X POST http://localhost:8080/orders \
	  -H "Content-Type: application/json" \
	  -d '{"user_id":"user-123","product_id":"prod-456","quantity":2,"amount":99.99}' | jq