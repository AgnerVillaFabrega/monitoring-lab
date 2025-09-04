# Laboratorio de Monitoreo Centralizado

Este proyecto implementa un stack completo de observabilidad centralizada simulando un entorno multi-cluster con Kubernetes en Docker Desktop.

## 🏗️ Arquitectura

### Stack Central de Monitoreo (Namespace `monitoring`)
- **Grafana**: Visualización y dashboards
- **Prometheus**: Recolección y almacenamiento de métricas
- **Alertmanager**: Gestión de alertas
- **Loki**: Agregación de logs
- **Tempo**: Distributed tracing

### Aplicaciones de Ejemplo
- **App1 (Go)**: API con métricas Prometheus y trazas OpenTelemetry
- **App2 (Python FastAPI)**: API con observabilidad completa

### Clusters Simulados
- **Cluster1** (namespace `app1`): Infraestructura para App1
- **Cluster2** (namespace `app2`): Infraestructura para App2

Cada cluster incluye:
- **kube-state-metrics**: Estado de objetos Kubernetes
- **node-exporter**: Métricas de sistema
- **Prometheus Agent**: Recolección y envío al Prometheus central
- **Fluent Bit**: Recolección y envío de logs a Loki

## 🚀 Inicio Rápido

### Prerrequisitos
- Docker Desktop con Kubernetes habilitado
- kubectl configurado
- make (opcional, pero recomendado)
- Git

### Despliegue Completo (Método Recomendado)
```bash
# Ver todos los comandos disponibles
make help

# Despliegue completo con logging automático
make deploy

# Verificar estado
make status
```

### Despliegue por Componentes
```bash
# Limpiar recursos existentes
make clean

# Solo stack de monitoreo
make monitoring

# Solo aplicaciones
make apps

# Solo infraestructura de clusters
make clusters

# Despliegue rápido sin logging extenso
make quick-deploy
```

## 📊 Accesos

Una vez desplegado, podrás acceder a:

- **Grafana**: http://localhost:30000 (admin/admin)
- **Prometheus**: http://localhost:30090
- **Alertmanager**: http://localhost:30093
- **Tempo**: http://localhost:30200
- **Prometheus Agent Cluster1**: http://localhost:30091
- **Prometheus Agent Cluster2**: http://localhost:30092

## 📈 Dashboards Incluidos

1. **Cluster Overview**: Métricas de nodos y estado del cluster
2. **Application Overview**: Métricas de negocio y rendimiento de apps
3. **Logs Explorer**: Búsqueda y análisis de logs
4. **Traces Explorer**: Visualización de trazas distribuidas
5. **Alerts Overview**: Estado y detalles de alertas

## 🔧 Configuración

### Generación de Tráfico Automático

Las aplicaciones incluyen generadores de tráfico automático que:
- Crean requests HTTP periódicos
- Generan errores simulados (10-15%)
- Producen logs estructurados en JSON
- Crean trazas distribuidas
- Simulan ráfagas de tráfico

### Métricas Personalizadas

**App1 (Go)**:
- `http_requests_total`: Contador de requests HTTP
- `http_request_duration_seconds`: Latencia de requests
- `app1_business_metric`: Métricas de negocio
- `app1_errors_total`: Contador de errores

**App2 (Python)**:
- `http_requests_total`: Contador de requests HTTP
- `http_request_duration_seconds`: Latencia de requests
- `app2_business_metric`: Métricas de negocio
- `app2_errors_total`: Contador de errores

### Logs Estructurados

Todas las aplicaciones producen logs en formato JSON con:
```json
{
  \"timestamp\": \"2023-XX-XXTXX:XX:XX\",
  \"level\": \"info|warn|error\",
  \"service\": \"app1|app2\",
  \"message\": \"Mensaje descriptivo\",
  \"trace_id\": \"ID de la traza\"
}
```

## 🔍 Verificación

