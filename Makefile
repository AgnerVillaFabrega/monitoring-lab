# E-commerce Monitoring Stack Makefile
# This file provides convenient commands to manage the monitoring stack

.PHONY: help build up down logs clean restart status health tidy deps

# Default target
help: ## Show this help message
	@echo "E-commerce Monitoring Stack Management"
	@echo "======================================"
	@echo ""
	@echo "Available commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make up          # Start all services"
	@echo "  make logs        # Show all logs"
	@echo "  make health      # Check service health"
	@echo "  make down        # Stop all services"

build: ## Build all Docker images
	@echo "🔨 Building Docker images..."
	docker compose build --no-cache

up: ## Start all services
	@echo "🚀 Starting e-commerce monitoring stack..."
	docker compose up -d
	@echo "✅ Services started!"
	@echo ""
	@echo "🌐 Access URLs:"
	@echo "  Grafana:       http://localhost:3000 (admin/admin123)"
	@echo "  User Service:  http://localhost:8081/health"
	@echo "  Product Service: http://localhost:8082/health"  
	@echo "  Order Service: http://localhost:8083/health"
	@echo ""
	@echo "📊 Dashboards will be available at:"
	@echo "  - E-commerce Logs"
	@echo "  - E-commerce Distributed Tracing"
	@echo "  - E-commerce Business Metrics"

down: ## Stop all services
	@echo "🛑 Stopping all services..."
	docker compose down
	@echo "✅ Services stopped!"

restart: ## Restart all services
	@echo "🔄 Restarting services..."
	docker compose down
	docker compose up -d
	@echo "✅ Services restarted!"

logs: ## Show logs for all services
	docker compose logs -f

logs-service: ## Show logs for a specific service (usage: make logs-service SERVICE=user-service)
	@if [ -z "$(SERVICE)" ]; then \
		echo "❌ Please specify SERVICE. Example: make logs-service SERVICE=user-service"; \
		exit 1; \
	fi
	docker compose logs -f $(SERVICE)

status: ## Show status of all services
	@echo "📊 Service Status:"
	@echo "=================="
	docker compose ps

health: ## Check health of all services
	@echo "🏥 Health Check Results:"
	@echo "======================="
	@echo ""
	@echo "Infrastructure Services:"
	@echo "-----------------------"
	@curl -s http://localhost:3000/api/health | jq '.status // "❌ Grafana not responding"' || echo "❌ Grafana not responding"
	@curl -s http://localhost:3100/ready | grep -q "ready" && echo "✅ Loki is ready" || echo "❌ Loki not ready"
	@curl -s http://localhost:3200/ready | grep -q "ready" && echo "✅ Tempo is ready" || echo "❌ Tempo not ready"
	@echo ""
	@echo "Application Services:"
	@echo "--------------------"
	@curl -s http://localhost:8081/health | jq '.status // "❌ User Service not responding"' || echo "❌ User Service not responding"
	@curl -s http://localhost:8082/health | jq '.status // "❌ Product Service not responding"' || echo "❌ Product Service not responding"
	@curl -s http://localhost:8083/health | jq '.status // "❌ Order Service not responding"' || echo "❌ Order Service not responding"

clean: ## Remove all containers, volumes and images
	@echo "🧹 Cleaning up..."
	@echo "⚠️  This will remove all containers, volumes, and built images!"
	@read -p "Are you sure? (y/N): " confirm && [ "$$confirm" = "y" ] || exit 1
	docker compose down -v
	docker system prune -af --volumes
	docker volume rm ecommerce-grafana-data ecommerce-loki-data ecommerce-tempo-data 2>/dev/null || true
	@echo "✅ Cleanup completed!"

tidy: ## Run go mod tidy for all services
	@echo "📦 Running go mod tidy for all services..."
	@for service in user-service product-service order-service traffic-generator; do \
		echo "  Processing $$service..."; \
		cd services/$$service && go mod tidy && cd ../..; \
	done
	@echo "✅ All go.mod files updated!"

deps: ## Download dependencies for all services
	@echo "📥 Downloading dependencies for all services..."
	@for service in user-service product-service order-service traffic-generator; do \
		echo "  Downloading for $$service..."; \
		cd services/$$service && go mod download && cd ../..; \
	done
	@echo "✅ All dependencies downloaded!"

test-endpoints: ## Test all service endpoints
	@echo "🧪 Testing Service Endpoints:"
	@echo "============================"
	@echo ""
	@echo "User Service Tests:"
	@echo "curl -X POST http://localhost:8081/auth/login -H 'Content-Type: application/json' -d '{\"email\":\"john@example.com\",\"password\":\"password123\"}'"
	@curl -X POST http://localhost:8081/auth/login -H 'Content-Type: application/json' -d '{"email":"john@example.com","password":"password123"}' | jq '.'
	@echo ""
	@echo "Product Service Tests:"
	@echo "curl http://localhost:8082/products"
	@curl -s http://localhost:8082/products | jq '.products[0:2]'
	@echo ""
	@echo "Order Service Tests:"
	@echo "curl http://localhost:8083/orders"
	@curl -s http://localhost:8083/orders | jq '.'

dev: ## Start services in development mode (with rebuild)
	@echo "🔧 Starting in development mode..."
	docker compose up -d --build
	@echo "✅ Development environment ready!"

prod: ## Start services in production mode
	@echo "🏭 Starting in production mode..."
	docker compose -f docker-compose.yml up -d
	@echo "✅ Production environment ready!"

backup: ## Create backup of persistent data
	@echo "💾 Creating backup..."
	@mkdir -p backups
	@timestamp=$$(date +%Y%m%d_%H%M%S); \
	docker run --rm -v ecommerce-grafana-data:/data -v $(PWD)/backups:/backup alpine tar czf /backup/grafana_$$timestamp.tar.gz -C /data . && \
	docker run --rm -v ecommerce-loki-data:/data -v $(PWD)/backups:/backup alpine tar czf /backup/loki_$$timestamp.tar.gz -C /data . && \
	echo "✅ Backup created in backups/ directory"

restore: ## Restore from backup (usage: make restore BACKUP_DATE=20240125_143000)
	@if [ -z "$(BACKUP_DATE)" ]; then \
		echo "❌ Please specify BACKUP_DATE. Example: make restore BACKUP_DATE=20240125_143000"; \
		ls backups/ 2>/dev/null || echo "No backups found"; \
		exit 1; \
	fi
	@echo "📥 Restoring from backup $(BACKUP_DATE)..."
	docker run --rm -v ecommerce-grafana-data:/data -v $(PWD)/backups:/backup alpine tar xzf /backup/grafana_$(BACKUP_DATE).tar.gz -C /data
	docker run --rm -v ecommerce-loki-data:/data -v $(PWD)/backups:/backup alpine tar xzf /backup/loki_$(BACKUP_DATE).tar.gz -C /data
	@echo "✅ Restore completed!"

install-tools: ## Install required tools (make, jq, curl)
	@echo "🔧 Installing required tools..."
	@if command -v apt-get >/dev/null 2>&1; then \
		sudo apt-get update && sudo apt-get install -y make jq curl; \
	elif command -v yum >/dev/null 2>&1; then \
		sudo yum install -y make jq curl; \
	elif command -v brew >/dev/null 2>&1; then \
		brew install make jq curl; \
	else \
		echo "❌ Could not detect package manager. Please install make, jq, and curl manually."; \
		exit 1; \
	fi
	@echo "✅ Tools installed!"