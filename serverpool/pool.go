package serverpool

import (
	"fmt"
	"load-balancer/backend"
	"load-balancer/utils"

	"go.uber.org/zap"
)

// ServerPool defines methods for managing backend servers and selecting a backend according to a load balancing strategy.
type ServerPool interface {
	GetBackends() []backend.Backend
	GetNextValidPeer() backend.Backend
	AddBackend(backend.Backend)
	GetServerPoolSize() int
}

// NewServerPool creates a new server pool using provided load balancing strategy (default round-robin).
// Returns an error if the strategy is unsupported.
func NewServerPool(strategy utils.LBStrategy) (ServerPool, error) {
	switch strategy {
	case utils.RoundRobin:
		return &roundRobinServerPool{
			backends: make([]backend.Backend, 0),
			current:  0,
		}, nil
	case utils.LeastConnected:
		return &lcServerPool{
			backends: make([]backend.Backend, 0),
		}, nil

	default:
		utils.Logger.Error("invalid server pool strategy", zap.Int("strategy", int(strategy)))
		return nil, fmt.Errorf("invalid strategy: %d", strategy)
	}
}
