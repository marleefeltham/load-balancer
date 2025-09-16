package serverpool

import (
	"load-balancer/backend"
	"load-balancer/utils"
	"net/url"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test creating Round Robin pool
func TestRoundRobinPoolCreation(t *testing.T) {
	sp, err := NewServerPool(utils.RoundRobin)
	require.NoError(t, err, "failed to create server pool")

	assert.Equal(t, 0, sp.GetServerPoolSize(), "new pool should be empty")
	assert.Empty(t, sp.GetBackends(), "new pool should have no backends")
}

// Test creating Least Connection pool
func TestLeastConnectionPoolCreation(t *testing.T) {
	sp, err := NewServerPool(utils.LeastConnected)
	require.NoError(t, err, "failed to create server pool")

	assert.Equal(t, 0, sp.GetServerPoolSize(), "new pool should be empty")
	assert.Empty(t, sp.GetBackends(), "new pool should have no backends")
}

// Test invalid load balancing strategy
func TestNewServerPool_InvalidStrategy(t *testing.T) {
	invalid := utils.LBStrategy(999)

	sp, err := NewServerPool(invalid)

	assert.Nil(t, sp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid strategy")
}

// Test AddBackend()
func TestAddBackend(t *testing.T) {
	sp, err := NewServerPool(utils.RoundRobin)
	require.NoError(t, err, "failed to create server pool")

	url, err := url.Parse("http://127.0.0.1:8081")
	require.NoError(t, err, "failed to parse url")

	b := backend.NewBackend(url)
	sp.AddBackend(b)

	backends := sp.GetBackends()

	assert.Equal(t, 1, sp.GetServerPoolSize())
	assert.Len(t, backends, 1)
	assert.Equal(t, b.GetURL().String(), backends[0].GetURL().String())
}

// Test GetBackends()
func TestGetBackends(t *testing.T) {
	sp, err := NewServerPool(utils.RoundRobin)
	require.NoError(t, err, "failed to create server pool")

	url1, err := url.Parse("http://127.0.0.1:8081")
	require.NoError(t, err, "failed to parse url 1")
	b1 := backend.NewBackend(url1)
	sp.AddBackend(b1)

	url2, err := url.Parse("http://127.0.0.1:8082")
	require.NoError(t, err, "failed to parse url 2")
	b2 := backend.NewBackend(url2)
	sp.AddBackend(b2)

	backends := sp.GetBackends()

	// Length and order should match
	assert.Len(t, backends, 2)
	assert.Equal(t, b1.GetURL().String(), backends[0].GetURL().String())
	assert.Equal(t, b2.GetURL().String(), backends[1].GetURL().String())
}

// Test GetNextValidPeer() on empty pool
func TestGetNextValidPeer_EmptyPool(t *testing.T) {
	sp, err := NewServerPool(utils.RoundRobin)
	require.NoError(t, err, "failed to create server pool")

	assert.Nil(t, sp.GetNextValidPeer())
}

// Test GetNextValidPeer() skips dead backend
func TestGetNextValidPeer_SkipsDead(t *testing.T) {
	sp, err := NewServerPool(utils.RoundRobin)
	require.NoError(t, err, "failed to create server pool")

	url1, err := url.Parse("http://127.0.0.1:8081")
	require.NoError(t, err, "failed to parse url 1")
	b1 := backend.NewBackend(url1)
	b1.SetAlive(false)
	sp.AddBackend(b1)

	url2, err := url.Parse("http://127.0.0.1:8082")
	require.NoError(t, err, "failed to parse url 2")
	b2 := backend.NewBackend(url2)
	b2.SetAlive(true)
	sp.AddBackend(b2)

	peer := sp.GetNextValidPeer()

	assert.Equal(t, b2.GetURL().String(), peer.GetURL().String())
}

// Test GetNextValidPeer() when backend is dead
func TestGetNextValidPeer_AllDead(t *testing.T) {
	sp, err := NewServerPool(utils.RoundRobin)
	require.NoError(t, err, "failed to create server pool")

	url1, err := url.Parse("http://127.0.0.1:8081")
	require.NoError(t, err, "failed to parse url 1")
	b1 := backend.NewBackend(url1)
	b1.SetAlive(false)
	sp.AddBackend(b1)

	assert.Nil(t, sp.GetNextValidPeer())
}

// Test concurrent AddBackend() and GetNextValidPeer()
func TestConcurrentAccess(t *testing.T) {
	sp, err := NewServerPool(utils.RoundRobin)
	require.NoError(t, err, "failed to create server pool")

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			u, err := url.Parse("http://127.0.0.1:808" + strconv.Itoa(i))
			require.NoError(t, err, "failed to parse url")
			b := backend.NewBackend(u)
			sp.AddBackend(b)
		}(i)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			peer := sp.GetNextValidPeer()
			if peer != nil {
				assert.True(t, peer.IsAlive())
			}
		}()
	}
	wg.Wait()

	backends := sp.GetBackends()
	assert.NotEmpty(t, backends)

	for _, b := range backends {
		assert.True(t, b.IsAlive())
	}
}

// Test that external code does not affect internal backends slice
func TestGetBackendsImmutability(t *testing.T) {
	sp, err := NewServerPool(utils.RoundRobin)
	require.NoError(t, err, "failed to create server pool")

	url1, err := url.Parse("http://127.0.0.1:8081")
	require.NoError(t, err, "failed to parse url 1")
	b1 := backend.NewBackend(url1)
	sp.AddBackend(b1)

	url2, err := url.Parse("http://127.0.0.1:8082")
	require.NoError(t, err, "failed to parse url 2")
	b2 := backend.NewBackend(url2)
	sp.AddBackend(b2)

	backends := sp.GetBackends()
	assert.Len(t, backends, 2)

	backends[0] = nil
	backendsAfter := sp.GetBackends()

	// Length and order should match
	assert.Equal(t, b1, backendsAfter[0])
}
