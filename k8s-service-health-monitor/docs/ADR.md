# Architectural Decision Records

## ADR-001: Use Kubernetes API Server as Single Source of Truth

**Date**: 2025-11-21

**Status**: Accepted

### Context

We need to persist health check results and uptime metrics. Options include:
1. External database (PostgreSQL, InfluxDB)
2. In-memory only (ephemeral)
3. Kubernetes API Server (etcd) via HealthCheck `.status` field

### Decision

Store all data in HealthCheck `.status` field. Use in-memory circular buffers for uptime calculations, persisting only aggregated percentages.

### Rationale

- **Kubernetes-native**: Follows standard K8s patterns
- **Zero dependencies**: No external database to manage
- **RBAC integration**: Automatic permission system
- **High availability**: etcd provides replication
- **Operational simplicity**: One less system to operate
- **Size constraints acceptable**: Aggregated metrics fit within status size limits (~256KB)

### Consequences

**Positive**:
- Simple deployment (no DB setup)
- Standard K8s operational model
- Built-in backup/restore (etcd snapshots)
- Works in air-gapped environments

**Negative**:
- Limited historical data (optimized for 30-day windows)
- Can't query across time series efficiently
- Status field size constraints

**Mitigation**:
- Future: Optional Prometheus export for long-term analytics
- Downsample older data (5min averages beyond 24h)

---

## ADR-002: Separate Controller and Dashboard Binaries

**Date**: 2025-11-21

**Status**: Accepted

### Context

Controller and dashboard can be:
1. Single binary with mode flag
2. Separate binaries in same repo
3. Separate repositories

### Decision

Separate binaries (`cmd/controller/main.go` and `cmd/dashboard/main.go`) in same repository.

### Rationale

**Different Characteristics**:
- Controller: CPU-intensive (check execution), needs write access
- Dashboard: I/O-intensive (WebSocket), needs only read access
- Different scaling requirements
- Different RBAC permissions

