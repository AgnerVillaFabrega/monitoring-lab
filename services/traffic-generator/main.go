package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
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
	serviceName = "traffic-generator"
	tracer      trace.Tracer
	httpClient  = &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
		Timeout:   30 * time.Second,
	}
)

type User struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)
}

func main() {
	ctx := context.Background()

	// Initialize tracer
	shutdown, err := initTracer(ctx)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize tracer")
	}
	defer shutdown()

	tracer = otel.Tracer(serviceName)

	logrus.WithField("service", serviceName).Info("Starting traffic generator")

	// Wait for services to be ready
	waitForServices()

	// Start different traffic patterns
	go generateUserTraffic()
	go generateProductTraffic()
	go generateOrderTraffic()
	go generateHealthChecks()
	go generateAdvancedUserTraffic()
	go generateAdvancedProductTraffic()
	go generateAdvancedOrderTraffic()

	// Keep the program running
	select {}
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

func waitForServices() {
	services := []string{
		"http://user-service:8081/health",
		"http://product-service:8082/health",
		"http://order-service:8083/health",
	}

	for _, service := range services {
		for {
			resp, err := http.Get(service)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				logrus.WithField("service", service).Info("Service is ready")
				break
			}
			if resp != nil {
				resp.Body.Close()
			}
			logrus.WithField("service", service).Info("Waiting for service to be ready...")
			time.Sleep(5 * time.Second)
		}
	}

	logrus.Info("All services are ready, starting traffic generation")
}

func generateUserTraffic() {
	userEmails := []string{
		"customer1@example.com",
		"customer2@example.com",
		"buyer@example.com",
		"shopper@example.com",
		"user123@example.com",
	}

	for range time.Tick(5 * time.Second) {
			ctx, span := tracer.Start(context.Background(), "user_workflow")
			
			scenario := rand.Intn(100)
			
			if scenario < 30 {
				// Login scenario
				email := userEmails[rand.Intn(len(userEmails))]
				span.SetAttributes(
					attribute.String("workflow.type", "login"),
					attribute.String("user.email", email),
				)
				
				success := performLogin(ctx, email)
				if success {
					// Get user profile after successful login
					getUserProfile(ctx, rand.Intn(5)+1)
				}
				
			} else if scenario < 60 {
				// Registration scenario
				newEmail := fmt.Sprintf("newuser%d@example.com", rand.Intn(1000))
				span.SetAttributes(
					attribute.String("workflow.type", "registration"),
					attribute.String("user.email", newEmail),
				)
				
				performRegistration(ctx, newEmail)
				
		} else {
			// Get user favorites
			userID := rand.Intn(5) + 1
			span.SetAttributes(
				attribute.String("workflow.type", "favorites"),
				attribute.Int("user.id", userID),
			)
			
			getUserFavorites(ctx, userID)
		}
		
		span.End()
	}
}

func generateProductTraffic() {
	searchTerms := []string{
		"laptop", "phone", "shoes", "coffee", "headphones", "watch", "backpack",
	}
	
	categories := []string{
		"Electronics", "Sports", "Home", "Travel",
	}

	for range time.Tick(4 * time.Second) {
			ctx, span := tracer.Start(context.Background(), "product_workflow")
			
			scenario := rand.Intn(100)
			
			if scenario < 25 {
				// Browse all products
				span.SetAttributes(attribute.String("workflow.type", "browse_all"))
				getAllProducts(ctx)
				
			} else if scenario < 50 {
				// Search products
				term := searchTerms[rand.Intn(len(searchTerms))]
				span.SetAttributes(
					attribute.String("workflow.type", "search"),
					attribute.String("search.term", term),
				)
				searchProducts(ctx, term, "")
				
			} else if scenario < 75 {
				// Browse by category
				category := categories[rand.Intn(len(categories))]
				span.SetAttributes(
					attribute.String("workflow.type", "browse_category"),
					attribute.String("product.category", category),
				)
				searchProducts(ctx, "", category)
				
		} else {
			// Get specific product and inventory
			productID := rand.Intn(8) + 1
			span.SetAttributes(
				attribute.String("workflow.type", "product_details"),
				attribute.Int("product.id", productID),
			)
			
			getProduct(ctx, productID)
			getInventory(ctx, productID)
		}
		
		span.End()
	}
}

