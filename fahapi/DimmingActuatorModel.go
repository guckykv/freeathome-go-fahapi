package fahapi

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"time"
)

type DimmingActuatorUnit struct {
	UnitData
	On              bool
	OnSet           bool
	DimmingValue    int
	DimmingValueSet bool
	Force           bool
	ForceSet        bool
}

const UntTypeDimmingActuator UnitTypeConst = "AcDimmin"

func CastDAU(u Unit) *DimmingActuatorUnit {
	if typeSave, ok := u.(*DimmingActuatorUnit); ok {
		return typeSave
	}
	log.Print("CastDAU - wrong type\n")
	return nil
}

func (dau *DimmingActuatorUnit) updateUnitFromOutDatapoint(outPut *InOutPut) bool {
	changed := false

	switch *outPut.PairingID {
	case 0x0100: // AL_INFO_ON_OFF (Reflects the binary state of the actuator)
		on := *outPut.Value != "0"
		if on != dau.On {
			dau.On = on
			dau.OnSet = true
			changed = true
		}
	case 0x0101: // AL_INFO_FORCE (Indicates the cause of forced operation (0 = not forced))
		force := *outPut.Value != "0"
		if force != dau.Force {
			dau.Force = force
			dau.ForceSet = true
			changed = true
		}
	case 0x0110: // AL_INFO_ACTUAL_DIMMING_VALUE
		dimmingValue, _ := strconv.Atoi(*outPut.Value)
		if math.Abs(float64(dau.DimmingValue-dimmingValue)) >= 1 {
			dau.DimmingValue = dimmingValue
			dau.DimmingValueSet = true
			changed = true
		}
	case 0x0111: // AL_INFO_ERROR (Indicates load failures / short circuits / etc)
	}

	return changed
}

func (dau *DimmingActuatorUnit) resetChanged() {
	dau.OnSet = false
	dau.DimmingValueSet = false
	dau.ForceSet = false
}

func (dau *DimmingActuatorUnit) String() string {
	on := "OFF"
	if dau.On {
		on = "ON "
	}
	force := ""
	if dau.Force {
		force = " (forced)"
	}
	return fmt.Sprintf("%s %s: %s %2d%%%s", dau.prtUnitHead(), *dau.GetChannel().DisplayName, on, dau.DimmingValue, force)
}

func dimmingActuatorFactory(deviceId string, device *Device, channelId string) Unit {

	floor, room := GetFloorRoom(device, device.Channels[channelId])

	dau := DimmingActuatorUnit{
		UnitData: UnitData{
			SerialNumber: deviceId,
			ChannelId:    channelId,
			Type:         UntTypeDimmingActuator,
			Device:       device,
			Floor:        floor,
			Room:         room,
			LastUpdate:   time.Now(),
		},
	}

	for _, inOut := range device.Channels[channelId].Outputs {
		dau.updateUnitFromOutDatapoint(inOut)
	}

	return &dau
}
