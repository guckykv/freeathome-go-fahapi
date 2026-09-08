package fahapi

import (
	"io"
	"log"
	"testing"
	"time"
)

// testClient is a client that logs nowhere.
func testClient(t *testing.T) *Client {
	t.Helper()
	return New(Config{
		Host:   "sysap.invalid",
		Logger: log.New(io.Discard, "", 0),
	})
}

func out(pairingID int, value string) *InOutPut {
	return &InOutPut{PairingID: &pairingID, Value: &value}
}

func ptr(s string) *string { return &s }

// fastPings shortens the keepalive timing so a test does not have to wait out
// the production interval.
func fastPings(t *testing.T) {
	t.Helper()
	oldPing, oldPong := pingInterval, pongWait
	pingInterval, pongWait = 50*time.Millisecond, 2*time.Second
	t.Cleanup(func() { pingInterval, pongWait = oldPing, oldPong })
}