func generateOrderTraffic() {
	for range time.Tick(10 * time.Second) {
			ctx, span := tracer.Start(context.Background(), "order_workflow")
			
			scenario := rand.Intn(100)
			
			if scenario < 40 {
				// Complete order flow
				span.SetAttributes(attribute.String("workflow.type", "complete_order"))
				
				userID := rand.Intn(5) + 1
				orderID := createOrder(ctx, userID)
				
				if orderID > 0 {
					// Process payment
					processPayment(ctx, orderID)
					
					// Check order status
					time.Sleep(2 * time.Second)
					getOrder(ctx, orderID)
					
					// Update order status
					updateOrderStatus(ctx, orderID, "shipped")
				}
				
			} else if scenario < 70 {
				// Check user orders
				userID := rand.Intn(5) + 1
				span.SetAttributes(
					attribute.String("workflow.type", "check_orders"),
					attribute.Int("user.id", userID),
				)
				getUserOrders(ctx, userID)
				
		} else {
			// Browse all orders (admin scenario)
			span.SetAttributes(attribute.String("workflow.type", "browse_orders"))
			getAllOrders(ctx)
		}
		
		span.End()
	}
}

func generateHealthChecks() {
	services := []string{
		"http://user-service:8081/health",
		"http://product-service:8082/health",
		"http://order-service:8083/health",
	}

	for range time.Tick(20 * time.Second) {
			ctx, span := tracer.Start(context.Background(), "health_checks")
			span.SetAttributes(attribute.String("workflow.type", "health_monitoring"))
			
		for _, service := range services {
			makeRequest(ctx, "GET", service, nil)
		}
		
		span.End()
	}
}

// User service calls
func performLogin(ctx context.Context, email string) bool {
	childCtx, span := tracer.Start(ctx, "login_request")
	defer span.End()

	payload := map[string]string{
		"email":    email,
		"password": "password123",
	}
	
	resp, err := makeRequest(childCtx, "POST", "http://user-service:8081/auth/login", payload)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return false
	}
	
	success := resp.StatusCode == http.StatusOK
	span.SetAttributes(
		attribute.Bool("login.success", success),
		attribute.Int("http.status_code", resp.StatusCode),
	)
	
	return success
}

func performRegistration(ctx context.Context, email string) bool {
	childCtx, span := tracer.Start(ctx, "registration_request")
	defer span.End()

	payload := map[string]string{
		"email":    email,
		"name":     fmt.Sprintf("User %d", rand.Intn(1000)),
		"password": "password123",
	}
	
	resp, err := makeRequest(childCtx, "POST", "http://user-service:8081/auth/register", payload)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return false
	}
	
	success := resp.StatusCode == http.StatusCreated
	span.SetAttributes(
		attribute.Bool("registration.success", success),
		attribute.Int("http.status_code", resp.StatusCode),
	)
	
	return success
}

func getUserProfile(ctx context.Context, userID int) {
	childCtx, span := tracer.Start(ctx, "get_user_profile")
	defer span.End()

	url := fmt.Sprintf("http://user-service:8081/users/%d/profile", userID)
	resp, err := makeRequest(childCtx, "GET", url, nil)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(
		attribute.Int("user.id", userID),
		attribute.Int("http.status_code", resp.StatusCode),
	)
}

func getUserFavorites(ctx context.Context, userID int) {
	childCtx, span := tracer.Start(ctx, "get_user_favorites")
	defer span.End()

	url := fmt.Sprintf("http://user-service:8081/users/%d/favorites", userID)
	resp, err := makeRequest(childCtx, "GET", url, nil)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(
		attribute.Int("user.id", userID),
		attribute.Int("http.status_code", resp.StatusCode),
	)
}

// Product service calls
func getAllProducts(ctx context.Context) {
	childCtx, span := tracer.Start(ctx, "get_all_products")
	defer span.End()

	resp, err := makeRequest(childCtx, "GET", "http://product-service:8082/products", nil)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
}

func searchProducts(ctx context.Context, query, category string) {
	childCtx, span := tracer.Start(ctx, "search_products")
	defer span.End()

	url := "http://product-service:8082/products/search"
	if query != "" || category != "" {
		url += "?"
		if query != "" {
			url += fmt.Sprintf("q=%s", query)
		}
		if category != "" {
			if query != "" {
				url += "&"
			}
			url += fmt.Sprintf("category=%s", category)
		}
	}
	
	resp, err := makeRequest(childCtx, "GET", url, nil)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(
		attribute.String("search.query", query),
		attribute.String("search.category", category),
		attribute.Int("http.status_code", resp.StatusCode),
	)
}

