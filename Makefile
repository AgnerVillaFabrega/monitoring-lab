# Makefile para el laboratorio Zabbix en Kubernetes
# Monitoreo de cluster RKE2

.PHONY: help deploy clean status logs port-forward restart verify install-deps

NAMESPACE = monitoring
KUBECTL = kubectl
KUSTOMIZE = kustomize
NFS_SERVER ?= 192.168.1.100
NFS_PATH ?= /exports/zabbix-logs

# Colores para output
GREEN = \033[0;32m
YELLOW = \033[1;33m
RED = \033[0;31m
BLUE = \033[0;34m
NC = \033[0m

help: ## Mostrar ayuda
	@echo "$(GREEN)Laboratorio Zabbix - Kubernetes$(NC)"
	@echo "=================================="
	@echo "NFS Server: $(NFS_SERVER)"
	@echo "NFS Path: $(NFS_PATH)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "$(BLUE)%-20s$(NC) %s\n", $$1, $$2}'

check-deps: ## Verificar dependencias
	@echo "$(YELLOW)Verificando dependencias...$(NC)"
	@command -v kubectl >/dev/null 2>&1 || { echo "$(RED)ERROR: kubectl no está instalado$(NC)"; exit 1; }
	@kubectl cluster-info >/dev/null 2>&1 || { echo "$(RED)ERROR: No se puede conectar al cluster$(NC)"; exit 1; }
	@echo "$(GREEN)✓ Dependencias verificadas$(NC)"

namespace: check-deps ## Crear namespace monitoring
	@echo "$(YELLOW)Creando namespace $(NAMESPACE)...$(NC)"
	@$(KUBECTL) apply -f k8s/zabbix/base/namespace.yaml
	@$(KUBECTL) wait --for=condition=Active namespace/$(NAMESPACE) --timeout=30s
	@echo "$(GREEN)✓ Namespace $(NAMESPACE) creado$(NC)"

postgres: namespace ## Desplegar PostgreSQL
	@echo "$(YELLOW)Desplegando PostgreSQL...$(NC)"
	@$(KUBECTL) apply -f k8s/zabbix/base/postgresql.yaml
	@echo "$(BLUE)Esperando a que PostgreSQL esté listo...$(NC)"
	@$(KUBECTL) wait --for=condition=ready pod -l app=postgres -n $(NAMESPACE) --timeout=300s
	@echo "$(GREEN)✓ PostgreSQL desplegado$(NC)"

zabbix-server: postgres ## Desplegar Zabbix Server
	@echo "$(YELLOW)Desplegando Zabbix Server con NFS: $(NFS_SERVER):$(NFS_PATH)$(NC)"
	@envsubst < k8s/zabbix/base/zabbix-server.yaml | $(KUBECTL) apply -f -
	@echo "$(BLUE)Esperando a que Zabbix Server esté listo...$(NC)"
	@$(KUBECTL) wait --for=condition=available deployment/zabbix-server -n $(NAMESPACE) --timeout=300s
	@echo "$(GREEN)✓ Zabbix Server desplegado$(NC)"

zabbix-web: zabbix-server ## Desplegar Zabbix Web UI
	@echo "$(YELLOW)Desplegando Zabbix Web UI...$(NC)"
	@$(KUBECTL) apply -f k8s/zabbix/base/zabbix-web.yaml
	@echo "$(BLUE)Esperando a que Zabbix Web UI esté listo...$(NC)"
	@$(KUBECTL) wait --for=condition=available deployment/zabbix-web -n $(NAMESPACE) --timeout=300s
	@echo "$(GREEN)✓ Zabbix Web UI desplegado$(NC)"

zabbix-agents: zabbix-web ## Desplegar Zabbix Agents (DaemonSet)
	@echo "$(YELLOW)Desplegando Zabbix Agents en todos los nodos...$(NC)"
	@$(KUBECTL) apply -f k8s/zabbix/base/zabbix-agent.yaml
	@echo "$(BLUE)Esperando a que los Zabbix Agents estén listos...$(NC)"
	@$(KUBECTL) rollout status daemonset/zabbix-agent -n $(NAMESPACE) --timeout=300s
	@echo "$(GREEN)✓ Zabbix Agents desplegados$(NC)"

auto-discovery: zabbix-agents ## Desplegar sistema de auto-discovery
	@echo "$(YELLOW)Desplegando auto-discovery...$(NC)"
	@$(KUBECTL) apply -f k8s/zabbix/auto-discovery/autodiscovery-config.yaml
	@echo "$(BLUE)Esperando a que auto-discovery esté listo...$(NC)"
	@$(KUBECTL) wait --for=condition=available deployment/zabbix-autodiscovery -n $(NAMESPACE) --timeout=300s
	@echo "$(GREEN)✓ Auto-discovery desplegado$(NC)"

nfs-storage: ## Configurar almacenamiento NFS para logs Parquet
	@echo "$(YELLOW)Configurando NFS storage para logs en $(NFS_SERVER):$(NFS_PATH)...$(NC)"
	@export NFS_SERVER=$(NFS_SERVER) NFS_PATH=$(NFS_PATH) && envsubst < k8s/zabbix/nfs-parquet/nfs-pv.yaml | $(KUBECTL) apply -f -
	@export NFS_SERVER=$(NFS_SERVER) NFS_PATH=$(NFS_PATH) && envsubst < k8s/zabbix/nfs-parquet/parquet-exporter.yaml | $(KUBECTL) apply -f -
	@echo "$(GREEN)✓ NFS storage configurado$(NC)"

grafana-integration: ## Configurar integración con Grafana
	@echo "$(YELLOW)Configurando integración con Grafana...$(NC)"
	@$(KUBECTL) apply -f k8s/zabbix/grafana-integration/grafana-datasource.yaml
	@echo "$(GREEN)✓ Integración con Grafana configurada$(NC)"

deploy: auto-discovery nfs-storage grafana-integration ## Desplegar todo el stack de Zabbix
	@echo "$(GREEN)🎉 Deployment de Zabbix completado!$(NC)"
	@echo ""
	@make status
	@make access-info

status: ## Mostrar estado de todos los componentes
	@echo "$(BLUE)📊 Estado de los componentes:$(NC)"
	@echo ""
	@echo "$(YELLOW)Pods:$(NC)"
	@$(KUBECTL) get pods -n $(NAMESPACE) -o wide
	@echo ""
	@echo "$(YELLOW)Services:$(NC)"
	@$(KUBECTL) get svc -n $(NAMESPACE)
	@echo ""
	@echo "$(YELLOW)Deployments:$(NC)"
	@$(KUBECTL) get deployments -n $(NAMESPACE)
	@echo ""
	@echo "$(YELLOW)DaemonSet:$(NC)"
	@$(KUBECTL) get daemonset -n $(NAMESPACE)

logs: ## Ver logs de Zabbix Server
	@echo "$(BLUE)📋 Logs de Zabbix Server:$(NC)"
	@$(KUBECTL) logs -f deployment/zabbix-server -n $(NAMESPACE)

logs-web: ## Ver logs de Zabbix Web UI
	@echo "$(BLUE)📋 Logs de Zabbix Web UI:$(NC)"
	@$(KUBECTL) logs -f deployment/zabbix-web -n $(NAMESPACE)

logs-autodiscovery: ## Ver logs del auto-discovery
	@echo "$(BLUE)📋 Logs de Auto-discovery:$(NC)"
	@$(KUBECTL) logs -f deployment/zabbix-autodiscovery -n $(NAMESPACE)

port-forward: ## Port forward para acceder a Zabbix Web UI
	@echo "$(GREEN)🌐 Iniciando port-forward a Zabbix Web UI...$(NC)"
	@echo "Accede a: http://localhost:8080"
	@echo "Usuario: Admin | Password: zabbix"
	@echo "$(YELLOW)Presiona Ctrl+C para detener$(NC)"
	@$(KUBECTL) port-forward -n $(NAMESPACE) svc/zabbix-web 8080:80

port-forward-postgres: ## Port forward para acceder a PostgreSQL
	@echo "$(GREEN)🗄️ Iniciando port-forward a PostgreSQL...$(NC)"
	@echo "Conexión: localhost:5432"
	@echo "Database: zabbix | User: zabbix | Password: zabbix123"
	@echo "$(YELLOW)Presiona Ctrl+C para detener$(NC)"
	@$(KUBECTL) port-forward -n $(NAMESPACE) svc/postgres 5432:5432

restart-server: ## Reiniciar Zabbix Server
	@echo "$(YELLOW)Reiniciando Zabbix Server...$(NC)"
	@$(KUBECTL) rollout restart deployment/zabbix-server -n $(NAMESPACE)
	@$(KUBECTL) rollout status deployment/zabbix-server -n $(NAMESPACE)
	@echo "$(GREEN)✓ Zabbix Server reiniciado$(NC)"

restart-web: ## Reiniciar Zabbix Web UI
	@echo "$(YELLOW)Reiniciando Zabbix Web UI...$(NC)"
	@$(KUBECTL) rollout restart deployment/zabbix-web -n $(NAMESPACE)
	@$(KUBECTL) rollout status deployment/zabbix-web -n $(NAMESPACE)
	@echo "$(GREEN)✓ Zabbix Web UI reiniciado$(NC)"

restart-agents: ## Reiniciar Zabbix Agents
	@echo "$(YELLOW)Reiniciando Zabbix Agents...$(NC)"
	@$(KUBECTL) rollout restart daemonset/zabbix-agent -n $(NAMESPACE)
	@$(KUBECTL) rollout status daemonset/zabbix-agent -n $(NAMESPACE)
	@echo "$(GREEN)✓ Zabbix Agents reiniciados$(NC)"

restart-all: ## Reiniciar todos los componentes
	@make restart-server
	@make restart-web
	@make restart-agents

scale-server: ## Escalar Zabbix Server (ejemplo: make scale-server REPLICAS=2)
	@echo "$(YELLOW)Escalando Zabbix Server a $(or $(REPLICAS),1) replicas...$(NC)"
	@$(KUBECTL) scale deployment/zabbix-server --replicas=$(or $(REPLICAS),1) -n $(NAMESPACE)
	@$(KUBECTL) rollout status deployment/zabbix-server -n $(NAMESPACE)
	@echo "$(GREEN)✓ Zabbix Server escalado$(NC)"

verify: ## Verificar que todos los servicios estén funcionando
	@echo "$(BLUE)🔍 Verificando servicios...$(NC)"
	@echo ""
	@echo "$(YELLOW)Verificando PostgreSQL...$(NC)"
	@$(KUBECTL) get pods -l app=postgres -n $(NAMESPACE) -o jsonpath='{.items[0].status.phase}' | grep -q "Running" && echo "$(GREEN)✓ PostgreSQL OK$(NC)" || echo "$(RED)✗ PostgreSQL FAIL$(NC)"
	@echo ""
	@echo "$(YELLOW)Verificando Zabbix Server...$(NC)"
	@$(KUBECTL) get pods -l app=zabbix-server -n $(NAMESPACE) -o jsonpath='{.items[0].status.phase}' | grep -q "Running" && echo "$(GREEN)✓ Zabbix Server OK$(NC)" || echo "$(RED)✗ Zabbix Server FAIL$(NC)"
	@echo ""
	@echo "$(YELLOW)Verificando Zabbix Web UI...$(NC)"
	@$(KUBECTL) get pods -l app=zabbix-web -n $(NAMESPACE) -o jsonpath='{.items[0].status.phase}' | grep -q "Running" && echo "$(GREEN)✓ Zabbix Web UI OK$(NC)" || echo "$(RED)✗ Zabbix Web UI FAIL$(NC)"

clean: ## Eliminar todos los recursos de Zabbix
	@echo "$(RED)⚠️ Eliminando todos los recursos de Zabbix...$(NC)"
	@echo "$(YELLOW)Esta acción eliminará todos los datos. ¿Continuar? [y/N]$(NC)" && read ans && [ $${ans:-N} = y ]
	@$(KUBECTL) delete -f k8s/zabbix/grafana-integration/grafana-datasource.yaml --ignore-not-found=true
	@$(KUBECTL) delete -f k8s/zabbix/nfs-parquet/parquet-exporter.yaml --ignore-not-found=true
	@$(KUBECTL) delete -f k8s/zabbix/auto-discovery/autodiscovery-config.yaml --ignore-not-found=true
	@$(KUBECTL) delete -f k8s/zabbix/base/zabbix-agent.yaml --ignore-not-found=true
	@$(KUBECTL) delete -f k8s/zabbix/base/zabbix-web.yaml --ignore-not-found=true
	@$(KUBECTL) delete -f k8s/zabbix/base/zabbix-server.yaml --ignore-not-found=true
	@$(KUBECTL) delete -f k8s/zabbix/base/postgresql.yaml --ignore-not-found=true
	@$(KUBECTL) delete namespace $(NAMESPACE) --ignore-not-found=true
	@echo "$(GREEN)✓ Recursos eliminados$(NC)"

access-info: ## Mostrar información de acceso
	@echo "$(GREEN)🌐 Información de acceso:$(NC)"
	@echo ""
	@echo "$(BLUE)Zabbix Web UI:$(NC)"
	@echo "  Port Forward: make port-forward"
	@echo "  URL: http://localhost:8080"
	@echo "  Usuario: Admin"
	@echo "  Password: zabbix"
	@echo ""
	@echo "$(BLUE)PostgreSQL:$(NC)"
	@echo "  Port Forward: make port-forward-postgres"
	@echo "  Host: localhost:5432"
	@echo "  Database: zabbix"
	@echo "  Usuario: zabbix"
	@echo "  Password: zabbix123"
	@echo ""
	@echo "$(YELLOW)⚠️ IMPORTANTE: Cambiar credenciales por defecto en producción$(NC)"

# Target por defecto
.DEFAULT_GOAL := help