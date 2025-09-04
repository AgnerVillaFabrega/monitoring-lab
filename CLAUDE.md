# Documentación Técnica - Lab de Monitoreo

> **Nota**: Este laboratorio ahora utiliza un **Makefile unificado** para simplificar todas las operaciones.
> Consulta `make help` para ver todos los comandos disponibles.

## 🎯 Arquitectura Detallada

### Flujo de Datos de Observabilidad

```
┌─────────────┐    ┌──────────────┐    ┌─────────────────┐
│   App1/Go   │───▶│ Prometheus   │───▶│   Prometheus    │
│             │    │   Agent      │    │    Central     │
│  - Metrics  │    │  (Cluster1)  │    │  (Monitoring)  │
│  - Logs     │    └──────────────┘    └─────────────────┘
│  - Traces   │                                   ▲
└─────────────┘    ┌──────────────┐              │
        │          │              │              │
        ├─────────▶│  Fluent Bit  │─────────────▶│
        │          │  (Cluster1)  │              │
        │          └──────────────┘              │
        │                                        │
        └──────────────────────────────────────▶ │
                                                 │
┌─────────────┐    ┌──────────────┐              │
│ App2/Python │───▶│ Prometheus   │──────────────┘
│             │    │   Agent      │
│  - Metrics  │    │  (Cluster2)  │    ┌─────────────┐
│  - Logs     │    └──────────────┘    │    Loki     │
│  - Traces   │                        │ (Monitoring)│
└─────────────┘    ┌──────────────┐    └─────────────┘
        │          │              │           ▲
        ├─────────▶│  Fluent Bit  │──────────┘
        │          │  (Cluster2)  │
        │          └──────────────┘    ┌─────────────┐
        │                              │   Tempo     │
        └─────────────────────────────▶│ (Monitoring)│
                                       └─────────────┘
                                              ▲
                                              │
                                       ┌─────────────┐
                                       │  Grafana    │
                                       │ (Dashboard) │
                                       └─────────────┘
```

## 🔧 Configuración Multi-Cluster Local

### Endpoints de Conectividad

El stack central expone servicios usando NodePort para simular conectividad multi-cluster:

- **Prometheus Central**: `prometheus.monitoring.svc.cluster.local:9090`
- **Loki Central**: `loki.monitoring.svc.cluster.local:3100`
- **Tempo Central**: `tempo.monitoring.svc.cluster.local:4318/v1/traces`

### Remote Write Configuration

Los Prometheus Agents envían métricas usando `remote_write`:

```yaml
remote_write:
  - url: http://prometheus.monitoring.svc.cluster.local:9090/api/v1/write
    queue_config:
      max_samples_per_send: 1000
      max_shards: 200
      capacity: 2500
```

### Service Discovery

#### Kubernetes SD en Prometheus Agents

```yaml
scrape_configs:
  - job_name: 'app-pods'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names: [\"app1\"] # o [\"app2\"]
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
```

#### Labels Externos

Cada cluster se identifica con labels externos:
- `cluster`: cluster1/cluster2
- `region`: local

## 📊 Métricas Implementadas

### Métricas de Aplicación

#### App1 (Go)
```go
// HTTP Metrics
httpRequestsTotal = prometheus.NewCounterVec(...)
httpDuration = prometheus.NewHistogramVec(...)

// Business Metrics
businessMetric = prometheus.NewGaugeVec(...)
errorRate = prometheus.NewCounterVec(...)
```

#### App2 (Python)
```python
# HTTP Metrics
http_requests_total = Counter(...)
http_request_duration_seconds = Histogram(...)

# Business Metrics
app2_business_metric = Gauge(...)
app2_errors_total = Counter(...)
```

### Métricas de Infraestructura

#### Kube State Metrics
- Estado de pods: `kube_pod_status_phase`
- Estado de deployments: `kube_deployment_status_replicas`
- Estado de nodos: `kube_node_status_condition`

#### Node Exporter
- CPU: `node_cpu_seconds_total`
- Memoria: `node_memory_MemAvailable_bytes`
- Disco: `node_filesystem_avail_bytes`
- Red: `node_network_receive_bytes_total`

## 📝 Configuración de Logs

### Fluent Bit Pipeline

1. **Input**: Tail de archivos de log de contenedores
2. **Filter**: 
   - Kubernetes metadata enrichment
   - JSON parsing
   - Cluster labeling