func getProduct(ctx context.Context, productID int) {
	childCtx, span := tracer.Start(ctx, "get_product")
	defer span.End()

	url := fmt.Sprintf("http://product-service:8082/products/%d", productID)
	resp, err := makeRequest(childCtx, "GET", url, nil)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(
		attribute.Int("product.id", productID),
		attribute.Int("http.status_code", resp.StatusCode),
	)
}

func getInventory(ctx context.Context, productID int) {
	childCtx, span := tracer.Start(ctx, "get_inventory")
	defer span.End()

	url := fmt.Sprintf("http://product-service:8082/inventory/%d", productID)
	resp, err := makeRequest(childCtx, "GET", url, nil)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(
		attribute.Int("product.id", productID),
		attribute.Int("http.status_code", resp.StatusCode),
	)
}

// Order service calls
func createOrder(ctx context.Context, userID int) int {
	childCtx, span := tracer.Start(ctx, "create_order")
	defer span.End()

	// Create order with 1-3 random products
	itemCount := rand.Intn(3) + 1
	items := make([]map[string]int, itemCount)
	
	for i := 0; i < itemCount; i++ {
		items[i] = map[string]int{
			"product_id": rand.Intn(8) + 1,
			"quantity":   rand.Intn(3) + 1,
		}
	}
	
	payload := map[string]interface{}{
		"user_id": userID,
		"items":   items,
	}
	
	resp, err := makeRequest(childCtx, "POST", "http://order-service:8083/orders", payload)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return 0
	}
	
	span.SetAttributes(
		attribute.Int("user.id", userID),
		attribute.Int("items.count", itemCount),
		attribute.Int("http.status_code", resp.StatusCode),
	)
	
	if resp.StatusCode == http.StatusCreated {
		// Parse response to get order ID
		var result map[string]interface{}
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &result)
		
		if id, ok := result["id"]; ok {
			if orderID, ok := id.(float64); ok {
				return int(orderID)
			}
		}
	}
	
	return 0
}

func processPayment(ctx context.Context, orderID int) {
	childCtx, span := tracer.Start(ctx, "process_payment")
	defer span.End()

	payload := map[string]interface{}{
		"payment_method": []string{"credit_card", "paypal", "debit_card"}[rand.Intn(3)],
		"amount":         rand.Float64()*500 + 50,
	}
	
	url := fmt.Sprintf("http://order-service:8083/payments/%d", orderID)
	resp, err := makeRequest(childCtx, "POST", url, payload)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(
		attribute.Int("order.id", orderID),
		attribute.Int("http.status_code", resp.StatusCode),
	)
}

func getOrder(ctx context.Context, orderID int) {
	childCtx, span := tracer.Start(ctx, "get_order")
	defer span.End()

	url := fmt.Sprintf("http://order-service:8083/orders/%d", orderID)
	resp, err := makeRequest(childCtx, "GET", url, nil)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(
		attribute.Int("order.id", orderID),
		attribute.Int("http.status_code", resp.StatusCode),
	)
}

func updateOrderStatus(ctx context.Context, orderID int, status string) {
	childCtx, span := tracer.Start(ctx, "update_order_status")
	defer span.End()

	payload := map[string]string{
		"status": status,
	}
	
	url := fmt.Sprintf("http://order-service:8083/orders/%d/status", orderID)
	resp, err := makeRequest(childCtx, "PUT", url, payload)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(
		attribute.Int("order.id", orderID),
		attribute.String("order.status", status),
		attribute.Int("http.status_code", resp.StatusCode),
	)
}

func getUserOrders(ctx context.Context, userID int) {
	childCtx, span := tracer.Start(ctx, "get_user_orders")
	defer span.End()

	url := fmt.Sprintf("http://order-service:8083/orders/user/%d", userID)
	resp, err := makeRequest(childCtx, "GET", url, nil)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(
		attribute.Int("user.id", userID),
		attribute.Int("http.status_code", resp.StatusCode),
	)
}

func getAllOrders(ctx context.Context) {
	childCtx, span := tracer.Start(ctx, "get_all_orders")
	defer span.End()

	resp, err := makeRequest(childCtx, "GET", "http://order-service:8083/orders", nil)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
}

