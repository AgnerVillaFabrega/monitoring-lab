package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	serviceName = "product-service"
	servicePort = "8082"
	tracer      trace.Tracer
	httpClient  = &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
)

type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Category    string  `json:"category"`
	Stock       int     `json:"stock"`
	ImageURL    string  `json:"image_url"`
}

var products = []Product{
	{ID: 1, Name: "Laptop Gaming", Description: "High-performance gaming laptop", Price: 1299.99, Category: "Electronics", Stock: 15, ImageURL: "https://example.com/laptop.jpg"},
	{ID: 2, Name: "Smartphone Pro", Description: "Latest smartphone with AI camera", Price: 899.99, Category: "Electronics", Stock: 8, ImageURL: "https://example.com/phone.jpg"},
	{ID: 3, Name: "Running Shoes", Description: "Professional running shoes", Price: 159.99, Category: "Sports", Stock: 25, ImageURL: "https://example.com/shoes.jpg"},
	{ID: 4, Name: "Coffee Maker", Description: "Automatic espresso machine", Price: 299.99, Category: "Home", Stock: 12, ImageURL: "https://example.com/coffee.jpg"},
	{ID: 5, Name: "Wireless Headphones", Description: "Noise-cancelling headphones", Price: 199.99, Category: "Electronics", Stock: 20, ImageURL: "https://example.com/headphones.jpg"},
	{ID: 6, Name: "Yoga Mat", Description: "Premium yoga mat", Price: 49.99, Category: "Sports", Stock: 30, ImageURL: "https://example.com/yoga.jpg"},
	{ID: 7, Name: "Smart Watch", Description: "Fitness tracking smartwatch", Price: 249.99, Category: "Electronics", Stock: 5, ImageURL: "https://example.com/watch.jpg"},
	{ID: 8, Name: "Backpack", Description: "Travel backpack", Price: 79.99, Category: "Travel", Stock: 18, ImageURL: "https://example.com/backpack.jpg"},
}

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)
}

func main() {
	ctx := context.Background()

	shutdown, err := initTracer(ctx)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize tracer")
	}
	defer shutdown()

	tracer = otel.Tracer(serviceName)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(otelgin.Middleware(serviceName))
	r.Use(loggingMiddleware())

	r.GET("/health", healthHandler)
	r.GET("/products", getProductsHandler)
	r.GET("/products/:id", getProductHandler)
	r.GET("/products/search", searchProductsHandler)
	r.GET("/products/favorites/:user_id", getFavoritesHandler)
	r.GET("/inventory/:id", getInventoryHandler)
	r.POST("/inventory/:id/reserve", reserveInventoryHandler)
	r.POST("/inventory/:id/release", releaseInventoryHandler)
	r.GET("/products/trending", getTrendingProductsHandler)
	r.POST("/products/:id/view", recordProductViewHandler)
	r.GET("/products/category/:category", getProductsByCategoryHandler)
	r.PUT("/products/:id/price", updateProductPriceHandler)

	go generateAutomaticLogs()
	go simulateProductActivity()

	logrus.WithFields(logrus.Fields{
		"service": serviceName,
		"port":    servicePort,
	}).Info("Starting product service")

	if err := r.Run(":" + servicePort); err != nil {
		logrus.WithError(err).Fatal("Failed to start server")
	}
}

func initTracer(ctx context.Context) (func(), error) {
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("tempo:4317"),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion("1.0.0"),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func() {
		if err := tp.Shutdown(ctx); err != nil {
			logrus.WithError(err).Error("Error shutting down tracer provider")
		}
	}, nil
}

func loggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logrus.WithFields(logrus.Fields{
			"service":    serviceName,
			"method":     param.Method,
			"path":       param.Path,
			"status":     param.StatusCode,
			"latency":    param.Latency,
			"client_ip":  param.ClientIP,
			"user_agent": param.Request.UserAgent(),
			"request_id": param.Request.Header.Get("X-Request-ID"),
		}).Info("HTTP request")
		return ""
	})
}

func healthHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "health_check")
	defer span.End()

	span.SetAttributes(
		attribute.String("service.name", serviceName),
		attribute.String("endpoint", "/health"),
	)

	c.JSON(http.StatusOK, gin.H{
		"status":    "OK",
		"service":   serviceName,
		"timestamp": time.Now().UTC(),
		"trace_id":  span.SpanContext().TraceID().String(),
	})
}

func getProductsHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "get_products")
	defer span.End()

	span.SetAttributes(
		attribute.String("endpoint", "/products"),
		attribute.String("http.method", "GET"),
	)

	// Simulate database query latency
	queryTime := time.Duration(rand.Intn(300)+50) * time.Millisecond
	time.Sleep(queryTime)

	// Simulate cache miss scenario
	if rand.Intn(100) < 15 {
		logrus.WithFields(logrus.Fields{
			"service":    serviceName,
			"endpoint":   "/products",
			"warning":    "cache_miss",
			"query_time": queryTime,
			"trace_id":   span.SpanContext().TraceID().String(),
		}).Warn("Cache miss - querying database directly")
	}

	// Simulate database connection issues
	if rand.Intn(100) < 5 {
		span.SetAttributes(attribute.String("error", "database_timeout"))
		logrus.WithFields(logrus.Fields{
			"service":  serviceName,
			"endpoint": "/products",
			"error":    "database_timeout",
			"timeout":  "5s",
			"trace_id": span.SpanContext().TraceID().String(),
		}).Error("Database query timeout")
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database timeout"})
		return
	}

	span.SetAttributes(
		attribute.Int("products.count", len(products)),
		attribute.String("query.duration", queryTime.String()),
	)

	logrus.WithFields(logrus.Fields{
		"service":       serviceName,
		"endpoint":      "/products",
		"product_count": len(products),
		"query_time":    queryTime,
		"trace_id":      span.SpanContext().TraceID().String(),
	}).Info("Products retrieved successfully")

	c.JSON(http.StatusOK, gin.H{
		"products": products,
		"total":    len(products),
		"cached":   rand.Intn(100) > 15,
	})
}

func getProductHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "get_product")
	defer span.End()

	productID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_product_id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	span.SetAttributes(
		attribute.Int("product.id", productID),
		attribute.String("endpoint", "/products/:id"),
	)

	// Find product
	for _, product := range products {
		if product.ID == productID {
			logrus.WithFields(logrus.Fields{
				"service":    serviceName,
				"endpoint":   "/products/:id",
				"product_id": productID,
				"trace_id":   span.SpanContext().TraceID().String(),
			}).Info("Product retrieved successfully")
			
			c.JSON(http.StatusOK, product)
			return
		}
	}

	span.SetAttributes(attribute.String("error", "product_not_found"))
	logrus.WithFields(logrus.Fields{
		"service":    serviceName,
		"endpoint":   "/products/:id",
		"product_id": productID,
		"error":      "product_not_found",
		"trace_id":   span.SpanContext().TraceID().String(),
	}).Warn("Product not found")

	c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
}

func searchProductsHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "search_products")
	defer span.End()

	query := c.Query("q")
	category := c.Query("category")

	span.SetAttributes(
		attribute.String("endpoint", "/products/search"),
		attribute.String("search.query", query),
		attribute.String("search.category", category),
	)

	// Simulate search index query
	searchTime := time.Duration(rand.Intn(500)+100) * time.Millisecond
	time.Sleep(searchTime)

	var results []Product
	
	// Simple search implementation
	for _, product := range products {
		match := false
		
		if query != "" {
			if strings.Contains(strings.ToLower(product.Name), strings.ToLower(query)) ||
			   strings.Contains(strings.ToLower(product.Description), strings.ToLower(query)) {
				match = true
			}
		}
		
		if category != "" && strings.ToLower(product.Category) == strings.ToLower(category) {
			match = true
		}
		
		if query == "" && category == "" {
			match = true
		}
		
		if match {
			results = append(results, product)
		}
	}

	span.SetAttributes(
		attribute.Int("search.results", len(results)),
		attribute.String("search.duration", searchTime.String()),
	)

	logrus.WithFields(logrus.Fields{
		"service":      serviceName,
		"endpoint":     "/products/search",
		"query":        query,
		"category":     category,
		"result_count": len(results),
		"search_time":  searchTime,
		"trace_id":     span.SpanContext().TraceID().String(),
	}).Info("Product search completed")

	c.JSON(http.StatusOK, gin.H{
		"products":    results,
		"total":       len(results),
		"query":       query,
		"category":    category,
		"search_time": searchTime,
	})
}

func getFavoritesHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "get_user_favorites")
	defer span.End()

	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_user_id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	span.SetAttributes(
		attribute.Int("user.id", userID),
		attribute.String("endpoint", "/products/favorites/:user_id"),
	)

	// Simulate getting user favorites from database
	time.Sleep(time.Duration(rand.Intn(200)+50) * time.Millisecond)

	// Return random favorites for demo
	var favorites []Product
	favoriteCount := rand.Intn(4) + 1
	
	for i := 0; i < favoriteCount && i < len(products); i++ {
		favorites = append(favorites, products[rand.Intn(len(products))])
	}

	span.SetAttributes(attribute.Int("favorites.count", len(favorites)))

	logrus.WithFields(logrus.Fields{
		"service":         serviceName,
		"endpoint":        "/products/favorites/:user_id",
		"user_id":         userID,
		"favorites_count": len(favorites),
		"trace_id":        span.SpanContext().TraceID().String(),
	}).Info("User favorites retrieved successfully")

	c.JSON(http.StatusOK, gin.H{
		"user_id":   userID,
		"favorites": favorites,
		"total":     len(favorites),
	})
}

func getInventoryHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "get_inventory")
	defer span.End()

	productID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_product_id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	span.SetAttributes(
		attribute.Int("product.id", productID),
		attribute.String("endpoint", "/inventory/:id"),
	)

	// Find product
	for _, product := range products {
		if product.ID == productID {
			// Simulate inventory check latency
			time.Sleep(time.Duration(rand.Intn(100)+20) * time.Millisecond)
			
			inventory := gin.H{
				"product_id":     product.ID,
				"available":      product.Stock,
				"reserved":       rand.Intn(5),
				"reorder_level":  10,
				"last_updated":   time.Now().Add(-time.Duration(rand.Intn(60)) * time.Minute),
				"warehouse":      fmt.Sprintf("WH-%d", rand.Intn(5)+1),
			}
			
			span.SetAttributes(attribute.Int("inventory.available", product.Stock))
			
			logrus.WithFields(logrus.Fields{
				"service":    serviceName,
				"endpoint":   "/inventory/:id",
				"product_id": productID,
				"stock":      product.Stock,
				"trace_id":   span.SpanContext().TraceID().String(),
			}).Info("Inventory retrieved successfully")
			
			c.JSON(http.StatusOK, inventory)
			return
		}
	}

	span.SetAttributes(attribute.String("error", "product_not_found"))
	c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
}

func reserveInventoryHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "reserve_inventory")
	defer span.End()

	productID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_product_id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var request struct {
		Quantity int `json:"quantity"`
		OrderID  int `json:"order_id"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		span.SetAttributes(attribute.String("error", "invalid_request"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	span.SetAttributes(
		attribute.Int("product.id", productID),
		attribute.Int("quantity", request.Quantity),
		attribute.Int("order.id", request.OrderID),
		attribute.String("endpoint", "/inventory/:id/reserve"),
	)

	// Find product and check stock
	for i, product := range products {
		if product.ID == productID {
			if product.Stock < request.Quantity {
				span.SetAttributes(attribute.String("error", "insufficient_stock"))
				logrus.WithFields(logrus.Fields{
					"service":       serviceName,
					"endpoint":      "/inventory/:id/reserve",
					"product_id":    productID,
					"order_id":      request.OrderID,
					"requested":     request.Quantity,
					"available":     product.Stock,
					"error":         "insufficient_stock",
					"trace_id":      span.SpanContext().TraceID().String(),
				}).Warn("Insufficient stock for reservation")
				
				c.JSON(http.StatusConflict, gin.H{"error": "Insufficient stock"})
				return
			}
			
			// Reserve inventory
			products[i].Stock -= request.Quantity
			
			span.SetAttributes(
				attribute.Int("inventory.reserved", request.Quantity),
				attribute.Int("inventory.remaining", products[i].Stock),
			)
			
			logrus.WithFields(logrus.Fields{
				"service":       serviceName,
				"endpoint":      "/inventory/:id/reserve",
				"product_id":    productID,
				"order_id":      request.OrderID,
				"quantity":      request.Quantity,
				"remaining":     products[i].Stock,
				"trace_id":      span.SpanContext().TraceID().String(),
			}).Info("Inventory reserved successfully")
			
			c.JSON(http.StatusOK, gin.H{
				"product_id":        productID,
				"reserved_quantity": request.Quantity,
				"remaining_stock":   products[i].Stock,
				"order_id":          request.OrderID,
				"reservation_id":    fmt.Sprintf("RES-%d-%d", productID, request.OrderID),
			})
			return
		}
	}

	span.SetAttributes(attribute.String("error", "product_not_found"))
	c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
}

func releaseInventoryHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "release_inventory")
	defer span.End()

	productID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_product_id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var request struct {
		Quantity int `json:"quantity"`
		OrderID  int `json:"order_id"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		span.SetAttributes(attribute.String("error", "invalid_request"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	span.SetAttributes(
		attribute.Int("product.id", productID),
		attribute.Int("quantity", request.Quantity),
		attribute.String("endpoint", "/inventory/:id/release"),
	)

	// Find product and release stock
	for i, product := range products {
		if product.ID == productID {
			products[i].Stock += request.Quantity
			
			span.SetAttributes(
				attribute.Int("inventory.released", request.Quantity),
				attribute.Int("inventory.total", products[i].Stock),
			)
			
			logrus.WithFields(logrus.Fields{
				"service":    serviceName,
				"endpoint":   "/inventory/:id/release",
				"product_id": productID,
				"order_id":   request.OrderID,
				"quantity":   request.Quantity,
				"new_total":  products[i].Stock,
				"trace_id":   span.SpanContext().TraceID().String(),
			}).Info("Inventory released successfully")
			
			c.JSON(http.StatusOK, gin.H{
				"product_id":       productID,
				"released_quantity": request.Quantity,
				"total_stock":      products[i].Stock,
				"order_id":         request.OrderID,
			})
			return
		}
	}

	span.SetAttributes(attribute.String("error", "product_not_found"))
	c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
}

func getTrendingProductsHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "get_trending_products")
	defer span.End()

	span.SetAttributes(attribute.String("endpoint", "/products/trending"))

	// Simulate cache miss and slow query
	if rand.Intn(100) < 20 {
		span.SetAttributes(attribute.String("cache.status", "miss"))
		time.Sleep(time.Duration(rand.Intn(500)+200) * time.Millisecond)
	}

	// Simulate trending calculation errors
	if rand.Intn(100) < 8 {
		span.SetAttributes(attribute.String("error", "analytics_service_error"))
		logrus.WithFields(logrus.Fields{
			"service":  serviceName,
			"endpoint": "/products/trending",
			"error":    "analytics_calculation_failed",
			"trace_id": span.SpanContext().TraceID().String(),
		}).Error("Failed to calculate trending products")
		
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Trending calculation service unavailable"})
		return
	}

	trendingCount := rand.Intn(5) + 3
	logrus.WithFields(logrus.Fields{
		"service":        serviceName,
		"endpoint":       "/products/trending",
		"trending_count": trendingCount,
		"calculation_time": fmt.Sprintf("%dms", rand.Intn(100)+50),
		"trace_id":       span.SpanContext().TraceID().String(),
	}).Info("Trending products calculated successfully")

	c.JSON(http.StatusOK, gin.H{
		"trending_products": trendingCount,
		"period": "24h",
		"products": []gin.H{}, // Empty for demo
	})
}

func recordProductViewHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "record_product_view")
	defer span.End()

	productID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_product_id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	span.SetAttributes(
		attribute.Int("product.id", productID),
		attribute.String("endpoint", "/products/:id/view"),
	)

	// Simulate analytics service failures
	if rand.Intn(100) < 5 {
		span.SetAttributes(attribute.String("error", "analytics_service_down"))
		logrus.WithFields(logrus.Fields{
			"service":    serviceName,
			"endpoint":   "/products/:id/view",
			"product_id": productID,
			"error":      "analytics_service_unavailable",
			"trace_id":   span.SpanContext().TraceID().String(),
		}).Error("Failed to record product view - analytics service down")
		
		c.JSON(http.StatusAccepted, gin.H{"message": "View recorded offline"})
		return
	}

	logrus.WithFields(logrus.Fields{
		"service":    serviceName,
		"endpoint":   "/products/:id/view",
		"product_id": productID,
		"user_agent": c.Request.UserAgent(),
		"trace_id":   span.SpanContext().TraceID().String(),
	}).Info("Product view recorded successfully")

	c.JSON(http.StatusOK, gin.H{
		"product_id": productID,
		"view_recorded": true,
		"timestamp": time.Now(),
	})
}

func getProductsByCategoryHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "get_products_by_category")
	defer span.End()

	category := c.Param("category")
	span.SetAttributes(
		attribute.String("product.category", category),
		attribute.String("endpoint", "/products/category/:category"),
	)

	// Simulate database query latency
	time.Sleep(time.Duration(rand.Intn(150)+50) * time.Millisecond)

	var categoryProducts []Product
	for _, product := range products {
		if strings.EqualFold(product.Category, category) {
			categoryProducts = append(categoryProducts, product)
		}
	}

	// Simulate category service errors
	if rand.Intn(100) < 6 {
		span.SetAttributes(attribute.String("error", "category_index_error"))
		logrus.WithFields(logrus.Fields{
			"service":  serviceName,
			"endpoint": "/products/category/:category",
			"category": category,
			"error":    "category_index_corruption",
			"trace_id": span.SpanContext().TraceID().String(),
		}).Error("Category index corruption detected")
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Category service error"})
		return
	}

	logrus.WithFields(logrus.Fields{
		"service":       serviceName,
		"endpoint":      "/products/category/:category",
		"category":      category,
		"product_count": len(categoryProducts),
		"trace_id":      span.SpanContext().TraceID().String(),
	}).Info("Products retrieved by category")

	c.JSON(http.StatusOK, gin.H{
		"category": category,
		"count":    len(categoryProducts),
		"products": categoryProducts,
	})
}

func updateProductPriceHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "update_product_price")
	defer span.End()

	productID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_product_id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var request struct {
		Price float64 `json:"price"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		span.SetAttributes(attribute.String("error", "invalid_request"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	span.SetAttributes(
		attribute.Int("product.id", productID),
		attribute.Float64("price.new", request.Price),
		attribute.String("endpoint", "/products/:id/price"),
	)

	// Simulate pricing service validations and failures
	if rand.Intn(100) < 10 {
		span.SetAttributes(attribute.String("error", "pricing_validation_failed"))
		logrus.WithFields(logrus.Fields{
			"service":    serviceName,
			"endpoint":   "/products/:id/price",
			"product_id": productID,
			"new_price":  request.Price,
			"error":      "pricing_policy_violation",
			"trace_id":   span.SpanContext().TraceID().String(),
		}).Error("Price update failed - pricing policy violation")
		
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Price violates pricing policy"})
		return
	}

	// Update product price
	for i, product := range products {
		if product.ID == productID {
			oldPrice := product.Price
			products[i].Price = request.Price
			
			span.SetAttributes(attribute.Float64("price.old", oldPrice))
			
			logrus.WithFields(logrus.Fields{
				"service":    serviceName,
				"endpoint":   "/products/:id/price",
				"product_id": productID,
				"old_price":  oldPrice,
				"new_price":  request.Price,
				"change_pct": ((request.Price - oldPrice) / oldPrice) * 100,
				"trace_id":   span.SpanContext().TraceID().String(),
			}).Info("Product price updated successfully")
			
			c.JSON(http.StatusOK, gin.H{
				"product_id": productID,
				"old_price":  oldPrice,
				"new_price":  request.Price,
				"updated_at": time.Now(),
			})
			return
		}
	}

	span.SetAttributes(attribute.String("error", "product_not_found"))
	c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
}

func simulateProductActivity() {
	ticker := time.NewTicker(12 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			random := rand.Intn(100)
			
			if random < 15 {
				logrus.WithFields(logrus.Fields{
					"service":     serviceName,
					"component":   "pricing_engine",
					"event":       "dynamic_pricing_update",
					"products":    rand.Intn(8) + 2,
					"avg_change":  fmt.Sprintf("%.2f%%", (rand.Float64()-0.5)*10),
					"trigger":     []string{"demand", "competition", "inventory", "season"}[rand.Intn(4)],
				}).Info("Dynamic pricing updates applied")
			} else if random < 30 {
				logrus.WithFields(logrus.Fields{
					"service":      serviceName,
					"component":    "recommendation_engine",
					"event":        "recommendation_generated",
					"user_sessions": rand.Intn(50) + 20,
					"avg_accuracy":  fmt.Sprintf("%.1f%%", rand.Float64()*15+80),
				}).Info("Product recommendations generated for active sessions")
			} else if random < 45 {
				logrus.WithFields(logrus.Fields{
					"service":       serviceName,
					"component":     "inventory_sync",
					"event":         "stock_level_updated",
					"products":      rand.Intn(15) + 5,
					"source":        []string{"warehouse", "supplier", "return"}[rand.Intn(3)],
					"sync_duration": fmt.Sprintf("%dms", rand.Intn(300)+100),
				}).Info("Inventory levels synchronized")
			} else if random < 60 {
				logrus.WithFields(logrus.Fields{
					"service":   serviceName,
					"component": "product_views",
					"event":     "high_traffic_product",
					"product_id": rand.Intn(8) + 1,
					"views_per_min": rand.Intn(100) + 50,
					"conversion_rate": fmt.Sprintf("%.2f%%", rand.Float64()*8+2),
				}).Info("High traffic detected on product")
			}
		}
	}
}

func generateAutomaticLogs() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			random := rand.Intn(100)
			
			if random < 18 {
				logrus.WithFields(logrus.Fields{
					"service":   serviceName,
					"component": "inventory_manager",
					"error":     "low_stock_alert",
					"products":  rand.Intn(5) + 1,
					"threshold": 5,
					"affected_categories": []string{"Electronics", "Sports", "Home"}[rand.Intn(3)],
				}).Error("Multiple products below minimum stock threshold")
			} else if random < 30 {
				logrus.WithFields(logrus.Fields{
					"service":   serviceName,
					"component": "search_engine",
					"warning":   "slow_search_queries",
					"avg_time":  strconv.Itoa(rand.Intn(1200)+300) + "ms",
					"threshold": "300ms",
					"concurrent_searches": rand.Intn(25) + 10,
				}).Warn("Search queries performing slower than expected")
			} else if random < 45 {
				logrus.WithFields(logrus.Fields{
					"service":    serviceName,
					"component":  "cache_layer",
					"warning":    "cache_eviction_rate_high",
					"evictions":  rand.Intn(150) + 30,
					"cache_size": "512MB",
					"hit_rate":   fmt.Sprintf("%.1f%%", rand.Float64()*20+70),
				}).Warn("High cache eviction rate detected")
			} else if random < 55 {
				logrus.WithFields(logrus.Fields{
					"service":     serviceName,
					"component":   "image_service",
					"error":       "image_processing_failed",
					"failed_uploads": rand.Intn(8) + 2,
					"error_type":  []string{"format_invalid", "size_exceeded", "corrupted_file"}[rand.Intn(3)],
				}).Error("Product image processing failures")
			} else if random < 70 {
				logrus.WithFields(logrus.Fields{
					"service":      serviceName,
					"component":    "price_monitor",
					"event":        "competitor_price_change",
					"products":     rand.Intn(12) + 3,
					"avg_variance": fmt.Sprintf("%.2f%%", rand.Float64()*15+2),
					"market_trend": []string{"increase", "decrease", "stable"}[rand.Intn(3)],
				}).Info("Competitor price changes detected")
			} else {
				logrus.WithFields(logrus.Fields{
					"service":        serviceName,
					"component":      "product_catalog",
					"status":         "operational",
					"products":       len(products),
					"cache_hit_rate": strconv.Itoa(rand.Intn(20)+75) + "%",
					"search_qps":     rand.Intn(80) + 20,
					"active_categories": rand.Intn(6) + 4,
				}).Info("Product catalog operating normally")
			}
		}
	}
}