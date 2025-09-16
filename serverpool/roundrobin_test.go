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

// Test happy path (as-is sequential rotation)
func TestRoundRobin_Rotation(t *testing.T) {
	sp, err := NewServerPool(utils.RoundRobin)
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
		assert.Equal(t, backends[(i+1)%len(backends)], peer)
	}
}

// Test empty pool
func TestRoundRobin_EmptyPool(t *testing.T) {
	sp, err := NewServerPool(utils.RoundRobin)
	require.NoError(t, err, "failed to create server pool")

	assert.Nil(t, sp.GetNextValidPeer())
}

// Test pool with single backend
func TestRoundRobin_SinglePool(t *testing.T) {
	sp, err := NewServerPool(utils.RoundRobin)
	require.NoError(t, err, "failed to create server pool")

	u1, err := url.Parse("http://127.0.0.1:8081")
	require.NoError(t, err, "failed to parse url 1")
	b := backend.NewBackend(u1)
	sp.AddBackend(b)

	peer := sp.GetNextValidPeer()

	assert.Equal(t, b, peer)
}

// Test pool with dead backend
func TestRoundRobin_SkipDeadBackend(t *testing.T) {
	sp, err := NewServerPool(utils.RoundRobin)
	if err != nil {
		t.Fatalf("failed to create server pool: %v", err)
	}

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

// Test concurrent access
func TestRoundRobin_ConcurrentAccess(t *testing.T) {
	sp, err := NewServerPool(utils.RoundRobin)
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
func TestRoundRobin_PoolProperties(t *testing.T) {
	sp, err := NewServerPool(utils.RoundRobin)
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

func TestRoundRobin_MultipleDeadBackends(t *testing.T) {
	sp, err := NewServerPool(utils.RoundRobin)
	require.NoError(t, err, "failed to create server pool")

	backends := []backend.Backend{}

	for i := 0; i < 6; i++ {
		u, err := url.Parse("http://127.0.0.1:808" + strconv.Itoa(i+1))
		require.NoError(t, err, "failed to parse url")
		b := backend.NewBackend(u)
		sp.AddBackend(b)
		backends = append(backends, b)
	}

	backends[1].SetAlive(false)
	backends[3].SetAlive(false)
	backends[4].SetAlive(false)

	for i := 0; i < 15; i++ {
		peer := sp.GetNextValidPeer()
		assert.True(t, peer.IsAlive())
		assert.NotEqual(t, backends[1], peer)
		assert.NotEqual(t, backends[3], peer)
		assert.NotEqual(t, backends[4], peer)
	}
}

func TestRoundRobin_ReviveDeadBackends(t *testing.T) {
	sp, err := NewServerPool(utils.RoundRobin)
	require.NoError(t, err, "failed to create server pool")

	backends := []backend.Backend{}

	for i := 0; i < 6; i++ {
		u, err := url.Parse("http://127.0.0.1:808" + strconv.Itoa(i+1))
		require.NoError(t, err, "failed to parse url")
		b := backend.NewBackend(u)
		sp.AddBackend(b)
		backends = append(backends, b)
	}

	backends[1].SetAlive(false)
	backends[3].SetAlive(false)
	backends[4].SetAlive(false)

	for i := 0; i < 15; i++ {
		peer := sp.GetNextValidPeer()
		assert.True(t, peer.IsAlive())
		assert.NotEqual(t, backends[1], peer)
		assert.NotEqual(t, backends[3], peer)
		assert.NotEqual(t, backends[4], peer)
	}

	backends[3].SetAlive(true)

	for i := 0; i < 15; i++ {
		peer := sp.GetNextValidPeer()
		assert.True(t, peer.IsAlive())
		assert.NotEqual(t, backends[1], peer)
		assert.NotEqual(t, backends[4], peer)
	}
}

// Test attempting to add duplicate url
func TestRoundRobin_AddDuplicate(t *testing.T) {
	sp, err := NewServerPool(utils.RoundRobin)
	require.NoError(t, err, "failed to create server pool")

	u, err := url.Parse("http://127.0.0.1:8081")
	require.NoError(t, err, "failed to parse url")
	b := backend.NewBackend(u)

	sp.AddBackend(b)
	sp.AddBackend(b) // should be ignored

	backends := sp.GetBackends()
	assert.Len(t, backends, 1)
}
