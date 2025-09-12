package backend

import (
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBackendURL verifies that the backend stores the correct URL.
func TestBackendURL(t *testing.T) {
	u, _ := url.Parse("http://localhost:8080")
	b := NewBackend(u)
	assert.Equal(t, "http://localhost:8080", b.GetURL().String())
}

// TestBackendInitialAliveStatus verifies that a new backend starts as alive.
func TestBackendInitialAliveStatus(t *testing.T) {
	u, _ := url.Parse("http://localhost:8080")
	b := NewBackend(u)
	assert.True(t, b.IsAlive())
}

// TestBackendSetAlive verifies that SetAlive correctly updates the alive status.
func TestBackendSetAlive(t *testing.T) {
	u, _ := url.Parse("http://localhost:8080")
	b := NewBackend(u)
	b.SetAlive(false)
	assert.False(t, b.IsAlive())
	b.SetAlive(true)
	assert.True(t, b.IsAlive())
}

// TestBackendReverseProxyInitialization verifies that the reverse proxy is initialized.
func TestBackendReverseProxyInitialization(t *testing.T) {
	u, _ := url.Parse("http://localhost:8080")
	b := NewBackend(u)
	assert.IsType(t, &httputil.ReverseProxy{}, b.reverseProxy)
}
