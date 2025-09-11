package serverpool

import (
	"load-balancer/backend"
	"sync"
)

type lcServerPool struct {
	backends []backend.Backend
	mux      sync.RWMutex
}

func (s *lcServerPool) GetNextValidPeer() backend.Backend {
	s.mux.RLock()
	copied := make([]backend.Backend, len(s.backends))
	copy(copied, s.backends)
	s.mux.RUnlock()

	var lc backend.Backend

	// find least connected peer
	for _, b := range copied {
		// skip backends that are unalive
		if !b.IsAlive() {
			continue
		}
		// set the first alive backend
		if lc == nil {
			lc = b
			continue
		}
		// update the least connected peer
		if lc.GetActiveConnections() > b.GetActiveConnections() {
			lc = b
		}
	}

	return lc
}

func (s *lcServerPool) GetBackends() []backend.Backend {
	s.mux.RLock()
	defer s.mux.RUnlock()

	copied := make([]backend.Backend, len(s.backends))
	copy(copied, s.backends)

	return copied
}

func (s *lcServerPool) AddBackend(b backend.Backend) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.backends = append(s.backends, b)
}

func (s *lcServerPool) GetServerPoolSize() int {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return len(s.backends)
}
