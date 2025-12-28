package fahapi

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
)

type ShutterActuatorUnit struct {
	UnitData
	AbsolutePosBlinds int
	AbsolutePosSlats  int
	InfoMoveUpDown    int
	Force             bool
	ForceSet          bool
}

const UntTypeShutterActuator UnitTypeConst = "AcShuttr"

func CastSHAU(u Unit) *ShutterActuatorUnit {
	if typeSave, ok := u.(*ShutterActuatorUnit); ok {
		return typeSave
	}
	log.Print("CastSHAU - wrong type\n")
	return nil
}

func (shau *ShutterActuatorUnit) updateUnitFromOutDatapoint(outPut *InOutPut) bool {
	changed := false

	switch *outPut.PairingID {
	case 0x0101: // AL_INFO_FORCE (Indicates the cause of forced operation (0 = not forced))
		force := *outPut.Value != "0"
		if force != shau.Force {
			shau.Force = force
			shau.ForceSet = true
			changed = true
		}
	case 0x0111: // AL_INFO_ERROR (Indicates load failures / short circuits / etc)
	case 0x0120: // AL_INFO_MOVE_UP_DOWN (Indicates last moving direction and whether moving currently or not)
		infoMoveUpDown, _ := strconv.Atoi(*outPut.Value)
		shau.InfoMoveUpDown = infoMoveUpDown
		changed = true
	case 0x0121: // AL_CURRENT_ABSOLUTE_POSITION_BLINDS_PERCENTAGE (Indicate the current position of the sunblinds in percentage)
		absolutePosBlinds, _ := strconv.Atoi(*outPut.Value)
		if math.Abs(float64(shau.AbsolutePosBlinds-absolutePosBlinds)) >= 1 {
			shau.AbsolutePosBlinds = absolutePosBlinds
			changed = true
		}
	case 0x0122: // AL_CURRENT_ABSOLUTE_POSITION_SLATS_PERCENTAGE (Indicate the current position of the slats in percentage)
		absolutePosSlats, _ := strconv.Atoi(*outPut.Value)
		if math.Abs(float64(shau.AbsolutePosSlats-absolutePosSlats)) >= 1 {
			shau.AbsolutePosSlats = absolutePosSlats
			changed = true
		}
	}

	return changed
}

func (shau *ShutterActuatorUnit) resetChanged() {
	shau.ForceSet = false
}

func (shau *ShutterActuatorUnit) String() string {
	force := ""
	if shau.Force {
		force = " (forced)"
	}
	updown := ""
	switch shau.InfoMoveUpDown {
	case 0: // not moving
		updown = "not moving"
	case 2: // moves up
		updown = "moves up"
	case 3: // moves down
		updown = "moves down"
	}
	name := strings.TrimSpace(*shau.GetChannel().DisplayName)
	return fmt.Sprintf("%s %s: %3d%%, %3d%% %s%s", shau.prtUnitHead(), name, shau.AbsolutePosBlinds, shau.AbsolutePosSlats, updown, force)
}

func shutterActuatorFactory(deviceId string, device *Device, channelId string) Unit {
	shau := ShutterActuatorUnit{
		UnitData: unitDataFactory(deviceId, channelId, UntTypeShutterActuator),
	}

	for _, inOut := range device.Channels[channelId].Outputs {
		shau.updateUnitFromOutDatapoint(inOut)
	}

	return &shau
}
