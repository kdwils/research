# Developer Guide - Kubernetes Service Health Monitor

## Table of Contents

1. [Development Setup](#development-setup)
2. [Project Structure](#project-structure)
3. [Architecture Overview](#architecture-overview)
4. [Building and Testing](#building-and-testing)
5. [Controller Development](#controller-development)
6. [Dashboard Development](#dashboard-development)
7. [Testing Strategy](#testing-strategy)
8. [Contributing](#contributing)

## Development Setup

### Prerequisites

- **Go**: 1.22 or higher
- **Docker**: For building container images
- **kubectl**: Configured with access to a Kubernetes cluster
- **kind** or **minikube**: For local testing (recommended)
- **Git**: For version control

### Clone the Repository

```bash
git clone https://github.com/kdwils/k8s-service-health-monitor.git
cd k8s-service-health-monitor
```

### Install Dependencies

```bash
go mod download
```

### Set Up Local Cluster

Using kind:
```bash
kind create cluster --name health-monitor-dev

# Load local images
kind load docker-image k8s-service-health-monitor:latest --name health-monitor-dev
```

Using minikube:
```bash
minikube start --cpus=4 --memory=8192
```

### Install CRDs

```bash
kubectl apply -f config/crd/
```

## Project Structure

```
.
├── api/v1alpha1/              # API definitions
│   ├── groupversion_info.go   # API group registration
│   └── healthcheck_types.go   # HealthCheck CRD definition
│
├── cmd/
│   ├── controller/            # Controller binary
│   │   └── main.go
│   └── dashboard/             # Dashboard binary
│       └── main.go
│
├── internal/
│   ├── controller/            # Controller logic
│   │   └── healthcheck_controller.go
│   ├── discovery/             # Service and pod discovery
│   │   └── discovery.go
│   ├── healthcheck/           # Health check executors
│   │   └── executor.go        # HTTP, TCP, gRPC, Exec implementations
│   ├── metrics/               # Uptime tracking
│   │   └── uptime.go
│   └── dashboard/             # Dashboard server
│       └── server.go          # REST API + WebSocket
│
├── config/
│   ├── crd/                   # CRD manifests
│   ├── rbac/                  # RBAC configuration
│   ├── manager/               # Deployment manifests
│   └── samples/               # Example HealthChecks
│
├── docs/                      # Documentation
│   ├── architecture.md
│   ├── ADR.md
│   ├── crd-design.md
│   ├── user-guide.md
│   └── developer-guide.md
│
├── Dockerfile                 # Multi-binary container image
├── Makefile                   # Build automation
└── go.mod                     # Go dependencies
```

## Architecture Overview

### Components

```
┌─────────────────────────────────────────────────────┐
│ Controller (Deployment with 2 replicas)             │
│                                                     │
│  ┌──────────────────────────────────────────────┐  │
│  │  HealthCheckReconciler                       │  │
│  │  - Watches HealthCheck CRs                   │  │
│  │  - Reconciles every interval                 │  │
│  │  - Updates status                            │  │
│  └────────────┬─────────────────────────────────┘  │
│               │                                     │
│  ┌────────────▼─────────┐  ┌────────────────────┐  │
│  │ ServiceDiscovery     │  │ Executor           │  │
│  │ - Watches Services   │  │ - HTTPExecutor     │  │
│  │ - EndpointSlices     │  │ - TCPExecutor      │  │
│  │ - Builds pod mapping │  │ - GRPCExecutor     │  │
│  └──────────────────────┘  │ - ExecExecutor     │  │
│                            └────────────────────┘  │
│  ┌──────────────────────────────────────────────┐  │
│  │ UptimeTracker                                │  │
│  │ - In-memory circular buffers                 │  │
│  │ - Calculates uptime percentages              │  │
│  └──────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────┐
│ Dashboard (Deployment with 3 replicas)              │
│                                                     │
│  ┌──────────────────────────────────────────────┐  │
│  │ Server                                       │  │
│  │ - Watches HealthCheck CRs via informer       │  │
│  │ - REST API (GET endpoints)                   │  │
│  │ - WebSocket Hub (broadcasts updates)         │  │
│  └──────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
```

### Data Flow

1. **User creates HealthCheck CR** → API server
2. **Controller watches CR** → Reconcile triggered
3. **Discovery service** → Finds service and pods
4. **Executor** → Runs checks against each pod
5. **Results aggregated** → Status updated
6. **UptimeTracker** → Records results, calculates percentages
7. **Status written** → HealthCheck `.status` field
8. **Dashboard watches** → Informer receives update
9. **WebSocket broadcasts** → UI updated in real-time

## Building and Testing

### Build Binaries

```bash
# Build both binaries
make build

# Build only controller
make build-controller

# Build only dashboard
make build-dashboard

# Binaries output to: bin/controller, bin/dashboard
```

### Run Locally

**Controller**:
```bash
# Run against current kubeconfig cluster
make run-controller

# Or directly:
go run cmd/controller/main.go --zap-log-level=debug
```

**Dashboard**:
```bash
make run-dashboard

# Or directly:
go run cmd/dashboard/main.go --bind-address=:8080
```

### Build Docker Image

```bash
make docker-build IMG=k8s-service-health-monitor:dev
```

The Dockerfile builds both binaries and includes them in a single image with different entrypoints.

### Run Tests

```bash
# Run all tests
make test

# Run with coverage
go test ./... -coverprofile=coverage.out

# View coverage
go tool cover -html=coverage.out
```

### Format and Lint

```bash
# Format code
make fmt

# Vet code
make vet

# Run golangci-lint (if installed)
golangci-lint run
```

## Controller Development

### HealthCheck Controller

**File**: `internal/controller/healthcheck_controller.go`

**Key Methods**:

```go
func (r *HealthCheckReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
```

**Reconciliation Flow**:

1. **Fetch HealthCheck**: Get CR from API server
2. **Handle Deletion**: If deleted, cleanup and remove finalizer
3. **Ensure Finalizer**: Add if not present
4. **Discover Pods**: Get pods backing target service
5. **Execute Checks**: Run all checks against all pods
6. **Record Metrics**: Update uptime tracker
7. **Update Status**: Write results to `.status`
8. **Requeue**: Return with `RequeueAfter` based on interval

**Adding a New Check Type**:

1. **Define Type** in `api/v1alpha1/healthcheck_types.go`:
```go
// In CheckType
const CheckTypeDNS CheckType = "DNS"

// In Check struct
DNS *DNSCheck `json:"dns,omitempty"`

// New struct
type DNSCheck struct {
    Hostname string `json:"hostname"`
    Expected string `json:"expected"`
}
```

2. **Implement Executor** in `internal/healthcheck/executor.go`:
```go
func (e *Executor) executeDNSCheck(ctx context.Context, check *monitoringv1alpha1.DNSCheck, pod discovery.PodInfo) CheckResult {
    // DNS lookup implementation
    ips, err := net.LookupIP(check.Hostname)
    // Validate against check.Expected
    // Return CheckResult
}
```

3. **Add to Switch** in `ExecuteCheck`:
```go
case monitoringv1alpha1.CheckTypeDNS:
    result = e.executeDNSCheck(checkCtx, check.DNS, pod)
```

### Service Discovery

**File**: `internal/discovery/discovery.go`

The discovery system maintains a cache of service→pod mappings:

```go
type ServiceDiscovery struct {
    client client.Client
    cache  *discoveryCache  // Thread-safe cache
}

func (sd *ServiceDiscovery) GetPodsForService(ctx context.Context, namespace, serviceName string) ([]PodInfo, error)
```

**How it Works**:

1. Queries EndpointSlices labeled with service name
2. Extracts pod references and IPs
3. Caches results for performance
4. Automatically invalidates on changes

**Adding Networking Resource Discovery** (future):

```go
// Watch Ingress resources
func (sd *ServiceDiscovery) WatchIngress(ctx context.Context) error {
    // Watch Ingress objects
    // Extract backend service references
    // Track Ingress → Service mappings
}
```

### Uptime Tracking

**File**: `internal/metrics/uptime.go`

**Architecture**:

- **Circular Buffers**: One per time window per HealthCheck
- **In-Memory**: Fast access, no external dependencies
- **Automatic Pruning**: Old data removed when outside window

**Key Methods**:

```go
// Record a check result
func (ut *UptimeTracker) RecordCheck(namespace, name string, windows []string, success bool)

// Get uptime statistics
func (ut *UptimeTracker) GetUptimeStats(namespace, name string, windows []string) []UptimeStat
```

**Adding Persistence** (future enhancement):

```go
// Periodically snapshot to ConfigMap or custom storage
func (ut *UptimeTracker) Snapshot(ctx context.Context) error {
    // Serialize buffer state
    // Write to persistent storage
}

func (ut *UptimeTracker) Restore(ctx context.Context) error {
    // Read from persistent storage
    // Rebuild in-memory buffers
}
```

## Dashboard Development

### Server

**File**: `internal/dashboard/server.go`

**Components**:

1. **REST API**: Read-only endpoints for HealthCheck data
2. **WebSocket Hub**: Manages client connections and broadcasts
3. **Informer**: Watches HealthCheck CRs for changes

**REST Handlers**:

```go
func (s *Server) handleListHealthChecks(w http.ResponseWriter, r *http.Request)
func (s *Server) handleGetHealthCheck(w http.ResponseWriter, r *http.Request)
func (s *Server) handleClusterSummary(w http.ResponseWriter, r *http.Request)
```

**WebSocket Flow**:

1. Client connects: `ws://dashboard/ws/healthchecks`
2. Server upgrades connection
3. Client registered in hub
4. Informer detects HealthCheck change
5. `broadcastUpdate()` called
6. Message sent to all connected clients

**Adding a New Endpoint**:

```go
// In Server.Start()
api.HandleFunc("/services/{namespace}/{name}/history", s.handleServiceHistory).Methods("GET")

// Handler
func (s *Server) handleServiceHistory(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    namespace := vars["namespace"]
    serviceName := vars["name"]

    // Query HealthChecks for this service
    // Extract historical data
    // Return JSON
}
```

### Frontend Development

The current implementation includes a minimal HTML placeholder. For production:

**Recommended Stack**:
- **Framework**: React with TypeScript
- **State Management**: React Query (for API calls)
- **WebSocket**: Native WebSocket API or use-websocket hook
- **UI Library**: Tailwind CSS + shadcn/ui
- **Charts**: Recharts or Chart.js

**Building Production Frontend**:

```bash
# In frontend/
npm run build

# Embed in Go binary
//go:embed frontend/dist
var frontendFS embed.FS

// Serve embedded files
router.PathPrefix("/").Handler(http.FileServer(http.FS(frontendFS)))
```

## Testing Strategy

### Unit Tests

**Testing Check Executors**:

```go
// internal/healthcheck/executor_test.go
func TestHTTPExecutor(t *testing.T) {
    // Start test HTTP server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status":"healthy"}`))
    }))
    defer server.Close()

    // Create executor and check
    executor := NewExecutor(nil, nil)
    check := &monitoringv1alpha1.HTTPCheck{
        Port: server.Port(),
        Path: "/health",
        ExpectedBodyContains: "healthy",
    }

    // Execute check
    result := executor.executeHTTPCheck(context.Background(), check, podInfo)

    // Assert
    assert.True(t, result.Success)
}
```

**Testing Uptime Tracking**:

```go
func TestUptimeTracker(t *testing.T) {
    tracker := metrics.NewUptimeTracker()

    // Record successes
    for i := 0; i < 100; i++ {
        tracker.RecordCheck("default", "test", []string{"1h"}, true)
    }

    // Record failures
    for i := 0; i < 10; i++ {
        tracker.RecordCheck("default", "test", []string{"1h"}, false)
    }

    // Get stats
    stats := tracker.GetUptimeStats("default", "test", []string{"1h"})

    // Assert
    assert.Equal(t, 90.9, stats[0].Percentage, 0.1)
}
```

### Integration Tests

Using **envtest** (controller-runtime test environment):

```go
func TestHealthCheckController(t *testing.T) {
    // Setup envtest environment
    testEnv := &envtest.Environment{
        CRDDirectoryPaths: []string{"config/crd"},
    }
    cfg, err := testEnv.Start()
    require.NoError(t, err)
    defer testEnv.Stop()

    // Create client
    k8sClient, err := client.New(cfg, client.Options{Scheme: scheme})
    require.NoError(t, err)

    // Create test service
    svc := &corev1.Service{...}
    err = k8sClient.Create(ctx, svc)
    require.NoError(t, err)

    // Create HealthCheck
    hc := &monitoringv1alpha1.HealthCheck{...}
    err = k8sClient.Create(ctx, hc)
    require.NoError(t, err)

    // Wait for status update
    eventually(func() bool {
        k8sClient.Get(ctx, client.ObjectKeyFromObject(hc), hc)
        return hc.Status.Phase != ""
    }, timeout, interval)

    // Assert status
    assert.Equal(t, monitoringv1alpha1.HealthPhaseHealthy, hc.Status.Phase)
}
```

### End-to-End Tests

```bash
#!/bin/bash
# test/e2e/test.sh

# Create test cluster
kind create cluster --name e2e-test

# Deploy controller and dashboard
kubectl apply -f config/crd/
kubectl apply -f config/rbac/
kubectl apply -f config/manager/

# Wait for deployments
kubectl wait --for=condition=available deployment/health-monitor-controller -n health-monitor-system --timeout=60s

# Create test service
kubectl apply -f test/fixtures/test-service.yaml

# Create HealthCheck
kubectl apply -f test/fixtures/test-healthcheck.yaml

# Wait for status
kubectl wait --for=condition=ready healthcheck/test-healthcheck --timeout=60s

# Verify status
kubectl get healthcheck test-healthcheck -o jsonpath='{.status.phase}' | grep Healthy

# Cleanup
kind delete cluster --name e2e-test
```

## Contributing

### Code Style

- **Go**: Follow [Effective Go](https://golang.org/doc/effective_go.html)
- **Formatting**: Use `gofmt` and `goimports`
- **Linting**: Pass `golangci-lint`
- **Comments**: Document exported functions and types

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add DNS health check type
fix: prevent race condition in uptime tracker
docs: update API documentation
refactor: simplify executor interface
test: add integration tests for controller
```

### Pull Request Process

1. **Fork** the repository
2. **Create branch** from `main`:
   ```bash
   git checkout -b feat/my-feature
   ```
3. **Make changes** with tests
4. **Run tests**: `make test`
5. **Format code**: `make fmt`
6. **Commit** with descriptive message
7. **Push** to fork
8. **Create PR** with description

**PR Checklist**:
- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] Code formatted (`make fmt`)
- [ ] Tests passing (`make test`)
- [ ] Linked to issue (if applicable)

### Release Process

1. **Update version** in relevant files
2. **Create tag**:
   ```bash
   git tag -a v0.1.0 -m "Release v0.1.0"
   git push origin v0.1.0
   ```
3. **GitHub Actions** builds and publishes image
4. **Create GitHub Release** with changelog

## Advanced Topics

### Leader Election

The controller uses leader election for high availability:

```go
// cmd/controller/main.go
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
    LeaderElection:   true,
    LeaderElectionID: "health-monitor-controller-lock",
})
```

**How it Works**:
- Multiple controller replicas deployed
- One acquires lease and becomes leader
- Leader performs reconciliation
- On failure, another replica takes over

### Metrics and Monitoring

**Prometheus Metrics** (already exposed by controller-runtime):

```
# Reconciliation metrics
controller_runtime_reconcile_total{controller="healthcheck"}
controller_runtime_reconcile_errors_total{controller="healthcheck"}
controller_runtime_reconcile_duration_seconds{controller="healthcheck"}

# Queue metrics
workqueue_depth{name="healthcheck"}
workqueue_adds_total{name="healthcheck"}
```

**Adding Custom Metrics**:

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
    checksTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "health_monitor_checks_total",
            Help: "Total number of health checks executed",
        },
        []string{"namespace", "name", "check", "result"},
    )
)

