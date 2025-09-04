# Monitoring Lab Makefile
# ====================

.PHONY: help clean build deploy status logs setup-logging monitoring apps clusters check

# Default target
help:
	@echo "🚀 Monitoring Lab Commands"
	@echo "=========================="
	@echo ""
	@echo "Main Commands:"
	@echo "  make setup-logging  - Create logs directory"
	@echo "  make clean         - Clean all resources"
	@echo "  make build         - Build all Docker images"
	@echo "  make deploy        - Deploy complete lab"
	@echo "  make status        - Check deployment status"
	@echo ""
	@echo "Individual Components:"
	@echo "  make monitoring    - Deploy monitoring stack"
	@echo "  make apps          - Deploy applications"
	@echo "  make clusters      - Deploy cluster monitoring"
	@echo ""
	@echo "Build Individual Apps:"
	@echo "  make build-app1    - Build app1 images"
	@echo "  make build-app2    - Build app2 images"
	@echo ""
	@echo "Utilities:"
	@echo "  make logs          - Show recent deployment logs"
	@echo "  make check         - Run comprehensive status check"
	@echo ""

# Setup
setup-logging:
	@echo "📁 Creating logs directory..."
	@mkdir -p logs
	@echo "✅ Logs directory ready"

# Clean resources
clean:
	@echo "🧹 Cleaning all resources..."
	@kubectl delete namespace monitoring --ignore-not-found=true
	@kubectl delete namespace app1 --ignore-not-found=true  
	@kubectl delete namespace app2 --ignore-not-found=true
	@kubectl delete clusterrole kube-state-metrics --ignore-not-found=true
	@kubectl delete clusterrole kube-state-metrics-app2 --ignore-not-found=true
	@kubectl delete clusterrole prometheus-agent --ignore-not-found=true
	@kubectl delete clusterrole prometheus-agent-app2 --ignore-not-found=true
	@kubectl delete clusterrole fluent-bit-read --ignore-not-found=true
	@kubectl delete clusterrole fluent-bit-read-app2 --ignore-not-found=true
	@kubectl delete clusterrolebinding kube-state-metrics --ignore-not-found=true
	@kubectl delete clusterrolebinding kube-state-metrics-app2 --ignore-not-found=true
	@kubectl delete clusterrolebinding prometheus-agent --ignore-not-found=true
	@kubectl delete clusterrolebinding prometheus-agent-app2 --ignore-not-found=true
	@kubectl delete clusterrolebinding fluent-bit-read --ignore-not-found=true
	@kubectl delete clusterrolebinding fluent-bit-read-app2 --ignore-not-found=true
	@echo "🔧 Force cleaning stuck pods..."
	@kubectl delete pods --all -n monitoring --force --grace-period=0 2>/dev/null || true
	@kubectl delete pods --all -n app1 --force --grace-period=0 2>/dev/null || true
	@kubectl delete pods --all -n app2 --force --grace-period=0 2>/dev/null || true
	@echo "⏳ Waiting for resources to be deleted..."
	@sleep 15
	@echo "✅ Cleanup completed"

# Build all images
build: build-app1 build-app2
	@echo "✅ All images built successfully"

# Build app1 images
build-app1:
	@echo "🔨 Building App1 containers..."
	@cd apps/app1 && docker build -t app1:latest -f docker/Dockerfile .
	@cd apps/app1 && docker build -t app1-traffic:latest -f docker/Dockerfile.traffic .
	@echo "✅ App1 images built: app1:latest, app1-traffic:latest"

# Build app2 images  
build-app2:
	@echo "🔨 Building App2 containers..."
	@cd apps/app2 && docker build -t app2:latest -f docker/Dockerfile .
	@cd apps/app2 && docker build -t app2-traffic:latest -f docker/Dockerfile.traffic .
	@echo "✅ App2 images built: app2:latest, app2-traffic:latest"

# Deploy monitoring stack
monitoring:
	@echo "📊 Deploying monitoring stack..."
	@kubectl apply -f monitoring/namespace.yaml
	@echo "⏳ Waiting for namespace..."
	@sleep 5
	@kubectl apply -f monitoring/prometheus.yaml
	@kubectl apply -f monitoring/alertmanager.yaml
	@kubectl apply -f monitoring/loki.yaml
	@kubectl apply -f monitoring/tempo.yaml
	@kubectl apply -f monitoring/grafana-dashboards.yaml
	@kubectl apply -f monitoring/grafana.yaml
	@echo "⏳ Waiting for pods to be ready..."
	@kubectl wait --for=condition=ready pod -l app=grafana -n monitoring --timeout=300s || true
	@kubectl wait --for=condition=ready pod -l app=alertmanager -n monitoring --timeout=300s || true
	@echo "✅ Monitoring stack deployed"

# Deploy applications
apps: build-app1 build-app2
	@echo "🚀 Deploying applications..."
	@kubectl apply -f apps/app1/k8s/k8s-manifests.yaml
	@kubectl apply -f apps/app2/k8s/k8s-manifests.yaml
	@echo "✅ Applications deployed"

# Deploy cluster monitoring
clusters:
	@echo "🖥️ Deploying cluster monitoring..."
	@echo "📊 Deploying cluster1 monitoring..."
	@kubectl apply -f apps/app1/k8s/k8s-manifests.yaml
	@kubectl apply -f clusters/cluster1/kube-state-metrics.yaml
	@kubectl apply -f clusters/cluster1/node-exporter.yaml
	@kubectl apply -f clusters/cluster1/prometheus-agent.yaml
	@kubectl apply -f clusters/cluster1/fluent-bit.yaml
	@echo "📊 Deploying cluster2 monitoring..."
	@kubectl apply -f apps/app2/k8s/k8s-manifests.yaml
	@kubectl apply -f clusters/cluster2/kube-state-metrics.yaml
	@kubectl apply -f clusters/cluster2/node-exporter.yaml
	@kubectl apply -f clusters/cluster2/prometheus-agent.yaml
	@kubectl apply -f clusters/cluster2/fluent-bit.yaml
	@echo "✅ Cluster monitoring deployed"

