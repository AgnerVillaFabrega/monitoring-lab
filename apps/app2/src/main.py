import json
import logging
import os
import random
import time
from datetime import datetime
from typing import Dict, Any

import uvicorn
from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import JSONResponse
from prometheus_client import Counter, Histogram, Gauge, generate_latest, CONTENT_TYPE_LATEST
from starlette.responses import Response

from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.instrumentation.requests import RequestsInstrumentor
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.sdk.resources import Resource

# Configurar logging estructurado
class JSONFormatter(logging.Formatter):
    def format(self, record):
        log_entry = {
            "timestamp": datetime.utcnow().isoformat(),
            "level": record.levelname.lower(),
            "service": "app2",
            "message": record.getMessage(),
        }
        
        # Agregar trace_id si está disponible
        span = trace.get_current_span()
        if span.get_span_context().is_valid:
            log_entry["trace_id"] = format(span.get_span_context().trace_id, "032x")
        
        return json.dumps(log_entry)

# Configurar logger
logger = logging.getLogger()
handler = logging.StreamHandler()
handler.setFormatter(JSONFormatter())
logger.addHandler(handler)
logger.setLevel(logging.INFO)

# Métricas Prometheus
http_requests_total = Counter(
    'http_requests_total',
    'Total HTTP requests',
    ['method', 'endpoint', 'status_code']
)

http_request_duration_seconds = Histogram(
    'http_request_duration_seconds',
    'HTTP request duration in seconds',
    ['method', 'endpoint']
)

app2_business_metric = Gauge(
    'app2_business_metric',
    'Business metrics for app2',
    ['type']
)

app2_errors_total = Counter(
    'app2_errors_total',
    'Total errors in app2',
    ['type']
)

# Configurar OpenTelemetry
def setup_tracing():
    tempo_endpoint = os.getenv("TEMPO_ENDPOINT", "http://tempo:4318/v1/traces")
    
    resource = Resource.create({"service.name": "app2", "service.version": "1.0.0"})
    
    tracer_provider = TracerProvider(resource=resource)
    trace.set_tracer_provider(tracer_provider)
    
    otlp_exporter = OTLPSpanExporter(endpoint=tempo_endpoint)
    span_processor = BatchSpanProcessor(otlp_exporter)
    tracer_provider.add_span_processor(span_processor)
    
    return tracer_provider

# Configurar FastAPI
app = FastAPI(
    title="App2 - Monitoring Lab",
    description="Python FastAPI application with observability",
    version="1.0.0"
)

# Configurar instrumentación
tracer_provider = setup_tracing()
FastAPIInstrumentor.instrument_app(app, tracer_provider=tracer_provider)
RequestsInstrumentor().instrument()

tracer = trace.get_tracer(__name__)

# Middleware para métricas
@app.middleware("http")
async def metrics_middleware(request: Request, call_next):
    start_time = time.time()
    
    response = await call_next(request)
    
    duration = time.time() - start_time
    
    # Actualizar métricas
    http_requests_total.labels(
        method=request.method,
        endpoint=request.url.path,
        status_code=response.status_code
    ).inc()
    
    http_request_duration_seconds.labels(
        method=request.method,
        endpoint=request.url.path
    ).observe(duration)
    
    return response

@app.get("/metrics")
async def metrics():
    return Response(generate_latest(), media_type=CONTENT_TYPE_LATEST)

@app.get("/health")
async def health_check():
    logger.info("Health check requested")
    
    span = trace.get_current_span()
    trace_id = format(span.get_span_context().trace_id, "032x")
    
    app2_business_metric.labels(type="health_checks").inc()
    
    return {
        "message": "App2 is healthy",
        "timestamp": datetime.utcnow().isoformat(),
        "trace_id": trace_id
    }

