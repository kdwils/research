package healthcheck

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	monitoringv1alpha1 "github.com/kdwils/k8s-service-health-monitor/api/v1alpha1"
	"github.com/kdwils/k8s-service-health-monitor/internal/discovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpchealth "google.golang.org/grpc/health/grpc_health_v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CheckResult represents the result of a health check execution
type CheckResult struct {
	Success bool
	Message string
	Error   error
}

// Executor executes health checks
type Executor struct {
	clientset  *kubernetes.Clientset
	restConfig *rest.Config
	httpClient *http.Client
}

// NewExecutor creates a new health check executor
func NewExecutor(clientset *kubernetes.Clientset, restConfig *rest.Config) *Executor {
	return &Executor{
		clientset:  clientset,
		restConfig: restConfig,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// ExecuteCheck executes a single check against a pod
func (e *Executor) ExecuteCheck(ctx context.Context, check monitoringv1alpha1.Check, pod discovery.PodInfo, timeout time.Duration) CheckResult {
	log := log.FromContext(ctx)

	// Create context with timeout
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	log.V(1).Info("Executing check",
		"checkName", check.Name,
		"checkType", check.Type,
		"pod", pod.Name,
		"podIP", pod.IP)

	var result CheckResult

	switch check.Type {
	case monitoringv1alpha1.CheckTypeHTTP:
		result = e.executeHTTPCheck(checkCtx, check.HTTP, pod)
	case monitoringv1alpha1.CheckTypeTCP:
		result = e.executeTCPCheck(checkCtx, check.TCP, pod)
	case monitoringv1alpha1.CheckTypeGRPC:
		result = e.executeGRPCCheck(checkCtx, check.GRPC, pod)
	case monitoringv1alpha1.CheckTypeExec:
		result = e.executeExecCheck(checkCtx, check.Exec, pod)
	default:
		result = CheckResult{
			Success: false,
			Message: fmt.Sprintf("Unknown check type: %s", check.Type),
			Error:   fmt.Errorf("unknown check type: %s", check.Type),
		}
	}

	return result
}

// executeHTTPCheck executes an HTTP health check
func (e *Executor) executeHTTPCheck(ctx context.Context, check *monitoringv1alpha1.HTTPCheck, pod discovery.PodInfo) CheckResult {
	if check == nil {
		return CheckResult{Success: false, Message: "HTTP check configuration is nil"}
	}

	// Build URL
	scheme := "http"
	if check.Scheme != "" {
		scheme = strings.ToLower(check.Scheme)
	}

	host := pod.IP
	if check.Host != "" {
		host = check.Host
	}

	path := check.Path
	if path == "" {
		path = "/"
	}

	url := fmt.Sprintf("%s://%s:%d%s", scheme, host, check.Port, path)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return CheckResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create request: %v", err),
			Error:   err,
		}
	}

	// Add custom headers
	for _, header := range check.HTTPHeaders {
		req.Header.Add(header.Name, header.Value)
	}

	// Execute request
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return CheckResult{
			Success: false,
			Message: fmt.Sprintf("HTTP request failed: %v", err),
			Error:   err,
		}
	}
	defer resp.Body.Close()

	// Check status code
	expectedStatuses := check.ExpectedStatus
	if len(expectedStatuses) == 0 {
		// Default: 200-399
		if resp.StatusCode < 200 || resp.StatusCode >= 400 {
			return CheckResult{
				Success: false,
				Message: fmt.Sprintf("HTTP %d: Unexpected status code", resp.StatusCode),
			}
		}
	} else {
		statusOK := false
		for _, expected := range expectedStatuses {
			if resp.StatusCode == int(expected) {
				statusOK = true
				break
			}
		}
		if !statusOK {
			return CheckResult{
				Success: false,
				Message: fmt.Sprintf("HTTP %d: Status code not in expected list", resp.StatusCode),
			}
		}
	}

	// Check body if needed
	if check.ExpectedBodyRegex != "" || check.ExpectedBodyContains != "" {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return CheckResult{
				Success: false,
				Message: fmt.Sprintf("Failed to read response body: %v", err),
				Error:   err,
			}
		}

		bodyStr := string(body)

		// Check regex pattern
		if check.ExpectedBodyRegex != "" {
			matched, err := regexp.MatchString(check.ExpectedBodyRegex, bodyStr)
			if err != nil {
				return CheckResult{
					Success: false,
					Message: fmt.Sprintf("Invalid regex pattern: %v", err),
					Error:   err,
				}
			}
			if !matched {
				return CheckResult{
					Success: false,
					Message: fmt.Sprintf("HTTP %d: Response body does not match regex pattern", resp.StatusCode),
				}
			}
		}

		// Check substring
		if check.ExpectedBodyContains != "" {
			if !strings.Contains(bodyStr, check.ExpectedBodyContains) {
				return CheckResult{
					Success: false,
					Message: fmt.Sprintf("HTTP %d: Response body does not contain expected substring", resp.StatusCode),
				}
			}
		}
	}

	return CheckResult{
		Success: true,
		Message: fmt.Sprintf("HTTP %d: OK", resp.StatusCode),
	}
}

