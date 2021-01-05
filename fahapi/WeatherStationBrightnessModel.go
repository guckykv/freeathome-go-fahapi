package fahapi

import (
	"fmt"
	"log"
	"math"
	"strconv"
)

const luminanceLevel = 0.02 // change has to be bigger than that: logarithm change of lux

type WeatherStationBrightnessUnit struct {
	UnitData
	Luminance         float64
	LuminanceSet      bool
	LuminanceAlarm    bool
	LuminanceAlarmSet bool
}

const UntTypeWeatherStationBrightness UnitTypeConst = "SeWBrig"

func CastWSB(u Unit) *WeatherStationBrightnessUnit {
	if typeSave, ok := u.(*WeatherStationBrightnessUnit); ok {
		return typeSave
	}
	log.Print("CastWSB - wrong type\n")
	return nil
}

func (ws *WeatherStationBrightnessUnit) String() string {
	alarm := ""
	if ws.LuminanceAlarm {
		alarm = " Sonne"
	}
	return fmt.Sprintf("%s Luminance %.2f lux%s", ws.prtUnitHead(), ws.Luminance, alarm)
}

func (ws *WeatherStationBrightnessUnit) updateUnitFromOutDatapoint(outPut *InOutPut) bool {
	changed := false

	// f 41
	switch *outPut.PairingID {
	case 0x0004: // AL_SCENE_CONTROL (Recall or learn the set value related to encoded scene number)
	case 0x0402: // AL_BRIGHTNESS_ALARM
		alarm := *outPut.Value != "0"
		if alarm != ws.LuminanceAlarm {
			ws.LuminanceAlarm = alarm
			ws.LuminanceAlarmSet = true
			changed = true
		}
	case 0x0403: // AL_BRIGHTNESS_LEVEL
		luminance, _ := strconv.ParseFloat(*outPut.Value, 64)
		if math.Abs(math.Log(ws.Luminance)-math.Log(luminance)) >= luminanceLevel && luminance != 0.0 {
			ws.Luminance = luminance
			ws.LuminanceSet = true
			changed = true
		}

	}

	return changed
}

func (ws *WeatherStationBrightnessUnit) resetChanged() {
	ws.LuminanceSet = false
	ws.LuminanceAlarmSet = false
}

func weatherStationBrightnessFactory(deviceId string, device *Device, channelId string) Unit {
	ws := WeatherStationBrightnessUnit{
		UnitData: unitDataFactory(deviceId, channelId, UntTypeWeatherStationBrightness),
	}

	for _, inOut := range device.Channels[channelId].Outputs {
		ws.updateUnitFromOutDatapoint(inOut)
	}

	return &ws
}
