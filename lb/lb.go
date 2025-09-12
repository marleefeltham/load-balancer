package lb

import (
	"fmt"
	"load-balancer/serverpool"
	"load-balancer/utils"
	"net/http"
)

// Use contextKey for type safe context values.
type contextKey string

const RetryAttemptedKey contextKey = "retry_attempted"

// AllowRetry checks if the RetryAttemptedKey is in the request context.
// Returns true (retry is allowed) if it is not in the context.
func AllowRetry(r *http.Request) bool {
	if _, ok := r.Context().Value(RetryAttemptedKey).(bool); ok {
		return false
	}
	return true
}

// LoadBalancer interface wraos a server pool for handling HTTP requests.
type LoadBalancer interface {
	http.Handler
}

// loadBalancer implements LoadBalancer by delegating requests to a server pool.
type loadBalancer struct {
	sp serverpool.ServerPool
}

// ServeHTTP selects the next available backend server from the server pool and forwards the request.
// If there is no backend available, it responds with "service unavailable".
func (lb *loadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// pick the next server to serve
	peer := lb.sp.GetNextValidPeer()
	if peer == nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}
	peer.ServeHTTP(w, r)
}

// NewLoadBalancer constructs a load balancer with provided server pool.
// If the server pool is nil, then NewLoadBalancer will create a new server pool using round-robin strategy.
func NewLoadBalancer(sp serverpool.ServerPool) LoadBalancer {
	if sp == nil {
		pool, err := serverpool.NewServerPool(utils.GetLBStrategy("round-robin"))
		if err != nil {
			fmt.Printf("%s\n", err)
			return nil
		}
		return &loadBalancer{sp: pool}
	}

	return &loadBalancer{
		sp: sp,
	}
}
