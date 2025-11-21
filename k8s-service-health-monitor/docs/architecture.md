# Service Health Monitor - System Architecture

## System Overview

The Service Health Monitor consists of three main components deployed in a Kubernetes cluster:

1. **Controller**: Discovers services/pods, executes healthchecks, manages state
2. **Dashboard Backend**: REST/WebSocket API for health data access
3. **Dashboard Frontend**: Web UI for visualization and monitoring

```
┌─────────────────────────────────────────────────────────────────┐
│                      Kubernetes Cluster                         │
│                                                                 │
│  ┌────────────────────────────────────────────────────────┐   │
│  │                    Controller                           │   │
│  │                                                         │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌─────────────┐  │   │
│  │  │  Discovery   │  │ HealthCheck  │  │   Metrics   │  │   │
│  │  │  Controller  │  │  Controller  │  │  Aggregator │  │   │
│  │  └──────────────┘  └──────────────┘  └─────────────┘  │   │
│  │         │                  │                  │         │   │
│  │         └──────────────────┴──────────────────┘         │   │
│  │                            │                            │   │
│  │                            ▼                            │   │
│  │                  ┌──────────────────┐                  │   │
│  │                  │ In-Memory Store  │                  │   │
│  │                  │  (shared cache)  │                  │   │
│  │                  └──────────────────┘                  │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌────────────────────────────────────────────────────────┐   │
│  │              Dashboard Backend                          │   │
│  │                                                         │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────────────┐    │   │
│  │  │   REST   │  │ WebSocket│  │  K8s API Client  │    │   │
│  │  │    API   │  │   Hub    │  │  (read status)   │    │   │
│  │  └──────────┘  └──────────┘  └──────────────────┘    │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                            │
                            │ HTTPS
                            ▼
                 ┌────────────────────┐
                 │  Dashboard Frontend │
                 │   (Browser / UI)    │
                 └────────────────────┘
```

## Component Details

### 1. Controller

**Purpose**: Core monitoring engine that discovers services, executes health checks, and maintains state.

#### Sub-Components

##### Discovery Controller
- **Watches**: Services, Endpoints/EndpointSlices, Pods, Ingress, Gateway, HTTPRoute
- **Reconciles**: Service discovery and pod mapping
- **Outputs**: Maintains internal service→pod mapping cache
- **Logic**:
  ```
  1. Service created/updated → extract label selectors
  2. Query EndpointSlices for matching pods
  3. Watch pod readiness and lifecycle events
  4. Update internal cache with service topology
  5. Trigger healthcheck controller for services with HealthChecks
  ```

##### HealthCheck Controller
- **Watches**: HealthCheck CRDs
- **Reconciles**: Healthcheck execution and status updates
- **Inputs**: Service→pod mappings from Discovery Controller
- **Outputs**: Updates HealthCheck `.status` field
- **Logic**:
  ```
  1. HealthCheck created/updated → parse check definitions
  2. Get target service pods from Discovery Controller
  3. Schedule checks based on interval
  4. Execute checks against each pod endpoint
  5. Aggregate results (per-check, per-pod, overall)
  6. Calculate uptime metrics
  7. Update HealthCheck status
  8. Emit Kubernetes events on state changes
  ```

##### Metrics Aggregator
- **Purpose**: Calculate and maintain uptime statistics
- **Inputs**: Check execution results over time
- **Outputs**: Uptime percentages for configured windows
- **Storage**: In-memory circular buffers per HealthCheck
- **Logic**:
  ```
  1. Maintain rolling window of check results
  2. For each configured window (1h, 24h, 7d, 30d):
     - Count total checks and successful checks
     - Calculate percentage
     - Update HealthCheck status
  3. Prune old data beyond longest window
  ```

#### Healthcheck Execution Engine

**Design**: Concurrent execution with rate limiting and timeouts

```go
type Executor interface {
    Execute(ctx context.Context, check Check, target Target) (*Result, error)
}

// Implementations
type HTTPExecutor struct{}
type TCPExecutor struct{}
type GRPCExecutor struct{}
type ExecExecutor struct{}
```

**Key Features**:
- **Concurrency**: Execute checks in parallel across pods
- **Rate Limiting**: Prevent overwhelming targets
- **Timeout Handling**: Respect per-check timeout settings
- **Context Cancellation**: Graceful shutdown
- **Retry Logic**: Exponential backoff for transient failures
- **Security**: Exec checks run in pod context, not controller

**Execution Flow**:
```
For each HealthCheck CR:
  For each pod backing the target service:
    For each check definition:
      Execute check concurrently (with semaphore limit)
      Record result with timestamp
      Update consecutive success/failure counters
      If threshold reached → update health status
```

