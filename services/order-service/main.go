package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
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
	serviceName = "order-service"
	servicePort = "8083"
	tracer      trace.Tracer
	httpClient  = &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
)

type Order struct {
	ID          int         `json:"id"`
	UserID      int         `json:"user_id"`
	Items       []OrderItem `json:"items"`
	Status      string      `json:"status"`
	Total       float64     `json:"total"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	PaymentID   string      `json:"payment_id,omitempty"`
	ShippingID  string      `json:"shipping_id,omitempty"`
}

type OrderItem struct {
	ProductID int     `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
	Name      string  `json:"name"`
}

type CreateOrderRequest struct {
	UserID int `json:"user_id"`
	Items  []struct {
		ProductID int `json:"product_id"`
		Quantity  int `json:"quantity"`
	} `json:"items"`
}

type Payment struct {
	ID            string  `json:"id"`
	OrderID       int     `json:"order_id"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"`
	PaymentMethod string  `json:"payment_method"`
	ProcessedAt   time.Time `json:"processed_at"`
}

var orders = []Order{}
var orderCounter = 1

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
	r.GET("/orders", getOrdersHandler)
	r.GET("/orders/:id", getOrderHandler)
	r.POST("/orders", createOrderHandler)
	r.PUT("/orders/:id/status", updateOrderStatusHandler)
	r.GET("/orders/user/:user_id", getUserOrdersHandler)
	r.POST("/payments/:id", processPaymentHandler)
	r.POST("/orders/:id/cancel", cancelOrderHandler)
	r.GET("/orders/:id/tracking", getOrderTrackingHandler)
	r.POST("/orders/:id/refund", processRefundHandler)
	r.GET("/analytics/orders", getOrderAnalyticsHandler)
	r.GET("/payments/:id", getPaymentHandler)

	go generateAutomaticLogs()
	go simulatePaymentActivity()
	go simulateOrderStatusUpdates()

	logrus.WithFields(logrus.Fields{
		"service": serviceName,
		"port":    servicePort,
	}).Info("Starting order service")

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

func getOrdersHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "get_orders")
	defer span.End()

	span.SetAttributes(
		attribute.String("endpoint", "/orders"),
		attribute.Int("orders.count", len(orders)),
	)

	logrus.WithFields(logrus.Fields{
		"service":     serviceName,
		"endpoint":    "/orders",
		"order_count": len(orders),
		"trace_id":    span.SpanContext().TraceID().String(),
	}).Info("Orders retrieved successfully")

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"total":  len(orders),
	})
}

func getOrderHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "get_order")
	defer span.End()

	orderID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_order_id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	span.SetAttributes(
		attribute.Int("order.id", orderID),
		attribute.String("endpoint", "/orders/:id"),
	)

	for _, order := range orders {
		if order.ID == orderID {
			logrus.WithFields(logrus.Fields{
				"service":  serviceName,
				"endpoint": "/orders/:id",
				"order_id": orderID,
				"trace_id": span.SpanContext().TraceID().String(),
			}).Info("Order retrieved successfully")
			
			c.JSON(http.StatusOK, order)
			return
		}
	}

	span.SetAttributes(attribute.String("error", "order_not_found"))
	c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
}

