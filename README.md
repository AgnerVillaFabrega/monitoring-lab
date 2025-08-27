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

## 📊 Dashboard Incluido

### E-commerce Lab - Dashboard Completo
Un dashboard integral que combina todas las capacidades de observabilidad en una vista unificada:

#### 🏥 Monitoreo de Servicios
- **Estado de salud** de todos los servicios (user, product, order, traffic-generator)
- **Indicadores visuales** con códigos de color (🟢 HEALTHY, 🟡 WARNING, 🔴 DOWN)
- **Conteo de requests** por servicio cada 5 minutos

#### 🚨 Análisis de Errores
- **Top errores y warnings** más frecuentes del sistema
- **Desglose por servicio** con mensajes detallados
- **Contadores de incidencias** ordenados por frecuencia
- **Nivel de severidad** (ERROR/WARNING) con indicadores visuales

#### 🚦 Métricas HTTP
- **Códigos de respuesta HTTP** en tiempo real por servicio
- **Visualización temporal** de códigos 2xx (verde), 4xx (naranja), 5xx (rojo)
- **Tasas de respuesta** apiladas para análisis de patrones

#### 💼 Eventos de Negocio
- **Métricas en tiempo real**: Logins, Registros, Órdenes, Pagos
- **Gráficos temporales** con colores diferenciados por tipo de evento
- **Stream de eventos** con trace IDs para correlación
- **Estadísticas** de último valor y máximo por métrica

#### 🐌 Análisis de Performance
- **Traces más lentas** del sistema (>100ms)
- **Tabla detallada** con duración y contexto de cada trace
- **Integración directa** con Tempo para análisis profundo

#### 📜 Logs en Tiempo Real
- **Stream unificado** de todos los servicios
- **Filtros dinámicos** por servicio usando variables de dashboard
- **Vista detallada** de logs con timestamps
- **Correlación automática** con trace context

#### ⚙️ Características Técnicas
- **Auto-refresh** cada 5 segundos
- **Variables de dashboard** para filtrado dinámico
- **Ventana temporal** configurable (por defecto: últimos 15 minutos)
- **Integración nativa** con Loki (logs) y Tempo (traces)

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
# Ver errores en tiempo real por línea de comandos
make logs-service SERVICE=user-service | grep ERROR

# En Grafana dashboard "E-commerce Lab - Dashboard Completo":
# - Revisar panel "Top Errores y Warnings del Sistema"
# - Filtrar por servicio en panel "Estado de Servicios E-commerce"
# - Ver correlación con trace IDs en "Stream de Eventos de Negocio"
# - Analizar códigos HTTP en panel "Códigos de Respuesta HTTP"
```

### 2. Análisis de Trazas Distribuidas
```bash
# En el dashboard "E-commerce Lab - Dashboard Completo":
# - Revisar sección "Traces Más Lentas" para identificar cuellos de botella
# - Hacer clic en cualquier trace para análisis detallado en Tempo
# - TraceQL queries útiles en Tempo:
{service.name="order-service"} && {span.name="create_order"}
{service.name="user-service"} && duration > 500ms
{error=true}
```

### 3. Monitoreo de Métricas de Negocio
```bash
# En sección "Métricas de Negocio - Eventos por Minuto":
# - Monitorear tasa de logins y registros de usuarios
# - Seguimiento de órdenes creadas por minuto
# - Análisis de pagos procesados
# - Stream de eventos en tiempo real con trace IDs para correlación
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
│   │   ├── dashboards/               # E-commerce Lab Dashboard completo
│   │   └── provisioning/             # Datasources automáticos (Loki + Tempo)
│   ├── loki/                         # Configuración de agregación de logs
│   └── tempo/                        # Configuración de trazas distribuidas
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