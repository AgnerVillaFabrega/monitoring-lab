# E-commerce Monitoring Stack

Un stack completo de monitoreo y observabilidad para microservicios de e-commerce, construido con **Grafana**, **Loki**, **Tempo** y servicios en **Go** con trazas distribuidas automáticas.

## 🏗️ Arquitectura

### Microservicios E-commerce
- **user-service** (puerto 8081) - Autenticación y gestión de usuarios
- **product-service** (puerto 8082) - Catálogo de productos e inventario  
- **order-service** (puerto 8083) - Procesamiento de órdenes y pagos
- **traffic-generator** - Genera tráfico automático realista

### Stack de Observabilidad
- **Grafana** (puerto 3000) - Dashboards y visualización
- **Loki** (puerto 3100) - Agregación y consulta de logs
- **Tempo** (puerto 3200) - Almacenamiento de trazas distribuidas
- **Promtail** (integrado) - Recolección de logs desde contenedores Docker

## 🚀 Inicio Rápido

### Prerrequisitos
- Docker y Docker Compose
- Make (opcional pero recomendado)
- 8GB RAM disponibles
- Puertos 3000, 3100, 3200, 8081-8083 libres

### 1. Clonar e Instalar
```bash
git clone <repository-url>
cd monitoring-stack

# Instalar herramientas requeridas (si no las tienes)
make install-tools

# Descargar dependencias Go
make deps
```

### 2. Levantar el Stack Completo
```bash
# Opción 1: Usando Makefile (recomendado)
make up

# Opción 2: Docker Compose directo
docker compose up -d
```

### 3. Acceder a los Servicios
- **Grafana**: http://localhost:3000
  - Usuario: `admin` 
  - Contraseña: `admin123`
- **Servicios**: 
  - User Service: http://localhost:8081/health
  - Product Service: http://localhost:8082/health
  - Order Service: http://localhost:8083/health

## 📊 Dashboards Incluidos

### 1. E-commerce Logs Dashboard
- **Logs por servicio** con filtros avanzados
- **Métricas de errores y warnings** en tiempo real
- **Correlación de logs** con trace IDs
- **Eventos de negocio** (registros, logins, órdenes)

### 2. E-commerce Distributed Tracing  
- **Mapa de dependencias** entre servicios
- **Métricas de latencia** P95/P99
- **Búsqueda de trazas** con TraceQL
- **Análisis de performance** por endpoint

### 3. E-commerce Business Metrics
- **KPIs de negocio**: registros, logins, órdenes, pagos
- **Tasas de éxito** para operaciones críticas
- **Actividad de productos**: búsquedas, visualizaciones
- **Stream de eventos** de negocio

> **Nota**: Actualmente se incluye 1 dashboard principal (E-commerce Lab Overview) que integra logs, trazas y métricas de negocio en una vista unificada.

## 🔄 Flujos Automáticos

El **traffic-generator** simula usuarios reales con patrones de tráfico avanzados:

### Flujos Automáticos Incluidos
- **User flows**: Registro, login, perfiles, favoritos
- **Product flows**: Navegación, búsquedas, consultas de inventario  
- **Order flows**: Creación de órdenes completas con pago
- **Advanced flows**: Preferencias, productos trending, reembolsos, analytics

### Patrones de Tráfico Realistas

### Flujos de Usuario (cada 5s)
- Login con credenciales existentes
- Registro de nuevos usuarios  
- Consulta de perfiles y favoritos

### Flujos de Producto (cada 4s)
- Navegación de catálogo completo
- Búsquedas por término y categoría
- Consulta de inventario y detalles

### Flujos de Órdenes (cada 10s)
- **Flujo completo**: crear orden → procesar pago → actualizar estado
- Consulta de órdenes por usuario
- Administración de órdenes

### Comunicación Inter-Servicios
Todos los servicios están instrumentados con **OpenTelemetry** para trazabilidad completa:
- **order-service** → **user-service** (validar usuario en cada orden)
- **order-service** → **product-service** (reservar inventario)
- **user-service** → **product-service** (obtener favoritos del usuario)
- **traffic-generator** → **todos los servicios** (simula tráfico realista)

