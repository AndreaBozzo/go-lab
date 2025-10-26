/*
internal/proxy/backend.go
Package proxy provides backend server management with health checking.
*/

package proxy

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Backend represents a backend server
type Backend struct {
	URL         *url.URL
	Weight      int
	Healthy     bool
	FailCount   int
	mu          sync.RWMutex
	healthURL   string
	lastCheck   time.Time
	client      *http.Client
}

// BackendPool manages a pool of backend servers
type BackendPool struct {
	backends        []*Backend
	mu              sync.RWMutex
	healthCheckPath string
	healthInterval  time.Duration
	maxFails        int
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewBackend creates a new backend instance
func NewBackend(urlStr string, weight int) (*Backend, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid backend URL %s: %w", urlStr, err)
	}

	return &Backend{
		URL:       parsedURL,
		Weight:    weight,
		Healthy:   true, // Start as healthy
		FailCount: 0,
		healthURL: urlStr + "/health",
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}, nil
}

// NewBackendPool creates a new backend pool
func NewBackendPool(backends []*Backend, healthCheckInterval time.Duration) *BackendPool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &BackendPool{
		backends:        backends,
		healthCheckPath: "/health",
		healthInterval:  healthCheckInterval,
		maxFails:        3, // Mark unhealthy after 3 consecutive failures
		ctx:             ctx,
		cancel:          cancel,
	}

	return pool
}

// Start begins health checking for all backends
func (bp *BackendPool) Start() {
	// Initial health check
	bp.checkAllBackends()

	// Periodic health checks
	ticker := time.NewTicker(bp.healthInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-bp.ctx.Done():
				return
			case <-ticker.C:
				bp.checkAllBackends()
			}
		}
	}()
}

// Stop stops the health checking
func (bp *BackendPool) Stop() {
	bp.cancel()
}

// checkAllBackends performs health checks on all backends
func (bp *BackendPool) checkAllBackends() {
	var wg sync.WaitGroup
	for _, backend := range bp.backends {
		wg.Add(1)
		go func(b *Backend) {
			defer wg.Done()
			bp.checkBackend(b)
		}(backend)
	}
	wg.Wait()
}

// checkBackend performs a health check on a single backend
func (bp *BackendPool) checkBackend(backend *Backend) {
	backend.mu.Lock()
	defer backend.mu.Unlock()

	ctx, cancel := context.WithTimeout(bp.ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", backend.healthURL, nil)
	if err != nil {
		backend.markUnhealthy()
		return
	}

	resp, err := backend.client.Do(req)
	if err != nil {
		backend.markUnhealthy()
		log.Printf("Health check failed for %s: %v", backend.URL.String(), err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		backend.markHealthy()
		if backend.FailCount > 0 {
			log.Printf("Backend %s recovered", backend.URL.String())
		}
	} else {
		backend.markUnhealthy()
		log.Printf("Health check failed for %s: status %d", backend.URL.String(), resp.StatusCode)
	}

	backend.lastCheck = time.Now()
}

// markHealthy marks the backend as healthy (must be called with lock held)
func (b *Backend) markHealthy() {
	b.Healthy = true
	b.FailCount = 0
}

// markUnhealthy increments fail count and marks unhealthy if threshold reached (must be called with lock held)
func (b *Backend) markUnhealthy() {
	b.FailCount++
	if b.FailCount >= 3 {
		if b.Healthy {
			log.Printf("Backend %s marked unhealthy after %d failures", b.URL.String(), b.FailCount)
		}
		b.Healthy = false
	}
}

// IsHealthy returns whether the backend is healthy (thread-safe)
func (b *Backend) IsHealthy() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Healthy
}

// GetURL returns the backend URL (thread-safe)
func (b *Backend) GetURL() *url.URL {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.URL
}

// GetHealthyBackends returns all healthy backends from the pool
func (bp *BackendPool) GetHealthyBackends() []*Backend {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	var healthy []*Backend
	for _, backend := range bp.backends {
		if backend.IsHealthy() {
			healthy = append(healthy, backend)
		}
	}
	return healthy
}

// GetAllBackends returns all backends (for admin/debug purposes)
func (bp *BackendPool) GetAllBackends() []*Backend {
	bp.mu.RLock()
	defer bp.mu.RUnlock()
	return bp.backends
}