func createOrderHandler(c *gin.Context) {
	ctx, span := tracer.Start(c.Request.Context(), "create_order")
	defer span.End()

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.SetAttributes(attribute.String("error", "invalid_request"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	span.SetAttributes(
		attribute.Int("user.id", req.UserID),
		attribute.Int("items.count", len(req.Items)),
		attribute.String("endpoint", "/orders"),
	)

	// Validate user exists
	userValid, err := validateUser(ctx, req.UserID)
	if err != nil || !userValid {
		span.SetAttributes(attribute.String("error", "user_validation_failed"))
		logrus.WithFields(logrus.Fields{
			"service":  serviceName,
			"endpoint": "/orders",
			"user_id":  req.UserID,
			"error":    "user_validation_failed",
			"trace_id": span.SpanContext().TraceID().String(),
		}).Error("User validation failed during order creation")
		
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user"})
		return
	}

	// Create order items and validate products
	var orderItems []OrderItem
	var total float64

	for _, item := range req.Items {
		// Get product details
		product, err := getProductDetails(ctx, item.ProductID)
		if err != nil {
			span.SetAttributes(attribute.String("error", "product_fetch_failed"))
			logrus.WithFields(logrus.Fields{
				"service":    serviceName,
				"endpoint":   "/orders",
				"product_id": item.ProductID,
				"error":      "product_fetch_failed",
				"trace_id":   span.SpanContext().TraceID().String(),
			}).Error("Failed to fetch product details")
			
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Product %d not found", item.ProductID)})
			return
		}

		// Reserve inventory
		reserved, err := reserveInventory(ctx, item.ProductID, item.Quantity, orderCounter)
		if err != nil || !reserved {
			span.SetAttributes(attribute.String("error", "inventory_reservation_failed"))
			logrus.WithFields(logrus.Fields{
				"service":    serviceName,
				"endpoint":   "/orders",
				"product_id": item.ProductID,
				"quantity":   item.Quantity,
				"error":      "inventory_reservation_failed",
				"trace_id":   span.SpanContext().TraceID().String(),
			}).Error("Failed to reserve inventory")
			
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Insufficient stock for product %d", item.ProductID)})
			return
		}

		orderItem := OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     product.Price,
			Name:      product.Name,
		}
		orderItems = append(orderItems, orderItem)
		total += product.Price * float64(item.Quantity)
	}

	// Create order
	order := Order{
		ID:        orderCounter,
		UserID:    req.UserID,
		Items:     orderItems,
		Status:    "pending",
		Total:     total,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	orders = append(orders, order)
	orderCounter++

	span.SetAttributes(
		attribute.Int("order.id", order.ID),
		attribute.Float64("order.total", total),
		attribute.String("order.status", order.Status),
	)

	logrus.WithFields(logrus.Fields{
		"service":    serviceName,
		"endpoint":   "/orders",
		"order_id":   order.ID,
		"user_id":    req.UserID,
		"item_count": len(orderItems),
		"total":      total,
		"trace_id":   span.SpanContext().TraceID().String(),
	}).Info("Order created successfully")

	c.JSON(http.StatusCreated, order)
}

func updateOrderStatusHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "update_order_status")
	defer span.End()

	orderID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_order_id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var request struct {
		Status string `json:"status"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		span.SetAttributes(attribute.String("error", "invalid_request"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	span.SetAttributes(
		attribute.Int("order.id", orderID),
		attribute.String("order.new_status", request.Status),
		attribute.String("endpoint", "/orders/:id/status"),
	)

	// Find and update order
	for i, order := range orders {
		if order.ID == orderID {
			oldStatus := order.Status
			orders[i].Status = request.Status
			orders[i].UpdatedAt = time.Now()
			
			span.SetAttributes(attribute.String("order.old_status", oldStatus))
			
			logrus.WithFields(logrus.Fields{
				"service":    serviceName,
				"endpoint":   "/orders/:id/status",
				"order_id":   orderID,
				"old_status": oldStatus,
				"new_status": request.Status,
				"trace_id":   span.SpanContext().TraceID().String(),
			}).Info("Order status updated successfully")
			
			c.JSON(http.StatusOK, orders[i])
			return
		}
	}

	span.SetAttributes(attribute.String("error", "order_not_found"))
	c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
}

func getUserOrdersHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "get_user_orders")
	defer span.End()

	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_user_id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	span.SetAttributes(
		attribute.Int("user.id", userID),
		attribute.String("endpoint", "/orders/user/:user_id"),
	)

	var userOrders []Order
	for _, order := range orders {
		if order.UserID == userID {
			userOrders = append(userOrders, order)
		}
	}

	span.SetAttributes(attribute.Int("user.orders.count", len(userOrders)))

	logrus.WithFields(logrus.Fields{
		"service":     serviceName,
		"endpoint":    "/orders/user/:user_id",
		"user_id":     userID,
		"order_count": len(userOrders),
		"trace_id":    span.SpanContext().TraceID().String(),
	}).Info("User orders retrieved successfully")

	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"orders":  userOrders,
		"total":   len(userOrders),
	})
}

func processPaymentHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "process_payment")
	defer span.End()

	orderID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_order_id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var request struct {
		PaymentMethod string  `json:"payment_method"`
		Amount        float64 `json:"amount"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		span.SetAttributes(attribute.String("error", "invalid_request"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	span.SetAttributes(
		attribute.Int("order.id", orderID),
		attribute.String("payment.method", request.PaymentMethod),
		attribute.Float64("payment.amount", request.Amount),
		attribute.String("endpoint", "/payments/:id"),
	)

	// Find order
	var order *Order
	for i, o := range orders {
		if o.ID == orderID {
			order = &orders[i]
			break
		}
	}

	if order == nil {
		span.SetAttributes(attribute.String("error", "order_not_found"))
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Simulate payment processing time
	processingTime := time.Duration(rand.Intn(2000)+500) * time.Millisecond
	time.Sleep(processingTime)

	// Simulate payment failures
	if rand.Intn(100) < 15 {
		span.SetAttributes(attribute.String("error", "payment_failed"))
		logrus.WithFields(logrus.Fields{
			"service":         serviceName,
			"endpoint":        "/payments/:id",
			"order_id":        orderID,
			"payment_method":  request.PaymentMethod,
			"amount":          request.Amount,
			"processing_time": processingTime,
			"error":           "payment_gateway_declined",
			"trace_id":        span.SpanContext().TraceID().String(),
		}).Error("Payment processing failed")
		
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Payment declined"})
		return
	}

	// Create payment record
	payment := Payment{
		ID:            fmt.Sprintf("PAY-%d-%d", orderID, time.Now().Unix()),
		OrderID:       orderID,
		Amount:        request.Amount,
		Status:        "completed",
		PaymentMethod: request.PaymentMethod,
		ProcessedAt:   time.Now(),
	}

	// Update order
	order.PaymentID = payment.ID
	order.Status = "paid"
	order.UpdatedAt = time.Now()

	span.SetAttributes(
		attribute.String("payment.id", payment.ID),
		attribute.String("payment.status", payment.Status),
		attribute.String("processing.duration", processingTime.String()),
	)

	logrus.WithFields(logrus.Fields{
		"service":         serviceName,
		"endpoint":        "/payments/:id",
		"order_id":        orderID,
		"payment_id":      payment.ID,
		"payment_method":  request.PaymentMethod,
		"amount":          request.Amount,
		"processing_time": processingTime,
		"trace_id":        span.SpanContext().TraceID().String(),
	}).Info("Payment processed successfully")

	c.JSON(http.StatusOK, payment)
}

func getPaymentHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "get_payment")
	defer span.End()

	orderID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_order_id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	span.SetAttributes(
		attribute.Int("order.id", orderID),
		attribute.String("endpoint", "/payments/:id"),
	)

	// Find order with payment
	for _, order := range orders {
		if order.ID == orderID && order.PaymentID != "" {
			payment := Payment{
				ID:            order.PaymentID,
				OrderID:       order.ID,
				Amount:        order.Total,
				Status:        "completed",
				PaymentMethod: "credit_card", // Simulated
				ProcessedAt:   order.UpdatedAt,
			}
			
			logrus.WithFields(logrus.Fields{
				"service":    serviceName,
				"endpoint":   "/payments/:id",
				"order_id":   orderID,
				"payment_id": order.PaymentID,
				"trace_id":   span.SpanContext().TraceID().String(),
			}).Info("Payment retrieved successfully")
			
			c.JSON(http.StatusOK, payment)
			return
		}
	}

	span.SetAttributes(attribute.String("error", "payment_not_found"))
	c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
}

// Helper functions for service communication

func validateUser(ctx context.Context, userID int) (bool, error) {
	childCtx, span := tracer.Start(ctx, "validate_user_call")
	defer span.End()

	span.SetAttributes(
		attribute.String("http.method", "GET"),
		attribute.String("http.url", fmt.Sprintf("http://user-service:8081/users/%d", userID)),
		attribute.Int("user.id", userID),
	)

	req, _ := http.NewRequestWithContext(childCtx, "GET", fmt.Sprintf("http://user-service:8081/users/%d", userID), nil)
	otel.GetTextMapPropagator().Inject(childCtx, propagation.HeaderCarrier(req.Header))
	
	resp, err := httpClient.Do(req)
	if err != nil {
		span.SetAttributes(attribute.String("error", "request_failed"))
		return false, err
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
	return resp.StatusCode == http.StatusOK, nil
}

type ProductResponse struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
}

func getProductDetails(ctx context.Context, productID int) (*ProductResponse, error) {
	childCtx, span := tracer.Start(ctx, "get_product_details_call")
	defer span.End()

	span.SetAttributes(
		attribute.String("http.method", "GET"),
		attribute.String("http.url", fmt.Sprintf("http://product-service:8082/products/%d", productID)),
		attribute.Int("product.id", productID),
	)

	req, _ := http.NewRequestWithContext(childCtx, "GET", fmt.Sprintf("http://product-service:8082/products/%d", productID), nil)
	otel.GetTextMapPropagator().Inject(childCtx, propagation.HeaderCarrier(req.Header))
	
	resp, err := httpClient.Do(req)
	if err != nil {
		span.SetAttributes(attribute.String("error", "request_failed"))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
		return nil, fmt.Errorf("product not found")
	}

	var product ProductResponse
	if err := json.NewDecoder(resp.Body).Decode(&product); err != nil {
		span.SetAttributes(attribute.String("error", "decode_failed"))
		return nil, err
	}

	span.SetAttributes(
		attribute.String("product.name", product.Name),
		attribute.Float64("product.price", product.Price),
	)

	return &product, nil
}

func reserveInventory(ctx context.Context, productID, quantity, orderID int) (bool, error) {
	childCtx, span := tracer.Start(ctx, "reserve_inventory_call")
	defer span.End()

	span.SetAttributes(
		attribute.String("http.method", "POST"),
		attribute.String("http.url", fmt.Sprintf("http://product-service:8082/inventory/%d/reserve", productID)),
		attribute.Int("product.id", productID),
		attribute.Int("quantity", quantity),
		attribute.Int("order.id", orderID),
	)

	payload := map[string]int{
		"quantity": quantity,
		"order_id": orderID,
	}
	
	payloadBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(childCtx, "POST", fmt.Sprintf("http://product-service:8082/inventory/%d/reserve", productID), bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	otel.GetTextMapPropagator().Inject(childCtx, propagation.HeaderCarrier(req.Header))
	
	resp, err := httpClient.Do(req)
	if err != nil {
		span.SetAttributes(attribute.String("error", "request_failed"))
		return false, err
	}
	defer resp.Body.Close()

	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
	return resp.StatusCode == http.StatusOK, nil
}

func cancelOrderHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "cancel_order")
	defer span.End()

	orderID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_order_id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	span.SetAttributes(
		attribute.Int("order.id", orderID),
		attribute.String("endpoint", "/orders/:id/cancel"),
	)

	// Simulate cancellation policy checks and failures
	if rand.Intn(100) < 15 {
		span.SetAttributes(attribute.String("error", "cancellation_not_allowed"))
		logrus.WithFields(logrus.Fields{
			"service":  serviceName,
			"endpoint": "/orders/:id/cancel",
			"order_id": orderID,
			"error":    "order_already_shipped",
			"trace_id": span.SpanContext().TraceID().String(),
		}).Error("Order cancellation failed - order already shipped")
		
		c.JSON(http.StatusConflict, gin.H{"error": "Cannot cancel shipped order"})
		return
	}

	// Find and cancel order
	for i, order := range orders {
		if order.ID == orderID {
			orders[i].Status = "cancelled"
			orders[i].UpdatedAt = time.Now()
			
			logrus.WithFields(logrus.Fields{
				"service":    serviceName,
				"endpoint":   "/orders/:id/cancel",
				"order_id":   orderID,
				"user_id":    order.UserID,
				"total":      order.Total,
				"cancelled_at": time.Now(),
				"trace_id":   span.SpanContext().TraceID().String(),
			}).Info("Order cancelled successfully")
			
			c.JSON(http.StatusOK, gin.H{
				"order_id": orderID,
				"status":   "cancelled",
				"cancelled_at": time.Now(),
			})
			return
		}
	}

	span.SetAttributes(attribute.String("error", "order_not_found"))
	c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
}

func getOrderTrackingHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "get_order_tracking")
	defer span.End()

	orderID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_order_id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	span.SetAttributes(
		attribute.Int("order.id", orderID),
		attribute.String("endpoint", "/orders/:id/tracking"),
	)

	// Simulate tracking service failures
	if rand.Intn(100) < 8 {
		span.SetAttributes(attribute.String("error", "tracking_service_error"))
		logrus.WithFields(logrus.Fields{
			"service":  serviceName,
			"endpoint": "/orders/:id/tracking",
			"order_id": orderID,
			"error":    "external_tracking_api_timeout",
			"trace_id": span.SpanContext().TraceID().String(),
		}).Error("Failed to retrieve tracking information")
		
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Tracking service temporarily unavailable"})
		return
	}

	// Generate fake tracking info
	trackingSteps := []string{"order_confirmed", "processing", "shipped", "in_transit", "delivered"}
	currentStep := rand.Intn(len(trackingSteps))
	
	tracking := gin.H{
		"order_id": orderID,
		"current_status": trackingSteps[currentStep],
		"estimated_delivery": time.Now().Add(time.Duration(rand.Intn(5)+1) * 24 * time.Hour),
		"tracking_number": fmt.Sprintf("TRK-%d-%d", orderID, rand.Intn(10000)),
		"carrier": []string{"UPS", "FedEx", "DHL", "USPS"}[rand.Intn(4)],
	}

	logrus.WithFields(logrus.Fields{
		"service":        serviceName,
		"endpoint":       "/orders/:id/tracking",
		"order_id":       orderID,
		"current_status": trackingSteps[currentStep],
		"trace_id":       span.SpanContext().TraceID().String(),
	}).Info("Order tracking retrieved successfully")

	c.JSON(http.StatusOK, tracking)
}

func processRefundHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "process_refund")
	defer span.End()

	orderID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_order_id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var request struct {
		Amount float64 `json:"amount"`
		Reason string  `json:"reason"`
	}
	
	if err := c.ShouldBindJSON(&request); err != nil {
		span.SetAttributes(attribute.String("error", "invalid_request"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	span.SetAttributes(
		attribute.Int("order.id", orderID),
		attribute.Float64("refund.amount", request.Amount),
		attribute.String("endpoint", "/orders/:id/refund"),
	)

	// Simulate payment gateway refund failures
	if rand.Intn(100) < 12 {
		span.SetAttributes(attribute.String("error", "refund_failed"))
		logrus.WithFields(logrus.Fields{
			"service":  serviceName,
			"endpoint": "/orders/:id/refund",
			"order_id": orderID,
			"amount":   request.Amount,
			"error":    "payment_gateway_declined",
			"trace_id": span.SpanContext().TraceID().String(),
		}).Error("Refund processing failed - payment gateway declined")
		
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Refund could not be processed"})
		return
	}

	refundID := fmt.Sprintf("REF-%d-%d", orderID, time.Now().Unix())
	
	logrus.WithFields(logrus.Fields{
		"service":   serviceName,
		"endpoint":  "/orders/:id/refund",
		"order_id":  orderID,
		"refund_id": refundID,
		"amount":    request.Amount,
		"reason":    request.Reason,
		"processed_at": time.Now(),
		"trace_id":  span.SpanContext().TraceID().String(),
	}).Info("Refund processed successfully")

	c.JSON(http.StatusOK, gin.H{
		"refund_id": refundID,
		"order_id":  orderID,
		"amount":    request.Amount,
		"status":    "processed",
		"processed_at": time.Now(),
	})
}

func getOrderAnalyticsHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "get_order_analytics")
	defer span.End()

	span.SetAttributes(attribute.String("endpoint", "/analytics/orders"))

	// Simulate analytics calculation time
	time.Sleep(time.Duration(rand.Intn(300)+100) * time.Millisecond)

	// Simulate analytics service errors
	if rand.Intn(100) < 7 {
		span.SetAttributes(attribute.String("error", "analytics_calculation_error"))
		logrus.WithFields(logrus.Fields{
			"service":  serviceName,
			"endpoint": "/analytics/orders",
			"error":    "data_aggregation_timeout",
			"trace_id": span.SpanContext().TraceID().String(),
		}).Error("Failed to calculate order analytics")
		
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Analytics service unavailable"})
		return
	}

	analytics := gin.H{
		"total_orders": len(orders) + rand.Intn(1000),
		"revenue_today": rand.Float64() * 50000 + 10000,
		"avg_order_value": rand.Float64() * 200 + 50,
		"conversion_rate": fmt.Sprintf("%.2f%%", rand.Float64() * 5 + 2),
		"top_categories": []string{"Electronics", "Sports", "Home"},
		"payment_methods": map[string]int{
			"credit_card": rand.Intn(60) + 40,
			"paypal": rand.Intn(30) + 15,
			"apple_pay": rand.Intn(20) + 10,
		},
	}

	logrus.WithFields(logrus.Fields{
		"service":       serviceName,
		"endpoint":      "/analytics/orders",
		"total_orders":  analytics["total_orders"],
		"revenue_today": analytics["revenue_today"],
		"trace_id":      span.SpanContext().TraceID().String(),
	}).Info("Order analytics calculated successfully")

	c.JSON(http.StatusOK, analytics)
}

