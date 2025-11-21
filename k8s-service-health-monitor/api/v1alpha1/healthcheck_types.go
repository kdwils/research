/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HealthCheckSpec defines the desired state of HealthCheck
type HealthCheckSpec struct {
	// TargetRef references the Kubernetes resource to monitor (optional if Endpoints specified)
	// +kubebuilder:validation:Optional
	TargetRef *TargetReference `json:"targetRef,omitempty"`

	// Endpoints manually specifies target endpoints (external or override discovered)
	// When specified, these endpoints are used instead of service discovery
	// +kubebuilder:validation:Optional
	Endpoints []ManualEndpoint `json:"endpoints,omitempty"`

	// Interval specifies how often to execute health checks
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="30s"
	Interval metav1.Duration `json:"interval,omitempty"`

	// Timeout for each check execution
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="10s"
	Timeout metav1.Duration `json:"timeout,omitempty"`

	// FailureThreshold is the number of consecutive failures before marking unhealthy
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=3
	FailureThreshold int32 `json:"failureThreshold,omitempty"`

	// SuccessThreshold is the number of consecutive successes to mark healthy again
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	SuccessThreshold int32 `json:"successThreshold,omitempty"`

	// Checks is the list of health check definitions
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Checks []Check `json:"checks"`

	// Uptime tracking configuration
	// +kubebuilder:validation:Optional
	Uptime *UptimeConfig `json:"uptime,omitempty"`
}

// ManualEndpoint specifies a manual endpoint to check (external or override)
type ManualEndpoint struct {
	// Name of the endpoint (for identification)
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Address is the IP address or hostname
	// +kubebuilder:validation:Required
	Address string `json:"address"`

	// Labels for categorization and filtering
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`
}

// TargetReference identifies the target resource to monitor
type TargetReference struct {
	// Kind of the target resource (currently only Service supported)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Service
	Kind string `json:"kind"`

	// Name of the target resource
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace of the target resource (defaults to HealthCheck's namespace)
	// +kubebuilder:validation:Optional
	Namespace string `json:"namespace,omitempty"`
}

// Check defines a single health check
type Check struct {
	// Name of the check (must be unique within the HealthCheck)
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type of the health check
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=HTTP;TCP;gRPC;Exec
	Type CheckType `json:"type"`

	// HTTP check configuration
	// +kubebuilder:validation:Optional
	HTTP *HTTPCheck `json:"http,omitempty"`

	// TCP check configuration
	// +kubebuilder:validation:Optional
	TCP *TCPCheck `json:"tcp,omitempty"`

	// gRPC check configuration
	// +kubebuilder:validation:Optional
	GRPC *GRPCCheck `json:"grpc,omitempty"`

	// Exec check configuration
	// +kubebuilder:validation:Optional
	Exec *ExecCheck `json:"exec,omitempty"`
}

// CheckType is the type of health check
// +kubebuilder:validation:Enum=HTTP;TCP;gRPC;Exec
type CheckType string

const (
	CheckTypeHTTP CheckType = "HTTP"
	CheckTypeTCP  CheckType = "TCP"
	CheckTypeGRPC CheckType = "gRPC"
	CheckTypeExec CheckType = "Exec"
)

// HTTPCheck defines an HTTP-based health check
type HTTPCheck struct {
	// Scheme (HTTP or HTTPS)
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=HTTP;HTTPS
	// +kubebuilder:default="HTTP"
	Scheme string `json:"scheme,omitempty"`

	// Host to connect to (defaults to pod IP)
	// +kubebuilder:validation:Optional
	Host string `json:"host,omitempty"`

	// Port to connect to
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`

	// Path to request
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="/"
	Path string `json:"path,omitempty"`

	// HTTPHeaders to send with the request
	// +kubebuilder:validation:Optional
	HTTPHeaders []HTTPHeader `json:"httpHeaders,omitempty"`

	// ExpectedStatus codes (default: 200-399)
	// +kubebuilder:validation:Optional
	ExpectedStatus []int32 `json:"expectedStatus,omitempty"`

	// ExpectedBodyRegex pattern that must match response body
	// +kubebuilder:validation:Optional
	ExpectedBodyRegex string `json:"expectedBodyRegex,omitempty"`

	// ExpectedBodyContains substring that must appear in response body
	// +kubebuilder:validation:Optional
	ExpectedBodyContains string `json:"expectedBodyContains,omitempty"`
}

// HTTPHeader represents an HTTP header
type HTTPHeader struct {
	// Name of the header
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Value of the header
	// +kubebuilder:validation:Required
	Value string `json:"value"`
}

// TCPCheck defines a TCP connection health check
type TCPCheck struct {
	// Port to connect to
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`
}

// GRPCCheck defines a gRPC health check
type GRPCCheck struct {
	// Port to connect to
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`

	// Service name for gRPC health check
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="grpc.health.v1.Health"
	Service string `json:"service,omitempty"`

	// UseTLS indicates whether to use TLS
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	UseTLS bool `json:"useTLS,omitempty"`
}

// ExecCheck defines an exec-based health check
type ExecCheck struct {
	// Command to execute (exit code 0 = healthy)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Command []string `json:"command"`
}