3. **Output**: Envío a Loki con labels automáticos

### Ejemplo de Configuración

```conf
[INPUT]
    Name              tail
    Path              /var/log/containers/*app1*.log
    multiline.parser  docker, cri
    Tag               app1.*

[FILTER]
    Name                kubernetes
    Match               app1.*
    Merge_Log           On
    K8S-Logging.Parser  On

[FILTER]
    Name    modify
    Match   app1.*
    Add     cluster cluster1

[OUTPUT]
    Name            loki
    Match           app1.*
    Host            loki.monitoring.svc.cluster.local
    Port            3100
    Labels          job=fluent-bit,cluster=cluster1
```

## 🔍 Distributed Tracing

### OpenTelemetry Configuration

#### Go (App1)
```go
// OTLP HTTP Exporter
exporter, err := otlptracehttp.New(
    context.Background(),
    otlptracehttp.WithEndpoint(\"http://tempo:4318/v1/traces\"),
    otlptracehttp.WithInsecure(),
)

// Resource attributes
resource.NewWithAttributes(
    semconv.SchemaURL,
    semconv.ServiceNameKey.String(\"app1\"),
    semconv.ServiceVersionKey.String(\"1.0.0\"),
)
```

#### Python (App2)
```python
# OTLP Exporter
otlp_exporter = OTLPSpanExporter(
    endpoint=\"http://tempo:4318/v1/traces\"
)

# Resource
resource = Resource.create({
    \"service.name\": \"app2\",
    \"service.version\": \"1.0.0\"
})
```

## 🚨 Alerting Configuration

### Prometheus Rules

Crear archivo `alerts.yml`:
```yaml
groups:
  - name: application.rules
    rules:
    - alert: HighErrorRate
      expr: rate(http_requests_total{status_code=~\"5..\"}[5m]) > 0.1
      for: 2m
      labels:
        severity: warning
      annotations:
        summary: \"High error rate on {{ $labels.job }}\"
        
    - alert: HighLatency
      expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 0.5
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: \"High latency on {{ $labels.job }}\"
```

### Alertmanager Routes

```yaml
route:
  group_by: ['alertname', 'cluster']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'default'
  routes:
  - match:
      severity: critical
    receiver: 'critical-alerts'
  - match:
      cluster: cluster1
    receiver: 'cluster1-team'
```

## 🔧 Troubleshooting

### Comandos de Diagnóstico Rápido

```bash
# Verificar estado general
make status

# Verificación completa con conectividad
make check

# Ver logs de despliegue más recientes
make logs

# Limpiar y redesplegar
make clean && make deploy
```

### Verificar Conectividad Manual

```bash
# Verificar que Prometheus puede alcanzar targets
kubectl port-forward -n monitoring service/prometheus 9090:9090
# Ir a http://localhost:9090/targets

# Verificar logs de Prometheus Agent
kubectl logs -n app1 deployment/prometheus-agent

# Verificar logs de Fluent Bit
kubectl logs -n app1 daemonset/fluent-bit
```

### Debugging de Métricas

```bash
# Verificar métricas específicas de app
kubectl port-forward -n app1 service/app1-service 8080:8080
curl http://localhost:8080/metrics | grep app1_

# Verificar si las métricas llegan al Prometheus central
kubectl port-forward -n monitoring service/prometheus 9090:9090
# Query: {job=\"app1\"}
```

### Debugging de Logs

```bash
# Verificar que Loki recibe logs
kubectl port-forward -n monitoring service/loki 3100:3100
curl \"http://localhost:3100/loki/api/v1/query?query={job=\\\"fluent-bit\\\"}\"

# Verificar configuración de Fluent Bit
kubectl describe configmap -n app1 fluent-bit-config
```

### Debugging de Trazas

```bash
# Verificar que Tempo recibe trazas
kubectl port-forward -n monitoring service/tempo 3200:3200
curl \"http://localhost:3200/api/search?tags=service.name=app1\"

# Generar traza de prueba
kubectl port-forward -n app1 service/app1-service 8080:8080
curl http://localhost:8080/data
```

## 📈 Escalabilidad y Performance

### Tunning de Prometheus

