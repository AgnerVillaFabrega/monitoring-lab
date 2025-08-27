package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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
	serviceName = "user-service"
	servicePort = "8081"
	tracer      trace.Tracer
	jwtSecret   = "your-secret-key"
	httpClient  = &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
)

type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

var users = []User{
	{ID: 1, Email: "john@example.com", Name: "John Doe", Password: "password123"},
	{ID: 2, Email: "jane@example.com", Name: "Jane Smith", Password: "password123"},
	{ID: 3, Email: "alice@example.com", Name: "Alice Johnson", Password: "password123"},
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
	r.POST("/auth/login", loginHandler)
	r.POST("/auth/register", registerHandler)
	r.GET("/users/:id", getUserHandler)
	r.GET("/users/:id/profile", getUserProfileHandler)
	r.GET("/users/:id/favorites", getUserFavoritesHandler)
	r.POST("/users/:id/preferences", updateUserPreferencesHandler)
	r.GET("/users/search", searchUsersHandler)
	r.POST("/auth/refresh", refreshTokenHandler)

	go generateAutomaticLogs()

	logrus.WithFields(logrus.Fields{
		"service": serviceName,
		"port":    servicePort,
	}).Info("Starting user service")

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

func loginHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "user_login")
	defer span.End()

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.SetAttributes(attribute.String("error", "invalid_request"))
		logrus.WithFields(logrus.Fields{
			"service":  serviceName,
			"endpoint": "/auth/login",
			"error":    "invalid_request",
			"trace_id": span.SpanContext().TraceID().String(),
		}).Error("Invalid login request")
		
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	span.SetAttributes(
		attribute.String("user.email", req.Email),
		attribute.String("endpoint", "/auth/login"),
	)

	// Simulate database connection issues
	if rand.Intn(100) < 15 {
		span.SetAttributes(attribute.String("error", "database_connection_failed"))
		logrus.WithFields(logrus.Fields{
			"service":  serviceName,
			"endpoint": "/auth/login",
			"email":    req.Email,
			"error":    "database_connection_failed",
			"trace_id": span.SpanContext().TraceID().String(),
		}).Error("Database connection failed during login")
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Find user
	var user *User
	for _, u := range users {
		if u.Email == req.Email && u.Password == req.Password {
			user = &u
			break
		}
	}

	if user == nil {
		span.SetAttributes(attribute.String("error", "invalid_credentials"))
		logrus.WithFields(logrus.Fields{
			"service":  serviceName,
			"endpoint": "/auth/login",
			"email":    req.Email,
			"error":    "invalid_credentials",
			"trace_id": span.SpanContext().TraceID().String(),
		}).Warn("Invalid login attempt")
		
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		span.SetAttributes(attribute.String("error", "token_generation_failed"))
		logrus.WithFields(logrus.Fields{
			"service":  serviceName,
			"error":    "token_generation_failed",
			"trace_id": span.SpanContext().TraceID().String(),
		}).Error("Failed to generate JWT token")
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	span.SetAttributes(attribute.Int("user.id", user.ID))
	
	logrus.WithFields(logrus.Fields{
		"service":  serviceName,
		"endpoint": "/auth/login",
		"user_id":  user.ID,
		"email":    user.Email,
		"trace_id": span.SpanContext().TraceID().String(),
	}).Info("User logged in successfully")

	c.JSON(http.StatusOK, gin.H{
		"token":   tokenString,
		"user_id": user.ID,
		"email":   user.Email,
		"name":    user.Name,
	})
}

func registerHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "user_register")
	defer span.End()

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.SetAttributes(attribute.String("error", "invalid_request"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	span.SetAttributes(
		attribute.String("user.email", req.Email),
		attribute.String("endpoint", "/auth/register"),
	)

	// Check if user already exists
	for _, u := range users {
		if u.Email == req.Email {
			span.SetAttributes(attribute.String("error", "user_already_exists"))
			logrus.WithFields(logrus.Fields{
				"service":  serviceName,
				"endpoint": "/auth/register",
				"email":    req.Email,
				"error":    "user_already_exists",
				"trace_id": span.SpanContext().TraceID().String(),
			}).Warn("Attempt to register existing user")
			
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
			return
		}
	}

	// Create new user
	newUser := User{
		ID:       len(users) + 1,
		Email:    req.Email,
		Name:     req.Name,
		Password: req.Password,
	}
	users = append(users, newUser)

	span.SetAttributes(attribute.Int("user.id", newUser.ID))
	
	logrus.WithFields(logrus.Fields{
		"service":  serviceName,
		"endpoint": "/auth/register",
		"user_id":  newUser.ID,
		"email":    newUser.Email,
		"trace_id": span.SpanContext().TraceID().String(),
	}).Info("User registered successfully")

	c.JSON(http.StatusCreated, gin.H{
		"user_id": newUser.ID,
		"email":   newUser.Email,
		"name":    newUser.Name,
		"message": "User created successfully",
	})
}

func getUserHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "get_user")
	defer span.End()

	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_user_id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	span.SetAttributes(
		attribute.Int("user.id", userID),
		attribute.String("endpoint", "/users/:id"),
	)

	// Find user
	for _, user := range users {
		if user.ID == userID {
			logrus.WithFields(logrus.Fields{
				"service":  serviceName,
				"endpoint": "/users/:id",
				"user_id":  userID,
				"trace_id": span.SpanContext().TraceID().String(),
			}).Info("User retrieved successfully")
			
			user.Password = "" // Don't return password
			c.JSON(http.StatusOK, user)
			return
		}
	}

	span.SetAttributes(attribute.String("error", "user_not_found"))
	logrus.WithFields(logrus.Fields{
		"service":  serviceName,
		"endpoint": "/users/:id",
		"user_id":  userID,
		"error":    "user_not_found",
		"trace_id": span.SpanContext().TraceID().String(),
	}).Warn("User not found")

	c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
}

func getUserProfileHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "get_user_profile")
	defer span.End()

	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_user_id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	span.SetAttributes(
		attribute.Int("user.id", userID),
		attribute.String("endpoint", "/users/:id/profile"),
	)

	// Simulate slow database query
	time.Sleep(time.Duration(rand.Intn(200)+100) * time.Millisecond)

	// Find user
	for _, user := range users {
		if user.ID == userID {
			profile := gin.H{
				"id":           user.ID,
				"email":        user.Email,
				"name":         user.Name,
				"created_at":   "2024-01-01T00:00:00Z",
				"last_login":   time.Now().Add(-time.Duration(rand.Intn(24)) * time.Hour),
				"orders_count": rand.Intn(10),
				"total_spent":  rand.Float64() * 1000,
			}
			
			logrus.WithFields(logrus.Fields{
				"service":  serviceName,
				"endpoint": "/users/:id/profile",
				"user_id":  userID,
				"trace_id": span.SpanContext().TraceID().String(),
			}).Info("User profile retrieved successfully")

			c.JSON(http.StatusOK, profile)
			return
		}
	}

	span.SetAttributes(attribute.String("error", "user_not_found"))
	c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
}

func getUserFavoritesHandler(c *gin.Context) {
	ctx, span := tracer.Start(c.Request.Context(), "get_user_favorites")
	defer span.End()

	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_user_id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	span.SetAttributes(
		attribute.Int("user.id", userID),
		attribute.String("endpoint", "/users/:id/favorites"),
	)

	// Call product-service to get user's favorite products
	childCtx, childSpan := tracer.Start(ctx, "call_product_service")
	defer childSpan.End()

	childSpan.SetAttributes(
		attribute.String("http.method", "GET"),
		attribute.String("http.url", fmt.Sprintf("http://product-service:8082/products/favorites/%d", userID)),
	)

	req, _ := http.NewRequestWithContext(childCtx, "GET", fmt.Sprintf("http://product-service:8082/products/favorites/%d", userID), nil)
	
	// Inject trace context
	otel.GetTextMapPropagator().Inject(childCtx, propagation.HeaderCarrier(req.Header))
	
	resp, err := httpClient.Do(req)
	if err != nil {
		childSpan.SetAttributes(attribute.String("error", "service_call_failed"))
		logrus.WithFields(logrus.Fields{
			"service":        serviceName,
			"endpoint":       "/users/:id/favorites",
			"user_id":        userID,
			"error":          "product_service_call_failed",
			"target_service": "product-service",
			"trace_id":       span.SpanContext().TraceID().String(),
		}).Error("Failed to call product service")
		
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Product service unavailable"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		childSpan.SetAttributes(attribute.Int("http.status_code", resp.StatusCode))
		logrus.WithFields(logrus.Fields{
			"service":     serviceName,
			"user_id":     userID,
			"status_code": resp.StatusCode,
			"trace_id":    span.SpanContext().TraceID().String(),
		}).Warn("Product service returned non-200 status")
		
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Failed to get favorites"})
		return
	}

	var favorites interface{}
	if err := json.NewDecoder(resp.Body).Decode(&favorites); err != nil {
		childSpan.SetAttributes(attribute.String("error", "response_decode_failed"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode response"})
		return
	}

	logrus.WithFields(logrus.Fields{
		"service":        serviceName,
		"endpoint":       "/users/:id/favorites",
		"user_id":        userID,
		"target_service": "product-service",
		"trace_id":       span.SpanContext().TraceID().String(),
	}).Info("User favorites retrieved successfully")

	c.JSON(http.StatusOK, favorites)
}

func updateUserPreferencesHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "update_user_preferences")
	defer span.End()

	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		span.SetAttributes(attribute.String("error", "invalid_user_id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	span.SetAttributes(
		attribute.Int("user.id", userID),
		attribute.String("endpoint", "/users/:id/preferences"),
	)

	// Simulate slow preference update
	time.Sleep(time.Duration(rand.Intn(300)+100) * time.Millisecond)

	// Simulate update failures
	if rand.Intn(100) < 8 {
		span.SetAttributes(attribute.String("error", "preference_update_failed"))
		logrus.WithFields(logrus.Fields{
			"service":  serviceName,
			"endpoint": "/users/:id/preferences",
			"user_id":  userID,
			"error":    "database_constraint_violation",
			"trace_id": span.SpanContext().TraceID().String(),
		}).Error("Failed to update user preferences due to database constraint")
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update preferences"})
		return
	}

	logrus.WithFields(logrus.Fields{
		"service":  serviceName,
		"endpoint": "/users/:id/preferences",
		"user_id":  userID,
		"preferences_updated": rand.Intn(5) + 1,
		"trace_id": span.SpanContext().TraceID().String(),
	}).Info("User preferences updated successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "Preferences updated",
		"user_id": userID,
		"updated_fields": rand.Intn(5) + 1,
	})
}

func searchUsersHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "search_users")
	defer span.End()

	query := c.Query("q")
	limit := c.DefaultQuery("limit", "10")

	span.SetAttributes(
		attribute.String("search.query", query),
		attribute.String("search.limit", limit),
		attribute.String("endpoint", "/users/search"),
	)

	// Simulate search latency
	time.Sleep(time.Duration(rand.Intn(200)+50) * time.Millisecond)

	// Simulate search errors
	if rand.Intn(100) < 5 {
		span.SetAttributes(attribute.String("error", "search_service_timeout"))
		logrus.WithFields(logrus.Fields{
			"service":  serviceName,
			"endpoint": "/users/search",
			"query":    query,
			"error":    "elasticsearch_timeout",
			"trace_id": span.SpanContext().TraceID().String(),
		}).Error("User search timed out")
		
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Search service temporarily unavailable"})
		return
	}

	results := rand.Intn(25) + 1
	logrus.WithFields(logrus.Fields{
		"service":  serviceName,
		"endpoint": "/users/search",
		"query":    query,
		"results":  results,
		"trace_id": span.SpanContext().TraceID().String(),
	}).Info("User search completed")

	c.JSON(http.StatusOK, gin.H{
		"query": query,
		"results": results,
		"users": []gin.H{}, // Empty for demo
	})
}