func simulatePaymentActivity() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			random := rand.Intn(100)
			
			if random < 20 {
				logrus.WithFields(logrus.Fields{
					"service":     serviceName,
					"component":   "payment_gateway",
					"event":       "payment_declined",
					"declined_count": rand.Intn(8) + 2,
					"reason":      []string{"insufficient_funds", "expired_card", "fraud_detected", "limit_exceeded"}[rand.Intn(4)],
					"recovery_rate": fmt.Sprintf("%.1f%%", rand.Float64() * 30 + 10),
				}).Warn("Payment declines detected")
			} else if random < 35 {
				logrus.WithFields(logrus.Fields{
					"service":        serviceName,
					"component":      "payment_gateway",
					"event":          "payment_processed",
					"processed_count": rand.Intn(25) + 10,
					"total_amount":   rand.Float64() * 15000 + 5000,
					"avg_processing_time": fmt.Sprintf("%dms", rand.Intn(200) + 50),
				}).Info("Payments processed successfully")
			} else if random < 50 {
				logrus.WithFields(logrus.Fields{
					"service":    serviceName,
					"component":  "fraud_detection",
					"event":      "suspicious_activity",
					"flagged_transactions": rand.Intn(5) + 1,
					"risk_score": fmt.Sprintf("%.1f", rand.Float64() * 40 + 60),
				}).Warn("Suspicious payment activity detected")
			}
		}
	}
}

