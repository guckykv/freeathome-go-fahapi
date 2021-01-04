package fahapi

import (
	"bytes"
	"encoding/base64"
	json2 "encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// see https://developer.eu.mybuildings.abb.com/fah_local/reference/functionids/
type FunctionIdType string

const (
	FID_SWITCH_SENSOR                                  FunctionIdType = "0"
	FID_DIMMING_SENSOR                                 FunctionIdType = "1"
	FID_SWITCH_ACTUATOR                                FunctionIdType = "7"
	FID_DIMMING_ACTUATOR                               FunctionIdType = "12"
	FID_WINDOW_DOOR_SENSOR                             FunctionIdType = "f"
	FID_ROOM_TEMPERATURE_CONTROLLER_MASTER_WITHOUT_FAN FunctionIdType = "23"
	FID_BRIGHTNESS_SENSOR                              FunctionIdType = "41"
	FID_RAIN_SENSOR                                    FunctionIdType = "42"
	FID_TEMPERATURE_SENSOR                             FunctionIdType = "43"
	FID_WIND_SENSOR                                    FunctionIdType = "44"
)

type ApiRestConfigurationGet200ApplicationJsonResponse struct {
	ZeroSysAp *SysAP `json:"00000000-0000-0000-0000-000000000000"`
}

type ApiRestDatapointSysapSerialGet200ApplicationJsonResponse struct {
	ZeroSysAp struct {
		Values []string `json:"values,omitempty"`
	} `json:"00000000-0000-0000-0000-000000000000"`
}

type ApiRestDatapointSysapSerialPut200TextPlainResponse map[string]struct{ Result string }

type ApiRestDeviceSysapDeviceGet200ApplicationJsonResponse struct {
	ZeroSysAp *Devices `json:"00000000-0000-0000-0000-000000000000"`
}

// Channel defines model for Channel.
type Channel struct {
	DisplayName *string              `json:"displayName,omitempty"`
	Type        *string              `json:"type,omitempty"`
	FunctionID  *string              `json:"functionID,omitempty"` // FunctionIdType
	Inputs      map[string]*InOutPut `json:"inputs,omitempty"`
	Outputs     map[string]*InOutPut `json:"outputs,omitempty"`
	Floor       *string              `json:"floor,omitempty"`
	Room        *string              `json:"room,omitempty"`
}

type Device struct {
	DisplayName  *string `json:"displayName,omitempty"`
	Floor        *string `json:"floor,omitempty"`
	Room         *string `json:"room,omitempty"`
	Interface    *string `json:"interface,omitempty"`
	NativeId     *string `json:"nativeId,omitempty"`
	unresponsive *bool
	Channels     map[string]*Channel `json:"channels,omitempty"`
}

type Devicelist struct {
	AdditionalProperties []string `json:"00000000-0000-0000-0000-000000000000"`
}

type Devices struct {
	Devices map[string]*Device
}

// Error defines model for Error.
type Error struct {
	Code   *string `json:"code,omitempty"`
	Detail *string `json:"detail,omitempty"`
	Title  *string `json:"title,omitempty"`
}

type Rooms struct {
	Name *string `json:"name,omitempty"`
}

type Floors struct {
	Name  *string           `json:"name,omitempty"`
	Rooms map[string]*Rooms `json:"rooms,omitempty"`
}

// InOutPut defines model for InOutPut.
type InOutPut struct {
	PairingID *int    `json:"pairingID,omitempty"`
	Value     *string `json:"value,omitempty"`
}

// Users defines model for Users.
type Users struct {
	AdditionalProperties map[string]struct {
		Enabled              *bool     `json:"enabled,omitempty"`
		Flags                *[]string `json:"flags,omitempty"`
		GrantedPermissions   *[]string `json:"grantedPermissions,omitempty"`
		Jid                  *string   `json:"jid,omitempty"`
		Name                 *string   `json:"name,omitempty"`
		RequestedPermissions *[]string `json:"requestedPermissions,omitempty"`
		Role                 *string   `json:"role,omitempty"`
	} `json:"-"`
}

type SysAP struct {
	Devices   map[string]*Device `json:"devices,omitempty"`
	Error     *Error             `json:"error"`
	Floorplan struct {
		Floors map[string]*Floors `json:"floors,omitempty"`
	} `json:"floorplan,omitempty"`
	SysapName *string `json:"sysapName,omitempty"`
	Users     *Users  `json:"users,omitempty"`
}

type WebsocketMessage struct {
	ZeroSysAp struct {
		Datapoints      map[string]string      `json:"datapoints"`
		Devices         map[string]*Devices    `json:"devices"`
		DevicesAdded    []string               `json:"devicesAdded"`
		DevicesRemoved  []string               `json:"devicesRemoved"`
		ScenesTriggered map[string]interface{} `json:"scenesTriggered"`
	} `json:"00000000-0000-0000-0000-000000000000"`
}

/*
// VirtualDevice defines model for VirtualDevice.
type VirtualDevice struct {
	Properties *struct {
		Displayname *string `json:"displayname,omitempty"`
		Ttl         *string `json:"ttl,omitempty"`
	} `json:"properties,omitempty"`
	Type *VirtualDeviceType `json:"type,omitempty"`
}

// VirtualDeviceType defines model for VirtualDeviceType.
type VirtualDeviceType string

// List of VirtualDeviceType
const (
	VirtualDeviceType_BinarySensor              VirtualDeviceType = "BinarySensor"
	VirtualDeviceType_CODetector                VirtualDeviceType = "CODetector"
	VirtualDeviceType_CeilingFanActuator        VirtualDeviceType = "CeilingFanActuator"
	VirtualDeviceType_DimActuator               VirtualDeviceType = "DimActuator"
	VirtualDeviceType_FireDetector              VirtualDeviceType = "FireDetector"
	VirtualDeviceType_RTC                       VirtualDeviceType = "RTC"
	VirtualDeviceType_ShutterActuator           VirtualDeviceType = "ShutterActuator"
	VirtualDeviceType_SwitchingActuator         VirtualDeviceType = "SwitchingActuator"
	VirtualDeviceType_WeatherStation            VirtualDeviceType = "WeatherStation"
	VirtualDeviceType_Weather_BrightnessSensor  VirtualDeviceType = "Weather-BrightnessSensor"
	VirtualDeviceType_Weather_RainSensor        VirtualDeviceType = "Weather-RainSensor"
	VirtualDeviceType_Weather_TemperatureSensor VirtualDeviceType = "Weather-TemperatureSensor"
	VirtualDeviceType_Weather_WindSensor        VirtualDeviceType = "Weather-WindSensor"
	VirtualDeviceType_WindowActuator            VirtualDeviceType = "WindowActuator"
	VirtualDeviceType_WindowSensor              VirtualDeviceType = "WindowSensor"
)

// VirtualDevicesSuccess defines model for VirtualDevicesSuccess.
type VirtualDevicesSuccess struct {
	AdditionalProperties map[string]struct {
		Devices *VirtualDevicesSuccess_Devices `json:"devices,omitempty"`
	} `json:"-"`
}
*/

// ===============================================================================================

const ApiPathPrefix string = "/fhapi/v1"
const WebSocketPath string = "/fhapi/v1/api/ws"

type WebsocketUpdateCallbackFunc func(unitKeys []string)

var wsUpdateCallback WebsocketUpdateCallbackFunc

var FreeDevices map[string]*Device
var SysAPConfiguration *SysAP
var logger *log.Logger
var logLevel int

type apiConfiguration struct {
	Host           string
	Authentication string
}

var apiConfig = apiConfiguration{}

func ConfigureApi(host string, username string, password string, callback WebsocketUpdateCallbackFunc, loggerParam *log.Logger, logLevelParam int) {
	apiConfig.Host = host
	apiConfig.Authentication = "Basic: " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
	wsUpdateCallback = callback
	logger = loggerParam
	logLevel = logLevelParam
}

func ReadAndHydradteAllDevices() {
	configResult, err := GetConfiguration()
	if err != nil {
		logger.Fatalf("can't initialize f@h api: %s", err)
	}

	SysAPConfiguration = configResult
	FreeDevices = configResult.Devices

	hydrateAllDevices(FreeDevices)
}

func GetDeviceList() (*Devicelist, error) {
	httpUrl := fmt.Sprintf("http://%s%s%s", apiConfig.Host, ApiPathPrefix, "/api/rest/devicelist")
	json, err := loadUrl(httpUrl)
	if err != nil {
		return nil, err
	}
	var result Devicelist
	err = json2.Unmarshal(json, &result)
	return &result, err
}

func GetDevice(sysap string, deviceId string) (*Device, error) {
	httpUrl := fmt.Sprintf("http://%s%s%s/%s/%s", apiConfig.Host, ApiPathPrefix, "/api/rest/device", sysap, deviceId)
	json, err := loadUrl(httpUrl)
	if err != nil {
		return nil, err
	}
	var result ApiRestDeviceSysapDeviceGet200ApplicationJsonResponse
	err = json2.Unmarshal(json, &result)
	if err != nil {
		return nil, err
	}
	device := result.ZeroSysAp.Devices[deviceId]
	return device, err
}

func GetDatapoint(sysap string, deviceId string, channelId string, datapointId string) (string, error) {
	httpUrl := fmt.Sprintf("http://%s%s%s/%s/%s.%s.%s", apiConfig.Host, ApiPathPrefix, "/api/rest/datapoint", sysap, deviceId, channelId, datapointId)
	json, err := loadUrl(httpUrl)
	if err != nil {
		return "", err
	}
	var result ApiRestDatapointSysapSerialGet200ApplicationJsonResponse
	err = json2.Unmarshal(json, &result)
	point := result.ZeroSysAp.Values[0]
	return point, err
}

func PutDatapoint(sysap string, deviceId string, channelId string, datapointId string, value string) (bool, error) {
	httpUrl := fmt.Sprintf("http://%s%s%s/%s/%s.%s.%s", apiConfig.Host, ApiPathPrefix, "/api/rest/datapoint", sysap, deviceId, channelId, datapointId)

	var err error
	var bstr, body []byte
	bstr = []byte(value)

	if body, err = putRequest(httpUrl, bytes.NewBuffer(bstr)); err != nil {
		return false, err
	}

	var result ApiRestDatapointSysapSerialPut200TextPlainResponse
	if err = json2.Unmarshal(body, &result); err != nil {
		return false, err
	}
	ok := result[sysap].Result == "OK"
	return ok, nil
}

func GetConfiguration() (*SysAP, error) {
	httpUrl := fmt.Sprintf("http://%s%s%s", apiConfig.Host, ApiPathPrefix, "/api/rest/configuration")
	json, err := loadUrl(httpUrl)
	if err != nil {
		return nil, err
	}

	var result ApiRestConfigurationGet200ApplicationJsonResponse
	err = json2.Unmarshal(json, &result)
	if err != nil {
		return nil, err
	}

	return result.ZeroSysAp, err
}

func loadUrl(httpUrl string) ([]byte, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", httpUrl, nil)
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", apiConfig.Authentication)

	if logLevel > 1 {
		logger.Printf("getting %s ...\n", httpUrl)
	}

	var json []byte

	response, err := client.Do(req)
	if err != nil {
		logger.Printf("error getting %s: %s\n", httpUrl, err.Error())
		return nil, err
	}
	if response.StatusCode != 200 {
		return nil, fmt.Errorf("url %s returned code %d (%s)", httpUrl, response.StatusCode, response.Status)
	}

	json, err = ioutil.ReadAll(response.Body)

	return json, err
}

func putRequest(url string, data io.Reader) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPut, url, data)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", apiConfig.Authentication)
	var response *http.Response
	response, err = client.Do(req)
	if err != nil {
		return nil, err
	}

	var body []byte
	body, err = ioutil.ReadAll(response.Body)

	return body, err
}

