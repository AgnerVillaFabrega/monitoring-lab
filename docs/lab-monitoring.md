Estoy trabajando en mi máquina local con Kubernetes en Docker Desktop y necesito simular un entorno de monitoreo centralizado.

El objetivo es tener un cluster de monitoreo central que reciba métricas, logs y trazas de varios clusters/proyectos (simulados).

El stack que debe instalarse en el cluster central es:

   * Grafana (visualización y paneles).
   * Prometheus (métricas).
   * Alertmanager (alertas).
   * Loki (logs recientes).
   * Tempo (tracing distribuido).

🔹 Requisitos:

1. Generar los manifests Kubernetes (YAML/Helm) para desplegar este stack en un namespace `monitoring`.

2. Crear algunos servicios de ejemplo en “otros clusters simulados” (pueden ser namespaces separados como `app1` y `app2`), que generen automáticamente:

   * Métricas (ej. endpoint `/metrics`).
   * Logs (stdout con mensajes de info, warn y error).
   * Trazas (instrumentación básica hacia Tempo).
   * Tráfico simulado sin intervención manual (por ejemplo, un sidecar o job que envíe requests a la app de forma periódica, generando carga, errores y trazas).

   🔸 Lenguajes de las apps:
   - `app1` debe implementarse en **Go**, con librerías oficiales de Prometheus y OpenTelemetry.
   - `app2` debe implementarse en **Python (FastAPI)**, usando `prometheus_client` y OpenTelemetry.

   🔸 Recolección de observabilidad de apps:
   - **Métricas**: cada app debe exponer `/metrics` y configurarse con un **Prometheus Agent** que reenvíe estas métricas al Prometheus central.
   - **Logs**: los logs generados en stdout deben ser recolectados con **Fluent Bit** y enviados al Loki del cluster central.
   - **Trazas**: las apps deben estar instrumentadas con **OpenTelemetry** y enviar trazas directamente a Tempo en el cluster central.

3. Monitoreo de los clusters simulados:
   - Cada cluster donde se desplieguen apps debe incluir:
     * **kube-state-metrics** para exponer el estado de los objetos de Kubernetes.
     * **node-exporter** para exponer métricas de CPU, memoria, red y disco de los nodos.
   - Estas métricas deben ser recolectadas por **Prometheus Agent** en cada cluster y enviadas al **Prometheus central**.
   - De esta manera, el Grafana central podrá mostrar tanto el estado de las apps como el estado de los clusters donde corren.

4. Explicar cómo configurar esa integración multi-cluster en mi entorno local (ej. qué endpoints usar, cómo exponer los servicios del cluster central para que los agentes remotos envíen datos).

5. Generar o actualizar la documentación en los archivos `README.md` y `@CLAUDE.md` describiendo la arquitectura, cómo levantar el lab y cómo agregar más clusters de prueba.

6. Generar dashboards de Grafana preconfigurados para:
   - **Cluster Overview** (métricas de nodos y objetos Kubernetes).
   - **Application Overview** (métricas y tráfico de app1 y app2).
   - **Logs** (exploración en Loki).
   - **Trazas** (visualización en Tempo).
   - **Alertas** (vista de Alertmanager con ejemplos).

🔹 Estructura de directorios sugerida:

```
/monitoring        -> manifests del stack central (Grafana, Prometheus, Alertmanager, Loki, Tempo)  
/apps  
   /app1           -> ejemplo de aplicación en Go con métricas, logs, trazas y generador de tráfico automático  
   /app2           -> ejemplo de aplicación en Python con métricas, logs, trazas y generador de tráfico automático  
/clusters  
   /cluster1       -> configuración de kube-state-metrics, node-exporter y agentes de observabilidad para app1  
   /cluster2       -> configuración de kube-state-metrics, node-exporter y agentes de observabilidad para app2  
/docs              -> documentación generada (README.md, @CLAUDE.md, diagramas si aplica)
```

El enfoque debe ser modular y flexible, pero pensado para un lab en local, de forma que pueda practicar cómo integrar proyectos externos a un cluster de monitoreo centralizado, generando tráfico, logs y trazas de manera automática, y observando tanto los recursos de las aplicaciones como los del cluster donde se ejecutan.