### 2. Dashboard Backend

**Purpose**: Provide API access to health data for UI and external consumers.

#### Deployment
- **Binary**: `cmd/dashboard/main.go`
- **Image**: Same image as controller, different entrypoint
- **Deployment**: Separate Kubernetes Deployment
- **Service**: ClusterIP or LoadBalancer for access
- **RBAC**: Read-only access to HealthCheck CRs

#### API Endpoints

**REST API**:
```
GET /api/v1/healthchecks
  → List all HealthChecks across all namespaces
  Query params: namespace, labelSelector

GET /api/v1/healthchecks/{namespace}/{name}
  → Get specific HealthCheck with full status

GET /api/v1/services/{namespace}/{name}/health
  → Get health status for a specific service (auto-discover HealthChecks)

GET /api/v1/namespaces/{namespace}/health-summary
  → Summary stats for namespace

GET /api/v1/health-summary
  → Cluster-wide health summary
```

**WebSocket API**:
```
WS /ws/healthchecks
  → Real-time updates when HealthCheck status changes

Messages:
  {
    "type": "update",
    "resource": { /* HealthCheck object */ }
  }
  {
    "type": "delete",
    "namespace": "prod",
    "name": "frontend-api"
  }
```

#### Implementation Strategy

**Option A: Watch API Server** (CHOSEN)
```go
// Dashboard uses client-go to watch HealthCheck CRs
cache.NewInformer(
  listWatch,
  &HealthCheck{},
  resyncPeriod,
  cache.ResourceEventHandlerFuncs{
    AddFunc:    func(obj interface{}) { /* broadcast to WS clients */ },
    UpdateFunc: func(old, new interface{}) { /* broadcast */ },
    DeleteFunc: func(obj interface{}) { /* broadcast */ },
  },
)
```

**Advantages**:
- No coupling between controller and dashboard
- Dashboard can be scaled independently
- Standard Kubernetes pattern
- Works with RBAC

**Disadvantages**:
- Slight delay (watch latency ~100-500ms)
- Additional load on API server (minimal)

**Alternative (Rejected): Shared Database**
- Adds complexity and external dependency
- Not Kubernetes-native
- Data consistency challenges

**Alternative (Rejected): Direct gRPC Connection**
- Tight coupling between components
- Harder to scale dashboard
- Service discovery complexity

### 3. Dashboard Frontend

**Purpose**: Web UI for visualizing service health.

#### Technology Stack
- **Framework**: React (hooks-based)
- **Styling**: Tailwind CSS
- **Data Fetching**: React Query (for REST) + native WebSocket
- **Routing**: React Router
- **Charts**: Recharts or Chart.js
- **Build**: Vite
- **Deployment**: Served by dashboard backend as static assets

#### Views

**1. Overview Dashboard**
- Cluster-wide health summary
- Services by namespace
- Health distribution (healthy/degraded/unhealthy)
- Recent incidents timeline

**2. Service Health Detail**
- Service metadata (name, namespace, selectors)
- Overall health status with visual indicator
- Individual check results (all checks listed)
- Pod-level health breakdown (table)
- Uptime statistics (graphs for each window)
- Historical trends (last 24h check results)
- Events log (recent status changes)

**3. Namespace View**
- All services in namespace
- Quick health overview
- Filter and search capabilities

**4. HealthCheck CR View**
- Raw HealthCheck resource (YAML viewer)
- Edit/delete capabilities (future)

#### Real-Time Updates

**WebSocket Integration**:
```javascript
const ws = new WebSocket('ws://dashboard:8080/ws/healthchecks');

ws.onmessage = (event) => {
  const update = JSON.parse(event.data);

  if (update.type === 'update') {
    // Update React state with new health data
    updateHealthCheck(update.resource);
  }
};
```

**Fallback**: If WebSocket disconnects, fall back to polling (30s interval).

## Data Storage Strategy

### Choice: Kubernetes API Server as Source of Truth

**Decision**: Store all data in HealthCheck `.status` field. No external database.

**Rationale**:
- Kubernetes-native approach
- Automatic persistence (etcd)
- RBAC integration
- No external dependencies
- Consistent with K8s patterns

**Limitations**:
- `.status` size limits (~256KB per resource)
- Not suitable for long-term historical data (months/years)

### Uptime Data Storage

**Strategy**: In-memory circular buffers with status snapshots

```go
type UptimeTracker struct {
    windows map[time.Duration]*CircularBuffer
}

type CircularBuffer struct {
    size    int      // Max entries to keep
    results []bool   // Success/failure
    times   []time.Time
}
```

