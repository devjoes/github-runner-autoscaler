package health

import (
	"net"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestLive(t *testing.T) {
	h := NewHealth(config.Config{MemcachedServers: ""})
	w := httptest.NewRecorder()
	h.Livez().ServeHTTP(w, nil)
	assert.Equal(t, 200, w.Code)
}

func TestReadyNoCache(t *testing.T) {
	h := NewHealth(config.Config{MemcachedServers: ""})
	w := httptest.NewRecorder()

	for i := 0; i < 10; i++ {
		h.Readyz().ServeHTTP(w, nil)
		assert.Equal(t, 200, w.Code)
	}
}

func TestReadyFailure(t *testing.T) {
	h := NewHealth(config.Config{MemcachedServers: "127.0.0.1:1234"})
	w := httptest.NewRecorder()
	for i := 0; i < 10; i++ {
		h.Readyz().ServeHTTP(w, nil)
		assert.NotEqual(t, 200, w.Code)
	}
}

func TestReadySuccess(t *testing.T) {
	_, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", "11211"), time.Second)
	if err != nil {
		t.Skip("Skipping test - no memcached running on 127.0.0.1:11211")
		return
	}
	h := NewHealth(config.Config{MemcachedServers: "127.0.0.1:11211"})
	w := httptest.NewRecorder()
	for i := 0; i < 10; i++ {
		h.Readyz().ServeHTTP(w, nil)
		assert.Equal(t, 200, w.Code)
	}
}

func TestReadyPartialFailure(t *testing.T) {
	_, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", "11211"), time.Second)
	if err != nil {
		t.Skip("Skipping test - no memcached running on 127.0.0.1:11211")
		return
	}
	h := NewHealth(config.Config{MemcachedServers: "127.0.0.1:4321,127.0.0.1:11211,127.0.0.1:12345,127.0.0.1:1234"})
	w := httptest.NewRecorder()
	for i := 0; i < 10; i++ {
		h.Readyz().ServeHTTP(w, nil)
		assert.Equal(t, 200, w.Code)
	}
}