// Helper function to make HTTP requests
func makeRequest(ctx context.Context, method, url string, payload interface{}) (*http.Response, error) {
	var body io.Reader
	
	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(payloadBytes)
	}
	
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	// Inject trace context
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
	
	resp, err := httpClient.Do(req)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"service":  serviceName,
			"method":   method,
			"url":      url,
			"error":    err.Error(),
			"trace_id": trace.SpanFromContext(ctx).SpanContext().TraceID().String(),
		}).Error("HTTP request failed")
		return nil, err
	}
	
	// Always close the response body
	defer resp.Body.Close()
	
	// Read and discard the response body to prevent connection leaks
	io.Copy(io.Discard, resp.Body)
	
	logrus.WithFields(logrus.Fields{
		"service":     serviceName,
		"method":      method,
		"url":         url,
		"status_code": resp.StatusCode,
		"trace_id":    trace.SpanFromContext(ctx).SpanContext().TraceID().String(),
	}).Info("HTTP request completed")
	
	return resp, nil
}

// Advanced traffic generators using new endpoints
func generateAdvancedUserTraffic() {
	for range time.Tick(4 * time.Second) {
		ctx, span := tracer.Start(context.Background(), "advanced_user_traffic")
		
		// Randomly choose an advanced user operation
		switch rand.Intn(4) {
		case 0:
			// Update user preferences
			userID := rand.Intn(3) + 1
			updateUserPreferences(ctx, userID)
		case 1:
			// Search users
			searchUsers(ctx)
		case 2:
			// Refresh token
			refreshUserToken(ctx)
		default:
			// Get user profile (enhanced)
			userID := rand.Intn(3) + 1
			getUserProfile(ctx, userID)
		}
		
		span.End()
	}
}

func generateAdvancedProductTraffic() {
	for range time.Tick(3 * time.Second) {
		ctx, span := tracer.Start(context.Background(), "advanced_product_traffic")
		
		// Randomly choose an advanced product operation
		switch rand.Intn(5) {
		case 0:
			// Get trending products
			getTrendingProducts(ctx)
		case 1:
			// Record product view
			productID := rand.Intn(8) + 1
			recordProductView(ctx, productID)
		case 2:
			// Get products by category
			categories := []string{"Electronics", "Sports", "Home", "Travel"}
			category := categories[rand.Intn(len(categories))]
			getProductsByCategory(ctx, category)
		case 3:
			// Update product price
			productID := rand.Intn(8) + 1
			updateProductPrice(ctx, productID)
		default:
			// Regular product operations
			searchProducts(ctx, []string{"laptop", "phone", "shoes", "coffee"}[rand.Intn(4)], "")
		}
		
		span.End()
	}
}

func generateAdvancedOrderTraffic() {
	for range time.Tick(8 * time.Second) {
			ctx, span := tracer.Start(context.Background(), "advanced_order_traffic")
			
			// Randomly choose an advanced order operation
			switch rand.Intn(5) {
			case 0:
				// Cancel order
				orderID := rand.Intn(10) + 1
				cancelOrder(ctx, orderID)
			case 1:
				// Get order tracking
				orderID := rand.Intn(10) + 1
				getOrderTracking(ctx, orderID)
			case 2:
				// Process refund
				orderID := rand.Intn(10) + 1
				processRefund(ctx, orderID)
			case 3:
				// Get order analytics
				getOrderAnalytics(ctx)
		default:
			// Create and process new order
			userID := rand.Intn(3) + 1
			if orderID := createOrder(ctx, userID); orderID > 0 {
				processPayment(ctx, orderID)
			}
		}
		
		span.End()
	}
}

// New endpoint functions
func updateUserPreferences(ctx context.Context, userID int) {
	childCtx, span := tracer.Start(ctx, "update_user_preferences")
	defer span.End()

	payload := map[string]interface{}{
		"preferences": map[string]interface{}{
			"notifications":     rand.Intn(2) == 1,
			"marketing_emails":  rand.Intn(2) == 1,
			"theme":            []string{"light", "dark"}[rand.Intn(2)],
			"language":         []string{"en", "es", "fr"}[rand.Intn(3)],
		},
	}

	url := fmt.Sprintf("http://user-service:8081/users/%d/preferences", userID)
	resp, err := makeRequest(childCtx, "POST", url, payload)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(
		attribute.Int("user.id", userID),
		attribute.Int("http.status_code", resp.StatusCode),
	)
}