Cada request HTTP incluye **trace context propagation** automática.

## 🛠️ Comandos Make Útiles

```bash
# Gestión básica
make up              # Iniciar todos los servicios
make down            # Detener todos los servicios  
make restart         # Reiniciar servicios
make status          # Ver estado de contenedores

# Monitoreo y debug
make logs            # Ver todos los logs
make health          # Verificar salud de servicios
make test-endpoints  # Probar endpoints de API

# Desarrollo
make build           # Construir imágenes
make tidy            # Actualizar go.mod de servicios
make dev             # Modo desarrollo (rebuild)

# Mantenimiento
make clean           # Limpiar todo (¡cuidado!)
make backup          # Crear backup de datos
make restore         # Restaurar desde backup
```

## 🔍 Casos de Uso de Monitoreo

### 1. Análisis de Errores
```bash
# Ver errores en tiempo real
make logs-service SERVICE=user-service | grep ERROR

# En Grafana: usar dashboard "E-commerce Logs"
# - Filtrar por servicio y nivel de error
# - Ver correlación con trace IDs
# - Analizar patrones temporales
```

### 2. Trazas Distribuidas
```bash
# Buscar trazas específicas en Grafana "E-commerce Distributed Tracing"
# TraceQL queries útiles:
{service.name="order-service"} && {span.name="create_order"}
{service.name="user-service"} && duration > 500ms
{error=true}
```

### 3. Métricas de Negocio
```bash
# Dashboard "E-commerce Business Metrics"
# Monitorear:
# - Tasa de conversión (órdenes/visitas)
# - Éxito de pagos
# - Actividad de productos más populares
# - Patrones de registro de usuarios
```

### 4. Alertas y SLAs
Los logs incluyen métricas para configurar alertas en:
- Tasas de error > 5%
- Latencia P95 > 2s
- Fallos de pago > 10%
- Servicios no disponibles

## 🏗️ Estructura del Proyecto

```
monitoring-stack/
├── services/                          # Microservicios Go
│   ├── user-service/                 # Autenticación y usuarios
│   ├── product-service/              # Catálogo e inventario
│   ├── order-service/                # Órdenes y pagos
│   └── traffic-generator/            # Generador de tráfico
├── infrastructure/                    # Stack de observabilidad
│   ├── grafana/
│   │   ├── dashboards/               # Dashboard principal integrado
│   │   └── provisioning/             # Datasources automáticos
│   ├── loki/                         # Configuración de logs
│   └── tempo/                        # Configuración de trazas
├── docker-compose.yml                # Orquestación completa
├── Makefile                          # Comandos de gestión
└── README.md                         # Esta documentación
```

## 🔧 Configuración Avanzada

### Variables de Entorno
```bash
# En docker-compose.yml puedes ajustar:
GF_SECURITY_ADMIN_PASSWORD=tu_password
SERVICE_NAME=custom-service-name
```

### Personalizar Dashboards
1. Accede a Grafana → Dashboards
2. Duplica un dashboard existente
3. Personaliza queries y visualizaciones
4. Exporta como JSON para versionado

### Ajustar Retención de Datos
```yaml
# En infrastructure/loki/loki-config.yaml
limits_config:
  retention_period: 168h  # 7 días (ajustable)
```

## 🚨 Troubleshooting

### Servicios no inician
```bash
# Verificar puertos ocupados
netstat -tulpn | grep -E ':(3000|3100|3200|808[1-3])'

# Ver logs de inicio
make logs-service SERVICE=grafana
```

### Trazas no aparecen
```bash
# Verificar conectividad a Tempo
curl http://localhost:3200/ready

# Ver logs de servicios
make logs | grep "tempo"
```

### Performance lenta
```bash
# Verificar recursos
docker stats

# Reducir retención de logs en Loki
# Editar infrastructure/loki/loki-config.yaml
```

### Dashboards en blanco
```bash
# Verificar datasources
curl http://localhost:3100/ready  # Loki
curl http://localhost:3200/ready  # Tempo

# Reiniciar Grafana
docker compose restart grafana
```