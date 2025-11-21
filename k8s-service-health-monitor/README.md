# Kubernetes Service Health Monitor

An intelligent Kubernetes controller that automatically discovers services and their pods, executes custom health checks, tracks uptime metrics, and provides real-time visibility through a unified dashboard.

## Overview

While Kubernetes provides pod-level liveness and readiness probes, there's no native way to:
- Get a holistic view of service health across all backing pods
- Automatically discover and monitor service → pod relationships
- Track uptime and availability metrics for services over time
- Define custom health checks beyond what pod probes offer
- Visualize service health in a unified dashboard

**Service Health Monitor** solves these problems by providing:

- ✅ **Automatic Discovery**: Discovers services, pods, and networking resources (Ingress, Gateway API)
- ✅ **External Monitoring**: Monitor out-of-cluster services and endpoints
- ✅ **User-Defined Overrides**: Manual endpoint specification overrides discovered pods
- ✅ **Custom Health Checks**: HTTP, TCP, gRPC, and Exec checks with response validation
- ✅ **Uptime Tracking**: Historical uptime metrics across multiple time windows (1h, 24h, 7d, 30d)
- ✅ **Real-Time Dashboard**: React-based WebSocket-powered UI for live health monitoring
- ✅ **Kubernetes-Native**: Uses CRDs, follows K8s conventions, zero external dependencies
- ✅ **Production-Ready**: Leader election, RBAC, security best practices

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                  Kubernetes Cluster                          │
│                                                              │
│  ┌────────────────────┐        ┌─────────────────────────┐  │
│  │    Controller      │        │    Dashboard            │  │
│  │  - Discovers Svcs  │        │  - REST API             │  │
│  │  - Executes Checks │        │  - WebSocket Hub        │  │
│  │  - Tracks Uptime   │        │  - Web UI               │  │
│  └────────────────────┘        └─────────────────────────┘  │
│           │                              │                   │
│           └──────────────────────────────┘                   │
│                      │                                       │
│                HealthCheck CRs                               │
│              (stored in etcd)                                │
└─────────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Kubernetes cluster (1.24+)
- kubectl configured
- Docker (for building images)

### Installation

1. **Install CRDs**:
```bash
kubectl apply -f config/crd/
```

2. **Deploy Controller and Dashboard**:
```bash
kubectl apply -f config/rbac/
kubectl apply -f config/manager/
```

3. **Verify Installation**:
```bash
kubectl get pods -n health-monitor-system
```

### Create Your First HealthCheck

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

  interval: 30s
  timeout: 10s

  checks:
    - name: http-health
      type: HTTP
      http:
        port: 8080
        path: /health
        expectedStatus:
          - 200

  uptime:
    enabled: true
    windows:
      - 1h
      - 24h
      - 7d
```

Apply it:
```bash
kubectl apply -f my-healthcheck.yaml
```

Check status:
```bash
kubectl get healthcheck my-app-health -o yaml
```

### Access the Dashboard

Get the dashboard service URL:
```bash
kubectl get svc -n health-monitor-system health-monitor-dashboard
```

For local development with port-forward:
```bash
kubectl port-forward -n health-monitor-system svc/health-monitor-dashboard 8080:80
```

Then open http://localhost:8080 in your browser.

## Features

### Health Check Types

#### HTTP Checks
```yaml
checks:
  - name: api-health
    type: HTTP
    http:
      scheme: HTTPS
      port: 8443
      path: /api/health
      httpHeaders:
        - name: Authorization
          value: "Bearer token"
      expectedStatus:
        - 200
      expectedBodyRegex: '"status":\s*"healthy"'
```

#### TCP Checks
```yaml
checks:
  - name: database-connection
    type: TCP
    tcp:
      port: 5432
```

#### gRPC Checks
```yaml
checks:
  - name: grpc-service
    type: gRPC
    grpc:
      port: 9090
      service: "grpc.health.v1.Health"
      useTLS: true