// UptimeConfig configures uptime tracking
type UptimeConfig struct {
	// Enabled indicates whether uptime tracking is enabled
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Windows to track (e.g., "1h", "24h", "7d", "30d")
	// +kubebuilder:validation:Optional
	Windows []string `json:"windows,omitempty"`
}

// HealthCheckStatus defines the observed state of HealthCheck
type HealthCheckStatus struct {
	// Phase is the overall health status
	// +kubebuilder:validation:Enum=Healthy;Degraded;Unhealthy;Unknown
	Phase HealthPhase `json:"phase,omitempty"`

	// CheckResults contains results for each defined check
	// +kubebuilder:validation:Optional
	CheckResults []CheckResult `json:"checkResults,omitempty"`

	// PodHealth contains per-pod health breakdown
	// +kubebuilder:validation:Optional
	PodHealth []PodHealthStatus `json:"podHealth,omitempty"`

	// Uptime statistics for configured windows
	// +kubebuilder:validation:Optional
	Uptime []UptimeStat `json:"uptime,omitempty"`

	// ServiceEndpoints contains discovered endpoint information
	// +kubebuilder:validation:Optional
	ServiceEndpoints *EndpointInfo `json:"serviceEndpoints,omitempty"`

	// Conditions represent the latest available observations
	// +kubebuilder:validation:Optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration reflects the generation of the most recently observed HealthCheck
	// +kubebuilder:validation:Optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// HealthPhase represents the overall health status
// +kubebuilder:validation:Enum=Healthy;Degraded;Unhealthy;Unknown
type HealthPhase string

const (
	HealthPhaseHealthy   HealthPhase = "Healthy"
	HealthPhaseDegraded  HealthPhase = "Degraded"
	HealthPhaseUnhealthy HealthPhase = "Unhealthy"
	HealthPhaseUnknown   HealthPhase = "Unknown"
)

// CheckResult represents the result of a single check
type CheckResult struct {
	// Name of the check
	Name string `json:"name"`

	// Healthy indicates if the check is passing
	Healthy bool `json:"healthy"`

	// LastCheckTime is when the check was last executed
	LastCheckTime metav1.Time `json:"lastCheckTime,omitempty"`

	// LastTransitionTime is when the health status last changed
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// ConsecutiveSuccesses is the count of consecutive successful checks
	ConsecutiveSuccesses int32 `json:"consecutiveSuccesses,omitempty"`

	// ConsecutiveFailures is the count of consecutive failed checks
	ConsecutiveFailures int32 `json:"consecutiveFailures,omitempty"`

	// Message provides additional context about the check result
	Message string `json:"message,omitempty"`
}

// PodHealthStatus represents health status for a single pod or endpoint
type PodHealthStatus struct {
	// EndpointName is the name of the endpoint (pod name or manual endpoint name)
	EndpointName string `json:"endpointName"`

	// EndpointType indicates if this is a pod or manual endpoint
	// +kubebuilder:validation:Optional
	EndpointType string `json:"endpointType,omitempty"` // "pod" or "manual"

	// PodName is the name of the pod (for discovered pods)
	// +kubebuilder:validation:Optional
	PodName string `json:"podName,omitempty"`

	// PodIP is the IP address of the pod/endpoint
	PodIP string `json:"podIP,omitempty"`

	// Healthy indicates if all checks are passing for this pod
	Healthy bool `json:"healthy"`

	// ChecksHealthy is the count of passing checks
	ChecksHealthy int32 `json:"checksHealthy,omitempty"`

	// ChecksUnhealthy is the count of failing checks
	ChecksUnhealthy int32 `json:"checksUnhealthy,omitempty"`

	// LastCheckTime is when checks were last executed for this pod
	LastCheckTime metav1.Time `json:"lastCheckTime,omitempty"`
}

// UptimeStat represents uptime statistics for a time window
// +kubebuilder:validation:XValidation:rule="self.percentage >= 0 && self.percentage <= 100",message="percentage must be between 0 and 100"
type UptimeStat struct {
	// Window is the time window (e.g., "1h", "24h")
	Window string `json:"window"`

	// Percentage is the uptime percentage (0-100)
	// +kubebuilder:validation:Type=number
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=100
	Percentage float64 `json:"percentage"`

	// TotalChecks is the total number of checks in this window
	TotalChecks int64 `json:"totalChecks,omitempty"`

	// SuccessfulChecks is the number of successful checks
	SuccessfulChecks int64 `json:"successfulChecks,omitempty"`
}

// EndpointInfo contains service endpoint discovery information
type EndpointInfo struct {
	// Ready is the number of ready endpoints
	Ready int32 `json:"ready,omitempty"`

	// Total is the total number of endpoints
	Total int32 `json:"total,omitempty"`
}

// Condition types for HealthCheck
const (
	// ConditionTypeReady indicates that the HealthCheck is ready and executing
	ConditionTypeReady string = "Ready"

	// ConditionTypeDegraded indicates that some checks are failing
	ConditionTypeDegraded string = "Degraded"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=hc;healthcheck
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Target",type=string,JSONPath=`.spec.targetRef.name`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// HealthCheck is the Schema for the healthchecks API
type HealthCheck struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HealthCheckSpec   `json:"spec,omitempty"`
	Status HealthCheckStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HealthCheckList contains a list of HealthCheck
type HealthCheckList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HealthCheck `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HealthCheck{}, &HealthCheckList{})
}
