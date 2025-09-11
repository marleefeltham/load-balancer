package backend

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

// interface for interacting with the backend
type Backend interface {
	SetAlive(bool) // alter backend status
	IsAlive() bool // set backend status
	GetURL() *url.URL
	GetActiveConnections() int
	// Serve(http.ResponseWriter, *http.Request)
	http.Handler
}

type backend struct {
	url          *url.URL
	alive        bool
	mux          sync.RWMutex
	connections  int
	reverseProxy *httputil.ReverseProxy
}

func (b *backend) SetAlive(alive bool) {
	b.mux.Lock()
	b.alive = alive
	b.mux.Unlock()
}

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

// client --> reverse proxy --> load balancer --> server
// forward incoming client request to teh backend's reverse proxy
// reverseProxy.ServeHTTP rewrites the request to match the destination backend server
func (b *backend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b.reverseProxy.ServeHTTP(w, r)
}

func CheckBackendHealth(ctx context.Context, b Backend) bool {
	reqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, b.GetURL().String()+"/health", nil)
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

// constructor
// changed passing reverse proxy as a parameter
func NewBackend(u *url.URL) *backend {
	proxy := httputil.NewSingleHostReverseProxy(u)
	return &backend{
		url:          u,
		alive:        true,
		mux:          sync.RWMutex{},
		connections:  0,
		reverseProxy: proxy,
	}
}
