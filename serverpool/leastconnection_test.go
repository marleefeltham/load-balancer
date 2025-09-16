package serverpool

import (
	"load-balancer/backend"
	"load-balancer/utils"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test selecting idle backend
func TestLeastConnection_ActiveConnections(t *testing.T) {
	sp, err := NewServerPool(utils.LeastConnected)
	require.NoError(t, err, "failed to create server pool")

	// Simulate backend with long-running request
	slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	})
	s1 := httptest.NewServer(slow)
	defer s1.Close()

	u1, err := url.Parse(s1.URL)
	require.NoError(t, err, "failed to parse url 1")
	b1 := backend.NewBackend(u1)
	sp.AddBackend(b1)

	// Simulate idle backend
	fast := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s2 := httptest.NewServer(fast)
	defer s2.Close()

	u2, err := url.Parse(s2.URL)
	require.NoError(t, err, "failed to parse url 2")
	b2 := backend.NewBackend(u2)
	sp.AddBackend(b2)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		b1.ServeHTTP(w, req)
	}()

	time.Sleep(500 * time.Millisecond)

	peer := sp.GetNextValidPeer()
	assert.Equal(t, b2.GetURL().String(), peer.GetURL().String())

	wg.Wait()
}

// Test all backends idle -> select first backend
func TestLeastConnection_AllBackendsIdle(t *testing.T) {
	sp, err := NewServerPool(utils.LeastConnected)
	require.NoError(t, err, "failed to create server pool")

	backends := []backend.Backend{}

	for i := 0; i < 4; i++ {
		u, err := url.Parse("http://127.0.0.1:808" + strconv.Itoa(i+1))
		require.NoError(t, err, "failed to parse url")
		b := backend.NewBackend(u)
		sp.AddBackend(b)
		backends = append(backends, b)
	}

	for i := 0; i < 10; i++ {
		peer := sp.GetNextValidPeer()
		assert.Equal(t, backends[0], peer)
	}
}

// Test skipping dead backend with the fewest connections
func TestLeastConnection_SkipDeadBackend(t *testing.T) {
	sp, err := NewServerPool(utils.LeastConnected)
	require.NoError(t, err, "failed to create server pool")

	u1, err := url.Parse("http://127.0.0.1:8081")
	require.NoError(t, err, "failed to parse url 1")
	b1 := backend.NewBackend(u1)
	sp.AddBackend(b1)

	u2, err := url.Parse("http://127.0.0.1:8082")
	require.NoError(t, err, "failed to parse url 2")
	b2 := backend.NewBackend(u2)
	b2.SetAlive(false)

	sp.AddBackend(b2)

	peer := sp.GetNextValidPeer()
	assert.Equal(t, b1, peer) // should skip b2
}

// Test all backends busy
func TestLeastConnection_AllBackendsBusy(t *testing.T) {
	sp, err := NewServerPool(utils.LeastConnected)
	require.NoError(t, err, "failed to create server pool")

	backends := []backend.Backend{}

	for i := 0; i < 3; i++ {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(200 * time.Millisecond) })
		s := httptest.NewServer(handler)
		defer s.Close()

		u, err := url.Parse("http://127.0.0.1:808" + strconv.Itoa(i+1))
		require.NoError(t, err, "failed to parse url")
		b := backend.NewBackend(u)
		sp.AddBackend(b)
		backends = append(backends, b)

	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	// Make all backends busy
	var wg sync.WaitGroup
	for _, b := range backends {
		wg.Add(1)
		go func(b backend.Backend) {
			defer wg.Done()
			w := httptest.NewRecorder()
			b.ServeHTTP(w, req)
		}(b)
	}

	time.Sleep(50 * time.Millisecond)

	peer := sp.GetNextValidPeer()
	require.NotNil(t, peer)

	wg.Wait()

	// Counter should be 0 after requests finish
	for _, b := range backends {
		assert.Equal(t, 0, b.GetActiveConnections())
	}
}

// Test concurrent access
func TestLeastConnection_ConcurrentAccess(t *testing.T) {
	sp, err := NewServerPool(utils.LeastConnected)
	require.NoError(t, err, "failed to create server pool")

	for i := 0; i < 5; i++ {
		u, err := url.Parse("http://127.0.0.1:808" + strconv.Itoa(i))
		require.NoError(t, err, "failed to parse url")
		sp.AddBackend(backend.NewBackend(u))
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sp.GetNextValidPeer()
		}()
	}

	wg.Wait()
}

// Test server pool properties: GetBackends() and GetServerPoolSize()
func TestLeastConnection_PoolProperties(t *testing.T) {
	sp, err := NewServerPool(utils.LeastConnected)
	require.NoError(t, err, "failed to create server pool")

	assert.Equal(t, 0, sp.GetServerPoolSize())

	u1, err := url.Parse("http://127.0.0.1:8081")
	require.NoError(t, err, "failed to parse url 1")
	b1 := backend.NewBackend(u1)
	sp.AddBackend(b1)

	u2, err := url.Parse("http://127.0.0.1:8082")
	require.NoError(t, err, "failed to parse url 2")
	b2 := backend.NewBackend(u2)

	sp.AddBackend(b2)
	backends := sp.GetBackends()

	assert.Equal(t, 2, sp.GetServerPoolSize())
	assert.Len(t, backends, 2)
	assert.Equal(t, b1, backends[0])
	assert.Equal(t, b2, backends[1])
}

// Test attempting to add duplicate url
func TestLeastConnection_AddDuplicate(t *testing.T) {
	sp, err := NewServerPool(utils.LeastConnected)
	require.NoError(t, err, "failed to create server pool")

	u, err := url.Parse("http://127.0.0.1:8081")
	require.NoError(t, err, "failed to parse url")
	b := backend.NewBackend(u)

	sp.AddBackend(b)
	sp.AddBackend(b) // should be ignored

	backends := sp.GetBackends()
	assert.Len(t, backends, 1)
}
