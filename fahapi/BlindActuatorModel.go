package fahapi

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
)

type BlindActuatorUnit struct {
	UnitData
	AbsolutePosBlinds int
	InfoMoveUpDown    int
	Force             bool
	ForceSet          bool
}

const UntTypeBlindActuator UnitTypeConst = "ActBlind"

func CastBAU(u Unit) *BlindActuatorUnit {
	if typeSave, ok := u.(*BlindActuatorUnit); ok {
		return typeSave
	}
	log.Print("CastBAU - wrong type\n")
	return nil
}

func (bau *BlindActuatorUnit) updateUnitFromOutDatapoint(outPut *InOutPut) bool {
	changed := false

	switch *outPut.PairingID {
	case 0x0101: // AL_INFO_FORCE (Indicates the cause of forced operation (0 = not forced))
		force := *outPut.Value != "0"
		if force != bau.Force {
			bau.Force = force
			bau.ForceSet = true
			changed = true
		}
	case 0x0111: // AL_INFO_ERROR (Indicates load failures / short circuits / etc)
	case 0x0120: // AL_INFO_MOVE_UP_DOWN (Indicates last moving direction and whether moving currently or not)
		infoMoveUpDown, _ := strconv.Atoi(*outPut.Value)
		bau.InfoMoveUpDown = infoMoveUpDown
		changed = true
	case 0x0121: // AL_CURRENT_ABSOLUTE_POSITION_BLINDS_PERCENTAGE (Indicate the current position of the sunblinds in percentage)
		absolutePosBlinds, _ := strconv.Atoi(*outPut.Value)
		if math.Abs(float64(bau.AbsolutePosBlinds-absolutePosBlinds)) >= 1 {
			bau.AbsolutePosBlinds = absolutePosBlinds
			changed = true
		}
	}

	return changed
}

func (bau *BlindActuatorUnit) resetChanged() {
	bau.ForceSet = false
}

func (bau *BlindActuatorUnit) String() string {
	force := ""
	if bau.Force {
		force = " (forced)"
	}
	updown := ""
	switch bau.InfoMoveUpDown {
	case 0: // not moving
		updown = "not moving"
	case 2: // moves up
		updown = "moves up"
	case 3: // moves down
		updown = "moves down"
	}
	name := strings.TrimSpace(*bau.GetChannel().DisplayName)
	return fmt.Sprintf("%s %s: %3d%% %s%s", bau.prtUnitHead(), name,  bau.AbsolutePosBlinds, updown, force)
}

func blindActuatorFactory(deviceId string, device *Device, channelId string) Unit {
	bau := BlindActuatorUnit{
		UnitData: unitDataFactory(deviceId, channelId, UntTypeBlindActuator),
	}

	for _, inOut := range device.Channels[channelId].Outputs {
		bau.updateUnitFromOutDatapoint(inOut)
	}

	return &bau
}
