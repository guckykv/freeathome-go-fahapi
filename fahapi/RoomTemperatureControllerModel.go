package fahapi

import (
	"fmt"
	"log"
	"strconv"
	"time"
)

type RoomTemperatureControllerUnit struct {
	UnitData
	ActualDegree    float64
	TargetDegree    float64
	Active          int
	Capacity        int
	ActualDegreeSet bool
	TargetDegreeSet bool
	ActiveSet       bool
	CapacitySet     bool
}

const UntTypeRoomTemperatureController UnitTypeConst = "CoRoTemp"

func CastRTC(u Unit) *RoomTemperatureControllerUnit {
	if typeSave, ok := u.(*RoomTemperatureControllerUnit); ok {
		return typeSave
	}
	log.Print("CastRTC - wrong type\n")
	return nil
}

func (rtc *RoomTemperatureControllerUnit) updateUnitFromOutDatapoint(outPut *InOutPut) bool {
	changed := false
	switch *outPut.PairingID {
	case 0x0030: // AL_ACTUATING_VALUE_HEATING (Determines the through flow volume of the control valve)
		capacity, _ := strconv.Atoi(*outPut.Value)
		if capacity != rtc.Capacity {
			rtc.Capacity = capacity
			rtc.CapacitySet = true
			rtc.LastUpdate = time.Now()
			changed = true
		}
	case 0x0031: // AL_FAN_COIL_LEVEL
	case 0x0032: // AL_ACTUATING_VALUE_COOLING (Determines the through flow volume of the control valve)
	case 0x0033: // AL_SET_POINT_TEMPERATURE (Defines the displayed set point Temperature of the system)
		target, _ := strconv.ParseFloat(*outPut.Value, 64)
		if target != rtc.TargetDegree {
			rtc.TargetDegree = target
			rtc.TargetDegreeSet = true
			rtc.LastUpdate = time.Now()
			changed = true
		}
	case 0x0034: // AL_RELATIVE_SET_POINT_TEMPERATURE
	case 0x0036: // AL_STATE_INDICATION (states: on/off heating/cooling; eco/comfort; frost/not frost)
	case 0x0037: // AL_FAN_MANUAL_ON_OFF
	case 0x0038: // AL_CONTROLLER_ON_OFF (Switches controller on or off. Off means protection mode)
	case 0x0039: // AL_RELATIVE_SET_POINT_REQUEST
	case 0x003A: // AL_ECO_ON_OFF
	case 0x0040: // AL_FAN_STAGE_REQUEST
	case 0x0042: // AL_CONTROLLER_ON_OFF_REQUEST
	case 0x0111: // AL_INFO_ERROR
		if *outPut.Value != "0" {
			log.Fatalf("Device Error %s", *outPut.Value)
		}
	case 0x0130: // AL_MEASURED_TEMPERATURE
		actual, _ := strconv.ParseFloat(*outPut.Value, 64)
		if actual != rtc.ActualDegree {
			rtc.ActualDegree = actual
			rtc.ActualDegreeSet = true
			rtc.LastUpdate = time.Now()
			changed = true
		}
	case 0x0131: // AL_INFO_VALUE_HEATING
	case 0x0136: // AL_ACTUATING_FAN_STAGE_HEATING
	case 0x0143: // AL_ACTUATING_VALUE_ADD_HEATING
	case 0x0144: // AL_ACTUATING_VALUE_ADD_COOLING
	case 0x0147: // AL_ACTUATING_FAN_STAGE_COOLING
	case 0x014B: // AL_HEATING_ACTIVE
		var active int
		if *outPut.Value == "1" {
			active = 1
		} else {
			active = 0
		}
		if active != rtc.Active {
			rtc.Active = active
			rtc.ActiveSet = true
			rtc.LastUpdate = time.Now()
			changed = true
		}
	case 0x014C: // AL_COOLING_ACTIVE
	case 0x014D: // AL_HEATING_DEMAND
	case 0x014E: // AL_COOLING_DEMAND
	}

	return changed
}

func (rtc *RoomTemperatureControllerUnit) resetChanged() {
	rtc.ActiveSet = false
	rtc.ActualDegreeSet = false
	rtc.TargetDegreeSet = false
	rtc.CapacitySet = false
}

func (rtc *RoomTemperatureControllerUnit) String() string {
	active := "off"
	if rtc.Active == 1 {
		active = "on "
	}
	return fmt.Sprintf("%s %2.2f°C, %2.2f°C, %s (%d%%)", rtc.prtUnitHead(), rtc.ActualDegree, rtc.TargetDegree, active, rtc.Capacity)
}

func roomTemperatureControllerFactory(deviceId string, device *Device, channelId string) Unit {

	floor, room := GetFloorRoom(device, device.Channels[channelId])

	rtc := RoomTemperatureControllerUnit{
		UnitData: UnitData{
			SerialNumber: deviceId,
			ChannelId:    channelId,
			Type:         "CoRoTemp",
			Device:       device,
			Floor:        floor,
			Room:         room,
			LastUpdate:   time.Now(),
		},
	}

	for _, inOut := range device.Channels[channelId].Outputs {
		rtc.updateUnitFromOutDatapoint(inOut)
	}

	return &rtc
}
