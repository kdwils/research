package discovery

import (
	"context"
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PodInfo contains information about a discovered pod
type PodInfo struct {
	Name      string
	Namespace string
	IP        string
	Ready     bool
}

// ServiceDiscovery manages service→pod mappings
type ServiceDiscovery struct {
	client client.Client
	cache  *discoveryCache
}

// discoveryCache stores service→pod mappings
type discoveryCache struct {
	mu   sync.RWMutex
	data map[string][]PodInfo // key: "namespace/serviceName"
}

// NewServiceDiscovery creates a new ServiceDiscovery instance
func NewServiceDiscovery(client client.Client) *ServiceDiscovery {
	return &ServiceDiscovery{
		client: client,
		cache: &discoveryCache{
			data: make(map[string][]PodInfo),
		},
	}
}

// GetPodsForService returns the pods backing a service
func (sd *ServiceDiscovery) GetPodsForService(ctx context.Context, namespace, serviceName string) ([]PodInfo, error) {
	log := log.FromContext(ctx)

	// Check cache first
	key := fmt.Sprintf("%s/%s", namespace, serviceName)
	sd.cache.mu.RLock()
	if pods, ok := sd.cache.data[key]; ok {
		sd.cache.mu.RUnlock()
		return pods, nil
	}
	sd.cache.mu.RUnlock()

	// Cache miss - discover pods
	pods, err := sd.discoverPods(ctx, namespace, serviceName)
	if err != nil {
		return nil, err
	}

	// Update cache
	sd.cache.mu.Lock()
	sd.cache.data[key] = pods
	sd.cache.mu.Unlock()

	log.Info("Discovered pods for service",
		"namespace", namespace,
		"service", serviceName,
		"podCount", len(pods))

	return pods, nil
}

// discoverPods discovers pods for a service using EndpointSlices
func (sd *ServiceDiscovery) discoverPods(ctx context.Context, namespace, serviceName string) ([]PodInfo, error) {
	// Get EndpointSlices for this service
	endpointSliceList := &discoveryv1.EndpointSliceList{}
	err := sd.client.List(ctx, endpointSliceList,
		client.InNamespace(namespace),
		client.MatchingLabels{"kubernetes.io/service-name": serviceName})

	if err != nil {
		return nil, fmt.Errorf("failed to list EndpointSlices: %w", err)
	}

	// Extract pod information from endpoints
	pods := make([]PodInfo, 0)
	seenPods := make(map[string]bool) // Deduplicate

	for _, slice := range endpointSliceList.Items {
		for _, endpoint := range slice.Endpoints {
			// Get pod reference
			if endpoint.TargetRef == nil || endpoint.TargetRef.Kind != "Pod" {
				continue
			}

			podKey := fmt.Sprintf("%s/%s", endpoint.TargetRef.Namespace, endpoint.TargetRef.Name)
			if seenPods[podKey] {
				continue
			}
			seenPods[podKey] = true

			// Determine if pod is ready
			ready := endpoint.Conditions.Ready != nil && *endpoint.Conditions.Ready

			// Get pod IP
			var ip string
			if len(endpoint.Addresses) > 0 {
				ip = endpoint.Addresses[0]
			}

			pods = append(pods, PodInfo{
				Name:      endpoint.TargetRef.Name,
				Namespace: endpoint.TargetRef.Namespace,
				IP:        ip,
				Ready:     ready,
			})
		}
	}

	return pods, nil
}

// InvalidateCache removes a service from the cache
func (sd *ServiceDiscovery) InvalidateCache(namespace, serviceName string) {
	key := fmt.Sprintf("%s/%s", namespace, serviceName)
	sd.cache.mu.Lock()
	delete(sd.cache.data, key)
	sd.cache.mu.Unlock()
}

// UpdateServiceCache updates the cache for a service
func (sd *ServiceDiscovery) UpdateServiceCache(ctx context.Context, namespace, serviceName string) error {
	pods, err := sd.discoverPods(ctx, namespace, serviceName)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s/%s", namespace, serviceName)
	sd.cache.mu.Lock()
	sd.cache.data[key] = pods
	sd.cache.mu.Unlock()

	return nil
}

// GetServiceInfo retrieves service metadata
func (sd *ServiceDiscovery) GetServiceInfo(ctx context.Context, namespace, serviceName string) (*corev1.Service, error) {
	service := &corev1.Service{}
	err := sd.client.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      serviceName,
	}, service)

	if err != nil {
		return nil, fmt.Errorf("failed to get service: %w", err)
	}

	return service, nil
}