// executeTCPCheck executes a TCP connection check
func (e *Executor) executeTCPCheck(ctx context.Context, check *monitoringv1alpha1.TCPCheck, pod discovery.PodInfo) CheckResult {
	if check == nil {
		return CheckResult{Success: false, Message: "TCP check configuration is nil"}
	}

	address := fmt.Sprintf("%s:%d", pod.IP, check.Port)

	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return CheckResult{
			Success: false,
			Message: fmt.Sprintf("TCP connection failed: %v", err),
			Error:   err,
		}
	}
	conn.Close()

	return CheckResult{
		Success: true,
		Message: "TCP connection successful",
	}
}

// executeGRPCCheck executes a gRPC health check
func (e *Executor) executeGRPCCheck(ctx context.Context, check *monitoringv1alpha1.GRPCCheck, pod discovery.PodInfo) CheckResult {
	if check == nil {
		return CheckResult{Success: false, Message: "gRPC check configuration is nil"}
	}

	target := fmt.Sprintf("%s:%d", pod.IP, check.Port)

	// Setup gRPC connection
	opts := []grpc.DialOption{
		grpc.WithBlock(),
	}

	if !check.UseTLS {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.DialContext(ctx, target, opts...)
	if err != nil {
		return CheckResult{
			Success: false,
			Message: fmt.Sprintf("gRPC connection failed: %v", err),
			Error:   err,
		}
	}
	defer conn.Close()

	// Call health check
	healthClient := grpchealth.NewHealthClient(conn)
	serviceName := check.Service
	if serviceName == "" {
		serviceName = ""
	}

	resp, err := healthClient.Check(ctx, &grpchealth.HealthCheckRequest{
		Service: serviceName,
	})

	if err != nil {
		return CheckResult{
			Success: false,
			Message: fmt.Sprintf("gRPC health check failed: %v", err),
			Error:   err,
		}
	}

	if resp.Status != grpchealth.HealthCheckResponse_SERVING {
		return CheckResult{
			Success: false,
			Message: fmt.Sprintf("gRPC health check: %s", resp.Status.String()),
		}
	}

	return CheckResult{
		Success: true,
		Message: fmt.Sprintf("gRPC health check: %s", resp.Status.String()),
	}
}

// executeExecCheck executes a command in the pod
func (e *Executor) executeExecCheck(ctx context.Context, check *monitoringv1alpha1.ExecCheck, pod discovery.PodInfo) CheckResult {
	if check == nil {
		return CheckResult{Success: false, Message: "Exec check configuration is nil"}
	}

	// Note: This requires finding the container name, which we'll assume is the first container
	// In a production implementation, this should be configurable

	req := e.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec")

	req.VersionedParams(&corev1.PodExecOptions{
		Command: check.Command,
		Stdout:  true,
		Stderr:  true,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(e.restConfig, "POST", req.URL())
	if err != nil {
		return CheckResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create executor: %v", err),
			Error:   err,
		}
	}

	// Capture output
	var stdout, stderr strings.Builder
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	if err != nil {
		return CheckResult{
			Success: false,
			Message: fmt.Sprintf("Command execution failed: %v, stderr: %s", err, stderr.String()),
			Error:   err,
		}
	}

	return CheckResult{
		Success: true,
		Message: fmt.Sprintf("Command executed successfully: %s", stdout.String()),
	}
}