```

#### Exec Checks
```yaml
checks:
  - name: custom-validation
    type: Exec
    exec:
      command:
        - /bin/sh
        - -c
        - "pg_isready -h localhost"
```

### External / Out-of-Cluster Health Checks

Monitor services outside your Kubernetes cluster by specifying manual endpoints:

```yaml
apiVersion: monitoring.k8s.io/v1alpha1
kind: HealthCheck
metadata:
  name: external-api-health
spec:
  # Manual endpoints - external services or out-of-cluster targets
  endpoints:
    - name: api-server-1
      address: api1.example.com
      labels:
        region: us-east-1
    - name: api-server-2
      address: api2.example.com
      labels:
        region: us-west-2

  checks:
    - name: https-endpoint
      type: HTTP
      http:
        scheme: HTTPS
        port: 443
        path: /api/v1/health
```

**User-Defined Overrides**: If both `targetRef` and `endpoints` are specified, manual endpoints take priority and override service discovery. This allows you to:
- Monitor external APIs and services
- Test specific backend instances
- Override discovered pods with custom targets
- Monitor databases, message queues, or any TCP/HTTP service outside the cluster

### Multiple Checks Per Service

You can define multiple checks to comprehensively validate service health:

```yaml
checks:
  - name: tcp-connection
    type: TCP
    tcp:
      port: 8080
  - name: health-endpoint
    type: HTTP
    http:
      port: 8080
      path: /health
  - name: api-functionality
    type: HTTP
    http:
      port: 8080
      path: /api/status
      expectedBodyContains: "operational"
```

All checks must pass for the service to be considered healthy.

### Uptime Tracking

Automatically tracks and calculates uptime percentages:

```yaml
uptime:
  enabled: true
  windows:
    - 1h    # Last hour
    - 24h   # Last 24 hours
    - 7d    # Last 7 days
    - 30d   # Last 30 days
```

View in status:
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

### Configurable Thresholds

Control when services are marked unhealthy:

```yaml
failureThreshold: 3    # 3 consecutive failures → unhealthy
successThreshold: 1    # 1 success → healthy again
```

### Service Discovery

Automatically discovers:
- Services and their backing pods
- EndpointSlices for real-time pod mapping
- Pod readiness status
- Service endpoints (ready/total)

### Dashboard Features

- **Real-Time Updates**: WebSocket-powered live health status
- **Service Overview**: All services with health indicators
- **Detailed View**: Per-service health with pod breakdown
- **Uptime Graphs**: Visual representation of uptime trends
- **Check Results**: Individual check status and messages
- **Filtering**: By namespace, health status, labels

### API Endpoints

**REST API**:
- `GET /api/v1/healthchecks` - List all HealthChecks
- `GET /api/v1/healthchecks/{namespace}/{name}` - Get specific HealthCheck
- `GET /api/v1/health-summary` - Cluster-wide summary
- `GET /api/v1/namespaces/{namespace}/health-summary` - Namespace summary

**WebSocket**:
- `WS /ws/healthchecks` - Real-time HealthCheck updates

## Documentation

- **[User Guide](docs/user-guide.md)**: Comprehensive usage documentation
- **[Developer Guide](docs/developer-guide.md)**: Architecture and development guide
- **[CRD Design](docs/crd-design.md)**: Detailed CRD specification
- **[Architecture](docs/architecture.md)**: System architecture and design decisions
- **[ADR](docs/ADR.md)**: Architectural decision records

## Examples

See [config/samples/](config/samples/) for complete examples:
- `http-healthcheck.yaml` - HTTP health check with multiple validations
- `multi-check-example.yaml` - Comprehensive examples (HTTP, TCP, gRPC, Exec)
- `external-healthcheck.yaml` - External/out-of-cluster health checks with manual endpoints

## Development

### Build from Source

```bash
# Build binaries
make build

# Run tests
make test

# Build Docker image
make docker-build IMG=your-registry/health-monitor:tag

