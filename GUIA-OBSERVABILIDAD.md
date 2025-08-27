# Guía Completa: Stack de Observabilidad E-commerce

## 📋 Resumen del Stack

| Componente | Puerto | Función Principal | Tecnología |
|------------|--------|-------------------|------------|
| **Grafana** | 3000 | Visualización unificada de logs, métricas y trazas | Grafana OSS |
| **Loki** | 3100 | Agregación y consulta de logs centralizados | Grafana Loki |
| **Tempo** | 3200 | Almacenamiento y consulta de trazas distribuidas | Grafana Tempo |
| **Promtail** | - | Recolección automática de logs de Docker | Grafana Promtail |
| **3 Microservicios Go** | 8081-8083 | Servicios de negocio con instrumentación completa | Go + Gin + OpenTelemetry |
| **Traffic Generator** | - | Simulación de tráfico realista y patrones de uso | Go + OpenTelemetry |

**Credenciales Grafana**: `admin/admin123`

## 🔄 Flujo Completo de Observabilidad

### Arquitectura Visual
```
[HTTP Request] → [Microservicio Go] → [OpenTelemetry Tracer] → [Tempo]
                       ↓                                          ↑
                 [Logrus Logger] → [Docker stdout] → [Promtail] → [Loki]
                                                                   ↓
                                                              [Grafana]
                                                         (Correlación por trace_id)
```

### Proceso Detallado Paso a Paso

1. **Generación de Request**
   - Traffic Generator o usuario real envía HTTP request
   - Request incluye headers HTTP estándar

2. **Recepción en Microservicio**
   - Gin middleware de OpenTelemetry intercepta automáticamente
   - Se crea un nuevo **span** con trace_id único
   - Se extrae trace context si viene de otro servicio (propagación)

3. **Instrumentación Automática**
   - Cada handler HTTP se instrumenta automáticamente
   - Se registran: método, URL, status code, duración
   - Errores se capturan como span events

4. **Logging Estructurado**
   - Logrus genera logs JSON con campos estructurados
   - **trace_id** se incluye automáticamente en cada log
   - Logs salen por Docker stdout

5. **Recolección de Logs**
   - Promtail lee logs de Docker daemon (`/var/run/docker.sock`)
   - Parsea JSON y extrae labels (service, level, etc.)
   - Envía a Loki via HTTP API

6. **Envío de Trazas**
   - OpenTelemetry SDK exporta spans a Tempo
   - Protocolo: OTLP gRPC (puerto 4317)
   - Batching automático para eficiencia

7. **Visualización en Grafana**
   - Datasources configurados: Loki (logs) + Tempo (trazas)
   - **Correlación automática** por trace_id
   - Dashboard unificado con múltiples perspectivas

## 🏗️ Componentes Técnicos Detallados

### Grafana - Centro de Observabilidad
- **Dashboard Principal**: "E-commerce Lab Overview"
- **Datasources automáticos**: Loki y Tempo pre-configurados
- **Funcionalidades clave**:
  - Log exploration con LogQL
  - Trace search con TraceQL
  - Correlación automática logs ↔ trazas
  - Service map generado automáticamente
  - Alerting (configuración manual)

### Loki - Agregación de Logs
- **Arquitectura**: Index + Chunks storage
- **Indexing**: Por labels (service, level, host)
- **Formato**: JSON logs con campos estructurados
- **LogQL queries**: Similar a PromQL pero para logs
- **Retención**: Configurable (default: ilimitada en desarrollo)

### Tempo - Trazas Distribuidas  
- **Protocolo**: OTLP (OpenTelemetry Protocol) sobre gRPC
- **Storage**: Local filesystem (desarrollo) / S3 (producción)
- **TraceQL**: Lenguaje de consulta para trazas complejas
- **Service Graph**: Generado automáticamente desde spans

### Promtail - Log Collector
- **Input**: Docker container logs via socket
- **Processing**: Label extraction, JSON parsing
- **Output**: HTTP push a Loki
- **Configuration**: YAML declarativo

### Microservicios Go - Fuentes de Telemetría
Cada servicio implementa el mismo patrón de instrumentación:

