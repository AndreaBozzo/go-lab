/*
internal/proxy/proxy.go
Package proxy provides reverse proxy functionality for the API Gateway.
*/

package proxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ProxyHandler handles reverse proxy requests
type ProxyHandler struct {
	balancer LoadBalancer
	timeout  time.Duration
	client   *http.Client
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(balancer LoadBalancer, timeout time.Duration) *ProxyHandler {
	// Custom HTTP client with connection pooling
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
		// Don't follow redirects automatically
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &ProxyHandler{
		balancer: balancer,
		timeout:  timeout,
		client:   client,
	}
}

// Handle proxies the request to a backend server
func (ph *ProxyHandler) Handle(c *gin.Context) {
	// Select backend using load balancer
	backend, err := ph.balancer.NextBackend()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "No backend servers available",
		})
		return
	}

	// Store backend info in context for logging middleware
	c.Set("backend", backend.GetURL().String())

	// Build target URL
	targetURL := ph.buildTargetURL(backend.GetURL(), c.Request.URL)

	// Create proxy request
	proxyReq, err := ph.createProxyRequest(c.Request, targetURL)
	if err != nil {
		log.Printf("Failed to create proxy request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create proxy request",
		})
		return
	}

	// Add forwarding headers
	ph.setForwardingHeaders(proxyReq, c.Request)

	// Execute request with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), ph.timeout)
	defer cancel()
	proxyReq = proxyReq.WithContext(ctx)

	// Perform the request
	resp, err := ph.client.Do(proxyReq)
	if err != nil {
		log.Printf("Proxy request failed for backend %s: %v", backend.GetURL().String(), err)
		c.JSON(http.StatusBadGateway, gin.H{
			"error": "Backend request failed",
		})
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	// Set status code
	c.Status(resp.StatusCode)

	// Stream response body
	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		log.Printf("Failed to copy response body: %v", err)
	}
}

// buildTargetURL constructs the target backend URL
func (ph *ProxyHandler) buildTargetURL(backendURL *url.URL, requestURL *url.URL) string {
	target := *backendURL
	target.Path = requestURL.Path
	target.RawQuery = requestURL.RawQuery
	return target.String()
}

// createProxyRequest creates a new HTTP request for the backend
func (ph *ProxyHandler) createProxyRequest(original *http.Request, targetURL string) (*http.Request, error) {
	// Create new request with same method and body
	req, err := http.NewRequest(original.Method, targetURL, original.Body)
	if err != nil {
		return nil, err
	}

	// Copy headers
	for key, values := range original.Header {
		// Skip hop-by-hop headers
		if isHopByHopHeader(key) {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	return req, nil
}

// setForwardingHeaders sets X-Forwarded-* headers
func (ph *ProxyHandler) setForwardingHeaders(proxyReq *http.Request, originalReq *http.Request) {
	// X-Forwarded-For
	clientIP := getClientIP(originalReq)
	if prior, ok := proxyReq.Header["X-Forwarded-For"]; ok {
		clientIP = strings.Join(prior, ", ") + ", " + clientIP
	}
	proxyReq.Header.Set("X-Forwarded-For", clientIP)

	// X-Real-IP
	proxyReq.Header.Set("X-Real-IP", getClientIP(originalReq))

	// X-Forwarded-Proto
	proto := "http"
	if originalReq.TLS != nil {
		proto = "https"
	}
	proxyReq.Header.Set("X-Forwarded-Proto", proto)

	// X-Forwarded-Host
	proxyReq.Header.Set("X-Forwarded-Host", originalReq.Host)
}

// getClientIP extracts the client IP from the request
func getClientIP(req *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP header
	if xri := req.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return host
}

// isHopByHopHeader checks if a header is hop-by-hop
// These headers are meaningful only for a single transport-level connection
func isHopByHopHeader(header string) bool {
	hopByHopHeaders := map[string]bool{
		"Connection":          true,
		"Keep-Alive":          true,
		"Proxy-Authenticate":  true,
		"Proxy-Authorization": true,
		"Te":                  true,
		"Trailers":            true,
		"Transfer-Encoding":   true,
		"Upgrade":             true,
	}
	return hopByHopHeaders[header]
}

// RouteProxy represents a proxy handler for a specific route
type RouteProxy struct {
	pool    *BackendPool
	handler *ProxyHandler
}

// NewRouteProxy creates a new route proxy with its own backend pool
func NewRouteProxy(backendURLs []string, weights []int, timeout time.Duration) (*RouteProxy, error) {
	if len(backendURLs) == 0 {
		return nil, fmt.Errorf("at least one backend URL is required")
	}

	// Create backends
	var backends []*Backend
	for i, urlStr := range backendURLs {
		weight := 1
		if i < len(weights) {
			weight = weights[i]
		}

		backend, err := NewBackend(urlStr, weight)
		if err != nil {
			return nil, err
		}
		backends = append(backends, backend)
	}

	// Create backend pool
	pool := NewBackendPool(backends, 10*time.Second)

	// Create load balancer
	balancer := NewRoundRobinBalancer(pool)

	// Create proxy handler
	handler := NewProxyHandler(balancer, timeout)

	return &RouteProxy{
		pool:    pool,
		handler: handler,
	}, nil
}

// Start starts health checking for this route's backends
func (rp *RouteProxy) Start() {
	rp.pool.Start()
}

// Stop stops health checking
func (rp *RouteProxy) Stop() {
	rp.pool.Stop()
}

// Handler returns the Gin handler function
func (rp *RouteProxy) Handler() gin.HandlerFunc {
	return rp.handler.Handle
}

// GetPool returns the backend pool for this route (for admin/testing)
func (rp *RouteProxy) GetPool() *BackendPool {
	return rp.pool
}