func simulateOrderStatusUpdates() {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			random := rand.Intn(100)
			
			if random < 25 {
				logrus.WithFields(logrus.Fields{
					"service":    serviceName,
					"component":  "fulfillment",
					"event":      "orders_shipped",
					"shipped_count": rand.Intn(15) + 5,
					"avg_fulfillment_time": fmt.Sprintf("%.1fh", rand.Float64() * 12 + 2),
					"carrier_distribution": map[string]int{
						"ups": rand.Intn(8) + 2,
						"fedex": rand.Intn(6) + 1,
						"usps": rand.Intn(4) + 1,
					},
				}).Info("Orders shipped to customers")
			} else if random < 45 {
				logrus.WithFields(logrus.Fields{
					"service":    serviceName,
					"component":  "order_processing",
					"event":      "orders_completed",
					"completed_count": rand.Intn(20) + 8,
					"customer_satisfaction": fmt.Sprintf("%.1f/5.0", rand.Float64() * 1.5 + 3.5),
				}).Info("Orders completed successfully")
			} else if random < 55 {
				logrus.WithFields(logrus.Fields{
					"service":    serviceName,
					"component":  "inventory_allocation",
					"event":      "stock_reserved",
					"orders_pending": rand.Intn(10) + 3,
					"reservation_conflicts": rand.Intn(3),
				}).Info("Inventory reserved for pending orders")
			}
		}
	}
}

