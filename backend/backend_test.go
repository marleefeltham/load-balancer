package backend

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBackendURL verifies that the backend stores the correct URL.
func TestBackendURL(t *testing.T) {
	u, err := url.Parse("http://127.0.0.1:8080")
	require.NoError(t, err, "failed to parse url")

	b := NewBackend(u)

	assert.Equal(t, "http://127.0.0.1:8080", b.GetURL().String())
}

// TestBackendInitialAliveStatus verifies that a new backend starts as alive.
func TestBackendInitialAliveStatus(t *testing.T) {
	u, err := url.Parse("http://127.0.0.1:8080")
	require.NoError(t, err, "failed to parse url")

	b := NewBackend(u)

	assert.True(t, b.IsAlive())
}

// TestBackendSetAlive verifies that SetAlive correctly updates the alive status.
func TestBackendSetAlive(t *testing.T) {
	u, err := url.Parse("http://127.0.0.1:8080")
	require.NoError(t, err, "failed to parse url")

	b := NewBackend(u)

	b.SetAlive(false)
	assert.False(t, b.IsAlive())

	b.SetAlive(true)
	assert.True(t, b.IsAlive())
}

// TestBackendReverseProxyInitialization verifies that the reverse proxy is initialized.
func TestBackendReverseProxyInitialization(t *testing.T) {
	u, err := url.Parse("http://127.0.0.1:8080")
	require.NoError(t, err, "failed to parse url")

	b := NewBackend(u)

	assert.IsType(t, &httputil.ReverseProxy{}, b.reverseProxy)
}

func TestRequestForwarding(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	})

	s := httptest.NewServer(handler)
	defer s.Close()

	u, err := url.Parse(s.URL)
	require.NoError(t, err, "failed to parse url")
	b := NewBackend(u)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	b.ServeHTTP(rr, req)

	res := rr.Result()
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "hello", string(body))
}

func TestBackend_ProxyErrorHandling(t *testing.T) {
	u, err := url.Parse("http://127.0.0.1:65535")
	require.NoError(t, err, "failed to parse url")
	b := NewBackend(u)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	b.ServeHTTP(rr, req)

	res := rr.Result()
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	assert.NotEqual(t, http.StatusOK, res.StatusCode)
	assert.Contains(t, string(body), "proxy error")
}

func TestBackend_ActiveConnections(t *testing.T) {
	slow := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})
	s := httptest.NewServer(slow)
	defer s.Close()

	u, err := url.Parse(s.URL)
	require.NoError(t, err, "failed to parse url")
	b := NewBackend(u)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			b.ServeHTTP(w, req)
		}()
	}

	time.Sleep(50 * time.Millisecond)

	// Active connections should be > 0 while requests are in flight
	assert.Greater(t, b.GetActiveConnections(), 0)

	wg.Wait()

	// Counter should be 0 after requests finish
	assert.Equal(t, 0, b.GetActiveConnections())
}
