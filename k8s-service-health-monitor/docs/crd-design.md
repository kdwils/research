# HealthCheck CRD Design

## Overview

The `HealthCheck` Custom Resource Definition allows users to define custom health checks for Kubernetes services. The controller watches these resources and executes checks against service endpoints.

## API Version and Group

- **Group**: `monitoring.k8s.io`
- **Version**: `v1alpha1` (initial release)
- **Kind**: `HealthCheck`
- **Plural**: `healthchecks`
- **Singular**: `healthcheck`
- **ShortNames**: `hc`, `healthcheck`

## CRD Schema

### Complete Example

```yaml
apiVersion: monitoring.k8s.io/v1alpha1
kind: HealthCheck
metadata:
  name: frontend-api-health
  namespace: production
spec:
  # Target service to monitor
  targetRef:
    kind: Service
    name: frontend-api
    namespace: production

  # Check execution interval
  interval: 30s

  # Timeout for each check execution
  timeout: 10s

  # Number of consecutive failures before marking unhealthy
  failureThreshold: 3

  # Number of consecutive successes to mark healthy again
  successThreshold: 1

  # Check definitions (at least one required)
  checks:
    - name: http-health-endpoint
      type: HTTP
      http:
        scheme: HTTP  # HTTP or HTTPS
        host: ""      # Empty = pod IP, or specify custom host
        port: 8080
        path: /health
        httpHeaders:
          - name: X-Custom-Header
            value: health-check
        expectedStatus:
          - 200
          - 204
        expectedBodyRegex: '"status":\s*"healthy"'

    - name: http-api-functionality
      type: HTTP
      http:
        scheme: HTTPS
        port: 8443
        path: /api/v1/status
        expectedStatus:
          - 200
        expectedBodyContains: "operational"

    - name: grpc-health
      type: gRPC
      grpc:
        port: 9090
        service: "health.v1.Health"  # gRPC service name

    - name: tcp-connection
      type: TCP
      tcp:
        port: 5432

    - name: custom-script
      type: Exec
      exec:
        command:
          - /bin/sh
          - -c
          - |
            response=$(curl -s http://localhost:8080/metrics)
            echo "$response" | grep -q "app_ready 1"
        # Exit code 0 = healthy, non-zero = unhealthy

  # Uptime tracking configuration
  uptime:
    enabled: true
    windows:
      - 1h
      - 24h
      - 7d
      - 30d

status:
  # Overall health status
  phase: Healthy  # Healthy, Degraded, Unhealthy, Unknown

  # Individual check results
  checkResults:
    - name: http-health-endpoint
      healthy: true
      lastCheckTime: "2025-11-21T10:30:00Z"
      lastTransitionTime: "2025-11-21T09:00:00Z"
      consecutiveSuccesses: 120
      consecutiveFailures: 0
      message: "HTTP 200: OK"

    - name: http-api-functionality
      healthy: true
      lastCheckTime: "2025-11-21T10:30:00Z"
      lastTransitionTime: "2025-11-21T09:00:00Z"
      consecutiveSuccesses: 120
      consecutiveFailures: 0
      message: "HTTP 200: Response contains 'operational'"

    - name: grpc-health
      healthy: true
      lastCheckTime: "2025-11-21T10:30:00Z"
      lastTransitionTime: "2025-11-21T09:00:00Z"
      consecutiveSuccesses: 120
      consecutiveFailures: 0
      message: "gRPC health check: SERVING"

    - name: tcp-connection
      healthy: true
      lastCheckTime: "2025-11-21T10:30:00Z"
      lastTransitionTime: "2025-11-21T09:00:00Z"
      consecutiveSuccesses: 120
      consecutiveFailures: 0
      message: "TCP connection successful"

    - name: custom-script
      healthy: false
      lastCheckTime: "2025-11-21T10:30:00Z"
      lastTransitionTime: "2025-11-21T10:25:00Z"
      consecutiveSuccesses: 0
      consecutiveFailures: 2
      message: "Exit code 1: grep pattern not found"

  # Pod-level health breakdown
  podHealth:
    - podName: frontend-api-7d4b9c8f-abc123
      podIP: 10.244.1.15
      healthy: true
      checksHealthy: 4
      checksUnhealthy: 1
      lastCheckTime: "2025-11-21T10:30:00Z"

    - podName: frontend-api-7d4b9c8f-def456
      podIP: 10.244.2.20
      healthy: true
      checksHealthy: 5
      checksUnhealthy: 0
      lastCheckTime: "2025-11-21T10:30:00Z"

  # Uptime statistics
  uptime:
    - window: 1h
      percentage: 98.5
      totalChecks: 120
      successfulChecks: 118

    - window: 24h
      percentage: 99.2
      totalChecks: 2880
      successfulChecks: 2857

    - window: 7d
      percentage: 99.8
      totalChecks: 20160
      successfulChecks: 20120

    - window: 30d
      percentage: 99.7
      totalChecks: 86400
      successfulChecks: 86141

  # Service endpoint information (discovered)
  serviceEndpoints:
    ready: 2
    total: 2

  # Kubernetes-standard conditions
  conditions:
    - type: Ready
      status: "True"
      lastTransitionTime: "2025-11-21T09:00:00Z"
      reason: AllChecksHealthy
      message: "All health checks passing on all pods"

    - type: Degraded
      status: "False"
      lastTransitionTime: "2025-11-21T09:00:00Z"
      reason: NoIssues
      message: "Service operating normally"
```

