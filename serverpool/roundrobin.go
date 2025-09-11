package serverpool

import (
	"load-balancer/backend"
	"sync"
)

type roundRobinServerPool struct {
	backends []backend.Backend // slice of servers
	mux      sync.RWMutex      // RWMutex in read-heavy scenarios (lb has many reads)
	current  int
}

// find next valid peer by rotating peers and return first alive (next valid) peer
func (s *roundRobinServerPool) GetNextValidPeer() backend.Backend {
	s.mux.Lock()
	defer s.mux.Unlock()

	n := len(s.backends)
	if n == 0 {
		return nil
	}

	for i := 0; i < n; i++ {
		s.current = (s.current + 1) % n
		peer := s.backends[s.current]
		if peer.IsAlive() {
			return peer
		}
	}

	return nil
}

// returns backend servers in round robin pool
func (s *roundRobinServerPool) GetBackends() []backend.Backend {
	s.mux.RLock()
	defer s.mux.RUnlock()

	copied := make([]backend.Backend, len(s.backends))
	copy(copied, s.backends)

	return copied
}

// adds new backend server to round robin pool
func (s *roundRobinServerPool) AddBackend(b backend.Backend) {
	s.mux.Lock()
	defer s.mux.Unlock()
	s.backends = append(s.backends, b)
}

// returns number of servers in the round robin pool
func (s *roundRobinServerPool) GetServerPoolSize() int {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return len(s.backends)
}
