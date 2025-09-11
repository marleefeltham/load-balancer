package lb

import (
	"load-balancer/serverpool"
	"net/http"
)

/*
load balancer interface

- create the server pool
- lb->server ->
- establish the connection to interact with the servers in the server pool
*/

// use contextKey for type safe context values
type contextKey string

const RetryAttemptedKey contextKey = "retry_attempted"

func AllowRetry(r *http.Request) bool {
	if _, ok := r.Context().Value(RetryAttemptedKey).(bool); ok {
		return false
	}
	return true
}

// server pool (state and scheduling)
type LoadBalancer interface {
	http.Handler
}

type loadBalancer struct {
	sp serverpool.ServerPool
}

func (lb *loadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// pick the next server to serve
	peer := lb.sp.GetNextValidPeer()
	if peer == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	}
	peer.ServeHTTP(w, r)
}

func NewLoadBalancer(sp serverpool.ServerPool) LoadBalancer {
	return &loadBalancer{
		sp: sp,
	}
}