func init() {
    metrics.Registry.MustRegister(checksTotal)
}

// In executor
checksTotal.WithLabelValues(namespace, name, checkName, "success").Inc()
```

### Debugging

**Enable Debug Logging**:
```bash
go run cmd/controller/main.go --zap-log-level=debug
```

**Remote Debugging**:
```go
// Add to main.go
import _ "net/http/pprof"

go func() {
    http.ListenAndServe("localhost:6060", nil)
}()
```

Access: http://localhost:6060/debug/pprof/

**Tracing** (future):
```go
import "go.opentelemetry.io/otel"

// Instrument reconciliation
ctx, span := tracer.Start(ctx, "reconcile")
defer span.End()
```

### Performance Optimization

**Reduce API Calls**:
```go
// Use informer cache instead of direct Get
hc := &monitoringv1alpha1.HealthCheck{}
err := r.Get(ctx, req.NamespacedName, hc)

// Cache is automatically used by controller-runtime
```

**Concurrent Check Execution**:
```go
// Use goroutines with semaphore
sem := make(chan struct{}, 10) // Max 10 concurrent
var wg sync.WaitGroup

for _, pod := range pods {
    for _, check := range checks {
        wg.Add(1)
        go func(p PodInfo, c Check) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()

            result := executor.ExecuteCheck(ctx, c, p, timeout)
            // Handle result
        }(pod, check)
    }
}
wg.Wait()
```

## Resources

- **Kubebuilder Book**: https://book.kubebuilder.io/
- **controller-runtime**: https://github.com/kubernetes-sigs/controller-runtime
- **Kubernetes API Conventions**: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md
- **Sample Controller**: https://github.com/kubernetes/sample-controller

## Questions?

- **GitHub Discussions**: Ask questions and share ideas
- **Issues**: Report bugs or request features
- **Slack**: #health-monitor on Kubernetes Slack