### Verificar Despliegue
```bash
# Usando Makefile (recomendado)
make status          # Estado básico
make check          # Verificación completa con conectividad

# Manualmente
kubectl get pods -n monitoring
kubectl get pods -n app1
kubectl get pods -n app2
```

### Verificar Métricas
```bash
# Verificar endpoints de métricas
kubectl port-forward -n app1 service/app1-service 8080:8080
curl http://localhost:8080/metrics

kubectl port-forward -n app2 service/app2-service 8000:8000
curl http://localhost:8000/metrics
```

### Verificar Logs
```bash
# Ver logs de las aplicaciones
kubectl logs -f -n app1 -l app=app1
kubectl logs -f -n app2 -l app=app2

# Ver logs de despliegue
make logs
```

## 🧹 Limpieza

```bash
# Usando Makefile (recomendado)
make clean

# Manualmente
kubectl delete namespace app1
kubectl delete namespace app2
kubectl delete namespace monitoring
```

## 📁 Estructura del Proyecto

```
├── Makefile                    # Comandos unificados
├── monitoring/                 # Stack central de monitoreo
│   ├── dashboards/            # Dashboards preconfigurados
│   ├── prometheus.yaml        # Configuración Prometheus
│   ├── grafana.yaml          # Configuración Grafana
│   ├── loki.yaml             # Configuración Loki
│   ├── tempo.yaml            # Configuración Tempo
│   └── alertmanager.yaml     # Configuración Alertmanager
├── apps/
│   ├── app1/                  # Aplicación Go
│   │   ├── cmd/
│   │   │   ├── app1/         # Código aplicación principal
│   │   │   └── traffic-generator/  # Generador de tráfico
│   │   ├── docker/           # Dockerfiles
│   │   └── k8s/             # Manifests Kubernetes
│   └── app2/                 # Aplicación Python FastAPI
│       ├── src/             # Código fuente Python
│       ├── docker/          # Dockerfiles
│       └── k8s/            # Manifests Kubernetes
├── clusters/
│   ├── cluster1/            # Infraestructura cluster1
│   └── cluster2/            # Infraestructura cluster2
└── logs/                   # Logs de despliegue (generados automáticamente)
```

## 📋 Comandos Disponibles

### Comandos Principales
```bash
make help          # Ver todos los comandos disponibles
make deploy        # Despliegue completo con logging
make clean         # Limpiar todos los recursos
make status        # Estado básico de pods
make check         # Verificación completa
```

### Build de Aplicaciones
```bash
make build         # Construir todas las imágenes
make build-app1    # Solo app1
make build-app2    # Solo app2
```

### Componentes Individuales
```bash
make monitoring    # Solo stack de monitoreo
make apps          # Solo aplicaciones
make clusters      # Solo infraestructura de clusters
```

### Utilidades
```bash
make logs          # Ver logs de despliegues
make quick-deploy  # Despliegue rápido sin logging extenso
make setup-logging # Crear directorio de logs
```

## 🔧 Personalización

### Agregar Nuevos Clusters

1. Crear directorio `clusters/clusterX/`
2. Copiar manifests de cluster existente
3. Modificar namespaces y labels
4. Actualizar configuración de Prometheus central

### Modificar Intervalos de Scraping

Editar `prometheus.yml` en cada Prometheus Agent:
```yaml
global:
  scrape_interval: 15s  # Cambiar aquí
```

### Ajustar Retención de Datos

**Prometheus**:
```yaml
args:
  - '--storage.tsdb.retention.time=200h'  # Cambiar aquí
```

**Loki**: Ver configuración en `loki.yml`

## 📚 Recursos Adicionales

- [Documentación de Prometheus](https://prometheus.io/docs/)
- [Documentación de Grafana](https://grafana.com/docs/)
- [OpenTelemetry](https://opentelemetry.io/)
- [Loki](https://grafana.com/docs/loki/)
- [Tempo](https://grafana.com/docs/tempo/)

## 🐛 Solución de Problemas

Ver [CLAUDE.md](CLAUDE.md) para instrucciones detalladas de troubleshooting y configuración avanzada.