```go
// Tracer initialization
tracer := otel.Tracer("service-name")

// HTTP middleware
router.Use(otelgin.Middleware("service-name"))

// HTTP client instrumentation  
httpClient := &http.Client{
    Transport: otelhttp.NewTransport(http.DefaultTransport)
}

// Structured logging with trace context
logrus.WithField("trace_id", span.SpanContext().TraceID().String())
```

## 🌐 Servicios y Comunicación

### user-service (Puerto 8081)
**Funciones**:
- Autenticación JWT (login/logout)
- Registro de usuarios
- Gestión de perfiles
- Endpoints de favoritos

**Comunicación Saliente**:
- → product-service (obtener favoritos del usuario)

### product-service (Puerto 8082)  
**Funciones**:
- Catálogo de productos
- Búsquedas y filtros
- Gestión de inventario
- Productos trending

**Comunicación**: Solo recibe requests (no hace llamadas salientes)

### order-service (Puerto 8083)
**Funciones**:
- Creación de órdenes
- Procesamiento de pagos
- Estados de orden
- Histórico de compras

**Comunicación Saliente**:
- → user-service (validar usuario existente)
- → product-service (reservar inventario)

### traffic-generator
**Funciones**:
- Genera tráfico realista 24/7
- Simula múltiples usuarios simultáneos
- Incluye failures y timeouts intencionales

**Patrones de Tráfico**:
- **User flows** (~5s): login, registro, consulta perfiles
- **Product flows** (~4s): browning, búsquedas, inventario  
- **Order flows** (~10s): creación orden completa + pago
- **Advanced flows** (variable): analytics, recomendaciones, refunds

## 🔍 Trazabilidad Distribuida en Acción

### ¿Cómo Funciona la Correlación?

Cuando **order-service** crea una orden:

1. **Request inicial**: `POST /orders` → genera `trace_id: abc123`
2. **Span raíz**: "create_order" en order-service
3. **Child span 1**: HTTP call a user-service → hereda trace_id
4. **Child span 2**: HTTP call a product-service → hereda trace_id  
5. **Logs correlacionados**: Todos los logs de estos 3 servicios incluyen `trace_id: abc123`

### Ejemplo de Service Map Resultante
```
[traffic-generator] → [order-service] → [user-service]
                           ↓
                      [product-service]
```

**Métricas automáticas por conexión**:
- Request rate (req/s)
- Error rate (%)
- Latencia P50/P95/P99
- Success rate

## 🎯 Casos de Uso Prácticos

### 1. Debugging de Error de Producción

**Escenario**: "Los usuarios no pueden completar órdenes desde las 14:30"

**Proceso de investigación**:

1. **Grafana Dashboard** → sección "Business Metrics"
2. **Identificar patrón**: Orders success rate cayó de 95% a 30%
3. **Log Explorer** → filtrar por:
   ```
   {service="order-service"} |= "error" | json | level="ERROR"
   ```
4. **Encontrar trace_id** en log de error específico
5. **Trace Explorer** → buscar trace_id completo
6. **Analizar spans**: ¿Cuál falló? ¿user-service? ¿product-service?
7. **Log drill-down**: Ver logs específicos de span problemático

**Resultado típico**: En 2-3 minutos identificas la causa raíz

### 2. Optimización de Performance

**Escenario**: "La app está lenta, usuarios se quejan"

**Proceso de análisis**:

1. **Service Map**: Identificar servicios con alta latencia
2. **Trace search**: 
   ```
   {duration > 2s} && {service.name="order-service"}
   ```
3. **Span analysis**: ¿Qué operación consume más tiempo?
4. **Log correlation**: Ver logs de esa operación específica
5. **Pattern identification**: ¿Es siempre la misma query? ¿Timeout de red?

### 3. Monitoreo Proactivo

**Alertas sugeridas** (configuración manual en Grafana):
- Error rate > 5% por 2 minutos
- Latencia P95 > 3 segundos por 5 minutos  
- Service unavailable por 30 segundos
- Órdenes fallidas > 20% por 5 minutos

## 📊 Métricas Clave Disponibles

### Métricas Técnicas (Automáticas)
- **Request Rate**: requests/segundo por servicio y endpoint
- **Error Rate**: % de HTTP 4xx/5xx por servicio
- **Latency Distribution**: P50, P95, P99 por operación
- **Service Dependencies**: Map de llamadas inter-servicios
- **Span Duration**: Tiempo por operación específica

