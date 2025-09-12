package serverpool

import (
	"load-balancer/backend"
	"sync"
)

// lcServerPool implements the ServerPool interface with least connections strategy.
type lcServerPool struct {
	backends []backend.Backend // slice of servers
	mux      sync.RWMutex      // RWMutex in read-heavy scenarios (lb has many reads)
}

// GetNextValidPeer returns the next alive backend server using least connections.
// Returns nil if there is no alive backend found.
func (s *lcServerPool) GetNextValidPeer() backend.Backend {
	s.mux.RLock()
	// Copy the backend slice to avoid holding the lock during selection.
	copied := make([]backend.Backend, len(s.backends))
	copy(copied, s.backends)
	s.mux.RUnlock()

	var lc backend.Backend

	// Find least connected peer
	for _, b := range copied {
		// Skip backends that are not alive
		if !b.IsAlive() {
			continue
		}
		// Set the first alive backend
		if lc == nil {
			lc = b
			continue
		}
		// Update the least connected peer
		if lc.GetActiveConnections() > b.GetActiveConnections() {
			lc = b
		}
	}

	return lc
}

// GetBackends returns a copy of all backend servers in the least connections pool.
func (s *lcServerPool) GetBackends() []backend.Backend {
	s.mux.RLock()
	defer s.mux.RUnlock()

	copied := make([]backend.Backend, len(s.backends))
	copy(copied, s.backends)

	return copied
}

// AddBackend adds new backend server to the least connections pool.
func (s *lcServerPool) AddBackend(b backend.Backend) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.backends = append(s.backends, b)
}

// GetServerPoolSize returns the current number of servers in the least connections pool.
func (s *lcServerPool) GetServerPoolSize() int {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return len(s.backends)
}