# Full deployment with logging
deploy: setup-logging clean
	@echo "🚀 Starting complete deployment..."
	@$(eval TIMESTAMP := $(shell date "+%Y%m%d_%H%M%S"))
	@$(eval LOG_FILE := logs/deploy_$(TIMESTAMP).log)
	@echo "📝 Logging to: $(LOG_FILE)"
	@echo "=== DEPLOYMENT START ===" | tee $(LOG_FILE)
	@echo "Timestamp: $$(date)" | tee -a $(LOG_FILE)
	@echo "" | tee -a $(LOG_FILE)
	@echo "🔍 Checking Kubernetes connectivity..." | tee -a $(LOG_FILE)
	@kubectl cluster-info >/dev/null 2>&1 || (echo "❌ Kubernetes not available" | tee -a $(LOG_FILE) && exit 1)
	@echo "✅ Kubernetes cluster detected" | tee -a $(LOG_FILE)
	@echo "" | tee -a $(LOG_FILE)
	@echo "[1/3] 📊 Deploying monitoring stack..." | tee -a $(LOG_FILE)
	@$(MAKE) monitoring 2>&1 | tee -a $(LOG_FILE)
	@echo "" | tee -a $(LOG_FILE)
	@echo "[2/3] 🚀 Deploying applications..." | tee -a $(LOG_FILE)
	@$(MAKE) apps 2>&1 | tee -a $(LOG_FILE)
	@echo "" | tee -a $(LOG_FILE)
	@echo "[3/3] 🖥️ Deploying cluster monitoring..." | tee -a $(LOG_FILE)
	@$(MAKE) clusters 2>&1 | tee -a $(LOG_FILE)
	@echo "" | tee -a $(LOG_FILE)
	@echo "=== DEPLOYMENT COMPLETE ===" | tee -a $(LOG_FILE)
	@echo "Timestamp: $$(date)" | tee -a $(LOG_FILE)
	@echo "" | tee -a $(LOG_FILE)
	@echo "🎉 Deployment completed!" | tee -a $(LOG_FILE)
	@echo "📝 Full log saved to: $(LOG_FILE)" | tee -a $(LOG_FILE)
	@echo "" | tee -a $(LOG_FILE)
	@echo "🌐 Access points:" | tee -a $(LOG_FILE)
	@echo "  • Grafana:      http://localhost:30000 (admin/admin)" | tee -a $(LOG_FILE)
	@echo "  • Prometheus:   http://localhost:30090" | tee -a $(LOG_FILE)
	@echo "  • Alertmanager: http://localhost:30093" | tee -a $(LOG_FILE)
	@echo "  • Tempo:        http://localhost:30200" | tee -a $(LOG_FILE)

# Check deployment status
status:
	@echo "🔍 Deployment Status"
	@echo "===================="
	@echo "Timestamp: $$(date)"
	@echo ""
	@echo "📊 Monitoring namespace:"
	@kubectl get pods -n monitoring -o wide 2>/dev/null || echo "  No monitoring namespace found"
	@echo ""
	@echo "🔧 App1 namespace:"
	@kubectl get pods -n app1 -o wide 2>/dev/null || echo "  No app1 namespace found"
	@echo ""
	@echo "🐍 App2 namespace:"
	@kubectl get pods -n app2 -o wide 2>/dev/null || echo "  No app2 namespace found"
	@echo ""
	@echo "🌐 NodePort services:"
	@kubectl get services --all-namespaces | grep NodePort || echo "  No NodePort services found"

# Comprehensive status check
check: status
	@echo ""
	@echo "📝 Pods with issues:"
	@kubectl get pods --all-namespaces | grep -v Running | grep -v Completed | grep -v STATUS || echo "  ✅ All pods are running"
	@echo ""
	@echo "🌐 Service connectivity:"
	@echo -n "  Grafana (30000): "
	@curl -s -o /dev/null -w "%{http_code}" http://localhost:30000 2>/dev/null | grep -q "200\|302" && echo "✅ Accessible" || echo "❌ Not accessible"
	@echo -n "  Prometheus (30090): "
	@curl -s -o /dev/null -w "%{http_code}" http://localhost:30090 2>/dev/null | grep -q "200\|302" && echo "✅ Accessible" || echo "❌ Not accessible"
	@echo -n "  Alertmanager (30093): "
	@curl -s -o /dev/null -w "%{http_code}" http://localhost:30093 2>/dev/null | grep -q "200\|302" && echo "✅ Accessible" || echo "❌ Not accessible"

# Show recent logs
logs:
	@echo "📋 Recent deployment logs:"
	@echo "========================="
	@ls -la logs/ 2>/dev/null || echo "No logs directory found. Run 'make setup-logging' first."
	@echo ""
	@echo "💡 To view a specific log:"
	@echo "  cat logs/deploy_YYYYMMDD_HHMMSS.log"
	@echo ""
	@echo "🔄 To follow logs in real-time during deployment:"
	@echo "  tail -f logs/deploy_YYYYMMDD_HHMMSS.log"

# Quick deployment (without extensive logging)
quick-deploy: clean monitoring apps clusters
	@echo "🎉 Quick deployment completed!"
	@$(MAKE) status