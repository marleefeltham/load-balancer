package serverpool

import (
	"fmt"
	"load-balancer/backend"
	"load-balancer/utils"

	"go.uber.org/zap"
)

// interface for a server pool
type ServerPool interface {
	GetBackends() []backend.Backend
	GetNextValidPeer() backend.Backend
	AddBackend(backend.Backend)
	GetServerPoolSize() int
}

// constructor for a server pool using config load balancing strategy (default LC)
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
