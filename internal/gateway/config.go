/*
internal/gateway/config.go
Package gateway provides configuration structures and loading for the API Gateway.
*/

package gateway

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the complete gateway configuration
type Config struct {
	Server       ServerConfig       `yaml:"server"`
	Logging      LoggingConfig      `yaml:"logging"`
	RateLimiting RateLimitingConfig `yaml:"rate_limiting"`
	CORS         CORSConfig         `yaml:"cors"`
	Routes       []RouteConfig      `yaml:"routes"`
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

// LoggingConfig contains logging settings
type LoggingConfig struct {
	Database string `yaml:"database"`
	Level    string `yaml:"level"`
}

// RateLimitingConfig contains rate limiting settings
type RateLimitingConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerSecond int  `yaml:"requests_per_second"`
	Burst             int  `yaml:"burst"`
}

// CORSConfig contains CORS settings
type CORSConfig struct {
	Enabled        bool     `yaml:"enabled"`
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers"`
}

// RouteConfig represents a single route configuration
type RouteConfig struct {
	Path      string          `yaml:"path"`
	Backends  []BackendConfig `yaml:"backends"`
	Methods   []string        `yaml:"methods"`
	RateLimit int             `yaml:"rate_limit"` // Per-route rate limit (requests per second)
}

// BackendConfig represents a backend server configuration
type BackendConfig struct {
	URL    string `yaml:"url"`
	Weight int    `yaml:"weight"` // For weighted load balancing
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Server.ReadTimeout == 0 {
		config.Server.ReadTimeout = 30 * time.Second
	}
	if config.Server.WriteTimeout == 0 {
		config.Server.WriteTimeout = 30 * time.Second
	}
	if config.Server.ShutdownTimeout == 0 {
		config.Server.ShutdownTimeout = 10 * time.Second
	}
	if config.Logging.Database == "" {
		config.Logging.Database = "gateway.db"
	}
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}

	return &config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if len(c.Routes) == 0 {
		return fmt.Errorf("no routes configured")
	}

	for i, route := range c.Routes {
		if route.Path == "" {
			return fmt.Errorf("route %d: path is required", i)
		}
		if len(route.Backends) == 0 {
			return fmt.Errorf("route %d: at least one backend is required", i)
		}
		for j, backend := range route.Backends {
			if backend.URL == "" {
				return fmt.Errorf("route %d, backend %d: URL is required", i, j)
			}
			if backend.Weight <= 0 {
				c.Routes[i].Backends[j].Weight = 1 // Default weight
			}
		}
	}

	return nil
}