**Operational Benefits**:
- Scale independently
- Upgrade separately (dashboard changes don't require controller restart)
- Different resource limits
- Security: Dashboard has read-only RBAC

**Development Benefits**:
- Shared code (API types, clients)
- Single release artifact
- Unified versioning

### Consequences

**Positive**:
- Clear separation of concerns
- Optimal resource allocation
- Security isolation
- Independent scaling

**Negative**:
- Slightly more complex deployment (two Deployments)
- Two processes to monitor

**Mitigation**:
- Same container image, different entrypoints
- Helm chart simplifies deployment

---

## ADR-003: Dashboard Watches API Server (Not Shared Database)

**Date**: 2025-11-21

**Status**: Accepted

### Context

Dashboard needs health data. Options:
1. Watch HealthCheck CRs via Kubernetes API
2. Shared database between controller and dashboard
3. Direct gRPC connection to controller
4. Message queue (NATS, Kafka)

### Decision

Dashboard uses client-go informers to watch HealthCheck resources from API server.

### Rationale

- **Decoupled**: Controller and dashboard are independent
- **Standard pattern**: Common in K8s ecosystem
- **Scalability**: Dashboard can scale horizontally
- **Simplicity**: No additional infrastructure
- **Latency acceptable**: Watch updates within 100-500ms
- **Resilient**: Automatic reconnection, resync

### Consequences

**Positive**:
- No coupling between components
- Dashboard can restart without data loss
- Standard K8s operational model
- RBAC enforced automatically

**Negative**:
- Slight latency vs direct communication
- Additional API server load (minimal for watching)

**Performance Impact**:
- Watching 1000 HealthChecks = negligible API server load
- Informer caching reduces API calls

---

## ADR-004: In-Memory Uptime Calculation with Status Snapshots

**Date**: 2025-11-21

**Status**: Accepted

### Context

Uptime metrics require historical data. Options:
1. Store every check result in status
2. In-memory circular buffers with aggregated snapshots
3. External time-series database

### Decision

Maintain in-memory circular buffers of check results. Persist only calculated uptime percentages in `.status.uptime[]`.

### Rationale

**Memory Efficiency**:
- 30d @ 30s interval = ~86K entries
- With downsampling: ~10K entries (5min averages after 24h)
- Per-service cost: ~100KB in memory

**Status Field Size**:
- Storing all results: 86K × 10 bytes = 860KB (exceeds limits)
- Storing percentages: 4 windows × 50 bytes = 200 bytes ✓

**Accuracy**:
- Recent data (24h): Full granularity (30s)
- Older data (>24h): 5min averages (still accurate)

### Consequences

**Positive**:
- Fits within K8s resource limits
- Fast calculations (in-memory)
- Accurate uptime metrics

**Negative**:
- Data lost on controller restart (rebuilds from last snapshot)
- Can't query historical check details

**Mitigation**:
- On restart: Use last known uptime % from status as baseline
- Gradually refill buffers with new checks
- Metric accuracy fully restored within longest window (30d)

---

## ADR-005: Multi-Type Check Support in Single HealthCheck CR

**Date**: 2025-11-21

**Status**: Accepted

### Context

Health checks can be HTTP, TCP, gRPC, or Exec. Options:
1. Single check type per HealthCheck CR
2. Multiple check types per HealthCheck CR
3. Separate CR types (HTTPHealthCheck, TCPHealthCheck, etc.)

### Decision

Allow multiple checks of any type within a single HealthCheck CR via `.spec.checks[]` array.

### Rationale

**User Experience**:
- Services typically need multiple validations (TCP + HTTP + content)
- Single CR is more manageable than multiple CRs
- Logical grouping of related checks

**Example Use Case**:
```yaml
checks:
  - type: TCP       # Database is accepting connections
  - type: HTTP      # Health endpoint responds
  - type: HTTP      # API returns valid data
  - type: Exec      # Custom business logic validation
```

**Aggregation**:
- Service is healthy only if ALL checks pass
- Provides comprehensive health validation

### Consequences

**Positive**:
- Comprehensive service validation
- Fewer resources to manage
- Logical grouping
- Easier RBAC (one resource to secure)

**Negative**:
- Larger CR size
- All-or-nothing health status

**Mitigation**:
- Individual check results in `.status.checkResults[]` for debugging
- Future: Optional per-check criticality levels

---

## ADR-006: Service-Level Focus (Not Pod-Level)

**Date**: 2025-11-21

**Status**: Accepted

### Context

Monitoring can target:
1. Individual pods
2. Services (pod aggregation)
3. Deployments/StatefulSets

### Decision

Primary abstraction is **Service** (`targetRef.kind: Service`). Collect pod-level data but report service-level health.

### Rationale

**User Mental Model**:
- Users think in terms of services, not individual pods
- Services are the unit of deployment and traffic routing
- Kubernetes probes already cover pod-level health

**Architecture Alignment**:
- Ingress → Service
- Gateway → Service
- DNS → Service

**Use Case Validation**:
- "Is my API healthy?" → Service question
- "Can users access the frontend?" → Service question

### Consequences

**Positive**:
- Matches user expectations
- Aligns with K8s networking model
- Clear aggregation boundary

**Negative**:
- Can't directly monitor pods without services
- Doesn't cover DaemonSets, StatefulSets directly

**Mitigation**:
- `.status.podHealth[]` provides pod-level breakdown
- Future: Support `targetRef.kind: Pod` for edge cases
- StatefulSets typically have headless services (still supported)

---

## ADR-007: CEL Validation Instead of Webhooks

**Date**: 2025-11-21

**Status**: Accepted

### Context

CRD validation can use:
1. Admission webhooks (ValidatingWebhook)
2. CEL (Common Expression Language) in CRD schema
3. Controller-side validation only

### Decision

Use CEL validation rules (`x-kubernetes-validations`) in CRD OpenAPI schema.

### Rationale

**Operational Simplicity**:
- No webhook deployment required
- No certificate management
- No webhook latency

**2025 Best Practice**:
- CEL is standard in modern K8s (1.25+)
- Recommended by Kubebuilder docs
- Used extensively in Gateway API

**Example**:
```yaml
x-kubernetes-validations:
  - rule: "self.interval.duration >= '5s'"
    message: "Interval must be at least 5 seconds"
```

### Consequences

**Positive**:
- Simpler deployment (one less component)
- Faster validation (no network call)
- Declarative rules
- Automatic enforcement

**Negative**:
- Less flexibility than webhook code
- Limited to what CEL can express

**Assessment**: CEL covers all critical validations for v1alpha1.

---

## ADR-008: Exec Check Security Model

**Date**: 2025-11-21

**Status**: Accepted with Safeguards

### Context

Exec checks run arbitrary commands. Security concerns:
1. Allow with warnings
2. Disallow completely
3. Restricted subset only

### Decision

Allow Exec checks with **strong documentation warnings** and **RBAC protection**.

### Rationale

**Functionality Requirement**:
- Some health checks require custom logic
- Can't always be expressed as HTTP/TCP/gRPC
- Example: Check Prometheus metrics, verify database records, validate files

**Security Model**:
- Commands run in **target pod context**, not controller
- Controller only reads exit code and output
- Same security boundary as `kubectl exec`

**Protection Layers**:
1. **RBAC**: Only authorized users can create HealthCheck CRs
2. **Documentation**: Clear warning about security implications
3. **Future**: `--allow-exec-checks` controller flag (opt-in)

### Consequences

**Positive**:
- Maximum flexibility for custom checks
- Solves complex validation scenarios
- Consistent with K8s security model

**Negative**:
- Potential for misuse if RBAC is too permissive
- Could be used for unintended pod access

**Safeguards**:
```yaml
# Recommended RBAC: Only cluster admins
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: healthcheck-admin
rules:
  - apiGroups: ["monitoring.k8s.io"]
    resources: ["healthchecks"]
    verbs: ["create", "update", "delete"]
```

**Documentation**:
- ⚠️ Warn users about Exec check security
- Recommend HTTP/TCP/gRPC when possible
- Explain RBAC best practices

---

## ADR-009: Leader Election for Controller High Availability

**Date**: 2025-11-21

**Status**: Accepted

### Context

Running multiple controller replicas requires coordination to avoid duplicate check execution.

### Decision

Use Kubernetes leader election (active-passive model). Only leader executes checks.

### Rationale

**Avoid Duplicate Work**:
- Two controllers would execute same checks → double load
- Could cause race conditions on status updates

**Standard Pattern**:
- Built into controller-runtime
- Uses Lease resources
- Automatic failover

**Implementation**:
```go
mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
    LeaderElection:   true,
    LeaderElectionID: "health-monitor-controller-lock",
})
```

### Consequences

**Positive**:
- High availability (automatic failover)
- No duplicate work
- Standard K8s pattern

**Negative**:
- Only one active controller (passive replicas are idle)
- Can't horizontally scale check execution

**Future Optimization**:
- Shard HealthChecks across multiple controllers
- Each controller handles subset (by namespace or hash)

---

## ADR-010: WebSocket for Real-Time Dashboard Updates

**Date**: 2025-11-21

**Status**: Accepted

### Context

Dashboard needs real-time updates. Options:
1. Polling (REST API every N seconds)
2. WebSocket (bidirectional connection)
3. Server-Sent Events (SSE)

### Decision

Use WebSocket with polling fallback.

### Rationale

**Real-Time Requirements**:
- Health status changes should appear immediately
- Uptime percentages update every check interval

**WebSocket Advantages**:
- True real-time (sub-second updates)
- Efficient (no repeated HTTP overhead)
- Bidirectional (future: client subscriptions)

**Polling Fallback**:
- If WebSocket fails (proxy, firewall)
- Degrade to 30s polling gracefully

### Consequences

**Positive**:
- Instant updates (great UX)
- Efficient bandwidth usage
- Modern web standard

**Negative**:
- Requires connection management
- Proxies may interfere

**Mitigation**:
```javascript
// Automatic reconnection
ws.onclose = () => {
  setTimeout(() => reconnect(), 5000);
};

// Fallback to polling
if (ws.readyState !== WebSocket.OPEN) {
  setInterval(() => fetchHealthChecks(), 30000);
}
```

---

## Summary of Decisions

| ADR | Decision | Impact |
|-----|----------|--------|
| 001 | K8s API as source of truth | Simplicity, zero dependencies |
| 002 | Separate binaries | Independent scaling, security |
| 003 | Dashboard watches API | Decoupling, standard pattern |
| 004 | In-memory uptime tracking | Performance, size constraints |
| 005 | Multi-type checks | Comprehensive validation |
| 006 | Service-level focus | User alignment |
| 007 | CEL validation | Operational simplicity |
| 008 | Allow Exec checks | Flexibility with safeguards |
| 009 | Leader election | High availability |
| 010 | WebSocket updates | Real-time UX |

These decisions collectively create a **Kubernetes-native, production-ready, operationally simple** health monitoring system.