# Push to registry
make docker-push IMG=your-registry/health-monitor:tag
```

### Local Development

Run controller locally:
```bash
make run-controller
```

Run dashboard backend locally:
```bash
make run-dashboard
```

### Frontend Development

The dashboard frontend is a React + TypeScript application located in `web/`:

```bash
cd web

# Install dependencies
npm install

# Run development server (proxies to backend on :8080)
npm run dev

# Build for production
npm run build
```

Built files are output to `web/dist/` which the dashboard server serves when running with:
```bash
./bin/dashboard --static-files=./web/dist
```

See [web/README.md](web/README.md) for more details.

### Project Structure

```
.
├── api/v1alpha1/          # CRD API definitions
├── cmd/
│   ├── controller/        # Controller binary entrypoint
│   └── dashboard/         # Dashboard binary entrypoint
├── internal/
│   ├── controller/        # HealthCheck controller
│   ├── discovery/         # Service and pod discovery
│   ├── healthcheck/       # Check executors (HTTP, TCP, gRPC, Exec)
│   ├── metrics/           # Uptime tracking
│   └── dashboard/         # Dashboard server (REST + WebSocket)
├── config/
│   ├── crd/              # CRD manifests
│   ├── rbac/             # RBAC roles and bindings
│   ├── manager/          # Controller and dashboard deployments
│   └── samples/          # Example HealthCheck CRs
└── docs/                 # Documentation
```

## Security Considerations

### Exec Checks

**⚠️ Warning**: Exec checks execute arbitrary commands in pods. Only authorized users should create HealthCheck resources.

**Best Practices**:
1. Restrict HealthCheck creation via RBAC
2. Use HTTP/TCP/gRPC checks when possible
3. Review exec commands carefully
4. Consider enabling `--allow-exec-checks=false` flag (future feature)

### RBAC

Controller requires:
- Read/write access to HealthCheck CRs
- Read access to Services, Pods, EndpointSlices
- Pod exec permissions (for Exec checks)

Dashboard requires:
- Read-only access to HealthCheck CRs
- Read access to Services

See [config/rbac/](config/rbac/) for complete RBAC configuration.

## Performance

**Scalability Targets**:
- 1,000 services with 10,000 pods
- 100,000 health checks per minute
- 100 concurrent WebSocket clients per dashboard replica

**Resource Requirements**:
- Controller: 100m CPU / 128Mi RAM (request), 1 CPU / 1Gi RAM (limit)
- Dashboard: 50m CPU / 64Mi RAM (request), 500m CPU / 512Mi RAM (limit)

## Roadmap

### v1alpha1 (Current)
- [x] Service-level health monitoring
- [x] HTTP, TCP, gRPC, Exec checks
- [x] Uptime tracking
- [x] Dashboard with WebSocket

### v1beta1 (Planned)
- [ ] Pod and Deployment targets
- [ ] Alert integrations (webhooks, Slack, PagerDuty)
- [ ] Advanced scheduling (time-based checks)
- [ ] Prometheus metrics export

### v1 (Future)
- [ ] Multi-cluster health checks
- [ ] Custom metrics export
- [ ] Distributed tracing integration
- [ ] Performance optimizations for 10k+ services

## Contributing

Contributions are welcome! Please see [docs/developer-guide.md](docs/developer-guide.md) for development setup and guidelines.

## License

Apache 2.0 License - See LICENSE file for details.

## Acknowledgments

Built using:
- [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) - Kubernetes operator framework
- [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) - Controller library
- [gorilla/websocket](https://github.com/gorilla/websocket) - WebSocket implementation
- [gRPC](https://grpc.io/) - gRPC health checks

## Contact

For questions, issues, or feedback:
- GitHub Issues: [github.com/kdwils/k8s-service-health-monitor/issues](https://github.com/kdwils/k8s-service-health-monitor/issues)
- Discussions: [github.com/kdwils/k8s-service-health-monitor/discussions](https://github.com/kdwils/k8s-service-health-monitor/discussions)
