package fahapi

import (
	"fmt"
	"log"
)

type SwitchSensorUnit struct {
	UnitData
	On    bool
	OnSet bool
}

const UntTypeSwitchSensor UnitTypeConst = "SeSwitch"

func CastSSU(u Unit) *SwitchSensorUnit {
	if typeSave, ok := u.(*SwitchSensorUnit); ok {
		return typeSave
	}
	log.Print("CastSSU - wrong type\n")
	return nil
}

func (ssu *SwitchSensorUnit) updateUnitFromOutDatapoint(outPut *InOutPut) bool {
	changed := false

	switch *outPut.PairingID {
	case 0x0001: // AL_SWITCH_ON_OFF (Binary Switch value)
		on := *outPut.Value != "0"
		if on != ssu.On {
			ssu.On = on
			ssu.OnSet = true
			changed = true
		}
	case 0x0002: // AL_TIMED_START_STOP (For staircase lighning or movement detection)
	case 0x0003: // AL_FORCED
	case 0x0004: // AL_SCENE_CONTROL
	case 0x0010: // AL_RELATIVE_SET_VALUE_CONTROL
	case 0x0020: // AL_MOVE_UP_DOWN
	case 0x0021: // AL_STOP_STEP_UP_DOWN
	case 0x0028: // AL_FORCED_UP_DOWN
	case 0x0440: // AL_MEDIA_PLAY
	case 0x0441: // AL_MEDIA_PAUSE
	case 0x0442: // AL_MEDIA_NEXT
	case 0x0443: // AL_MEDIA_PREVIOUS
	case 0x0444: // AL_MEDIA_PLAY_MODE
	case 0x0445: // AL_MEDIA_MUTE
	case 0x0446: // AL_RELATIVE_VOLUME_CONTROL
	case 0x0447: // AL_ABSOLUTE_VOLUME_CONTROL
	case 0x0448: // AL_GROUP_MEMBERSHIP
	case 0x0449: // AL_PLAY_FAVORITE
	case 0x044a: // AL_PLAY_NEXT_FAVORITE
	case 0x0460: // AL_PLAYBACK_STATUS
	case 0x0160: // AL_RELATIVE_FAN_SPEED_CONTROL
	case 0x0161: // AL_ABSOLUTE_FAN_SPEED_CONTROL
	case 0xf101: // AL_SWITCH_ENTITY_ON_OFF (Switch entity On/Off; Entity control e.g. activate an alert or timer program)
	}

	return changed
}

func (ssu *SwitchSensorUnit) resetChanged() {
	ssu.OnSet = false
}

func (ssu *SwitchSensorUnit) String() string {
	on := "OFF"
	if ssu.On {
		on = "ON "
	}
	return fmt.Sprintf("%s %s: %s ", ssu.prtUnitHead(), *ssu.GetChannel().DisplayName, on)
}

func switchSensorFactory(deviceId string, device *Device, channelId string) Unit {
	wds := SwitchSensorUnit{
		UnitData: unitDataFactory(deviceId, channelId, UntTypeSwitchSensor),
	}

	for _, inOut := range device.Channels[channelId].Outputs {
		wds.updateUnitFromOutDatapoint(inOut)
	}

	return &wds
}