```yaml
# Aumentar retención
args:
  - '--storage.tsdb.retention.time=720h'
  - '--storage.tsdb.retention.size=50GB'

# Optimizar para carga alta
global:
  scrape_interval: 30s
  evaluation_interval: 30s
```

### Configuración de Loki

```yaml
# Ajustar chunks
schema_config:
  configs:
    - from: 2020-10-24
      store: boltdb-shipper
      object_store: filesystem
      schema: v11
      index:
        prefix: index_
        period: 24h

# Limites de ingestion
limits_config:
  ingestion_rate_mb: 4
  ingestion_burst_size_mb: 6
```

### Performance de Fluent Bit

```yaml
# Optimizar buffers
[SERVICE]
    Flush         5
    Grace         30
    
[INPUT]
    Name              tail
    Mem_Buf_Limit     10MB
    Skip_Long_Lines   On
    Refresh_Interval  10
```

## 🔐 Seguridad

### RBAC Configurations

Los manifests incluyen configuraciones RBAC mínimas:
- ServiceAccounts dedicadas
- ClusterRoles con permisos específicos
- ClusterRoleBindings limitados

### Network Policies (Opcional)

Ejemplo de NetworkPolicy para limitar tráfico:
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: app1-network-policy
spec:
  podSelector:
    matchLabels:
      app: app1
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: monitoring
    ports:
    - protocol: TCP
      port: 8080
```

## 🚀 Extensiones

### Agregar Nuevas Aplicaciones

1. **Crear estructura organizada**:
   ```bash
   mkdir -p apps/app3/{src,docker,k8s}
   ```

2. **Instrumentación necesaria**:
   - Exponer métricas en `/metrics`
   - Configurar logs estructurados en JSON
   - Agregar trazas OpenTelemetry
   - Crear manifests K8s con annotations Prometheus

3. **Integrar en Makefile**:
   - Agregar target `build-app3` 
   - Actualizar target `apps` para incluir nueva aplicación
   - Crear configuración de cluster correspondiente

### Integrar con Servicios Externos

Para simular servicios externos en el lab:
```bash
# Crear servicio external
kubectl create service externalname external-api --external-name api.example.com
```

### Métricas Personalizadas

Ejemplo para agregar métricas de base de datos:
```go
dbConnections := prometheus.NewGaugeVec(
    prometheus.GaugeOpts{
        Name: \"db_connections_active\",
        Help: \"Active database connections\",
    },
    []string{\"database\", \"pool\"},
)
```

## 📊 Dashboards Personalizados

### Importar Dashboards Existentes

1. Descargar JSON de Grafana.com
2. Modificar datasources UIDs
3. Importar via UI o ConfigMap

### Crear Dashboards Programáticamente

Usar [grafana-operator](https://github.com/grafana-operator/grafana-operator) o APIs:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: custom-dashboard
  labels:
    grafana_dashboard: \"1\"
data:
  dashboard.json: |
    {
      \"dashboard\": {...},
      \"overwrite\": true
    }
```

## 🛠️ Gestión del Proyecto con Makefile

### Ventajas del Makefile Unificado

1. **Simplicidad**: Un solo comando para cada operación
2. **Logging Automático**: Registro automático de despliegues
3. **Modularidad**: Componentes individuales ejecutables
4. **Reproducibilidad**: Comandos consistentes
5. **Debugging**: Verificación integrada con `make check`

### Flujo de Trabajo Típico

```bash
# 1. Ver opciones disponibles
make help

# 2. Despliegue inicial
make deploy

# 3. Verificar estado
make check

# 4. Durante desarrollo/debugging
make clean && make quick-deploy

# 5. Ver logs si hay problemas
make logs
```

### Estructura Interna de Apps

Las aplicaciones ahora siguen una estructura estándar:

**App1 (Go)**:
```
apps/app1/
├── go.mod, go.sum          # Dependencias
├── cmd/                    # Executables
│   ├── app1/              # Aplicación principal
│   └── traffic-generator/ # Generador de tráfico
├── docker/                # Dockerfiles
└── k8s/                  # Manifests Kubernetes
```

**App2 (Python)**:
```
apps/app2/
├── requirements.txt       # Dependencias
├── src/                  # Código fuente
├── docker/              # Dockerfiles
└── k8s/                # Manifests Kubernetes
```

Esta documentación cubre todos los aspectos técnicos necesarios para entender, operar y extender el laboratorio de monitoreo.