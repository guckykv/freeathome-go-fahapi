package fahapi

import (
	"context"
	json2 "encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

// StartWebSocketLoop öffnet einen Websocket, startet eine go-Routine zum lesen
// von Events und aktualisiert in einer Schleife alle Devices.
// Es wird ein Websocket ohne TLS geöffnet mit Basic Auth.
// Die 
func StartWebSocketLoop(refreshTime int) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGHUP)

	// 1. URL und Header vorbereiten
	u := url.URL{Scheme: "ws", Host: apiConfig.Host, Path: WebSocketPath}
	if logLevel > 0 {
		logger.Printf("connecting to %s", u.String())
	}

	header := http.Header{}
	header.Set("Authorization", apiConfig.Authentication)
	// Origin beibehalten, da es oft bei lokalen APIs benötigt wird
	header.Set("Origin", "http://"+strings.Split(apiConfig.Host, ":")[0])

	// 2. Lokalen Dialer erstellen und konfigurieren
	// Dies vermeidet Race Conditions durch die Modifikation des DefaultDialer
	dialer := &websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		// Sub-Protokoll hinzufügen (oftmals 'fhapi' bei free@home)
		Subprotocols: []string{"fhapi"},
	}

	// 3. Verbindung aufbauen
	// Verwenden Sie DialContext, um den Kontext für Timeouts zu nutzen
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	c, _, err := dialer.DialContext(ctx, u.String(), header)

	if err != nil {
		return fmt.Errorf("WebSocket Dial failed: %w", err)
	}
	defer c.Close()

	// 4. Ersten Ping (Keep-Alive) sofort senden
	// Dies verhindert Timeouts, falls der Server sofortigen Heartbeat erwartet
	err = c.WriteMessage(websocket.PingMessage, nil)
	if err != nil {
		logger.Println("initial write ping failed:", err)
		return err
	}

	done := make(chan struct{})

	// Go-Routine für das Lesen von Nachrichten
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				logger.Printf("read error: %v", err)
				return
			}

			// ... (Restliche Verarbeitungslogik)
			if logLevel == 3 { // debug out
				fmt.Printf("%s\n", message)
			}
			var result WebsocketMessage
			err = json2.Unmarshal(message, &result)
			if err != nil {
				logger.Printf("WS unmarshall error: %s\n", err)
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
		case <-ticker.C:
			// 5. Ping-Nachricht als Keep-Alive senden
			// Dies ist der korrekte Weg, den WebSocket-Heartbeat zu implementieren
			err := c.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				logger.Println("ticker write failed:", err)
				return err
			}
			ticks++
			if ticks > refreshTime {
				ticks = 0
				treatAllUnitsAsUpdated(false) // regulary flush all units
			}
		case sig := <-interrupt:
			logger.Println("interrupt", sig)

			if sig.String() == "hangup" {
				treatAllUnitsAsUpdated(true)
			} else {
				// Sauberes Schliessen
				err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				if err != nil {
					logger.Println("write close:", err)
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
	logger.Println("[processWebsocketMessage]") // roac
	if wsUpdateMessageCallback != nil {
		logger.Println("[wsUpdateMessageCallback]") // roac
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
			logger.Fatalf("illegal message %x: illegal datapoint format %s", message, updDatapoint)
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
				logger.Printf("error: [updateDevices] No device %s found and failed to load it: %s\n", deviceId, err)
			}
			continue
		}
		if channel, ok = device.Channels[channelId]; !ok {
			if logLevel > 1 {
				logger.Printf("warning: [updateDevices] No channel %s for device %s\n", channelId, deviceId)
			}
			continue
		}
		if outPoint, ok = channel.Outputs[outDatapointId]; !ok {
			if logLevel > 1 {
				logger.Printf("warning: [updateDevices] No out datapoint %s for device %s and channel %s\n", outDatapointId, deviceId, channelId)
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
		logger.Printf("Add new %sdevice %s (resulting in %d new Units)\n", virtual, deviceId, len(newUnitKeys))
		for _, key := range newUnitKeys {
			logger.Println(UnitMap[key].String())
		}
	}

	return
}
