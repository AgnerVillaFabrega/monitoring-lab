# Laboratorio Zabbix - Cluster RKE2

## Objetivo
Crear un laboratorio de monitoreo completo con:
- Instalación de Zabbix a nivel de cluster
- Monitoreo de nodos del cluster (instancias/IPs)
- Integración con nodos adicionales (load balancers, etc.)
- Integración Zabbix → Grafana

## Arquitectura del Cluster
- **Tipo:** RKE2 (administrado internamente)
- **Nodos:** Auto-discovery para N cantidad de nodos
- **Grafana:** Dentro del mismo cluster
- **Storage:** NFS para logs en formato Parquet

## Requisitos Definidos

### Monitoreo
- **Métricas:** CPU, Memoria, Red de nodos/máquinas
- **Auto-discovery:** Reconocimiento automático de nodos
- **Grafana:** Integración con Grafana existente en cluster

### Storage
- **Logs Zabbix:** NFS con formato Parquet
- **Base datos:** PostgreSQL para metadatos Zabbix

## Opciones de Instalación Zabbix

### Opción 1: Zabbix en Cluster (Recomendado)
**Ventajas:**
- Gestión centralizada con Kubernetes
- Escalabilidad automática
- Respaldos y recuperación integrados
- Actualizaciones simplificadas

**Componentes:**
- Zabbix Server (Deployment)
- Zabbix Web UI (Deployment + Service)
- PostgreSQL/MySQL (StatefulSet)
- Zabbix Agent (DaemonSet en nodos)

### Opción 2: Zabbix en VM externa
**Ventajas:**
- Independencia del cluster
- Acceso directo sin ingress

**Desventajas:**
- Gestión manual
- Punto único de falla
- Escalabilidad limitada

## Deployment con Makefile

### Comandos principales
```bash
# Ver ayuda completa
make help

# Desplegar todo el stack
make deploy

# Ver estado de componentes
make status

# Acceder a Zabbix Web UI
make port-forward

# Ver logs
make logs

# Limpiar todo
make clean
```

### Comandos por componente
```bash
# Solo PostgreSQL
make postgres

# Solo Zabbix Server
make zabbix-server

# Solo Web UI
make zabbix-web

# Solo Agents
make zabbix-agents

# Auto-discovery
make auto-discovery

# NFS Storage
make nfs-storage
```

### Comandos de mantenimiento
```bash
# Reiniciar servicios
make restart-server
make restart-web
make restart-agents
make restart-all

# Verificar servicios
make verify

# Escalar server
make scale-server REPLICAS=2

# Ver información de acceso
make access-info
```

## Arquitectura Propuesta

```
┌─────────────────────────────────────────────────────────┐
│                    RKE2 Cluster                        │
├─────────────────────────────────────────────────────────┤
│ Namespace: monitoring                                   │
│                                                         │
│ ┌─────────────────┐  ┌─────────────────┐               │
│ │   Zabbix Server │  │    PostgreSQL   │               │
│ │   (Deployment)  │──│  (StatefulSet)  │               │
│ └─────────────────┘  └─────────────────┘               │
│           │                                             │
│ ┌─────────────────┐  ┌─────────────────┐               │
│ │   Zabbix Web    │  │     Grafana     │               │
│ │   (Deployment)  │  │   (Existing)    │               │
│ └─────────────────┘  └─────────────────┘               │
│           │                    │                       │
│ ┌─────────────────────────────────────────┐             │
│ │        Zabbix Agent                     │             │
│ │        (DaemonSet)                      │             │
│ │  ┌─────┐  ┌─────┐  ┌─────┐  ┌─────┐    │             │
│ │  │Node1│  │Node2│  │Node3│  │NodeN│    │             │
│ │  └─────┘  └─────┘  └─────┘  └─────┘    │             │
│ └─────────────────────────────────────────┘             │
└─────────────────────────────────────────────────────────┘
                          │
                ┌─────────────────┐
                │   NFS Storage   │
                │ (Parquet Logs)  │
                └─────────────────┘
```

## Componentes

### 1. Zabbix Server
- **Deployment** con replicas configurables
- **Auto-discovery** vía Kubernetes API
- **Exportador de logs** a NFS en formato Parquet

### 2. PostgreSQL
- **StatefulSet** para persistencia
- **PVC** para datos de Zabbix

### 3. Zabbix Agent
- **DaemonSet** en todos los nodos
- **Métricas:** CPU, RAM, Red, Disco
- **Auto-registro** con Zabbix Server

### 4. Integración Grafana
- **Datasource** Zabbix para Grafana
- **Dashboards** pre-configurados

## Plan de Implementación
- [x] Definir arquitectura
- [x] Crear Kubernetes manifests
- [x] Configurar auto-discovery
- [x] Setup NFS + Parquet export  
- [x] Integración con Grafana
- [x] Crear Makefile para deployment
- [ ] Probar en cluster real
- [ ] Documentar troubleshooting

## Inicio Rápido

1. **Configurar IP del servidor NFS:**
```bash
# Opción 1: Variable de entorno
export NFS_SERVER=192.168.1.100
export NFS_PATH=/exports/zabbix-logs

# Opción 2: Directamente en el comando
make deploy NFS_SERVER=192.168.1.100 NFS_PATH=/exports/zabbix-logs
```

2. **Verificar cluster y desplegar:**
```bash
kubectl cluster-info
make check-deps
make deploy
```

3. **Acceder a Zabbix:**
```bash
make port-forward
# Abrir http://localhost:8080
# Usuario: Admin | Password: zabbix
```

4. **Verificar funcionamiento:**
```bash
make verify
make status
```