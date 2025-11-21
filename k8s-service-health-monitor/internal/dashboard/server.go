package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	monitoringv1alpha1 "github.com/kdwils/k8s-service-health-monitor/api/v1alpha1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// Server provides the dashboard API
type Server struct {
	client client.Client
	cache  cache.Cache
	hub    *WebSocketHub
}

// NewServer creates a new dashboard server
func NewServer(client client.Client, cache cache.Cache) *Server {
	return &Server{
		client: client,
		cache:  cache,
		hub:    NewWebSocketHub(),
	}
}

// Start starts the dashboard server
func (s *Server) Start(ctx context.Context, addr string) error {
	// Start WebSocket hub
	go s.hub.Run()

	// Start watching for changes
	go s.watchHealthChecks(ctx)

	// Setup routes
	router := mux.NewRouter()

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/healthchecks", s.handleListHealthChecks).Methods("GET")
	api.HandleFunc("/healthchecks/{namespace}/{name}", s.handleGetHealthCheck).Methods("GET")
	api.HandleFunc("/namespaces/{namespace}/health-summary", s.handleNamespaceSummary).Methods("GET")
	api.HandleFunc("/health-summary", s.handleClusterSummary).Methods("GET")

	// WebSocket route
	router.HandleFunc("/ws/healthchecks", s.handleWebSocket)

	// Serve static files (frontend) - would be embedded in production
	router.PathPrefix("/").HandlerFunc(s.handleIndex)

	// Start server
	log.FromContext(ctx).Info("Starting dashboard server", "address", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	return server.ListenAndServe()
}

// handleListHealthChecks lists all HealthCheck resources
func (s *Server) handleListHealthChecks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	namespace := r.URL.Query().Get("namespace")
	labelSelector := r.URL.Query().Get("labelSelector")

	healthCheckList := &monitoringv1alpha1.HealthCheckList{}

	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}
	if labelSelector != "" {
		selector, err := labels.Parse(labelSelector)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid label selector: %v", err), http.StatusBadRequest)
			return
		}
		opts = append(opts, client.MatchingLabelsSelector{Selector: selector})
	}

	if err := s.client.List(ctx, healthCheckList, opts...); err != nil {
		http.Error(w, fmt.Sprintf("Failed to list HealthChecks: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(healthCheckList)
}

// handleGetHealthCheck gets a specific HealthCheck
func (s *Server) handleGetHealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	namespace := vars["namespace"]
	name := vars["name"]

	healthCheck := &monitoringv1alpha1.HealthCheck{}
	if err := s.client.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, healthCheck); err != nil {
		http.Error(w, fmt.Sprintf("Failed to get HealthCheck: %v", err), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(healthCheck)
}

// handleNamespaceSummary provides health summary for a namespace
func (s *Server) handleNamespaceSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	namespace := vars["namespace"]

	healthCheckList := &monitoringv1alpha1.HealthCheckList{}
	if err := s.client.List(ctx, healthCheckList, client.InNamespace(namespace)); err != nil {
		http.Error(w, fmt.Sprintf("Failed to list HealthChecks: %v", err), http.StatusInternalServerError)
		return
	}

	summary := calculateSummary(healthCheckList.Items)
	summary["namespace"] = namespace

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// handleClusterSummary provides cluster-wide health summary
func (s *Server) handleClusterSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	healthCheckList := &monitoringv1alpha1.HealthCheckList{}
	if err := s.client.List(ctx, healthCheckList); err != nil {
		http.Error(w, fmt.Sprintf("Failed to list HealthChecks: %v", err), http.StatusInternalServerError)
		return
	}

	summary := calculateSummary(healthCheckList.Items)
	summary["scope"] = "cluster"

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// calculateSummary calculates health summary statistics
func calculateSummary(healthChecks []monitoringv1alpha1.HealthCheck) map[string]interface{} {
	summary := map[string]interface{}{
		"total":     len(healthChecks),
		"healthy":   0,
		"degraded":  0,
		"unhealthy": 0,
		"unknown":   0,
	}

	for _, hc := range healthChecks {
		switch hc.Status.Phase {
		case monitoringv1alpha1.HealthPhaseHealthy:
			summary["healthy"] = summary["healthy"].(int) + 1
		case monitoringv1alpha1.HealthPhaseDegraded:
			summary["degraded"] = summary["degraded"].(int) + 1
		case monitoringv1alpha1.HealthPhaseUnhealthy:
			summary["unhealthy"] = summary["unhealthy"].(int) + 1
		default:
			summary["unknown"] = summary["unknown"].(int) + 1
		}
	}

	return summary
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Log.Error(err, "Failed to upgrade WebSocket connection")
		return
	}

	client := &WebSocketClient{
		hub:  s.hub,
		conn: conn,
		send: make(chan []byte, 256),
	}

	s.hub.register <- client

	go client.writePump()
	go client.readPump()
}

// handleIndex serves the frontend
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// In production, this would serve embedded static files
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>Service Health Monitor</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .healthy { color: green; }
        .degraded { color: orange; }
        .unhealthy { color: red; }
        .unknown { color: gray; }
    </style>
</head>
<body>
    <h1>Service Health Monitor Dashboard</h1>
    <p>API Endpoints:</p>
    <ul>
        <li>GET /api/v1/healthchecks - List all health checks</li>
        <li>GET /api/v1/healthchecks/{namespace}/{name} - Get specific health check</li>
        <li>GET /api/v1/health-summary - Cluster-wide summary</li>
        <li>WS /ws/healthchecks - WebSocket for real-time updates</li>
    </ul>
    <p>For production deployment, build a proper frontend using React or similar framework.</p>
</body>
</html>
`)
}

// watchHealthChecks watches for HealthCheck changes and broadcasts via WebSocket
func (s *Server) watchHealthChecks(ctx context.Context) {
	// Create informer for HealthChecks
	informer, err := s.cache.GetInformer(ctx, &monitoringv1alpha1.HealthCheck{})
	if err != nil {
		log.Log.Error(err, "Failed to get informer for HealthChecks")
		return
	}

	// Add event handlers
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			hc := obj.(*monitoringv1alpha1.HealthCheck)
			s.broadcastUpdate("add", hc)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			hc := newObj.(*monitoringv1alpha1.HealthCheck)
			s.broadcastUpdate("update", hc)
		},
		DeleteFunc: func(obj interface{}) {
			hc := obj.(*monitoringv1alpha1.HealthCheck)
			s.broadcastUpdate("delete", hc)
		},
	})
}

// broadcastUpdate broadcasts a HealthCheck update to all WebSocket clients
func (s *Server) broadcastUpdate(eventType string, hc *monitoringv1alpha1.HealthCheck) {
	message := map[string]interface{}{
		"type":     eventType,
		"resource": hc,
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Log.Error(err, "Failed to marshal WebSocket message")
		return
	}

	s.hub.broadcast <- data
}

// WebSocketHub manages WebSocket clients
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan []byte
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mu         sync.RWMutex
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
	}
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// WebSocketClient represents a WebSocket client
type WebSocketClient struct {
	hub  *WebSocketHub
	conn *websocket.Conn
	send chan []byte
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

// readPump reads messages from the WebSocket connection
func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// writePump writes messages to the WebSocket connection
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
