package serverpool

import (
	"load-balancer/backend"
	"load-balancer/utils"
	"net/url"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPoolCreation(t *testing.T) {
	sp, _ := NewServerPool(utils.RoundRobin)
	url, _ := url.Parse("http://localhost:8080")
	b := backend.NewBackend(url)
	sp.AddBackend(b)

	assert.Equal(t, 1, sp.GetServerPoolSize())
}

func TestNextIndexIteration(t *testing.T) {
	sp, _ := NewServerPool(utils.RoundRobin)
	url, _ := url.Parse("http://localhost:8080")
	b := backend.NewBackend(url)
	sp.AddBackend(b)

	url, _ = url.Parse("http://localhost:8081")
	b2 := backend.NewBackend(url)
	sp.AddBackend(b2)

	url, _ = url.Parse("http://localhost:8082")
	b3 := backend.NewBackend(url)
	sp.AddBackend(b3)

	url, _ = url.Parse("http://localhost:8083")
	b4 := backend.NewBackend(url)
	sp.AddBackend(b4)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 4; i++ {
			sp.GetNextValidPeer()
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 3; i++ {
			sp.GetNextValidPeer()
		}
	}()

	wg.Wait()
	assert.Equal(t, b.GetURL().String(), sp.GetNextValidPeer().GetURL().String())
}
