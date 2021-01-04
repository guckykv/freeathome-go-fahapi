package fahapi

import (
	"fmt"
	"log"
	"strings"
	"time"
)

type SwitchActuatorUnit struct {
	UnitData
	On       bool
	OnSet    bool
	Force    bool
	ForceSet bool
}

const UntTypeSwitchActuator UnitTypeConst = "AcSwitch"

func CastSAU(u Unit) *SwitchActuatorUnit {
	if typeSave, ok := u.(*SwitchActuatorUnit); ok {
		return typeSave
	}
	log.Print("CastSAU - wrong type\n")
	return nil
}

func (sau *SwitchActuatorUnit) updateUnitFromOutDatapoint(outPut *InOutPut) bool {
	changed := false

	switch *outPut.PairingID {
	case 0x0100: // AL_INFO_ON_OFF (Reflects the binary state of the actuator)
		on := *outPut.Value != "0"
		if on != sau.On {
			sau.On = on
			sau.OnSet = true
			changed = true
		}
	case 0x0101: // AL_INFO_FORCE (Indicates the cause of forced operation (0 = not forced))
		force := *outPut.Value != "0"
		if force != sau.Force {
			sau.Force = force
			sau.ForceSet = true
			changed = true
		}
	}

	return changed
}

func (sau *SwitchActuatorUnit) resetChanged() {
	sau.OnSet = false
	sau.ForceSet = false
}

func (sau *SwitchActuatorUnit) String() string {
	on := "OFF"
	if sau.On {
		on = "ON "
	}
	force := ""
	if sau.Force {
		force = " (forced)"
	}
	name := strings.TrimSpace(*sau.GetChannel().DisplayName)
	return fmt.Sprintf("%s %s: %s%s", sau.prtUnitHead(), name, on, force)
}

func switchActuatorFactory(deviceId string, device *Device, channelId string) Unit {

	floor, room := GetFloorRoom(device, device.Channels[channelId])

	sau := SwitchActuatorUnit{
		UnitData: UnitData{
			SerialNumber: deviceId,
			ChannelId:    channelId,
			Type:         "AcSwitch",
			Device:       device,
			Floor:        floor,
			Room:         room,
			LastUpdate:   time.Now(),
		},
	}

	for _, inOut := range device.Channels[channelId].Outputs {
		sau.updateUnitFromOutDatapoint(inOut)
	}

	return &sau
}
