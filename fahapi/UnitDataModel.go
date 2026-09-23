package fahapi

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"
)

type UnitTypeConst string

// hydrated data structures
type UnitData struct {
	// client is the owner; it gives a unit access to logging without a global.
	client *Client

	SerialNumber string
	NativeId     *string
	ChannelId    string
	Type         UnitTypeConst
	Device       *Device
	Floor        string
	Room         string
	LastUpdate   time.Time
}

type Unit interface {
	GetChannel() *Channel
	GetUnitData() *UnitData
	String() string
	getUnitMapKey() string
	updateUnitFromOutDatapoint(outPut *InOutPut) bool
	resetChanged()
}

func (u *UnitData) logf(format string, v ...any) {
	if u != nil && u.client != nil {
		u.client.logf(format, v...)
		return
	}
	log.Printf(format, v...)
}

func (u *UnitData) verbose() bool {
	return u != nil && u.client != nil && u.client.logLevel > 1
}

// Str dereferences an optional string of the API model. Almost every field the
// SysAP returns is a pointer that may be absent, so consumers need this too.
func Str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// DisplayName is the name of the unit's channel, or "" when the channel or its
// name is missing.
func (u *UnitData) DisplayName() string {
	if ch := u.GetChannel(); ch != nil {
		return Str(ch.DisplayName)
	}
	return ""
}

// floatValue and intValue parse a datapoint value. A malformed value is
// reported and rejected rather than silently becoming 0, which would otherwise
// be passed on as a genuine measurement.
func (u *UnitData) floatValue(out *InOutPut) (float64, bool) {
	v, err := strconv.ParseFloat(*out.Value, 64)
	if err != nil {
		if u.verbose() {
			u.logf("warning: pairingID 0x%04x: %q is not a number, ignored\n", *out.PairingID, *out.Value)
		}
		return 0, false
	}
	return v, true
}

func (u *UnitData) intValue(out *InOutPut) (int, bool) {
	v, err := strconv.Atoi(*out.Value)
	if err != nil {
		if u.verbose() {
			u.logf("warning: pairingID 0x%04x: %q is not an integer, ignored\n", *out.PairingID, *out.Value)
		}
		return 0, false
	}
	return v, true
}

// applyOutput feeds one output datapoint into a unit. It absorbs the optional
// fields of the API model in one place, so no unit implementation has to guard
// against them and new device types inherit the checks.
func applyOutput(u Unit, out *InOutPut) bool {
	if u == nil || out == nil || out.PairingID == nil || out.Value == nil {
		return false
	}
	return u.updateUnitFromOutDatapoint(out)
}

func (u *UnitData) GetChannel() *Channel {
	if u.Device == nil {
		return nil
	}
	if channel, ok := u.Device.Channels[u.ChannelId]; ok {
		return channel
	}
	return nil
}

func (u *UnitData) GetUnitData() *UnitData {
	return u
}

func (u *UnitData) prtUnitHead() string {
	var updTimeFormat = "15:04:05"
	//return fmt.Sprintf("%3s %s@%s: %-40s", u.Type, u.getUnitMapKey(), u.LastUpdate.Format(updTimeFormat), name)
	nativeId := ""
	if u.NativeId != nil {
		nativeId = *u.NativeId // show only 8 chars of the native Id
	}
	return fmt.Sprintf("%s %-8s %s: %-11s / %-16s [%-8s] ", u.getUnitMapKey(), nativeId, u.LastUpdate.Format(updTimeFormat), u.Floor, u.Room, u.Type)
}

func getUnitMapKey(deviceId, channelId string) string {
	return fmt.Sprintf("%s.%s", deviceId, channelId)
}
func (u *UnitData) getUnitMapKey() string {
	return getUnitMapKey(u.SerialNumber, u.ChannelId)
}

func (c *Client) getUnit(deviceId, channelId string) Unit {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.units[getUnitMapKey(deviceId, channelId)]
}

// PrtAllUnits dumps every unit, sorted by floor and room.
func (c *Client) PrtAllUnits() {
	c.logf("------- BEGIN DUMP ALL UNITS\n")
	for _, key := range c.unitKeysSortedByFloorRoom() {
		if unit := c.Unit(key); unit != nil {
			c.logf("%s\n", unit.String())
		}
	}
	c.logf("------- END DUMP ALL UNITS\n")
}

// Sort
type ByFloorAndRoom []Unit

func (u ByFloorAndRoom) Len() int { return len(u) }
func (u ByFloorAndRoom) Less(i, j int) bool {
	return u[i].GetUnitData().Floor < u[j].GetUnitData().Floor ||
		(u[i].GetUnitData().Floor == u[j].GetUnitData().Floor && u[i].GetUnitData().Room < u[j].GetUnitData().Room)
}
func (u ByFloorAndRoom) Swap(i, j int) { u[i], u[j] = u[j], u[i] }

func (c *Client) unitKeysSortedByFloorRoom() []string {
	c.mu.RLock()
	units := make([]Unit, 0, len(c.units))
	for _, unit := range c.units {
		units = append(units, unit)
	}
	c.mu.RUnlock()

	sort.Sort(ByFloorAndRoom(units))

	keys := make([]string, len(units))
	for i, unit := range units {
		keys[i] = unit.getUnitMapKey()
	}
	return keys
}

// ####

