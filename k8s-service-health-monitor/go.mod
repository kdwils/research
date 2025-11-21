module github.com/kdwils/k8s-service-health-monitor

go 1.22

require (
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.1
	google.golang.org/grpc v1.64.0
	k8s.io/api v0.30.0
	k8s.io/apimachinery v0.30.0
	k8s.io/client-go v0.30.0
	sigs.k8s.io/controller-runtime v0.18.0
)