**Memory Usage**:
- 1h window @ 30s interval = 120 entries = ~2KB per service
- 30d window @ 30s interval = 86,400 entries = ~1.7MB per service
- Optimization: Downsample older data (5min averages beyond 24h)

**Status Field Updates**:
- Store aggregated percentages only (not raw data)
- Update on each reconciliation
- Lightweight and within size limits

**Future Enhancement**: Optional Prometheus export for long-term storage.

## Reconciliation Logic

### Discovery Controller

```go
func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    service := &corev1.Service{}
    if err := r.Get(ctx, req.NamespacedName, service); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Get EndpointSlices for this service
    endpointSlices := &discoveryv1.EndpointSliceList{}
    if err := r.List(ctx, endpointSlices,
        client.InNamespace(service.Namespace),
        client.MatchingLabels{"kubernetes.io/service-name": service.Name}); err != nil {
        return ctrl.Result{}, err
    }

    // Build pod mapping
    pods := extractPodsFromEndpoints(endpointSlices)

    // Update cache
    r.Cache.Set(service.Namespace + "/" + service.Name, pods)

    // Trigger HealthCheck reconciliation if exists
    triggerHealthCheckReconcile(service)

    return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}
```

### HealthCheck Controller

```go
func (r *HealthCheckReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    hc := &monitoringv1alpha1.HealthCheck{}
    if err := r.Get(ctx, req.NamespacedName, hc); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Handle deletion with finalizer
    if !hc.DeletionTimestamp.IsZero() {
        return r.handleDeletion(ctx, hc)
    }

    // Ensure finalizer
    if !controllerutil.ContainsFinalizer(hc, finalizerName) {
        controllerutil.AddFinalizer(hc, finalizerName)
        return ctrl.Result{Requeue: true}, r.Update(ctx, hc)
    }

    // Get target service and pods
    service, pods, err := r.getTargetServiceAndPods(ctx, hc)
    if err != nil {
        return ctrl.Result{}, err
    }

    // Execute health checks
    results := r.executeChecks(ctx, hc, pods)

    // Update status
    hc.Status = r.buildStatus(hc, results, service, pods)

    if err := r.Status().Update(ctx, hc); err != nil {
        return ctrl.Result{}, err
    }

    // Requeue based on interval
    return ctrl.Result{RequeueAfter: hc.Spec.Interval.Duration}, nil
}
```

## Scalability Considerations

### Controller
- **Horizontal Scaling**: Leader election for multiple replicas (active-passive)
- **Vertical Scaling**: Resource limits based on cluster size
- **Check Concurrency**: Semaphore limiting (default: 100 concurrent checks)
- **Rate Limiting**: Per-service check rate limits to prevent overwhelming

### Dashboard
- **Horizontal Scaling**: Stateless, can run multiple replicas
- **WebSocket Scaling**: Each replica maintains own connections
- **Load Balancing**: Standard Kubernetes Service with session affinity

### Performance Targets
- **Controller**: Handle 1,000 services with 10,000 pods
- **Checks**: Execute 100,000 checks/minute
- **Dashboard**: Support 100 concurrent WebSocket clients per replica
- **API Latency**: <100ms for REST endpoints, <500ms WebSocket broadcast

## Security Considerations

### RBAC