## Field Specifications

### Spec Fields

#### `targetRef` (required)
Reference to the Kubernetes resource being monitored.

- **`kind`**: Currently supports `Service` (future: `Pod`, `Deployment`)
- **`name`**: Name of the resource
- **`namespace`**: Namespace (defaults to HealthCheck's namespace)

#### `interval` (optional, default: `30s`)
How often to execute health checks. Format: Go duration string (`30s`, `1m`, `5m`).

#### `timeout` (optional, default: `10s`)
Maximum time for check execution before considering it failed.

#### `failureThreshold` (optional, default: `3`)
Consecutive failures before marking check as unhealthy.

#### `successThreshold` (optional, default: `1`)
Consecutive successes needed to transition back to healthy.

#### `checks` (required, min: 1)
Array of health check definitions.

##### Check Types

**HTTP Check**
```yaml
- name: my-http-check
  type: HTTP
  http:
    scheme: HTTP | HTTPS
    host: ""  # Optional: custom host header / target
    port: 8080
    path: /health
    httpHeaders:  # Optional
      - name: Header-Name
        value: header-value
    expectedStatus:  # Optional: default [200-399]
      - 200
      - 204
    expectedBodyRegex: ""  # Optional: regex pattern
    expectedBodyContains: ""  # Optional: substring match
```

**gRPC Check**
```yaml
- name: my-grpc-check
  type: gRPC
  grpc:
    port: 9090
    service: "grpc.health.v1.Health"  # Standard gRPC health service
    useTLS: false  # Optional
```

**TCP Check**
```yaml
- name: my-tcp-check
  type: TCP
  tcp:
    port: 5432
```

**Exec Check**
```yaml
- name: my-exec-check
  type: Exec
  exec:
    command:
      - /bin/sh
      - -c
      - "your-command-here"
```

#### `uptime` (optional)
Configuration for uptime tracking.

- **`enabled`**: Boolean, default `true`
- **`windows`**: Array of time windows (`1h`, `24h`, `7d`, `30d`)

### Status Fields

#### `phase` (string)
Overall health status: `Healthy`, `Degraded`, `Unhealthy`, `Unknown`

- **Healthy**: All checks passing on all pods
- **Degraded**: Some checks failing but service partially operational
- **Unhealthy**: Critical failures affecting service
- **Unknown**: Unable to determine health (initialization, errors)

#### `checkResults` (array)
Results for each defined check across all pods (aggregated).

#### `podHealth` (array)
Per-pod health breakdown showing which pods are healthy/unhealthy.

#### `uptime` (array)
Uptime percentages for configured time windows.

#### `serviceEndpoints` (object)
Discovery information about service endpoints.

#### `conditions` (array)
Standard Kubernetes conditions following conventions.

## Validation Rules (CEL)

```yaml
# In CRD definition
validation:
  openAPIV3Schema:
    # ... field definitions ...
    x-kubernetes-validations:
      - rule: "self.checks.size() > 0"
        message: "At least one health check must be defined"

      - rule: "self.interval.duration >= '5s'"
        message: "Interval must be at least 5 seconds"

      - rule: "self.timeout.duration < self.interval.duration"
        message: "Timeout must be less than interval"

      - rule: "self.failureThreshold >= 1"
        message: "Failure threshold must be at least 1"

      - rule: "self.successThreshold >= 1"
        message: "Success threshold must be at least 1"
```

## Design Rationale

### Why Service-Level Not Pod-Level?

Services are the logical unit users care about. While pod-level data is collected, the primary abstraction is service health. Users can still drill down to pod-level details in the dashboard.

### Multiple Checks Per HealthCheck

Allows comprehensive validation:
- Basic connectivity (TCP)
- HTTP endpoint availability
- Response content verification
- Business logic validation (Exec)

All checks must pass for overall health.

### Uptime Windows

Historical tracking with multiple windows provides:
- Short-term: Recent incidents (1h)
- Medium-term: Daily patterns (24h)
- Long-term: SLA tracking (7d, 30d)

### Status Design

Follows Kubernetes conventions:
- Top-level `phase` for quick status
- Detailed `checkResults` for diagnostics
- Standard `conditions` for integration with tools
- Pod-level breakdown for troubleshooting

## Future Extensions (v1beta1, v1)

Potential additions:
- Support for `targetRef.kind: Pod` and `Deployment`
- Alert integration (webhooks, Slack, PagerDuty)
- Advanced scheduling (time-based, day-of-week)
- Multi-cluster health checks
- Distributed tracing integration
- Custom metrics export to Prometheus

## Alternative Approaches Considered

### Single Check Per CRD
**Rejected**: Would require multiple HealthCheck resources per service, cluttering the API and making management harder.

### Built-in Alerting
**Deferred**: Keep initial version focused on monitoring and visibility. Alerting can be added later or handled by external tools watching HealthCheck status.

### Pod-First Design
**Rejected**: Services are the primary unit of deployment and user concern in Kubernetes.
