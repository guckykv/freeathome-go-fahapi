package fahapi

import (
	"fmt"
	"log"
	"time"
)

type WindowDoorSensorUnit struct {
	UnitData
	Open    bool
	OpenSet bool
}

const UntTypeWindowDoorSensor UnitTypeConst = "SeWindow"

func CastWDS(u Unit) *WindowDoorSensorUnit {
	if typeSave, ok := u.(*WindowDoorSensorUnit); ok {
		return typeSave
	}
	log.Print("CastWDS - wrong type\n")
	return nil
}

func (wds *WindowDoorSensorUnit) updateUnitFromOutDatapoint(outPut *InOutPut) bool {
	changed := false
	switch *outPut.PairingID {
	case 0x0001: // AL_SWITCH_ON_OFF (Binary Switch value)
	case 0x0002: // AL_TIMED_START_STOP (For staircase lighning or movement detection)
	case 0x0003: // AL_FORCED
	case 0x0004: // AL_SCENE_CONTROL
	case 0x0006: // AL_TIMED_MOVEMENT
	case 0x0010: // AL_RELATIVE_SET_VALUE_CONTROL
	case 0x0020: // AL_MOVE_UP_DOWN
	case 0x0021: // AL_STOP_STEP_UP_DOWN
	case 0x0025: // AL_WIND_ALARM
	case 0x0026: // AL_FROST_ALARM
	case 0x0027: // AL_RAIN_ALARM
	case 0x0028: // AL_FORCED_UP_DOWN
	case 0x0035: // AL_WINDOW_DOOR (Open = 1 / closed = 0)
		var open bool
		if *outPut.Value == "1" {
			open = true
		} else {
			open = false
		}
		if open != wds.Open {
			wds.Open = open
			wds.OpenSet = true
			wds.LastUpdate = time.Now()
			changed = true
		}

	case 0x0135: // AL_HEATING_COOLING
	}

	return changed
}

func (wds *WindowDoorSensorUnit) resetChanged() {
	wds.OpenSet = false
}

func (wds *WindowDoorSensorUnit) String() string {
	open := "zu"
	if wds.Open {
		open = "auf"
	}
	return fmt.Sprintf("%s %s: %s", wds.prtUnitHead(), *wds.GetChannel().DisplayName, open)
}

func windowDoorSensorFactory(deviceId string, device *Device, channelId string) Unit {
	wds := WindowDoorSensorUnit{
		UnitData: unitDataFactory(deviceId, channelId, UntTypeWindowDoorSensor),
	}

	for _, inOut := range device.Channels[channelId].Outputs {
		wds.updateUnitFromOutDatapoint(inOut)
	}

	return &wds
}