@app.get("/api/data")
async def get_data():
    with tracer.start_as_current_span("process_data") as span:
        logger.info("Processing data request")
        
        # Simular procesamiento
        processing_time = random.uniform(0.1, 0.5)
        time.sleep(processing_time)
        
        span.set_attribute("processing.duration", processing_time)
        
        # Simular errores ocasionales
        if random.random() < 0.15:
            logger.error("Random error occurred during data processing")
            app2_errors_total.labels(type="processing").inc()
            raise HTTPException(status_code=500, detail="Internal processing error")
        
        # Simular llamada externa
        with tracer.start_as_current_span("external_call") as external_span:
            external_time = random.uniform(0.05, 0.2)
            time.sleep(external_time)
            external_span.set_attribute("external.duration", external_time)
        
        trace_id = format(span.get_span_context().trace_id, "032x")
        app2_business_metric.labels(type="data_processed").inc()
        
        return {
            "message": "Data processed successfully",
            "timestamp": datetime.utcnow().isoformat(),
            "trace_id": trace_id,
            "processing_time": processing_time
        }

@app.get("/api/compute")
async def compute_task():
    with tracer.start_as_current_span("compute_task") as span:
        logger.info("Compute task started")
        
        # Simular tarea computacional intensiva
        compute_time = random.uniform(1.0, 3.0)
        iterations = random.randint(1000, 5000)
        
        span.set_attribute("compute.iterations", iterations)
        span.set_attribute("compute.duration", compute_time)
        
        # Simular trabajo
        result = 0
        for i in range(iterations):
            result += random.random()
            if i % 1000 == 0:
                time.sleep(compute_time / (iterations / 1000))
        
        trace_id = format(span.get_span_context().trace_id, "032x")
        app2_business_metric.labels(type="compute_tasks").inc()
        
        return {
            "message": "Compute task completed",
            "timestamp": datetime.utcnow().isoformat(),
            "trace_id": trace_id,
            "result": round(result, 2),
            "iterations": iterations
        }

@app.get("/api/database")
async def database_operation():
    with tracer.start_as_current_span("database_operation") as span:
        logger.info("Database operation started")
        
        # Simular operación de base de datos
        db_time = random.uniform(0.2, 1.0)
        
        with tracer.start_as_current_span("db_connect") as connect_span:
            time.sleep(0.1)
            connect_span.set_attribute("db.connection", "postgresql")
        
        with tracer.start_as_current_span("db_query") as query_span:
            query_span.set_attribute("db.statement", "SELECT * FROM users WHERE active = true")
            query_span.set_attribute("db.rows_affected", random.randint(10, 100))
            time.sleep(db_time)
        
        # Simular error de DB ocasional
        if random.random() < 0.08:
            logger.error("Database connection timeout")
            app2_errors_total.labels(type="database").inc()
            raise HTTPException(status_code=503, detail="Database temporarily unavailable")
        
        trace_id = format(span.get_span_context().trace_id, "032x")
        app2_business_metric.labels(type="db_operations").inc()
        
        return {
            "message": "Database operation completed",
            "timestamp": datetime.utcnow().isoformat(),
            "trace_id": trace_id,
            "query_time": round(db_time, 3)
        }

# Simulador de métricas en background
import asyncio
import threading

def metrics_simulator():
    while True:
        app2_business_metric.labels(type="cpu_usage").set(random.uniform(10, 90))
        app2_business_metric.labels(type="memory_usage").set(random.uniform(20, 80))
        app2_business_metric.labels(type="active_sessions").set(random.randint(5, 50))
        
        if random.random() < 0.03:
            app2_errors_total.labels(type="background").inc()
            logger.warning("Background task warning")
        
        time.sleep(15)

# Iniciar simulador en background
metrics_thread = threading.Thread(target=metrics_simulator, daemon=True)
metrics_thread.start()

if __name__ == "__main__":
    port = int(os.getenv("PORT", 8000))
    logger.info(f"App2 starting on port {port}")
    
    uvicorn.run(
        app,
        host="0.0.0.0",
        port=port,
        log_config=None  # Usar nuestro logger personalizado
    )