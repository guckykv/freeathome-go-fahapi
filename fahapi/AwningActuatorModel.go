package fahapi

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
)

type AwningActuatorUnit struct {
	UnitData
	AbsolutePosBlinds int
	InfoMoveUpDown    int
	Force             bool
	ForceSet          bool
}

const UntTypeAwningActuator UnitTypeConst = "AcAwning"

func CastAAU(u Unit) *AwningActuatorUnit {
	if typeSave, ok := u.(*AwningActuatorUnit); ok {
		return typeSave
	}
	log.Print("CastAAU - wrong type\n")
	return nil
}

func (aau *AwningActuatorUnit) updateUnitFromOutDatapoint(outPut *InOutPut) bool {
	changed := false

	switch *outPut.PairingID {
	case 0x0101: // AL_INFO_FORCE (Indicates the cause of forced operation (0 = not forced))
		force := *outPut.Value != "0"
		if force != aau.Force {
			aau.Force = force
			aau.ForceSet = true
			changed = true
		}
	case 0x0111: // AL_INFO_ERROR (Indicates load failures / short circuits / etc)
	case 0x0120: // AL_INFO_MOVE_UP_DOWN (Indicates last moving direction and whether moving currently or not)
		infoMoveUpDown, _ := strconv.Atoi(*outPut.Value)
		aau.InfoMoveUpDown = infoMoveUpDown
		changed = true
	case 0x0121: // AL_CURRENT_ABSOLUTE_POSITION_BLINDS_PERCENTAGE (Indicate the current position of the sunblinds in percentage)
		absolutePosBlinds, _ := strconv.Atoi(*outPut.Value)
		if math.Abs(float64(aau.AbsolutePosBlinds-absolutePosBlinds)) >= 1 {
			aau.AbsolutePosBlinds = absolutePosBlinds
			changed = true
		}
	}

	return changed
}

func (aau *AwningActuatorUnit) resetChanged() {
	aau.ForceSet = false
}

func (aau *AwningActuatorUnit) String() string {
	force := ""
	if aau.Force {
		force = " (forced)"
	}
	updown := ""
	switch aau.InfoMoveUpDown {
	case 0: // not moving
		updown = "not moving"
	case 2: // moves up
		updown = "moves up"
	case 3: // moves down
		updown = "moves down"
	}
	name := strings.TrimSpace(*aau.GetChannel().DisplayName)
	return fmt.Sprintf("%s %s: %3d%% %s%s", aau.prtUnitHead(), name,  aau.AbsolutePosBlinds, updown, force)
}

func awningActuatorFactory(deviceId string, device *Device, channelId string) Unit {
	aau := AwningActuatorUnit{
		UnitData: unitDataFactory(deviceId, channelId, UntTypeAwningActuator),
	}

	for _, inOut := range device.Channels[channelId].Outputs {
		aau.updateUnitFromOutDatapoint(inOut)
	}

	return &aau
}
