package main

import (
	"context"
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	monitoringv1alpha1 "github.com/kdwils/k8s-service-health-monitor/api/v1alpha1"
	"github.com/kdwils/k8s-service-health-monitor/internal/dashboard"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(monitoringv1alpha1.AddToScheme(scheme))
}

func main() {
	var bindAddress string

	flag.StringVar(&bindAddress, "bind-address", ":8080", "The address the dashboard HTTP server binds to.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	setupLog.Info("setting up dashboard")

	// Get Kubernetes config
	config := ctrl.GetConfigOrDie()

	// Create client
	k8sClient, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		setupLog.Error(err, "unable to create Kubernetes client")
		os.Exit(1)
	}

	// Create cache for watching resources
	k8sCache, err := cache.New(config, cache.Options{Scheme: scheme})
	if err != nil {
		setupLog.Error(err, "unable to create cache")
		os.Exit(1)
	}

	// Start cache
	ctx := ctrl.SetupSignalHandler()
	go func() {
		if err := k8sCache.Start(ctx); err != nil {
			setupLog.Error(err, "unable to start cache")
			os.Exit(1)
		}
	}()

	// Wait for cache sync
	if !k8sCache.WaitForCacheSync(ctx) {
		setupLog.Error(nil, "failed to wait for cache sync")
		os.Exit(1)
	}

	// Create and start dashboard server
	server := dashboard.NewServer(k8sClient, k8sCache)

	setupLog.Info("starting dashboard server", "address", bindAddress)
	if err := server.Start(context.Background(), bindAddress); err != nil {
		setupLog.Error(err, "problem running dashboard server")
		os.Exit(1)
	}
}