func refreshTokenHandler(c *gin.Context) {
	_, span := tracer.Start(c.Request.Context(), "refresh_token")
	defer span.End()

	span.SetAttributes(attribute.String("endpoint", "/auth/refresh"))

	// Simulate token refresh failures
	if rand.Intn(100) < 12 {
		span.SetAttributes(attribute.String("error", "invalid_refresh_token"))
		logrus.WithFields(logrus.Fields{
			"service":  serviceName,
			"endpoint": "/auth/refresh",
			"error":    "invalid_refresh_token",
			"trace_id": span.SpanContext().TraceID().String(),
		}).Warn("Token refresh failed - invalid refresh token")
		
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	logrus.WithFields(logrus.Fields{
		"service":  serviceName,
		"endpoint": "/auth/refresh",
		"trace_id": span.SpanContext().TraceID().String(),
	}).Info("Token refreshed successfully")

	c.JSON(http.StatusOK, gin.H{
		"access_token": "new_jwt_token_here",
		"expires_in": 3600,
	})
}

func generateAutomaticLogs() {
	ticker := time.NewTicker(8 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			random := rand.Intn(100)
			
			if random < 15 {
				logrus.WithFields(logrus.Fields{
					"service":   serviceName,
					"component": "auth_service",
					"error":     "jwt_verification_failed",
					"tokens":    rand.Intn(15) + 1,
					"user_agent": []string{"Chrome", "Firefox", "Safari", "Edge"}[rand.Intn(4)],
				}).Error("JWT token verification failed for multiple requests")
			} else if random < 25 {
				logrus.WithFields(logrus.Fields{
					"service":   serviceName,
					"component": "auth_service",
					"warning":   "high_failed_login_attempts",
					"attempts":  rand.Intn(30) + 15,
					"window":    "1min",
					"source_ip": fmt.Sprintf("192.168.1.%d", rand.Intn(255)),
				}).Warn("High number of failed login attempts detected")
			} else if random < 35 {
				logrus.WithFields(logrus.Fields{
					"service":     serviceName,
					"component":   "user_registration",
					"event":       "new_user_registered",
					"user_count":  rand.Intn(500) + 1000,
					"email_domain": []string{"gmail.com", "yahoo.com", "outlook.com", "hotmail.com"}[rand.Intn(4)],
				}).Info("New user registration completed")
			} else if random < 45 {
				logrus.WithFields(logrus.Fields{
					"service":      serviceName,
					"component":    "password_service",
					"event":        "password_reset_request",
					"requests_per_hour": rand.Intn(25) + 5,
				}).Info("Password reset requests processed")
			} else if random < 55 {
				logrus.WithFields(logrus.Fields{
					"service":   serviceName,
					"component": "session_manager",
					"error":     "session_expired",
					"expired_sessions": rand.Intn(20) + 5,
					"cleanup_duration": fmt.Sprintf("%dms", rand.Intn(200)+50),
				}).Warn("Cleaned up expired user sessions")
			} else if random < 70 {
				logrus.WithFields(logrus.Fields{
					"service":     serviceName,
					"component":   "user_activity",
					"event":       "profile_update",
					"updates_per_min": rand.Intn(15) + 3,
					"fields_updated": []string{"name", "email", "preferences", "avatar"}[rand.Intn(4)],
				}).Info("User profile updates processed")
			} else {
				logrus.WithFields(logrus.Fields{
					"service":       serviceName,
					"component":     "auth_service",
					"status":        "operational",
					"active_users":  rand.Intn(200) + 100,
					"login_success": strconv.Itoa(rand.Intn(80)+30) + "/min",
					"concurrent_sessions": rand.Intn(150) + 75,
				}).Info("Authentication service running normally")
			}
		}
	}
}