### Métricas de Negocio (En logs estructurados)
- **User Registration Rate**: nuevos usuarios por período
- **Login Success Rate**: % de logins exitosos vs fallidos  
- **Product Search Volume**: búsquedas por término/categoría
- **Order Completion Rate**: % órdenes finalizadas vs abandonadas
- **Payment Success Rate**: % pagos exitosos vs rechazados
- **Inventory Turnover**: productos más reservados

### KPIs Dashboard Incluidos
- Total de órdenes procesadas
- Revenue tracking (simulado)
- Productos más populares
- Patrones de tráfico por hora
- Health status de todos los servicios

## 🚀 Comandos de Operación Diaria

### Startup y Health Check
```bash
# Levantar stack completo
make up

# Verificar que todo esté funcionando
make health

# Ver el dashboard principal
open http://localhost:3000
```

### Debugging y Troubleshooting
```bash
# Ver logs de servicio específico
make logs-service SERVICE=order-service

# Ver logs con filtro de errores (usando herramientas locales)
make logs-service SERVICE=user-service | grep ERROR

# Ver estado detallado de contenedores
make status
```

### Testing y Validación
```bash  
# Probar endpoints manualmente
make test-endpoints

# Verificar conectividad a stack de observabilidad
curl http://localhost:3100/ready  # Loki
curl http://localhost:3200/ready  # Tempo  
curl http://localhost:3000/api/health  # Grafana
```

## ⚡ Quick Start para Aprendizaje

### Primer uso (5 minutos)
1. `make up` → esperar 30 segundos
2. Abrir http://localhost:3000 (admin/admin123)  
3. Ir a "E-commerce Lab Overview" dashboard
4. Observar métricas en tiempo real (traffic generator ya está corriendo)

### Simulación de error (learning exercise)
```bash
# Generar error intencional
curl -X POST http://localhost:8081/auth/login \
  -H "Content-Type: application/json" \  
  -d '{"email":"invalid","password":"wrong"}'

# Buscar en Grafana logs: level="ERROR" email="invalid"
# Encontrar trace_id en el log
# Buscar ese trace_id en Tempo para ver span completo
```

### Trace de request completo (learning exercise)
```bash
# Generar orden completa (user → product → order)
curl -X POST http://localhost:8083/orders \
  -H "Content-Type: application/json" \
  -d '{"user_id":1,"products":[{"id":1,"quantity":2}]}'

# En Grafana Tempo, buscar:
{service.name="order-service"} && {span.name="create_order"}

# Verás el service map completo: order→user + order→product
```

## 🏆 Ventajas de Este Stack

### Técnicas
- **Una sola herramienta**: Grafana para logs, métricas y trazas
- **Correlación automática**: trace_id enlaza todo
- **Open source**: Sin vendor lock-in
- **Escalable**: Loki y Tempo diseñados para volumen
- **Estándares**: OpenTelemetry es vendor-neutral

### De Negocio  
- **Mean Time To Detection**: De horas → minutos
- **Mean Time To Resolution**: De horas → minutos
- **Proactive monitoring**: Detectar antes que usuarios reporten
- **Cost effective**: Herramientas gratuitas vs soluciones comerciales
- **Learning curve**: Grafana es ampliamente conocido

### Educativas
- **Hands-on learning**: Stack completo funcionando en minutos
- **Real patterns**: Tráfico realista con failures y latencia
- **Industry standards**: OpenTelemetry, Grafana stack
- **Microservices patterns**: Service communication, distributed tracing
- **DevOps skills**: Docker, observability, monitoring

---

## 📚 Recursos de Aprendizaje

### Documentación Oficial
- [OpenTelemetry Docs](https://opentelemetry.io/docs/)
- [Grafana Observability](https://grafana.com/docs/)
- [LogQL Language](https://grafana.com/docs/loki/latest/logql/)
- [TraceQL Language](https://grafana.com/docs/tempo/latest/traceql/)

### Conceptos Clave a Dominar
1. **Distributed Tracing**: Spans, trace context, service maps
2. **Structured Logging**: JSON logs, log correlation  
3. **Observability vs Monitoring**: Diferencias y cuándo usar cada uno
4. **SRE Practices**: SLIs, SLOs, error budgets

**Stack Status**: ✅ Production-ready con patrones de la industria