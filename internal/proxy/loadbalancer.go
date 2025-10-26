/*
internal/proxy/loadbalancer.go
Package proxy provides load balancing algorithms for backend selection.
*/

package proxy

import (
	"fmt"
	"sync"
)

// LoadBalancer defines the interface for load balancing algorithms
type LoadBalancer interface {
	NextBackend() (*Backend, error)
}

// RoundRobinBalancer implements round-robin load balancing with weight support
type RoundRobinBalancer struct {
	pool    *BackendPool
	current int
	mu      sync.Mutex
}

// NewRoundRobinBalancer creates a new round-robin load balancer
func NewRoundRobinBalancer(pool *BackendPool) *RoundRobinBalancer {
	return &RoundRobinBalancer{
		pool:    pool,
		current: 0,
	}
}

// NextBackend returns the next healthy backend using round-robin algorithm
func (rr *RoundRobinBalancer) NextBackend() (*Backend, error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	healthyBackends := rr.pool.GetHealthyBackends()
	if len(healthyBackends) == 0 {
		return nil, fmt.Errorf("no healthy backends available")
	}

	// Expand backends based on weight for weighted round-robin
	// For example: backend with weight 2 appears twice in the list
	var weightedBackends []*Backend
	for _, backend := range healthyBackends {
		weight := backend.Weight
		if weight <= 0 {
			weight = 1
		}
		for i := 0; i < weight; i++ {
			weightedBackends = append(weightedBackends, backend)
		}
	}

	// Select next backend in round-robin fashion
	backend := weightedBackends[rr.current%len(weightedBackends)]
	rr.current++

	// Prevent overflow by resetting counter
	if rr.current >= len(weightedBackends)*1000 {
		rr.current = 0
	}

	return backend, nil
}

// Reset resets the round-robin counter
func (rr *RoundRobinBalancer) Reset() {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.current = 0
}