func (c *Client) GetFloorRoom(device *Device, channel *Channel) (string, string) {
	var floor, room string
	var floorId, roomId string

	if device == nil || channel == nil {
		return "", ""
	}

	if channel.Floor != nil {
		floorId = *channel.Floor
	} else {
		if device.Floor != nil {
			floorId = *device.Floor
		} else {
			return "", ""
		}
	}
	// The room may be missing on both channel and device even though a floor is
	// set; leave it empty rather than dereferencing nil.
	if channel.Room != nil {
		roomId = *channel.Room
	} else if device.Room != nil {
		roomId = *device.Room
	}

	if c.configuration == nil {
		return "", ""
	}

	if floorObject, ok := c.configuration.Floorplan.Floors[floorId]; ok {
		floor = *floorObject.Name
		if roomObject, ok := floorObject.Rooms[roomId]; ok {
			room = *roomObject.Name
		} else {
			room = "-"
		}
	}

	return floor, room
}

func (c *Client) unitDataFactory(deviceId, channelId string, unitType UnitTypeConst) UnitData {
	device := c.devices[deviceId]
	floor, room := c.GetFloorRoom(device, device.Channels[channelId])

	return UnitData{
		client:       c,
		SerialNumber: deviceId,
		NativeId:     device.NativeId,
		ChannelId:    channelId,
		Type:         unitType,
		Device:       device,
		Floor:        floor,
		Room:         room,
		LastUpdate:   time.Now(),
	}
}

func (c *Client) hydrateAllDevices() {
	c.mu.Lock()
	c.units = make(map[string]Unit, len(c.devices))
	for deviceId, device := range c.devices {
		c.hydrateDeviceLocked(deviceId, device)
	}
	c.mu.Unlock()

	c.treatAllUnitsAsUpdated(false) // initially handle all units as updated - e.g. send all to influx
}

// TreatAllUnitsAsUpdated reports every unit as updated even though nothing
// changed. Applications use this to force a full flush, typically on SIGHUP.
func (c *Client) TreatAllUnitsAsUpdated(forceLogging bool) {
	c.treatAllUnitsAsUpdated(forceLogging)
}

func (c *Client) treatAllUnitsAsUpdated(forceLogging bool) {
	if forceLogging || c.logLevel > 1 {
		c.logf("------- BEGIN TREAD AS UNITS AS UPDATED --- %d ---\n", c.tickRounds)
	} else if c.logLevel > 0 {
		c.logf("------- TICK EVENT %d - MARK ALL AS UPDATED\n", c.tickRounds)
	}

	keys := c.unitKeysSortedByFloorRoom()
	c.handleUpdatedUnits(keys, forceLogging || c.logLevel > 1)

	if c.logLevel > 1 {
		c.logf("------- END TREAD AS UNITS AS UPDATED --- %d ---\n", c.tickRounds)
	}
	c.tickRounds++
}

func (c *Client) handleUpdatedUnits(unitKeys []string, printDevices bool) {
	if c.unitCallback != nil {
		c.unitCallback(unitKeys) // tell someone what has changed
	}

	for _, key := range unitKeys {
		unit := c.Unit(key)
		if unit == nil {
			continue
		}
		if printDevices {
			c.logf("%s\n", unit)
		}
		unit.resetChanged()
	}
}

func (c *Client) reHydrateUnitValue(deviceId string, channelId string, newData *InOutPut) (string, bool) {
	key := getUnitMapKey(deviceId, channelId)
	unit := c.Unit(key)
	if unit == nil {
		//fmt.Printf("reHydrateUnitValue: no unit found for key %s.\n", key)
		return "", false
	}
	changed := applyOutput(unit, newData)
	if changed {
		unit.GetUnitData().LastUpdate = time.Now()
	}
	return key, changed
}

func (c *Client) hydrateDevice(deviceId string, device *Device) []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.hydrateDeviceLocked(deviceId, device)
}

// hydrateDeviceLocked expects c.mu to be held for writing.
func (c *Client) hydrateDeviceLocked(deviceId string, device *Device) []string {
	newUnitKeys := make([]string, 0, len(device.Channels))

	for channelId := range device.Channels {
		if unit := c.hydrateChannel(deviceId, device, channelId); unit != nil {
			key := unit.getUnitMapKey()
			c.units[key] = unit
			newUnitKeys = append(newUnitKeys, key)
		}
	}

	return newUnitKeys
}

func (c *Client) hydrateChannel(deviceId string, device *Device, channelId string) Unit {
	channel := device.Channels[channelId]
	if channel == nil || channel.FunctionID == nil {
		// Channels without a functionID exist on real hardware and are simply
		// not tracked.
		return nil
	}

	// Function IDs are hex, and the SysAP does not keep one letter case: current
	// firmware reports window/door sensors as "F", older firmware as "f", and
	// even mixes cases within one configuration ("1a" next to "5A").
	switch FunctionIdType(strings.ToLower(*channel.FunctionID)) {
	case FID_SWITCH_SENSOR:
		return switchSensorFactory(c, deviceId, device, channelId)

	case FID_DIMMING_SENSOR:
		return dimmingSensorFactory(c, deviceId, device, channelId)

	case FID_SWITCH_ACTUATOR:
		return switchActuatorFactory(c, deviceId, device, channelId)

	case FID_DIMMING_ACTUATOR:
		return dimmingActuatorFactory(c, deviceId, device, channelId)

	case FID_WINDOW_DOOR_SENSOR:
		return windowDoorSensorFactory(c, deviceId, device, channelId)

	case FID_ROOM_TEMPERATURE_CONTROLLER_MASTER_WITHOUT_FAN:
		return roomTemperatureControllerFactory(c, deviceId, device, channelId)

	case FID_BRIGHTNESS_SENSOR:
		return weatherStationBrightnessFactory(c, deviceId, device, channelId)

	case FID_RAIN_SENSOR:
		return weatherStationRainFactory(c, deviceId, device, channelId)

	case FID_TEMPERATURE_SENSOR:
		return weatherStationTemperatureFactory(c, deviceId, device, channelId)

	case FID_WIND_SENSOR:
		return weatherStationWindFactory(c, deviceId, device, channelId)

	}

	return nil
}
