package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	monitoringv1alpha1 "github.com/kdwils/k8s-service-health-monitor/api/v1alpha1"
	"github.com/kdwils/k8s-service-health-monitor/internal/discovery"
	"github.com/kdwils/k8s-service-health-monitor/internal/healthcheck"
	"github.com/kdwils/k8s-service-health-monitor/internal/metrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	healthCheckFinalizer = "monitoring.k8s.io/healthcheck-finalizer"
)

// HealthCheckReconciler reconciles a HealthCheck object
type HealthCheckReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Discovery     *discovery.ServiceDiscovery
	Executor      *healthcheck.Executor
	UptimeTracker *metrics.UptimeTracker
}

// +kubebuilder:rbac:groups=monitoring.k8s.io,resources=healthchecks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.k8s.io,resources=healthchecks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=monitoring.k8s.io,resources=healthchecks/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=services;endpoints;pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=discovery.k8s.io,resources=endpointslices,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *HealthCheckReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the HealthCheck instance
	hc := &monitoringv1alpha1.HealthCheck{}
	if err := r.Get(ctx, req.NamespacedName, hc); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !hc.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, hc)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(hc, healthCheckFinalizer) {
		controllerutil.AddFinalizer(hc, healthCheckFinalizer)
		if err := r.Update(ctx, hc); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Get target service and pods
	targetNamespace := hc.Spec.TargetRef.Namespace
	if targetNamespace == "" {
		targetNamespace = hc.Namespace
	}

	pods, err := r.Discovery.GetPodsForService(ctx, targetNamespace, hc.Spec.TargetRef.Name)
	if err != nil {
		log.Error(err, "Failed to discover pods for service",
			"service", hc.Spec.TargetRef.Name,
			"namespace", targetNamespace)

		// Update status to unknown
		r.updateStatusUnknown(ctx, hc, fmt.Sprintf("Failed to discover pods: %v", err))
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if len(pods) == 0 {
		log.Info("No pods found for service",
			"service", hc.Spec.TargetRef.Name,
			"namespace", targetNamespace)

		r.updateStatusUnknown(ctx, hc, "No pods found for service")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Execute health checks
	timeout := 10 * time.Second
	if hc.Spec.Timeout.Duration > 0 {
		timeout = hc.Spec.Timeout.Duration
	}

	checkResults := r.executeHealthChecks(ctx, hc, pods, timeout)

	// Calculate uptime windows
	windows := []string{"1h", "24h", "7d", "30d"}
	if hc.Spec.Uptime != nil && len(hc.Spec.Uptime.Windows) > 0 {
		windows = hc.Spec.Uptime.Windows
	}

	// Record check results for uptime tracking
	allHealthy := true
	for _, result := range checkResults {
		success := result.Healthy
		if !success {
			allHealthy = false
		}
	}

	if hc.Spec.Uptime == nil || hc.Spec.Uptime.Enabled {
		r.UptimeTracker.RecordCheck(hc.Namespace, hc.Name, windows, allHealthy)
	}

	// Build status
	r.updateStatus(ctx, hc, checkResults, pods, windows)

	// Requeue based on interval
	interval := 30 * time.Second
	if hc.Spec.Interval.Duration > 0 {
		interval = hc.Spec.Interval.Duration
	}

	return ctrl.Result{RequeueAfter: interval}, nil
}

// executeHealthChecks executes all checks against all pods
func (r *HealthCheckReconciler) executeHealthChecks(
	ctx context.Context,
	hc *monitoringv1alpha1.HealthCheck,
	pods []discovery.PodInfo,
	timeout time.Duration,
) []monitoringv1alpha1.CheckResult {

	log := log.FromContext(ctx)

	// Map to track check results (aggregated across pods)
	checkMap := make(map[string]*monitoringv1alpha1.CheckResult)

	// Initialize check results
	for _, check := range hc.Spec.Checks {
		checkMap[check.Name] = &monitoringv1alpha1.CheckResult{
			Name:                 check.Name,
			Healthy:              true,
			ConsecutiveSuccesses: 0,
			ConsecutiveFailures:  0,
			LastCheckTime:        metav1.NewTime(time.Now()),
		}
	}

	// Execute checks for each pod
	for _, pod := range pods {
		if !pod.Ready {
			log.V(1).Info("Skipping unhealthy pod", "pod", pod.Name)
			continue
		}

		for _, check := range hc.Spec.Checks {
			result := r.Executor.ExecuteCheck(ctx, check, pod, timeout)

			checkResult := checkMap[check.Name]

			if !result.Success {
				checkResult.Healthy = false
				checkResult.ConsecutiveFailures++
				checkResult.ConsecutiveSuccesses = 0
				checkResult.Message = result.Message
			} else {
				checkResult.ConsecutiveSuccesses++
			}

			// Update transition time if status changed
			if hc.Status.CheckResults != nil {
				for _, oldResult := range hc.Status.CheckResults {
					if oldResult.Name == check.Name && oldResult.Healthy != checkResult.Healthy {
						checkResult.LastTransitionTime = metav1.NewTime(time.Now())
						break
					}
				}
			}
		}
	}

	// Apply thresholds
	failureThreshold := int32(3)
	if hc.Spec.FailureThreshold > 0 {
		failureThreshold = hc.Spec.FailureThreshold
	}

	successThreshold := int32(1)
	if hc.Spec.SuccessThreshold > 0 {
		successThreshold = hc.Spec.SuccessThreshold
	}

	results := make([]monitoringv1alpha1.CheckResult, 0, len(checkMap))
	for _, result := range checkMap {
		// Apply thresholds
		if result.ConsecutiveFailures >= failureThreshold {
			result.Healthy = false
		} else if result.ConsecutiveSuccesses >= successThreshold {
			result.Healthy = true
		}

		results = append(results, *result)
	}

	return results
}

// updateStatus updates the HealthCheck status
func (r *HealthCheckReconciler) updateStatus(
	ctx context.Context,
	hc *monitoringv1alpha1.HealthCheck,
	checkResults []monitoringv1alpha1.CheckResult,
	pods []discovery.PodInfo,
	windows []string,
) {

	// Calculate overall phase
	phase := monitoringv1alpha1.HealthPhaseHealthy
	healthyChecks := 0
	totalChecks := len(checkResults)

	for _, result := range checkResults {
		if result.Healthy {
			healthyChecks++
		}
	}

	if healthyChecks == 0 {
		phase = monitoringv1alpha1.HealthPhaseUnhealthy
	} else if healthyChecks < totalChecks {
		phase = monitoringv1alpha1.HealthPhaseDegraded
	}

	// Build pod health status
	podHealthStatuses := make([]monitoringv1alpha1.PodHealthStatus, 0, len(pods))
	for _, pod := range pods {
		podHealthStatuses = append(podHealthStatuses, monitoringv1alpha1.PodHealthStatus{
			PodName:         pod.Name,
			PodIP:           pod.IP,
			Healthy:         pod.Ready,
			ChecksHealthy:   int32(healthyChecks),
			ChecksUnhealthy: int32(totalChecks - healthyChecks),
			LastCheckTime:   metav1.NewTime(time.Now()),
		})
	}

	// Get uptime stats
	var uptimeStats []monitoringv1alpha1.UptimeStat
	if hc.Spec.Uptime == nil || hc.Spec.Uptime.Enabled {
		uptimeStats = r.UptimeTracker.GetUptimeStats(hc.Namespace, hc.Name, windows)
	}

	// Build status
	hc.Status = monitoringv1alpha1.HealthCheckStatus{
		Phase:        phase,
		CheckResults: checkResults,
		PodHealth:    podHealthStatuses,
		Uptime:       uptimeStats,
		ServiceEndpoints: &monitoringv1alpha1.EndpointInfo{
			Ready: int32(len(pods)),
			Total: int32(len(pods)),
		},
		ObservedGeneration: hc.Generation,
	}

	// Update conditions
	readyCondition := metav1.Condition{
		Type:               monitoringv1alpha1.ConditionTypeReady,
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.NewTime(time.Now()),
		Reason:             "AllChecksHealthy",
		Message:            "All health checks passing",
		ObservedGeneration: hc.Generation,
	}

	if phase != monitoringv1alpha1.HealthPhaseHealthy {
		readyCondition.Status = metav1.ConditionFalse
		readyCondition.Reason = "ChecksFailing"
		readyCondition.Message = fmt.Sprintf("%d/%d checks failing", totalChecks-healthyChecks, totalChecks)
	}

	hc.Status.Conditions = []metav1.Condition{readyCondition}

	// Update status
	if err := r.Status().Update(ctx, hc); err != nil {
		log.FromContext(ctx).Error(err, "Failed to update HealthCheck status")
	}
}

// updateStatusUnknown sets status to unknown
func (r *HealthCheckReconciler) updateStatusUnknown(ctx context.Context, hc *monitoringv1alpha1.HealthCheck, message string) {
	hc.Status.Phase = monitoringv1alpha1.HealthPhaseUnknown
	hc.Status.Conditions = []metav1.Condition{
		{
			Type:               monitoringv1alpha1.ConditionTypeReady,
			Status:             metav1.ConditionUnknown,
			LastTransitionTime: metav1.NewTime(time.Now()),
			Reason:             "DiscoveryFailed",
			Message:            message,
			ObservedGeneration: hc.Generation,
		},
	}

	if err := r.Status().Update(ctx, hc); err != nil {
		log.FromContext(ctx).Error(err, "Failed to update HealthCheck status")
	}
}

// handleDeletion handles HealthCheck deletion
func (r *HealthCheckReconciler) handleDeletion(ctx context.Context, hc *monitoringv1alpha1.HealthCheck) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(hc, healthCheckFinalizer) {
		// Cleanup
		r.UptimeTracker.RemoveHealthCheck(hc.Namespace, hc.Name)

		// Remove finalizer
		controllerutil.RemoveFinalizer(hc, healthCheckFinalizer)
		if err := r.Update(ctx, hc); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *HealthCheckReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Initialize dependencies if not set
	if r.Discovery == nil {
		r.Discovery = discovery.NewServiceDiscovery(mgr.GetClient())
	}

	if r.UptimeTracker == nil {
		r.UptimeTracker = metrics.NewUptimeTracker()
	}

	if r.Executor == nil {
		config := mgr.GetConfig()
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return fmt.Errorf("failed to create clientset: %w", err)
		}
		r.Executor = healthcheck.NewExecutor(clientset, config)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1alpha1.HealthCheck{}).
		Complete(r)
}
