package backend

import (
	"context"
	"load-balancer/utils"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

// Backend interface defined the methods for interacting with the backend.
// Implements http.Handler to directly serve HTTP requests.
type Backend interface {
	SetAlive(bool) // alter backend status
	IsAlive() bool // set backend status
	GetURL() *url.URL
	GetActiveConnections() int
	http.Handler // allows backend to serve HTTP requests
}

// backend represents a single backend server.
type backend struct {
	url          *url.URL
	alive        bool                   // backend status
	mux          sync.RWMutex           // protect concurrent access (avoid race conditions)
	connections  int                    // number of active connections to the backend
	reverseProxy *httputil.ReverseProxy // rewrites and forwards request to the backend server
}

// SetAlive serves backend status.
func (b *backend) SetAlive(alive bool) {
	b.mux.Lock()
	b.alive = alive
	b.mux.Unlock()
}

// IsAlive checks backend status.
func (b *backend) IsAlive() bool {
	b.mux.RLock()
	alive := b.alive
	defer b.mux.RUnlock() // defer because alive may not be set
	return alive
}

func (b *backend) GetURL() *url.URL {
	return b.url
}

func (b *backend) GetActiveConnections() int {
	b.mux.RLock()
	connections := b.connections
	defer b.mux.RUnlock()
	return connections
}

// ServehTTP forwards incoming client request to the backend's reverse proxy.
// reverseProxy.ServeHTTP rewrites the request to match the destination backend server.
func (b *backend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Increment
	b.mux.Lock()
	b.connections++
	b.mux.Unlock()

	defer func() {
		// Decrement after request finishes
		b.mux.Lock()
		b.connections--
		b.mux.Unlock()
	}()

	b.reverseProxy.ServeHTTP(w, r)
}

// CheckBackendHealth sends a GET request to the backend to determine if it is reachable.
// Returns true if status is 200 and false otherwise.
func CheckBackendHealth(ctx context.Context, b Backend) bool {
	config, err := utils.GetLBConfig()
	if err != nil {
		return false
	}

	reqCtx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(config.BackendTimeout))
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, b.GetURL().String(), nil) // removed +"/health"
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (b *backend) SetErrorHandler(h func(http.ResponseWriter, *http.Request, error)) {
	b.reverseProxy.ErrorHandler = h
}

// NewBackend creates a new backend with the provided URL and initializes its reverse proxy.
func NewBackend(u *url.URL) *backend {
	proxy := httputil.NewSingleHostReverseProxy(u)

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "proxy error: "+err.Error(), http.StatusBadGateway)
	}

	return &backend{
		url:          u,
		alive:        true,
		mux:          sync.RWMutex{},
		connections:  0,
		reverseProxy: proxy,
	}
}
