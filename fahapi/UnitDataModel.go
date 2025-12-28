package fahapi

import (
	"fmt"
	"log"
	"sort"
	"time"
)

var UnitMap map[string]Unit

type UnitTypeConst string

// hydrated data structures
type UnitData struct {
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

func (u *UnitData) GetChannel() *Channel {
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

func getUnit(deviceId, channelId string) Unit {
	key := getUnitMapKey(deviceId, channelId)
	if unit, ok := UnitMap[key]; ok {
		return unit
	}
	return nil
}

func PrtAllUnits() {
	log.Println("------- BEGIN DUMP ALL UNITS")
	keys := getUnitMapKeysSortedByFloorRoom()
	for _, key := range keys {
		log.Println(UnitMap[key].String())
	}
	log.Println("------- END DUMP ALL UNITS")
}

// Sort
type ByFloorAndRoom []Unit

func (u ByFloorAndRoom) Len() int { return len(u) }
func (u ByFloorAndRoom) Less(i, j int) bool {
	return u[i].GetUnitData().Floor < u[j].GetUnitData().Floor ||
		(u[i].GetUnitData().Floor == u[j].GetUnitData().Floor && u[i].GetUnitData().Room < u[j].GetUnitData().Room)
}
func (u ByFloorAndRoom) Swap(i, j int) { u[i], u[j] = u[j], u[i] }

func getUnitMapKeysSortedByFloorRoom() []string {
	copyArray := make([]Unit, len(UnitMap))
	var i int = 0
	for _, unit := range UnitMap {
		copyArray[i] = unit
		i++
	}
	sort.Sort(ByFloorAndRoom(copyArray))

	keys := make([]string, len(UnitMap))

	i = 0
	for _, unit := range copyArray {
		keys[i] = unit.getUnitMapKey()
		i++
	}

	return keys
}

// ####

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

func unitDataFactory(deviceId, channelId string, unitType UnitTypeConst) UnitData {
	device := FreeDevices[deviceId]
	floor, room := GetFloorRoom(device, device.Channels[channelId])

	return UnitData{
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

func hydrateAllDevices(devices map[string]*Device) {
	UnitMap = make(map[string]Unit, len(devices))

	for deviceId, device := range devices {
		hydrateDevice(deviceId, device)
	}

	treatAllUnitsAsUpdated(false) // initially handle all units as updated - e.g. send all to influx
}

var countTickRounds = 0

func treatAllUnitsAsUpdated(forceLogging bool) {
	if forceLogging || logLevel > 1 {
		logger.Printf("------- BEGIN TREAT AS UNITS AS UPDATED --- %d ---\n", countTickRounds)
	} else if logLevel > 0 {
		logger.Printf("------- TICK EVENT %d - MARK ALL AS UPDATED\n", countTickRounds)
	}

	keys := getUnitMapKeysSortedByFloorRoom()
	handleUpdatedUnits(keys, forceLogging || logLevel > 1)

	if logLevel > 1 {
		logger.Printf("------- END TREAT AS UNITS AS UPDATED --- %d ---\n", countTickRounds)
	}
	countTickRounds++
}

func handleUpdatedUnits(unitKeys []string, printDevices bool) {
	if wsUpdateUnitCallback != nil {
		wsUpdateUnitCallback(unitKeys) // tell someone what has changed
	}

	for _, key := range unitKeys {
		unit := UnitMap[key]
		if printDevices {
			logger.Printf("[handleUpdatedUnits]%s\n", unit)
		}
		unit.resetChanged()
	}
}

func reHydrateUnitValue(deviceId string, channelId string, newData *InOutPut) (string, bool) {
	key := getUnitMapKey(deviceId, channelId)
	unit := UnitMap[key]
	if unit == nil {
		//fmt.Printf("reHydrateUnitValue: no unit found for key %s.\n", key)
		return "", false
	}
	changed := unit.updateUnitFromOutDatapoint(newData)
	if changed {
		unit.GetUnitData().LastUpdate = time.Now()
	}
	return key, changed
}

func hydrateDevice(deviceId string, device *Device) []string {
	var newUnitKeys []string
	newUnitKeys = make([]string, 0, len(device.Channels))

	for channelId := range device.Channels {
		if unit := hydrateChannel(deviceId, device, channelId); unit != nil {
			key := unit.getUnitMapKey()
			UnitMap[key] = unit
			newUnitKeys = append(newUnitKeys, key)
		}
	}

	return newUnitKeys
}

func hydrateChannel(deviceId string, device *Device, channelId string) Unit {
	switch FunctionIdType(*device.Channels[channelId].FunctionID) {
	case FID_SWITCH_SENSOR:
		return switchSensorFactory(deviceId, device, channelId)

	case FID_DIMMING_SENSOR:
		return dimmingSensorFactory(deviceId, device, channelId)

	case FID_SWITCH_ACTUATOR:
		return switchActuatorFactory(deviceId, device, channelId)

	case FID_DIMMING_ACTUATOR:
		return dimmingActuatorFactory(deviceId, device, channelId)

	case FID_WINDOW_DOOR_SENSOR:
		return windowDoorSensorFactory(deviceId, device, channelId)

	case FID_ROOM_TEMPERATURE_CONTROLLER_MASTER_WITHOUT_FAN:
		return roomTemperatureControllerFactory(deviceId, device, channelId)

	case FID_BRIGHTNESS_SENSOR:
		return weatherStationBrightnessFactory(deviceId, device, channelId)

	case FID_RAIN_SENSOR:
		return weatherStationRainFactory(deviceId, device, channelId)

	case FID_TEMPERATURE_SENSOR:
		return weatherStationTemperatureFactory(deviceId, device, channelId)

	case FID_WIND_SENSOR:
		return weatherStationWindFactory(deviceId, device, channelId)

	case FID_BLIND_ACTUATOR:
		return blindActuatorFactory(deviceId, device, channelId)

	case FID_SHUTTER_ACTUATOR:
		return shutterActuatorFactory(deviceId, device, channelId)

	case FID_AWNING_ACTUATOR:
		return awningActuatorFactory(deviceId, device, channelId)

	}

	return nil
}
