package fahapi

import (
	"io"
	"log"
	"testing"
	"time"
)

// quietApi points the package globals at a discarding logger. Every test needs
// it because the library keeps its configuration in package state.
func quietApi(t *testing.T) {
	t.Helper()
	logger = log.New(io.Discard, "", 0)
	logLevel = 0
	wsUpdateUnitCallback = nil
	wsUpdateMessageCallback = nil
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