func searchUsers(ctx context.Context) {
	childCtx, span := tracer.Start(ctx, "search_users")
	defer span.End()

	queries := []string{"john", "jane", "alice", "test", "user"}
	query := queries[rand.Intn(len(queries))]
	
	url := fmt.Sprintf("http://user-service:8081/users/search?q=%s&limit=%d", query, rand.Intn(20)+5)
	resp, err := makeRequest(childCtx, "GET", url, nil)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(
		attribute.String("search.query", query),
		attribute.Int("http.status_code", resp.StatusCode),
	)
}

func refreshUserToken(ctx context.Context) {
	childCtx, span := tracer.Start(ctx, "refresh_user_token")
	defer span.End()

	payload := map[string]string{
		"refresh_token": fmt.Sprintf("refresh_token_%d", rand.Intn(1000)),
	}

	resp, err := makeRequest(childCtx, "POST", "http://user-service:8081/auth/refresh", payload)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
}

func getTrendingProducts(ctx context.Context) {
	childCtx, span := tracer.Start(ctx, "get_trending_products")
	defer span.End()

	resp, err := makeRequest(childCtx, "GET", "http://product-service:8082/products/trending", nil)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
}

func recordProductView(ctx context.Context, productID int) {
	childCtx, span := tracer.Start(ctx, "record_product_view")
	defer span.End()

	payload := map[string]interface{}{
		"user_agent": []string{"Chrome", "Firefox", "Safari", "Edge"}[rand.Intn(4)],
		"referrer":   []string{"google.com", "direct", "facebook.com", "twitter.com"}[rand.Intn(4)],
	}

	url := fmt.Sprintf("http://product-service:8082/products/%d/view", productID)
	resp, err := makeRequest(childCtx, "POST", url, payload)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(
		attribute.Int("product.id", productID),
		attribute.Int("http.status_code", resp.StatusCode),
	)
}

func getProductsByCategory(ctx context.Context, category string) {
	childCtx, span := tracer.Start(ctx, "get_products_by_category")
	defer span.End()

	url := fmt.Sprintf("http://product-service:8082/products/category/%s", category)
	resp, err := makeRequest(childCtx, "GET", url, nil)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(
		attribute.String("product.category", category),
		attribute.Int("http.status_code", resp.StatusCode),
	)
}

func updateProductPrice(ctx context.Context, productID int) {
	childCtx, span := tracer.Start(ctx, "update_product_price")
	defer span.End()

	payload := map[string]interface{}{
		"price": rand.Float64()*500 + 50, // Random price between 50-550
	}

	url := fmt.Sprintf("http://product-service:8082/products/%d/price", productID)
	resp, err := makeRequest(childCtx, "PUT", url, payload)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(
		attribute.Int("product.id", productID),
		attribute.Int("http.status_code", resp.StatusCode),
	)
}

func cancelOrder(ctx context.Context, orderID int) {
	childCtx, span := tracer.Start(ctx, "cancel_order")
	defer span.End()

	url := fmt.Sprintf("http://order-service:8083/orders/%d/cancel", orderID)
	resp, err := makeRequest(childCtx, "POST", url, nil)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(
		attribute.Int("order.id", orderID),
		attribute.Int("http.status_code", resp.StatusCode),
	)
}

func getOrderTracking(ctx context.Context, orderID int) {
	childCtx, span := tracer.Start(ctx, "get_order_tracking")
	defer span.End()

	url := fmt.Sprintf("http://order-service:8083/orders/%d/tracking", orderID)
	resp, err := makeRequest(childCtx, "GET", url, nil)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(
		attribute.Int("order.id", orderID),
		attribute.Int("http.status_code", resp.StatusCode),
	)
}

func processRefund(ctx context.Context, orderID int) {
	childCtx, span := tracer.Start(ctx, "process_refund")
	defer span.End()

	payload := map[string]interface{}{
		"amount": rand.Float64()*200 + 20,
		"reason": []string{"damaged_item", "wrong_size", "not_as_described", "customer_request"}[rand.Intn(4)],
	}

	url := fmt.Sprintf("http://order-service:8083/orders/%d/refund", orderID)
	resp, err := makeRequest(childCtx, "POST", url, payload)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(
		attribute.Int("order.id", orderID),
		attribute.Int("http.status_code", resp.StatusCode),
	)
}

func getOrderAnalytics(ctx context.Context) {
	childCtx, span := tracer.Start(ctx, "get_order_analytics")
	defer span.End()

	resp, err := makeRequest(childCtx, "GET", "http://order-service:8083/analytics/orders", nil)
	if err != nil {
		span.SetAttributes(attribute.String("error", err.Error()))
		return
	}
	
	span.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
}