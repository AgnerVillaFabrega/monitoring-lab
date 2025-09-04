package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

var (
	// Prometheus metrics
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)
	
	httpDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)
	
	businessMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "app1_business_metric",
			Help: "Business metric example for app1",
		},
		[]string{"type"},
	)

	errorRate = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "app1_errors_total",
			Help: "Total number of errors in app1",
		},
		[]string{"type"},
	)
)

type Response struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	TraceID   string    `json:"trace_id"`
}

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpDuration)
	prometheus.MustRegister(businessMetric)
	prometheus.MustRegister(errorRate)
}

func setupTracing() (*trace.TracerProvider, error) {
	tempoEndpoint := os.Getenv("TEMPO_ENDPOINT")
	if tempoEndpoint == "" {
		tempoEndpoint = "http://tempo:4318/v1/traces"
	}

	exporter, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpoint(tempoEndpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("app1"),
			semconv.ServiceVersionKey.String("1.0.0"),
		)),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}

func logMessage(level, message string, traceID string) {
	logEntry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     level,
		"service":   "app1",
		"message":   message,
		"trace_id":  traceID,
	}
	
	logJSON, _ := json.Marshal(logEntry)
	fmt.Println(string(logJSON))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	span := oteltrace.SpanFromContext(r.Context())
	traceID := span.SpanContext().TraceID().String()
	
	logMessage("info", "Health check requested", traceID)
	
	response := Response{
		Message:   "App1 is healthy",
		Timestamp: time.Now(),
		TraceID:   traceID,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	
	httpRequestsTotal.WithLabelValues(r.Method, "/health", "200").Inc()
	httpDuration.WithLabelValues(r.Method, "/health").Observe(time.Since(start).Seconds())
	businessMetric.WithLabelValues("health_checks").Inc()
}

func dataHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	span := oteltrace.SpanFromContext(r.Context())
	traceID := span.SpanContext().TraceID().String()
	
	// Simular procesamiento con trazas
	ctx, processSpan := otel.Tracer("app1").Start(r.Context(), "process_data")
	processSpan.SetAttributes()
	
	// Simular trabajo
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	
	logMessage("info", "Processing data request", traceID)
	
	// Simular errores ocasionales
	if rand.Float32() < 0.1 {
		logMessage("error", "Random error occurred during data processing", traceID)
		errorRate.WithLabelValues("processing").Inc()
		processSpan.End()
		w.WriteHeader(http.StatusInternalServerError)
		httpRequestsTotal.WithLabelValues(r.Method, "/data", "500").Inc()
		return
	}
	
	processSpan.End()
	
	response := Response{
		Message:   "Data processed successfully",
		Timestamp: time.Now(),
		TraceID:   traceID,
	}
	
	// Simular llamada a otro servicio
	ctx, callSpan := otel.Tracer("app1").Start(ctx, "external_call")
	time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
	callSpan.End()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	
	httpRequestsTotal.WithLabelValues(r.Method, "/data", "200").Inc()
	httpDuration.WithLabelValues(r.Method, "/data").Observe(time.Since(start).Seconds())
	businessMetric.WithLabelValues("data_processed").Inc()
}

func slowHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	span := oteltrace.SpanFromContext(r.Context())
	traceID := span.SpanContext().TraceID().String()
	
	logMessage("info", "Slow endpoint called", traceID)
	
	// Simular operación lenta
	_, slowSpan := otel.Tracer("app1").Start(r.Context(), "slow_operation")
	time.Sleep(time.Duration(2+rand.Intn(3)) * time.Second)
	slowSpan.End()
	
	response := Response{
		Message:   "Slow operation completed",
		Timestamp: time.Now(),
		TraceID:   traceID,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	
	httpRequestsTotal.WithLabelValues(r.Method, "/slow", "200").Inc()
	httpDuration.WithLabelValues(r.Method, "/slow").Observe(time.Since(start).Seconds())
	businessMetric.WithLabelValues("slow_operations").Inc()
}

// Simulador de métricas de negocio
func metricsSimulator() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			businessMetric.WithLabelValues("cpu_usage").Set(rand.Float64() * 100)
			businessMetric.WithLabelValues("memory_usage").Set(rand.Float64() * 100)
			businessMetric.WithLabelValues("active_connections").Set(rand.Float64() * 50)
			
			if rand.Float32() < 0.05 {
				errorRate.WithLabelValues("background").Inc()
				logMessage("warn", "Background task warning", "")
			}
		}
	}
}

func main() {
	// Configurar trazas
	tp, err := setupTracing()
	if err != nil {
		log.Fatalf("Error setting up tracing: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	// Iniciar simulador de métricas en background
	go metricsSimulator()
	
	// Configurar rutas con instrumentación OpenTelemetry
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/data", dataHandler)
	mux.HandleFunc("/slow", slowHandler)
	
	// Envolver con instrumentación OpenTelemetry
	handler := otelhttp.NewHandler(mux, "app1")
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	logMessage("info", "App1 starting on port "+port, "")
	
	server := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}
	
	log.Fatal(server.ListenAndServe())
}