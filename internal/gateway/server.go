/*
internal/gateway/server.go
Package gateway provides HTTP server setup and routing for the API Gateway.
*/

package gateway

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/AndreaBozzo/go-lab/internal/middleware"
	"github.com/AndreaBozzo/go-lab/internal/proxy"
	"github.com/AndreaBozzo/go-lab/internal/storage"
	"github.com/gin-gonic/gin"
)

// Server represents the API Gateway server
type Server struct {
	config      *Config
	router      *gin.Engine
	httpServer  *http.Server
	routeProxies []*proxy.RouteProxy
	storage     storage.LogStorage
}

// NewServer creates a new API Gateway server
func NewServer(config *Config, store storage.LogStorage) (*Server, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	// Create router
	router := gin.New()

	// Disable automatic redirects for trailing slashes
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	server := &Server{
		config:  config,
		router:  router,
		storage: store,
	}

	// Setup middleware and routes
	if err := server.setupMiddleware(); err != nil {
		return nil, err
	}
	if err := server.setupRoutes(); err != nil {
		return nil, err
	}

	return server, nil
}

// setupMiddleware configures all middleware in the correct order
func (s *Server) setupMiddleware() error {
	// 1. Recovery middleware (should be first to catch all panics)
	s.router.Use(middleware.RecoveryMiddleware())

	// 2. CORS middleware (if enabled)
	if s.config.CORS.Enabled {
		corsConfig := middleware.CORSConfig{
			AllowedOrigins: s.config.CORS.AllowedOrigins,
			AllowedMethods: s.config.CORS.AllowedMethods,
			AllowedHeaders: s.config.CORS.AllowedHeaders,
		}
		s.router.Use(middleware.CORSMiddleware(corsConfig))
	}

	// 3. Logging middleware
	s.router.Use(middleware.LoggingMiddleware(s.storage))

	// 4. Global rate limiting (if enabled)
	if s.config.RateLimiting.Enabled {
		limiter := middleware.NewRateLimiter(
			s.config.RateLimiting.RequestsPerSecond,
			s.config.RateLimiting.Burst,
		)
		s.router.Use(middleware.RateLimitMiddleware(limiter))
	}

	return nil
}

// setupRoutes configures all routes from the configuration
func (s *Server) setupRoutes() error {
	// Health check endpoint
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// Admin endpoint to view backend status
	s.router.GET("/admin/backends", func(c *gin.Context) {
		backends := make(map[string]interface{})
		for i, rp := range s.routeProxies {
			routeBackends := []map[string]interface{}{}
			for _, backend := range rp.GetPool().GetAllBackends() {
				routeBackends = append(routeBackends, map[string]interface{}{
					"url":     backend.GetURL().String(),
					"healthy": backend.IsHealthy(),
					"weight":  backend.Weight,
				})
			}
			backends[s.config.Routes[i].Path] = routeBackends
		}
		c.JSON(http.StatusOK, gin.H{
			"backends": backends,
		})
	})

	// Configure proxy routes
	for _, routeConfig := range s.config.Routes {
		// Extract backend URLs and weights
		var backendURLs []string
		var weights []int
		for _, backend := range routeConfig.Backends {
			backendURLs = append(backendURLs, backend.URL)
			weights = append(weights, backend.Weight)
		}

		// Create route proxy
		routeProxy, err := proxy.NewRouteProxy(
			backendURLs,
			weights,
			s.config.Server.WriteTimeout,
		)
		if err != nil {
			return fmt.Errorf("failed to create proxy for route %s: %w", routeConfig.Path, err)
		}

		// Start health checks for this route
		routeProxy.Start()
		s.routeProxies = append(s.routeProxies, routeProxy)

		// Register route handlers for each method
		if len(routeConfig.Methods) == 0 {
			// If no methods specified, allow all common methods
			routeConfig.Methods = []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
		}

		for _, method := range routeConfig.Methods {
			s.router.Handle(method, routeConfig.Path, routeProxy.Handler())
		}

		log.Printf("Registered route: %s -> %v", routeConfig.Path, backendURLs)
	}

	return nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
	}

	log.Printf("Starting API Gateway on %s", addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down API Gateway...")

	// Stop health checks for all route proxies
	for _, rp := range s.routeProxies {
		rp.Stop()
	}

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	log.Println("API Gateway stopped gracefully")
	return nil
}