func generateAutomaticLogs() {
	ticker := time.NewTicker(12 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			random := rand.Intn(100)
			
			if random < 18 {
				logrus.WithFields(logrus.Fields{
					"service":   serviceName,
					"component": "payment_processor",
					"error":     "payment_gateway_timeout",
					"gateway":   []string{"stripe_api", "paypal_api", "square_api"}[rand.Intn(3)],
					"timeout":   fmt.Sprintf("%ds", rand.Intn(45)+15),
					"orders":    rand.Intn(8) + 2,
					"retry_attempts": rand.Intn(3) + 1,
				}).Error("Payment gateway timeout affecting multiple orders")
			} else if random < 28 {
				logrus.WithFields(logrus.Fields{
					"service":   serviceName,
					"component": "order_processor",
					"error":     "inventory_service_unavailable",
					"attempts":  rand.Intn(5) + 2,
					"orders":    rand.Intn(15) + 3,
					"fallback_enabled": rand.Intn(2) == 1,
				}).Error("Inventory service unavailable during order processing")
			} else if random < 40 {
				logrus.WithFields(logrus.Fields{
					"service":      serviceName,
					"component":    "payment_processor",
					"warning":      "high_payment_failure_rate",
					"failure_rate": fmt.Sprintf("%.1f%%", rand.Float64()*20+10),
					"threshold":    "8%",
					"window":       "5min",
					"main_reason":  []string{"insufficient_funds", "expired_card", "fraud_detected"}[rand.Intn(3)],
				}).Warn("Payment failure rate above normal threshold")
			} else if random < 55 {
				logrus.WithFields(logrus.Fields{
					"service":           serviceName,
					"component":         "order_fulfillment",
					"warning":           "slow_order_processing",
					"avg_processing":    strconv.Itoa(rand.Intn(2000)+1000) + "ms",
					"target":            "800ms",
					"pending_orders":    rand.Intn(20) + 5,
				}).Warn("Order processing time exceeding target")
			} else if random < 65 {
				logrus.WithFields(logrus.Fields{
					"service":    serviceName,
					"component":  "fraud_detection",
					"event":      "suspicious_order_pattern",
					"flagged_orders": rand.Intn(6) + 2,
					"risk_indicators": []string{"high_velocity", "unusual_location", "card_testing"}[rand.Intn(3)],
					"manual_review_required": rand.Intn(2) == 1,
				}).Warn("Suspicious order patterns detected")
			} else if random < 75 {
				logrus.WithFields(logrus.Fields{
					"service":      serviceName,
					"component":    "order_notifications",
					"event":        "notification_delivery_failed",
					"failed_count": rand.Intn(12) + 3,
					"channels":     []string{"email", "sms", "push"}[rand.Intn(3)],
					"retry_queue_size": rand.Intn(25) + 5,
				}).Error("Order notification delivery failures")
			} else {
				logrus.WithFields(logrus.Fields{
					"service":           serviceName,
					"component":         "order_service",
					"status":            "operational",
					"pending_orders":    rand.Intn(20) + 5,
					"processing_orders": rand.Intn(25) + 10,
					"completed_today":   rand.Intn(200) + 100,
					"payment_success":   fmt.Sprintf("%.1f%%", rand.Float64()*10+85),
					"avg_order_value":   fmt.Sprintf("$%.2f", rand.Float64()*150+50),
					"customer_satisfaction": fmt.Sprintf("%.1f/5.0", rand.Float64()*1.5+3.5),
				}).Info("Order service operating normally")
			}
		}
	}
}