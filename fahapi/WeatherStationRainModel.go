package fahapi

import (
	"fmt"
	"log"
	"math"
)

const rainLevel = 2 // rain percentage change has to be bigger than that

type WeatherStationRainUnit struct {
	UnitData
	RainAlarm         bool
	RainAlarmSet      bool
	RainPercentage    int
	RainPercentageSet bool
}

const UntTypeWeatherStationRain UnitTypeConst = "SeWRain"

func CastWSR(u Unit) *WeatherStationRainUnit {
	if typeSave, ok := u.(*WeatherStationRainUnit); ok {
		return typeSave
	}
	log.Printf("CastWSR: wrong unit type %T\n", u)
	return nil
}

func (ws *WeatherStationRainUnit) String() string {
	alarm := ""
	if ws.RainAlarm {
		alarm = " Regen"
	}
	return fmt.Sprintf("%s Rain %d%%%s", ws.prtUnitHead(), ws.RainPercentage, alarm)
}

func (ws *WeatherStationRainUnit) updateUnitFromOutDatapoint(outPut *InOutPut) bool {
	changed := false

	// f 42
	switch *outPut.PairingID {
	case 0x0004: // AL_SCENE_CONTROL (Recall or learn the set value related to encoded scene number)
	case 0x0027: // AL_RAIN_ALARM
		alarm := *outPut.Value != "0"
		if alarm != ws.RainAlarm {
			ws.RainAlarm = alarm
			ws.RainAlarmSet = true
			changed = true
		}
	case 0x0405: // AL_RAIN_SENSOR_ACTIVATION_PERCENTAGE
		rainPercentage, ok := ws.intValue(outPut)
		if !ok {
			return false
		}
		if math.Abs(float64(ws.RainPercentage-rainPercentage)) > rainLevel {
			ws.RainPercentage = rainPercentage
			ws.RainPercentageSet = true
			changed = true
		}
	case 0x0406: // AL_RAIN_SENSOR_FREQUENCY
	}

	return changed
}

func (ws *WeatherStationRainUnit) resetChanged() {
	ws.RainAlarmSet = false
	ws.RainPercentageSet = false
}

func weatherStationRainFactory(c *Client, deviceId string, device *Device, channelId string) Unit {
	ws := WeatherStationRainUnit{
		UnitData: c.unitDataFactory(deviceId, channelId, UntTypeWeatherStationRain),
	}

	for _, inOut := range device.Channels[channelId].Outputs {
		applyOutput(&ws, inOut)
	}

	return &ws
}
