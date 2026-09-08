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
func wsServer(t *testing.T, c *Client, serve func(conn *websocket.Conn)) {
	t.Helper()

	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/api/ws") {
			// The initial REST call of a test that hydrates first.
			w.WriteHeader(http.StatusNotFound)
			return
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		serve(conn)
	}))
	t.Cleanup(srv.Close)

	c.host = strings.TrimPrefix(srv.URL, "http://")
}

// A datapoint arriving over the websocket must reach the hydrated unit, and
// cancelling the context must end the loop.
func TestWebSocketLoopAppliesUpdateAndShutsDown(t *testing.T) {
	c := hydrateTestSysAP(t)

	applied := make(chan string, 4)

	wsServer(t, c, func(conn *websocket.Conn) {
		conn.WriteMessage(websocket.TextMessage,
			[]byte(`{"`+testSysApID+`":{"datapoints":{"DEV1/ch0000/odp0000":"0"}}}`))
		// Keep the connection open until the client closes it.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})

	c.unitCallback = func(keys []string) {
		for _, k := range keys {
			applied <- k
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- c.StartWebSocketLoop(ctx, 300) }()

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
	c := hydrateTestSysAP(t)

	var connections int32
	wsServer(t, c, func(conn *websocket.Conn) {
		n := atomic.AddInt32(&connections, 1)
		if n == 1 {
			// Drop the first connection immediately.
			return
		}
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- c.StartWebSocketLoop(ctx, 300) }()

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
	c := hydrateTestSysAP(t)

	fastPings(t)

	pinged := make(chan struct{}, 1)
	var once sync.Once

	wsServer(t, c, func(conn *websocket.Conn) {
		conn.SetPingHandler(func(string) error {
			once.Do(func() { close(pinged) })
			return conn.WriteMessage(websocket.PongMessage, nil)
		})
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- c.StartWebSocketLoop(ctx, 300) }()

	select {
	case <-pinged:
	case <-time.After(5 * time.Second):
		t.Fatal("no ping within the ping interval")
	}

	cancel()
	<-done
}

// Looking units up from another goroutine while the loop applies updates must be
// safe. Run with -race; this is what the accessors exist for.
//
// Note what this deliberately does not do: read the fields of a unit. The lock
// covers the maps, not the contents of a unit, which the loop keeps changing.
// Reading fields belongs in a callback. See the Client documentation.
func TestConcurrentReadersWhileUpdating(t *testing.T) {
	c := hydrateTestSysAP(t)

	wsServer(t, c, func(conn *websocket.Conn) {
		for i := 0; i < 200; i++ {
			value := "1"
			if i%2 == 0 {
				value = "0"
			}
			err := conn.WriteMessage(websocket.TextMessage,
				[]byte(`{"`+testSysApID+`":{"datapoints":{"DEV1/ch0000/odp0000":"`+value+`"}}}`))
			if err != nil {
				return
			}
		}
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- c.StartWebSocketLoop(ctx, 300) }()

	var readers sync.WaitGroup
	stop := make(chan struct{})
	for i := 0; i < 4; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				for key := range c.Units() {
					_ = key
				}
				_ = c.Unit("DEV1.ch0000")
				_ = c.Device("DEV1")
				_ = c.Configuration()
			}
		}()
	}

	time.Sleep(500 * time.Millisecond)
	close(stop)
	readers.Wait()

	cancel()
	<-done
}
