## go-lab - API Gateway

My personal Go playground as a Sunday project. An API Gateway built with Go to learn the language and compare with Rust and Python implementations.

## What Works

- **Reverse Proxy**: Forwards HTTP requests to backend services
- **Load Balancing**: Weighted round-robin (tested, works)
- **Health Checks**: Every 10s, marks unhealthy after 3 failures
- **Rate Limiting**: Token bucket algorithm (implemented, not stress tested)
- **CORS**: Configurable headers
- **Request Logging**: All requests saved to SQLite with metrics
- **Graceful Shutdown**: Waits for in-flight requests
- **Panic Recovery**: Catches panics, logs stack trace
- **Mock Servers**: 3 backends for testing

## Architecture

```
go-lab/
├── cmd/
│   ├── apigateway/          # Gateway entry point
│   └── mockserver/          # Mock backends for testing
├── internal/
│   ├── gateway/             # YAML config, server setup, routing
│   ├── proxy/               # Reverse proxy, load balancer, backend pool
│   ├── middleware/          # Logging, CORS, rate limit, recovery
│   ├── collector/           # Log entry structs
│   └── storage/             # SQLite storage
├── pkg/
│   └── logutil/             # Logger utils
└── config/
    ├── gateway.yaml         # Active config
    └── gateway.example.yaml # Example
```

**Stack:**
- HTTP: Gin
- Database: SQLite (modernc.org/sqlite - pure Go, no CGO)
- Rate Limiting: golang.org/x/time/rate
- Config: YAML

## Quick Start

### 1. Install dependencies

```bash
go mod download
```

### 2. Edit config

Edit [config/gateway.yaml](config/gateway.yaml):

```yaml
routes:
  - path: "/api/users/*filepath"    # NOTE: Gin needs *filepath not just *
    backends:
      - url: "http://localhost:9001"
        weight: 1
      - url: "http://localhost:9002"
        weight: 1
    methods: ["GET", "POST", "PUT", "DELETE"]
```

### 3. Start mock backends

**3 terminals:**
```bash
go run cmd/mockserver/main.go -port 9001 -name "UserService-1"
go run cmd/mockserver/main.go -port 9002 -name "UserService-2"
go run cmd/mockserver/main.go -port 9003 -name "OrderService"
```

### 4. Start gateway

```bash
go run cmd/apigateway/main.go
```

Gateway runs on `http://localhost:8080`

### 5. Test it

```bash
# Health check
curl http://localhost:8080/health

# Backend status
curl http://localhost:8080/admin/backends

# Get user (load balanced between 9001 and 9002)
curl http://localhost:8080/api/users/1

# Get order (routes to 9003)
curl http://localhost:8080/api/orders/1

# Test load balancing - watch "server" field change
curl http://localhost:8080/api/users/1
curl http://localhost:8080/api/users/1
curl http://localhost:8080/api/users/1
# Alternates between UserService-1 and UserService-2
```

## Configuration

### Server

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  shutdown_timeout: 10s
```

### Rate Limiting

```yaml
rate_limiting:
  enabled: true
  requests_per_second: 100
  burst: 50
```

**Note:** Bash loops aren't fast enough to hit rate limits. Need proper load tester (ab, wrk, hey).

### CORS

```yaml
cors:
  enabled: true
  allowed_origins: ["*"]
  allowed_methods: ["GET", "POST", "PUT", "DELETE"]
```

### Logging

All requests logged to SQLite:
- Method, path, status code
- Latency (milliseconds)
- Client IP, User-Agent
- Backend URL that handled it

Also printed to stdout:
```
2025/10/26 15:01:26 [INFO] GET /api/users/1 - 200 (35ms) - Backend: http://localhost:9001
```

## Load Balancing

Weighted round-robin, tested and working:

```yaml
backends:
  - url: "http://localhost:9001"
    weight: 2   # Gets 2x traffic
  - url: "http://localhost:9002"
    weight: 1   # Gets 1x traffic
```

## Health Checks

- Every 10 seconds → `GET /health` on each backend
- Marked unhealthy after 3 consecutive failures
- Unhealthy backends excluded from load balancing
- Auto-recovery when backend comes back

Tested: Kill a mock server → gateway detects and routes around it.

## Gotchas

### SQLite Driver

Uses `modernc.org/sqlite` (pure Go) instead of `mattn/go-sqlite3` (needs CGO).
- Pro: Works without C compiler, easier cross-compile
- Con: Slower compilation (~20-30s), slightly slower queries

### Gin Routing

**Important:** Gin needs named wildcards. Use `/api/users/*filepath` not `/api/users/*`.

Wildcard routes only match paths with something after the prefix:
- `/api/users/1` ✅
- `/api/users/` ✅
- `/api/users` ❌ 404

## Comparison: Go vs Rust vs Python

| Aspect | Go (this) | Rust (Axum) | Python (FastAPI) |
|--------|-----------|-------------|------------------|
| Concurrency | Goroutines | async/await | async/await |
| Type Safety | Static | Strong + ownership | Dynamic + hints |
| Performance | High | Highest | Medium |
| Compile Time | Fast (~5s) | Slow (~30s) | N/A |
| Error Handling | if err != nil | Result<T,E> | try/except |
| Middleware | Function chain | Tower layers | Decorators |
| Learning Curve | Low-Medium | High | Low |

**Go good for:**
- Learning curve vs performance balance
- Quick iteration
- Simple deployment (single binary)

**Rust good for:**
- Maximum performance needed
- Memory safety critical

**Python good for:**
- Prototyping
- Already Python team

## What's Missing

- [ ] JWT/Auth middleware
- [ ] Circuit breaker
- [ ] Prometheus metrics
- [ ] Distributed tracing
- [ ] Request/response transformation
- [ ] WebSocket proxying
- [ ] Service discovery

## Things I Learned

1. Gin wildcards need explicit names (`*filepath`)
2. modernc.org/sqlite avoids CGO pain on Windows
3. Health checks better in background goroutines
4. Context cancellation essential for clean shutdown
5. Mutex needed for shared state in concurrent code

## License

MIT
