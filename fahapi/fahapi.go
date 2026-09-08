package fahapi

import (
	"bytes"
	"encoding/base64"
	json2 "encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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
	DisplayName  *string             `json:"displayName,omitempty"`
	Floor        *string             `json:"floor,omitempty"`
	Room         *string             `json:"room,omitempty"`
	Interface    *string             `json:"interface,omitempty"`
	NativeId     *string             `json:"nativeId,omitempty"`
	Unresponsive *bool               `json:"unresponsive,omitempty"`
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

type VirtualDeviceProperties struct {
	Displayname string `json:"displayname,omitempty"`
	Ttl         string `json:"ttl,omitempty"`
}

// VirtualDevice defines model for VirtualDevice.
type VirtualDevice struct {
	Properties VirtualDeviceProperties `json:"properties,omitempty"`
	Type       VirtualDeviceType       `json:"type,omitempty"`
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
/*
type VirtualDevicesSuccessXXX struct {
	AdditionalProperties map[string]struct {
		Devices *VirtualDevicesSuccess_Devices `json:"devices,omitempty"`
	} `json:"-"`
}
*/
type VirtualDevicesSuccess struct {
	ZeroSysAp struct {
		Devices map[string]struct {
			Serial string `json:"serial,omitempty"`
		} `json:"devices,omitempty"`
	} `json:"00000000-0000-0000-0000-000000000000"`
}

// ===============================================================================================

// defaultSysApID is the only SysAP id the local API uses today.
const defaultSysApID = "00000000-0000-0000-0000-000000000000"

const ApiPathPrefix string = "/fhapi/v1"
const WebSocketPath string = "/fhapi/v1/api/ws"

type WebsocketUpdateUnitCallbackFunc func(unitKeys []string)
type WebsocketUpdateMessageCallbackFunc func(message WebsocketMessage)

// Config describes one System Access Point and how to report its updates.
type Config struct {
	// Host is the SysAP address, optionally with a port: "192.168.1.10".
	Host     string
	Username string
	Password string

	// UnitCallback receives the keys of the units that changed. MessageCallback
	// receives every websocket message before it is applied. Both are called
	// from the websocket loop's goroutine and must not block for long.
	UnitCallback    WebsocketUpdateUnitCallbackFunc
	MessageCallback WebsocketUpdateMessageCallbackFunc

	// Logger defaults to the standard logger. LogLevel: 0 quiet, 1 updates and
	// connection, 2 verbose, 3 dump raw websocket messages.
	Logger   *log.Logger
	LogLevel int

	// Timeout bounds a single HTTP request. Defaults to 10s.
	Timeout time.Duration
}

// Client talks to one System Access Point and holds the hydrated model of its
// devices. Create it with New.
//
// Concurrency: the websocket loop applies every update from a single goroutine,
// and the callbacks run in that goroutine too, so a callback sees a consistent
// model and may read unit fields freely. That is where reading belongs.
//
// From any other goroutine, use Unit, Units, Device and Configuration: they
// take a read lock and are safe. Their lock covers the maps, not the contents
// of a unit -- the loop keeps writing those fields, so reading them outside a
// callback is a data race. The "...Set" flags are valid only for the duration
// of the callback that reported them.
type Client struct {
	host           string
	authentication string
	httpClient     *http.Client
	dialer         *websocket.Dialer

	logger   *log.Logger
	logLevel int

	unitCallback    WebsocketUpdateUnitCallbackFunc
	messageCallback WebsocketUpdateMessageCallbackFunc

	mu            sync.RWMutex
	devices       map[string]*Device
	units         map[string]Unit
	configuration *SysAP

	tickRounds int
}

// New creates a client. It performs no I/O; call ReadAndHydrateAllDevices next.
func New(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}

	return &Client{
		host:           cfg.Host,
		authentication: "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Password)),
		// One client so connections are reused. Without a timeout an
		// unresponsive SysAP would block a call forever.
		httpClient: &http.Client{Timeout: timeout},
		dialer: &websocket.Dialer{
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: handshakeTimeout,
		},
		logger:          logger,
		logLevel:        cfg.LogLevel,
		unitCallback:    cfg.UnitCallback,
		messageCallback: cfg.MessageCallback,
		devices:         map[string]*Device{},
		units:           map[string]Unit{},
	}
}

func (c *Client) logf(format string, v ...any) {
	c.logger.Printf(format, v...)
}

// Unit returns the unit for a "deviceId.channelId" key, or nil.
func (c *Client) Unit(key string) Unit {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.units[key]
}

// Units is a snapshot of the unit map. The map is a copy, the units in it are
// not: their fields keep changing as updates arrive, and the "...Set" flags are
// only meaningful inside a callback.
func (c *Client) Units() map[string]Unit {
	c.mu.RLock()
	defer c.mu.RUnlock()

	units := make(map[string]Unit, len(c.units))
	for key, unit := range c.units {
		units[key] = unit
	}
	return units
}

// Device returns the raw device record, or nil.
func (c *Client) Device(deviceId string) *Device {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.devices[deviceId]
}

// Configuration is the SysAP configuration read by ReadAndHydrateAllDevices.
func (c *Client) Configuration() *SysAP {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.configuration
}

// ReadAndHydrateAllDevices loads the SysAP configuration and builds the Unit
// map from it. It must be called once before StartWebSocketLoop.
func (c *Client) ReadAndHydrateAllDevices() error {
	configResult, err := c.GetConfiguration()
	if err != nil {
		return fmt.Errorf("can't initialize f@h api: %w", err)
	}

	c.mu.Lock()
	c.configuration = configResult
	c.devices = configResult.Devices
	if c.devices == nil {
		c.devices = map[string]*Device{}
	}
	c.mu.Unlock()

	c.hydrateAllDevices()
	return nil
}

func (c *Client) GetDeviceList() (*Devicelist, error) {
	httpUrl := fmt.Sprintf("http://%s%s%s", c.host, ApiPathPrefix, "/api/rest/devicelist")
	json, err := c.loadUrl(httpUrl)
	if err != nil {
		return nil, err
	}
	var result Devicelist
	err = json2.Unmarshal(json, &result)
	return &result, err
}

func (c *Client) GetDevice(sysap string, deviceId string) (*Device, error) {
	httpUrl := fmt.Sprintf("http://%s%s%s/%s/%s", c.host, ApiPathPrefix, "/api/rest/device", sysap, deviceId)
	json, err := c.loadUrl(httpUrl)
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

func (c *Client) GetDatapoint(sysap string, deviceId string, channelId string, datapointId string) (string, error) {
	httpUrl := fmt.Sprintf("http://%s%s%s/%s/%s.%s.%s", c.host, ApiPathPrefix, "/api/rest/datapoint", sysap, deviceId, channelId, datapointId)
	json, err := c.loadUrl(httpUrl)
	if err != nil {
		return "", err
	}
	var result ApiRestDatapointSysapSerialGet200ApplicationJsonResponse
	if err = json2.Unmarshal(json, &result); err != nil {
		return "", err
	}
	if len(result.ZeroSysAp.Values) == 0 {
		return "", fmt.Errorf("datapoint %s.%s.%s returned no value", deviceId, channelId, datapointId)
	}
	return result.ZeroSysAp.Values[0], nil
}

func (c *Client) GetConfiguration() (*SysAP, error) {
	httpUrl := fmt.Sprintf("http://%s%s%s", c.host, ApiPathPrefix, "/api/rest/configuration")
	json, err := c.loadUrl(httpUrl)
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

func (c *Client) PutDatapoint(sysap string, deviceId string, channelId string, datapointId string, value string) (bool, error) {
	httpUrl := fmt.Sprintf("http://%s%s%s/%s/%s.%s.%s", c.host, ApiPathPrefix, "/api/rest/datapoint", sysap, deviceId, channelId, datapointId)

	var err error
	var bstr, body []byte
	bstr = []byte(value)

	if body, err = c.putRequest(httpUrl, bytes.NewBuffer(bstr)); err != nil {
		return false, err
	}

	var result ApiRestDatapointSysapSerialPut200TextPlainResponse
	if err = json2.Unmarshal(body, &result); err != nil {
		return false, err
	}
	ok := result[sysap].Result == "OK"
	return ok, nil
}

func (c *Client) PutVirtualDevice(sysap, serial string, message *VirtualDevice) (virtualSerial string, err error) {
	httpUrl := fmt.Sprintf("http://%s%s%s/%s/%s", c.host, ApiPathPrefix, "/api/rest/virtualdevice", sysap, serial)

	var messageString []byte
	messageString, err = json2.Marshal(message)

	var returnBody []byte
	if returnBody, err = c.putRequest(httpUrl, bytes.NewBuffer(messageString)); err != nil {
		return
	}

	var result VirtualDevicesSuccess
	if err = json2.Unmarshal(returnBody, &result); err != nil {
		return
	}

	for virtualSerial, devices := range result.ZeroSysAp.Devices {
		if devices.Serial == serial {
			return virtualSerial, nil
		}
	}
	return "", fmt.Errorf("virtual Device PUT returned no device with serial %s (%s)", serial, returnBody)
}

func (c *Client) loadUrl(httpUrl string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, httpUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", c.authentication)

	if c.logLevel > 1 {
		c.logf("getting %s ...\n", httpUrl)
	}

	response, err := c.httpClient.Do(req)
	if err != nil {
		c.logf("error getting %s: %s\n", httpUrl, err.Error())
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET url %s returned code %d (%s): %s",
			httpUrl, response.StatusCode, response.Status, errorBody(response.Body))
	}

	return io.ReadAll(response.Body)
}

func (c *Client) putRequest(url string, data io.Reader) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPut, url, data)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authentication)
	req.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PUT url %s returned code %d (%s): %s",
			url, response.StatusCode, response.Status, errorBody(response.Body))
	}

	return io.ReadAll(response.Body)
}

// errorBody reads the response body of a failed request so the reason can be
// part of the error. The SysAP puts usable diagnostics there.
func errorBody(body io.Reader) string {
	b, err := io.ReadAll(io.LimitReader(body, 2048))
	if err != nil || len(b) == 0 {
		return "<no body>"
	}
	return strings.TrimSpace(string(b))
}
