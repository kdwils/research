# Kubernetes Service Health Monitoring - Landscape Analysis

## Research Date
November 2025

## Executive Summary

This document summarizes research into existing Kubernetes health monitoring solutions, controller patterns, and architectural approaches to inform the design of an intelligent service health monitoring system.

## Existing Health Check Mechanisms

### Native Kubernetes Probes

Kubernetes provides three probe types at the **pod/container level**:

1. **Liveness Probes**: Detects deadlocks or irrecoverable states; triggers container restart
2. **Readiness Probes**: Determines if container can handle requests; controls endpoint inclusion
3. **Startup Probes**: Handles slow-starting applications before liveness takes over

**Key Limitation**: These operate at container level, not service level. No native aggregation across pods backing a service.

### Probe Implementation Methods

- **HTTP**: Status code checks (200-399 = success)
- **TCP**: Connection test to specific port
- **Exec**: Run command inside container
- **gRPC**: Native support since K8s 1.24+

## Controller Architecture Patterns

### Kubebuilder Best Practices (2025)

1. **Single Binary, Multiple Controllers**: Standard pattern uses one binary (`cmd/main.go`) with controller-runtime Manager registering multiple controllers
2. **Single Responsibility**: Each CRD kind gets its own controller for scalability and error isolation
3. **Finalizers**: Required for external cleanup (cloud resources, DNS, etc.)
4. **CEL Validation**: Standard practice in 2025 - express validation rules without admission webhooks
5. **Event-Driven**: Controllers should be idempotent and policy-aware

### Multi-Binary Architecture

While single binary is standard, separate binaries can coexist in same codebase:
- Separate `cmd/controller/main.go` and `cmd/dashboard/main.go`
- Share API types and common packages
- Deploy independently as separate Kubernetes Deployments

## Gateway/Ingress Discovery Patterns

### Kong's Gateway Discovery Pattern

Relevant pattern for service discovery:
- Controller watches Services matching specific labels/names
- Discovers Gateway instances via EndpointSlices
- Maintains real-time mapping as pods scale/restart
- Lower resource usage than sidecar pattern

### Gateway API Architecture

- **GatewayClass**: Defines controller implementation
- **Gateway**: Entry point for traffic routing
- **Routes** (HTTPRoute, etc.): Attach to Gateways, reference backend Services
- **Key Insight**: All networking resources (Ingress, Gateway, HTTPRoute) ultimately reference `kind: Service`

## Existing Health Check CRD Examples

### Argo CD Custom Health Checks

- Uses Lua scripts for custom health logic
- Checks resource `.status.conditions[]`
- Returns: "Healthy", "Degraded", "Progressing", "Suspended", "Missing", "Unknown"
- Integrates with cert-manager and other CRDs

### ClusterHealthCheck Operator Pattern

Example CRD structure:
```yaml
apiVersion: mycompany.com/v1
kind: ClusterHealthCheck
metadata:
  name: my-cluster-check
spec:
  clusterName: production
  interval: 30s
status:
  health: Healthy
  lastCheck: 2025-11-21T10:00:00Z
  conditions: []
```

## Gaps in Current Solutions

1. **No Service-Level Aggregation**: Probes monitor containers, not services
2. **No Automatic Discovery**: Manual configuration required for monitoring tools
3. **No Historical Tracking**: Probes are point-in-time; no built-in uptime metrics
4. **Limited Check Types**: Can't verify response content or business logic
5. **Separate Tooling**: Monitoring requires external systems (Prometheus, Datadog, etc.)

## Design Implications

### What We Should Adopt

1. **Finalizers**: For cleanup when HealthCheck CRs are deleted
2. **CEL Validation**: For CRD schema validation
3. **Gateway Discovery Pattern**: For discovering service→pod mappings via EndpointSlices
4. **Condition-Based Status**: Follow Kubernetes conventions for `.status.conditions[]`
5. **Multi-Binary Single Repo**: Controller and dashboard as separate binaries

### What We Should Avoid

1. **Admission Webhooks**: Use CEL validation instead where possible
2. **Single Controller for Multiple Kinds**: Maintain separation of concerns
3. **Tightly Coupled Components**: Dashboard should work independently of controller

### Unique Value Proposition

Our solution fills gaps by providing:
- **Automatic service discovery** and pod mapping
- **Service-level health** aggregation across backing pods
- **Custom healthchecks** beyond pod probes (content verification, business logic)
- **Historical uptime tracking** with configurable time windows
- **Unified dashboard** for real-time visibility
- **Networking awareness** (Ingress, Gateway API integration)

## References

- Kubebuilder Good Practices: https://book.kubebuilder.io/reference/good-practices
- Gateway API Spec: https://gateway-api.sigs.k8s.io/
- Kong Gateway Discovery: https://docs.konghq.com/kubernetes-ingress-controller/
- Argo CD Health Checks: https://argo-cd.readthedocs.io/en/latest/operator-manual/health/