**Controller Service Account**:
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: health-monitor-controller
rules:
  # Watch and manage HealthCheck CRs
  - apiGroups: ["monitoring.k8s.io"]
    resources: ["healthchecks"]
    verbs: ["get", "list", "watch", "update", "patch"]
  - apiGroups: ["monitoring.k8s.io"]
    resources: ["healthchecks/status"]
    verbs: ["get", "update", "patch"]

  # Discover services and pods
  - apiGroups: [""]
    resources: ["services", "endpoints", "pods"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["discovery.k8s.io"]
    resources: ["endpointslices"]
    verbs: ["get", "list", "watch"]

  # Networking resources
  - apiGroups: ["networking.k8s.io"]
    resources: ["ingresses"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["gateway.networking.k8s.io"]
    resources: ["gateways", "httproutes"]
    verbs: ["get", "list", "watch"]

  # Events
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["create", "patch"]
```

**Dashboard Service Account**:
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: health-monitor-dashboard
rules:
  # Read-only access to HealthCheck CRs
  - apiGroups: ["monitoring.k8s.io"]
    resources: ["healthchecks"]
    verbs: ["get", "list", "watch"]

  # Read service info for context
  - apiGroups: [""]
    resources: ["services"]
    verbs: ["get", "list"]
```

### Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: health-monitor-controller
spec:
  podSelector:
    matchLabels:
      app: health-monitor-controller
  policyTypes:
    - Egress
  egress:
    # Access to API server
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 6443

    # Access to pods for health checks
    - to:
        - podSelector: {}
      ports:
        - protocol: TCP
```

### Exec Check Security

**Risk**: Arbitrary command execution could be abused

**Mitigation**:
1. **Documentation**: Clearly warn about security implications
2. **RBAC**: Only cluster admins can create HealthCheck CRs (by default)
3. **Admission Policy** (optional): Webhook to restrict Exec checks
4. **Future**: Add `allowExecChecks` flag to controller, disabled by default

## Deployment Architecture

```yaml
# Controller Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: health-monitor-controller
spec:
  replicas: 2  # Active-passive via leader election
  selector:
    matchLabels:
      app: health-monitor-controller
  template:
    spec:
      serviceAccountName: health-monitor-controller
      containers:
        - name: controller
          image: health-monitor:latest
          command: ["/manager"]
          args:
            - --leader-elect
            - --health-probe-bind-address=:8081
            - --metrics-bind-address=:8080
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 1000m
              memory: 1Gi

---
# Dashboard Deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: health-monitor-dashboard
spec:
  replicas: 3  # Horizontal scaling
  selector:
    matchLabels:
      app: health-monitor-dashboard
  template:
    spec:
      serviceAccountName: health-monitor-dashboard
      containers:
        - name: dashboard
          image: health-monitor:latest
          command: ["/dashboard"]
          args:
            - --bind-address=:8080
          resources:
            requests:
              cpu: 50m
              memory: 64Mi
            limits:
              cpu: 500m
              memory: 512Mi
```

## Observability

### Metrics (Prometheus)

**Controller Metrics**:
```
# Reconciliation
health_monitor_reconcile_duration_seconds{controller="healthcheck"}
health_monitor_reconcile_errors_total{controller="healthcheck"}

# Health checks
health_monitor_checks_total{namespace, service, check_name, result}
health_monitor_check_duration_seconds{namespace, service, check_name, type}

# Service discovery
health_monitor_services_discovered_total
health_monitor_pods_monitored_total
```

**Dashboard Metrics**:
```
health_monitor_dashboard_requests_total{method, path, status}
health_monitor_dashboard_websocket_connections{state="active|idle"}
```

### Logging

**Structured Logging** (using zap):
```go
logger.Info("Executing health check",
    "namespace", hc.Namespace,
    "name", hc.Name,
    "service", hc.Spec.TargetRef.Name,
    "check", check.Name,
    "type", check.Type)
```

**Log Levels**:
- **Debug**: Individual check execution details
- **Info**: State transitions, reconciliation events
- **Warn**: Transient errors, retries
- **Error**: Permanent failures, crashes

### Tracing

**Future Enhancement**: OpenTelemetry integration for distributed tracing of check execution paths.

## Testing Strategy

### Unit Tests
- Check executors (HTTP, TCP, gRPC, Exec)
- Uptime calculation logic
- Status aggregation
- Reconciliation logic

### Integration Tests
- envtest for controller reconciliation
- Fake Kubernetes API server
- Mock check executors

### End-to-End Tests
- Deploy to kind/minikube cluster
- Create services and HealthChecks
- Verify status updates
- Test dashboard API and WebSocket

## Migration and Versioning

### API Versioning Strategy

**v1alpha1** (Initial):
- HealthCheck CRD with basic functionality
- Service target support only

**v1beta1** (Future):
- Add Pod and Deployment targets
- Alert integrations
- Advanced scheduling

**v1** (Stable):
- Production-ready API
- Conversion webhooks for backward compatibility

### Upgrade Path

Controller upgrades:
1. Install new CRD version (with conversion)
2. Rolling update controller deployment
3. Automatic status migration via webhook

## Alternatives Considered

### External Time-Series Database
**Considered**: Store uptime data in Prometheus, InfluxDB, or PostgreSQL

**Rejected**:
- Adds external dependency
- Increases complexity
- Not Kubernetes-native
- Initial version doesn't need years of history

**Future**: Optional Prometheus export for long-term storage

### Webhook-Based Checks
**Considered**: Allow HealthChecks to call external webhooks

**Rejected for v1alpha1**:
- Security implications (SSRF)
- Complexity in error handling
- Can be added later with proper security controls

### Unified Controller-Dashboard Binary
**Considered**: Single binary with mode flag

**Rejected**:
- Different scaling requirements
- Different resource needs
- Different RBAC permissions
- Separate binaries cleaner for ops

## Conclusion

This architecture provides:
- ✅ Kubernetes-native design
- ✅ No external dependencies
- ✅ Scalable to 1000s of services
- ✅ Real-time visibility
- ✅ Production-ready patterns
- ✅ Future extensibility

The design balances simplicity (no database), functionality (comprehensive health checks), and operability (standard K8s patterns).
