package fahapi

import (
	json2 "encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func StartWebSocketLoop(refreshTime int) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGHUP)

	u := url.URL{Scheme: "ws", Host: apiConfig.Host, Path: WebSocketPath}
	if logLevel > 0 {
		logf("connecting to %s", u.String())
	}

	header := http.Header{}
	header.Set("Authorization", apiConfig.Authentication)
	c, _, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		return err
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				logf("websocket read: %v", err)
				return
			}
			if logLevel == 3 { // debug out
				fmt.Printf("%s\n", message)
			}
			var result WebsocketMessage
			err = json2.Unmarshal(message, &result)
			if err != nil {
				logf("WS unmarshall error: %s\n", err)
			} else {
				processWebsocketMessage(result)
			}
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	ticks := 0

	for {
		select {
		case <-done:
			return nil
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				logf("ticker write: %v\n", err)
				return err
			}
			ticks++
			if ticks > refreshTime {
				ticks = 0
				// todo Maybe we should also refresh the whole UnitMap structure (re read the f@h configuration)
				treatAllUnitsAsUpdated(false) // regulary flush all units
			}
		case sig := <-interrupt:
			logf("interrupt: %v\n", sig)

			if sig.String() == "hangup" {
				treatAllUnitsAsUpdated(true)
			} else {
				// Cleanly close the connection by sending a close message and then
				// waiting (with timeout) for the server to close the connection.
				err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				if err != nil {
					logf("write close: %v\n", err)
					return err
				}
				select {
				case <-done:
				case <-time.After(time.Second):
				}
				return nil
			}
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
