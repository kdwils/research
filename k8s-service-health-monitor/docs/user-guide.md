# User Guide - Kubernetes Service Health Monitor

## Table of Contents

1. [Introduction](#introduction)
2. [Installation](#installation)
3. [Creating Health Checks](#creating-health-checks)
4. [Understanding Health Status](#understanding-health-status)
5. [Using the Dashboard](#using-the-dashboard)
6. [Best Practices](#best-practices)
7. [Troubleshooting](#troubleshooting)

## Introduction

The Kubernetes Service Health Monitor provides automated, comprehensive health monitoring for your Kubernetes services. Unlike pod probes which monitor individual containers, this system monitors services as logical units, automatically discovering backing pods and providing unified health visibility.

### Key Concepts

- **HealthCheck**: A Custom Resource that defines health checks for a service
- **Service**: The Kubernetes Service being monitored (target)
- **Checks**: Individual health validations (HTTP, TCP, gRPC, Exec)
- **Uptime**: Historical availability metrics over time windows
- **Phase**: Overall health status (Healthy, Degraded, Unhealthy, Unknown)

## Installation

### Prerequisites

- Kubernetes cluster version 1.24 or higher
- `kubectl` configured to access your cluster
- Cluster admin permissions (for initial setup)

### Step 1: Install CRDs

```bash
kubectl apply -f https://raw.githubusercontent.com/kdwils/k8s-service-health-monitor/main/config/crd/healthcheck-crd.yaml
```

Verify CRD installation:
```bash
kubectl get crd healthchecks.monitoring.k8s.io
```

### Step 2: Create Namespace

```bash
kubectl create namespace health-monitor-system
```

### Step 3: Install RBAC

```bash
kubectl apply -f config/rbac/role.yaml
kubectl apply -f config/rbac/role_binding.yaml
```

### Step 4: Deploy Controller and Dashboard

```bash
kubectl apply -f config/manager/controller.yaml
kubectl apply -f config/manager/dashboard.yaml
```

### Step 5: Verify Deployment

```bash
kubectl get pods -n health-monitor-system

# Expected output:
# NAME                                        READY   STATUS    RESTARTS   AGE
# health-monitor-controller-xxx-yyy          1/1     Running   0          30s
# health-monitor-controller-xxx-zzz          1/1     Running   0          30s
# health-monitor-dashboard-aaa-bbb           1/1     Running   0          30s
# health-monitor-dashboard-aaa-ccc           1/1     Running   0          30s
# health-monitor-dashboard-aaa-ddd           1/1     Running   0          30s
```

## Creating Health Checks

### Basic HTTP Health Check

The simplest HealthCheck monitors an HTTP endpoint:

```yaml
apiVersion: monitoring.k8s.io/v1alpha1
kind: HealthCheck
metadata:
  name: my-app-health
  namespace: default
spec:
  targetRef:
    kind: Service
    name: my-app

  checks:
    - name: health-endpoint
      type: HTTP
      http:
        port: 8080
        path: /health
```

Apply it:
```bash
kubectl apply -f healthcheck.yaml
```

### HTTP Check with Validation

Validate response status and body content:

```yaml
spec:
  checks:
    - name: api-health
      type: HTTP
      http:
        scheme: HTTPS
        port: 8443
        path: /api/health
        httpHeaders:
          - name: Authorization
            value: "Bearer ${TOKEN}"
        expectedStatus:
          - 200
          - 204
        expectedBodyRegex: '"status":\s*"ok"'
        expectedBodyContains: "healthy"
```

**Options**:
- `scheme`: HTTP or HTTPS (default: HTTP)
- `host`: Custom host header (default: pod IP)
- `port`: Port number (required)
- `path`: URL path (default: /)
- `httpHeaders`: Custom HTTP headers
- `expectedStatus`: List of valid status codes (default: 200-399)
- `expectedBodyRegex`: Regex pattern that must match response body
- `expectedBodyContains`: Substring that must appear in response body

### TCP Connection Check

Check if a TCP port accepts connections:

```yaml
spec:
  checks:
    - name: database-port
      type: TCP
      tcp:
        port: 5432
```

Useful for:
- Databases (PostgreSQL, MySQL, Redis)
- Message queues (RabbitMQ, Kafka)
- Any TCP service

### gRPC Health Check

Use standard gRPC health checking protocol:

```yaml
spec:
  checks:
    - name: grpc-service
      type: gRPC
      grpc:
        port: 9090
        service: "myapp.v1.MyService"
        useTLS: true
```

**Note**: Your service must implement the [gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md).

### Custom Exec Check

Execute a command inside the pod:

```yaml
spec:
  checks:
    - name: postgres-ready
      type: Exec
      exec:
        command:
          - /bin/sh
          - -c
          - "pg_isready -h localhost -p 5432"
```

**Exit code semantics**:
- Exit 0: Healthy
- Non-zero: Unhealthy

**⚠️ Security Warning**: Exec checks run commands in your pods. Only trusted users should create HealthChecks with Exec checks.

### Multiple Checks

Combine multiple check types for comprehensive validation:

```yaml
spec:
  checks:
    # Basic connectivity
    - name: tcp-connection
      type: TCP
      tcp:
        port: 8080

    # HTTP health endpoint
    - name: health
      type: HTTP
      http:
        port: 8080
        path: /health

    # API functionality
    - name: api-ready
      type: HTTP
      http:
        port: 8080
        path: /api/ready
        expectedBodyContains: "ready"

    # Custom validation
    - name: custom-check
      type: Exec
      exec:
        command: ["/app/health-check.sh"]
```

**Behavior**: Service is healthy only if ALL checks pass.

### Configuring Check Intervals

```yaml
spec:
  interval: 30s      # Run checks every 30 seconds
  timeout: 10s       # Timeout for each check execution
  failureThreshold: 3  # Mark unhealthy after 3 consecutive failures
  successThreshold: 1  # Mark healthy after 1 success
```

**Recommendations**:
- **interval**: 30s for most cases, 60s for expensive checks
- **timeout**: Should be less than interval
- **failureThreshold**: 3 prevents false positives from transient issues
- **successThreshold**: 1 allows quick recovery

### Uptime Tracking

Configure which time windows to track:

```yaml
spec:
  uptime:
    enabled: true
    windows:
      - 1h    # Last hour
      - 24h   # Last day
      - 7d    # Last week
      - 30d   # Last month
```

Disable uptime tracking:
```yaml
spec:
  uptime:
    enabled: false
```

## Understanding Health Status

### Health Phases

```yaml
status:
  phase: Healthy  # Healthy | Degraded | Unhealthy | Unknown
```

- **Healthy**: All checks passing on all pods
- **Degraded**: Some checks failing but service partially operational
- **Unhealthy**: Critical failures affecting service availability
- **Unknown**: Unable to determine health (no pods, discovery failed, etc.)

### Check Results

View individual check results:

```yaml
status:
  checkResults:
    - name: health-endpoint
      healthy: true
      lastCheckTime: "2025-11-21T10:30:00Z"
      lastTransitionTime: "2025-11-21T09:00:00Z"
      consecutiveSuccesses: 120
      consecutiveFailures: 0
      message: "HTTP 200: OK"

    - name: api-functionality
      healthy: false
      consecutiveSuccesses: 0
      consecutiveFailures: 3
      message: "HTTP 500: Internal Server Error"
```

### Pod-Level Health

See which pods are healthy:

```yaml
status:
  podHealth:
    - podName: my-app-abc123
      podIP: 10.244.1.15
      healthy: true
      checksHealthy: 3
      checksUnhealthy: 0
      lastCheckTime: "2025-11-21T10:30:00Z"
```

### Uptime Statistics

```yaml
status:
  uptime:
    - window: 1h
      percentage: 99.5
      totalChecks: 120
      successfulChecks: 119

    - window: 24h
      percentage: 99.8
      totalChecks: 2880
      successfulChecks: 2874
```

**Interpretation**:
- **percentage**: Uptime % (0-100)
- **totalChecks**: Number of checks executed
- **successfulChecks**: Number that passed

**SLA Calculation**: The `30d` window percentage can be used for SLA reporting.

### Kubernetes Conditions

Standard conditions:

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: AllChecksHealthy
      message: "All health checks passing on all pods"
      lastTransitionTime: "2025-11-21T09:00:00Z"
```

Integrates with tools that read Kubernetes conditions.

## Using the Dashboard

### Accessing the Dashboard

**Option 1: LoadBalancer** (if supported):
```bash
kubectl get svc -n health-monitor-system health-monitor-dashboard
# Access via EXTERNAL-IP
```

**Option 2: Port Forward** (local development):
```bash
kubectl port-forward -n health-monitor-system svc/health-monitor-dashboard 8080:80
```
Open http://localhost:8080

**Option 3: Ingress** (production):
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: health-monitor-dashboard
  namespace: health-monitor-system
spec:
  rules:
    - host: health.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: health-monitor-dashboard
                port:
                  number: 80
```

### Dashboard Features

1. **Cluster Overview**
   - Total services monitored
   - Health distribution (healthy/degraded/unhealthy)
   - Recent incidents

2. **Service List**
   - All monitored services
   - Health status indicators
   - Filter by namespace
   - Search by name

3. **Service Detail View**
   - Overall health phase
   - Individual check results
   - Pod-level breakdown
   - Uptime graphs
   - Historical trends

4. **Real-Time Updates**
   - Automatic refresh via WebSocket
   - Instant status changes
   - Live uptime updates

### API Usage

The dashboard exposes a REST API:

**List all HealthChecks**:
```bash
curl http://dashboard/api/v1/healthchecks
```

**Get specific HealthCheck**:
```bash
curl http://dashboard/api/v1/healthchecks/default/my-app-health
```

**Cluster-wide summary**:
```bash
curl http://dashboard/api/v1/health-summary
```

**Namespace summary**:
```bash
curl http://dashboard/api/v1/namespaces/production/health-summary
```

**WebSocket for real-time updates**:
```javascript
const ws = new WebSocket('ws://dashboard/ws/healthchecks');
ws.onmessage = (event) => {
  const update = JSON.parse(event.data);
  console.log(update.type, update.resource);
};
```

## Best Practices

### Health Check Design

1. **Start Simple**: Begin with basic HTTP/TCP checks, add complexity as needed
2. **Check What Matters**: Validate actual functionality, not just process existence
3. **Avoid Expensive Checks**: Don't run database queries in health checks
4. **Use Multiple Checks**: Combine TCP (connectivity) + HTTP (endpoint) + Exec (custom logic)
5. **Validate Responses**: Use `expectedBodyRegex` to ensure correct responses

### Interval Configuration

```yaml
# For critical services
interval: 30s
failureThreshold: 2

# For non-critical services
interval: 60s
failureThreshold: 3

# For expensive checks
interval: 300s  # 5 minutes
```

### Security

1. **Restrict RBAC**: Limit who can create HealthChecks
   ```yaml
   # Only allow specific users to create HealthChecks
   kind: RoleBinding
   roleRef:
     name: healthcheck-admin
   subjects:
     - kind: User
       name: ops-team
   ```

2. **Avoid Sensitive Data**: Don't put secrets in check configurations
   - Use Kubernetes secrets for tokens
   - Reference via environment variables

3. **Review Exec Checks**: Audit all Exec commands before deployment

4. **Use HTTPS**: Prefer HTTPS over HTTP for sensitive health endpoints

### Uptime Tracking

```yaml
# For SLA tracking
uptime:
  windows:
    - 24h   # Daily SLA
    - 7d    # Weekly SLA
    - 30d   # Monthly SLA

# For debugging recent issues
uptime:
  windows:
    - 1h    # Very recent
    - 6h    # Last few hours
    - 24h   # Today
```

### Namespace Organization

```yaml
# Development environment
metadata:
  namespace: dev

# Production (separate HealthChecks)
metadata:
  namespace: prod
```

Benefits:
- Different check intervals per environment
- Separate RBAC policies
- Independent uptime tracking

## Troubleshooting

### HealthCheck Not Executing

**Symptom**: Status never updates

**Check**:
```bash
# Verify controller is running
kubectl get pods -n health-monitor-system

# Check controller logs
kubectl logs -n health-monitor-system deployment/health-monitor-controller

# Verify HealthCheck was created
kubectl get healthcheck -A
```

**Common Causes**:
- Controller not running
- Service doesn't exist
- No pods backing the service
- RBAC permissions missing

### Checks Failing

**Symptom**: `phase: Unhealthy`

**Debug**:
```bash
# View check results
kubectl get healthcheck my-app-health -o jsonpath='{.status.checkResults}'

# Check pod health
kubectl get healthcheck my-app-health -o jsonpath='{.status.podHealth}'

# View detailed message
kubectl get healthcheck my-app-health -o yaml | grep message
```

**Common Causes**:
- Incorrect port number
- Wrong path
- Service not ready
- Network policies blocking access
- Timeout too short

### HTTP Checks Failing with Connection Refused

**Possible Causes**:
1. Wrong port number
2. Pod not listening on all interfaces (0.0.0.0)
3. Health endpoint not implemented

**Solution**:
```yaml
# Verify with a TCP check first
checks:
  - name: tcp-test
    type: TCP
    tcp:
      port: 8080  # Verify this is correct

# Then add HTTP check
  - name: http-test
    type: HTTP
    http:
      port: 8080
      path: /health
```

### Exec Checks Not Working

**Error**: "Failed to create executor"

**Causes**:
- RBAC permissions for pod/exec missing
- Pod doesn't have the specified command
- Command path incorrect

**Solution**:
```yaml
# Test command exists in pod
kubectl exec -it <pod-name> -- /bin/sh -c "which pg_isready"

# Use full command path
exec:
  command:
    - /usr/bin/pg_isready
    - -h
    - localhost
```

### Dashboard Not Showing Updates

**Symptom**: Dashboard shows old data

**Check**:
```bash
# Verify dashboard is running
kubectl get pods -n health-monitor-system | grep dashboard

# Check dashboard logs
kubectl logs -n health-monitor-system deployment/health-monitor-dashboard

# Test API directly
kubectl port-forward -n health-monitor-system svc/health-monitor-dashboard 8080:80
curl http://localhost:8080/api/v1/healthchecks
```

**Solutions**:
- Refresh browser (clear cache)
- Check WebSocket connection in browser dev tools
- Verify RBAC permissions for dashboard

### High Resource Usage

**Symptom**: Controller using too much CPU/memory

**Causes**:
- Too many HealthChecks
- Checks running too frequently
- Large number of pods per service

**Solutions**:
```yaml
# Increase interval
interval: 60s  # instead of 30s

# Reduce window tracking
uptime:
  windows:
    - 1h
    - 24h
    # Remove 7d, 30d if not needed

# Increase controller resources
resources:
  limits:
    cpu: 2000m
    memory: 2Gi
```

### Getting Help

1. **Check Logs**:
   ```bash
   kubectl logs -n health-monitor-system deployment/health-monitor-controller
   ```

2. **Describe HealthCheck**:
   ```bash
   kubectl describe healthcheck my-app-health
   ```

3. **View Events**:
   ```bash
   kubectl get events --sort-by='.lastTimestamp' | grep HealthCheck
   ```

4. **Enable Debug Logging**:
   ```yaml
   # In controller deployment
   args:
     - --zap-log-level=debug
   ```

5. **Community Support**:
   - GitHub Issues
   - Kubernetes Slack (#health-monitor)

## Next Steps

- **[Developer Guide](developer-guide.md)**: Learn about the architecture and contribute
- **[Examples](../config/samples/)**: See more complex HealthCheck examples
- **[Architecture](architecture.md)**: Understand system design