func GetFloorRoom(device *Device, channel *Channel) (string, string) {
	var floor, room string
	var floorId, roomId string

	if channel.Floor != nil {
		floorId = *channel.Floor
	} else {
		if device.Floor != nil {
			floorId = *device.Floor
		} else {
			return "", ""
		}
	}
	if channel.Room != nil {
		roomId = *channel.Room
	} else {
		roomId = *device.Room
	}

	if floorObject, ok := SysAPConfiguration.Floorplan.Floors[floorId]; ok {
		floor = *floorObject.Name
		if roomObject, ok := floorObject.Rooms[roomId]; ok {
			room = *roomObject.Name
		} else {
			room = "-"
		}
	}

	return floor, room
}

func StartWebSocketLoop(refreshTime int) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGHUP)

	u := url.URL{Scheme: "ws", Host: apiConfig.Host, Path: WebSocketPath}
	if logLevel > 0 {
		logger.Printf("connecting to %s", u.String())
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
				logger.Printf("read:", err)
				return
			}
			//fmt.Printf("WS message: \n%s\n", message)
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
		case t := <-ticker.C:
			err := c.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				logger.Println("ticker write:", err)
				return err
			}
			ticks++
			if ticks > refreshTime {
				ticks = 0
				// todo Maybe we should also refresh the whole UnitMap structure (re read the f@h configuration)
				treatAllUnitsAsUpdated(false) // regulary flush all units
			}
		case sig := <-interrupt:
			logger.Println("interrupt", sig)

			if sig.String() == "hangup" {
				treatAllUnitsAsUpdated(true)
			} else {
				// Cleanly close the connection by sending a close message and then
				// waiting (with timeout) for the server to close the connection.
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
			//fmt.Printf("updateDevices: No device %s in total device list\n", deviceId)
			continue
		}
		if channel, ok = device.Channels[channelId]; !ok {
			//fmt.Printf("updateDevices: No channel %s for device %s\n", channelId, deviceId)
			continue
		}
		if outPoint, ok = channel.Outputs[outDatapointId]; !ok {
			//fmt.Printf("updateDevices: No out datapoint %s for device %s and channel %d\n", outDatapointId, deviceId, channelId)
			continue
		}

		// 1) update the value in our device data structure
		updateDeviceDatapoint(outPoint, updValue)

		// 2) update the corresponding unit data structures
		key, changed := reHydrateUnitValue(deviceId, device, channelId, outPoint)

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
