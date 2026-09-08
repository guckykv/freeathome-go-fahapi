package fahapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// wsServer upgrades every request and hands the connection to serve.
func wsServer(t *testing.T, serve func(c *websocket.Conn)) *httptest.Server {
	t.Helper()

	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/api/ws") {
			// The initial REST call of a test that hydrates first.
			w.WriteHeader(http.StatusNotFound)
			return
		}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		serve(c)
	}))
	t.Cleanup(srv.Close)

	ConfigureApi(strings.TrimPrefix(srv.URL, "http://"), "user", "secret", nil, nil, logger, 0)
	return srv
}

// A datapoint arriving over the websocket must reach the hydrated unit, and
// cancelling the context must end the loop.
func TestWebSocketLoopAppliesUpdateAndShutsDown(t *testing.T) {
	hydrateTestSysAP(t)

	applied := make(chan string, 4)

	wsServer(t, func(c *websocket.Conn) {
		c.WriteMessage(websocket.TextMessage,
			[]byte(`{"`+testSysApID+`":{"datapoints":{"DEV1/ch0000/odp0000":"0"}}}`))
		// Keep the connection open until the client closes it.
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	})

	// wsServer calls ConfigureApi, which resets the callbacks.
	wsUpdateUnitCallback = func(keys []string) {
		for _, k := range keys {
			applied <- k
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- StartWebSocketLoop(ctx, 300) }()

	select {
	case key := <-applied:
		if key != "DEV1.ch0000" {
			t.Errorf("updated %q, want DEV1.ch0000", key)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no update arrived within 5s")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("StartWebSocketLoop returned %v, want nil on cancellation", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("loop did not return after the context was cancelled")
	}
}

// A connection that drops must be re-established.
func TestWebSocketLoopReconnects(t *testing.T) {
	hydrateTestSysAP(t)

	var connections int32
	wsServer(t, func(c *websocket.Conn) {
		n := atomic.AddInt32(&connections, 1)
		if n == 1 {
			// Drop the first connection immediately.
			return
		}
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- StartWebSocketLoop(ctx, 300) }()

	deadline := time.After(10 * time.Second)
	for atomic.LoadInt32(&connections) < 2 {
		select {
		case <-deadline:
			t.Fatalf("only %d connection(s), expected a reconnect", atomic.LoadInt32(&connections))
		case <-time.After(50 * time.Millisecond):
		}
	}

	cancel()
	<-done
}

// The library must ping, so a silent connection can be detected at all.
func TestWebSocketLoopSendsPings(t *testing.T) {
	hydrateTestSysAP(t)

	fastPings(t)

	pinged := make(chan struct{}, 1)
	var once sync.Once

	wsServer(t, func(c *websocket.Conn) {
		c.SetPingHandler(func(string) error {
			once.Do(func() { close(pinged) })
			return c.WriteMessage(websocket.PongMessage, nil)
		})
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- StartWebSocketLoop(ctx, 300) }()

	select {
	case <-pinged:
	case <-time.After(5 * time.Second):
		t.Fatal("no ping within the ping interval")
	}

	cancel()
	<-done
}
