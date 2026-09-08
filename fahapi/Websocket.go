package fahapi

import (
	"context"
	json2 "encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// pingInterval is how often a ping goes out; pongWait is how long a
	// connection may stay silent before it counts as dead. pongWait must be
	// comfortably larger than pingInterval.
	pingInterval = 20 * time.Second
	pongWait     = 60 * time.Second
	writeWait    = 10 * time.Second

	// DefaultDialer would wait 45s for a handshake, which delays noticing a
	// blackholed host far longer than necessary.
	handshakeTimeout = 10 * time.Second

	reconnectMin = 1 * time.Second
	reconnectMax = 60 * time.Second
	// A connection that survived this long is treated as healthy, so the
	// backoff starts over after it drops.
	connectionStable = 2 * time.Minute
)

var wsDialer = &websocket.Dialer{
	Proxy:            http.ProxyFromEnvironment,
	HandshakeTimeout: handshakeTimeout,
}

// StartWebSocketLoop keeps a websocket connection to the SysAP open and applies
// every update to the Unit model. It reconnects with a backoff until ctx is
// cancelled, and returns nil on that cancellation.
//
// refreshTime is the interval in seconds at which all units are reported as
// updated even when nothing changed.
func StartWebSocketLoop(ctx context.Context, refreshTime int) error {
	backoff := reconnectMin

	for {
		start := time.Now()
		err := runWebSocket(ctx, refreshTime)

		if ctx.Err() != nil {
			return nil
		}
		if time.Since(start) >= connectionStable {
			backoff = reconnectMin
		}

		logf("websocket connection lost (%v), reconnecting in %s\n", err, backoff)

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}

		if backoff *= 2; backoff > reconnectMax {
			backoff = reconnectMax
		}
	}
}

// runWebSocket serves one connection and returns as soon as it fails.
//
// The reader goroutine only receives frames; every change to FreeDevices and
// UnitMap happens in the loop below. That keeps the model in a single
// goroutine, which the previous version did not: it applied updates from the
// reader while the ticker path walked the same maps.
func runWebSocket(ctx context.Context, refreshTime int) error {
	u := url.URL{Scheme: "ws", Host: apiConfig.Host, Path: WebSocketPath}
	if logLevel > 0 {
		logf("connecting to %s\n", u.String())
	}

	header := http.Header{}
	header.Set("Authorization", apiConfig.Authentication)

	c, _, err := wsDialer.DialContext(ctx, u.String(), header)
	if err != nil {
		return err
	}
	defer c.Close()

	// Without a read deadline a half-open connection -- an access point reboot,
	// a dropped wifi link -- blocks ReadMessage forever: the process keeps
	// running but never receives another update. Every pong pushes it out.
	if err := c.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		return err
	}
	c.SetPongHandler(func(string) error {
		return c.SetReadDeadline(time.Now().Add(pongWait))
	})

	messages := make(chan []byte, 16)
	readErr := make(chan error, 1)

	go func() {
		defer close(messages)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				readErr <- err
				return
			}
			select {
			case messages <- message:
			case <-ctx.Done():
				return
			}
		}
	}()

	ping := time.NewTicker(pingInterval)
	defer ping.Stop()

	refresh := time.NewTicker(time.Duration(refreshTime) * time.Second)
	defer refresh.Stop()

	for {
		select {
		case <-ctx.Done():
			// Close politely and give the peer a moment to answer.
			_ = c.SetWriteDeadline(time.Now().Add(writeWait))
			_ = c.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			select {
			case <-readErr:
			case <-time.After(time.Second):
			}
			return nil

		case err := <-readErr:
			return err

		case message := <-messages:
			if logLevel == 3 {
				logf("%s\n", message)
			}
			var result WebsocketMessage
			if err := json2.Unmarshal(message, &result); err != nil {
				logf("WS unmarshall error: %s\n", err)
				continue
			}
			processWebsocketMessage(result)

		case <-ping.C:
			_ = c.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.WriteMessage(websocket.PingMessage, nil); err != nil {
				return fmt.Errorf("ping: %w", err)
			}

		case <-refresh.C:
			treatAllUnitsAsUpdated(false)
		}
	}
}

func processWebsocketMessage(message WebsocketMessage) {
	if wsUpdateMessageCallback != nil {
		wsUpdateMessageCallback(message) // tell someone about the new message
	}

	changedKeys := updateDevices(message)
	if len(changedKeys) > 0 {
		handleUpdatedUnits(changedKeys, logLevel > 0)
	}
}

func updateDevices(message WebsocketMessage) []string {
	changedMap := make(map[string]bool)

	for updDatapoint, updValue := range message.ZeroSysAp.Datapoints {
		split := strings.Split(updDatapoint, "/")
		if len(split) != 3 {
			// One malformed key must not take the process down; skip it.
			logf("warning: [updateDevices] illegal datapoint format %q, skipped\n", updDatapoint)
			continue
		}
		deviceId := split[0]
		channelId := split[1]
		outDatapointId := split[2]

		var device *Device
		var channel *Channel
		var outPoint *InOutPut
		var ok bool

		if device, ok = FreeDevices[deviceId]; !ok {
			var err error
			if device, err = addNewDevice(deviceId); err != nil {
				logf("error: [updateDevices] No device %s found and failed to load it: %s\n", deviceId, err)
			}
			continue
		}
		if channel, ok = device.Channels[channelId]; !ok {
			if logLevel > 1 {
				logf("warning: [updateDevices] No channel %s for device %s\n", channelId, deviceId)
			}
			continue
		}
		if outPoint, ok = channel.Outputs[outDatapointId]; !ok {
			if logLevel > 1 {
				logf("warning: [updateDevices] No out datapoint %s for device %s and channel %s\n", outDatapointId, deviceId, channelId)
			}
			continue
		}

		// 1) update the value in our device data structure
		updateDeviceDatapoint(outPoint, updValue)

		// 2) update the corresponding unit data structures
		key, changed := reHydrateUnitValue(deviceId, channelId, outPoint)

		if changed {
			changedMap[key] = true
		}
	}

	// unique list of all changed device.channel combinations
	changedKeys := make([]string, 0, len(changedMap))
	for k := range changedMap {
		changedKeys = append(changedKeys, k)
	}

	return changedKeys
}

func updateDeviceDatapoint(data *InOutPut, updValue string) {
	data.Value = &updValue
}

// new device is added to the system - add it to our Device and our Unit list
func addNewDevice(deviceId string) (device *Device, err error) {
	if device, err = GetDevice("00000000-0000-0000-0000-000000000000", deviceId); err != nil {
		return
	}

	FreeDevices[deviceId] = device
	newUnitKeys := hydrateDevice(deviceId, device)

	if logLevel > 0 {
		virtual := ""
		if device.NativeId != nil {
			virtual = fmt.Sprintf("virtual [%s] ", *device.NativeId)
		}
		logf("Add new %sdevice %s (resulting in %d new Units)\n", virtual, deviceId, len(newUnitKeys))
		for _, key := range newUnitKeys {
			logf("%s\n", UnitMap[key].String())
		}
	}

	return
}
