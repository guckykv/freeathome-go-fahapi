package fahapi

import (
	"fmt"
	"log"
	"strconv"
)

type WeatherStationWindUnit struct {
	UnitData
	Wind         float64
	WindSet      bool
	WindAlarm    bool
	WindAlarmSet bool
	WindForce    float64
	WindForceSet bool
}

const UntTypeWeatherStationWind UnitTypeConst = "SeWWind"

func CastWSW(u Unit) *WeatherStationWindUnit {
	if typeSave, ok := u.(*WeatherStationWindUnit); ok {
		return typeSave
	}
	log.Print("CastWSW - wrong type\n")
	return nil
}

func (ws *WeatherStationWindUnit) String() string {
	alarm := ""
	if ws.WindAlarm {
		alarm = " Sturm"
	}
	return fmt.Sprintf("%s Wind %.2f m/s %.2f force%s", ws.prtUnitHead(), ws.Wind, ws.WindForce, alarm)
}

func (ws *WeatherStationWindUnit) updateUnitFromOutDatapoint(outPut *InOutPut) bool {
	changed := false

	// f 44
	switch *outPut.PairingID {
	case 0x0004: // AL_SCENE_CONTROL (Recall or learn the set value related to encoded scene number)
	case 0x0025: // AL_WIND_ALARM
		alarm := *outPut.Value != "0"
		if alarm != ws.WindAlarm {
			ws.WindAlarm = alarm
			ws.WindAlarmSet = true
			changed = true
		}
	case 0x0401: // AL_WIND_FORCE
		windForce, _ := strconv.ParseFloat(*outPut.Value, 64)
		if windForce != ws.WindForce {
			ws.WindForce = windForce
			ws.WindForceSet = true
			changed = true
		}
	case 0x0404: // AL_WIND_SPEED
		wind, _ := strconv.ParseFloat(*outPut.Value, 64)
		if wind != ws.Wind {
			ws.Wind = wind
			ws.WindSet = true
			changed = true
		}
	}

	return changed
}

func (ws *WeatherStationWindUnit) resetChanged() {
	ws.WindSet = false
	ws.WindAlarmSet = false
	ws.WindForceSet = false
}

func weatherStationWindFactory(deviceId string, device *Device, channelId string) Unit {
	ws := WeatherStationWindUnit{
		UnitData: unitDataFactory(deviceId, channelId, UntTypeWeatherStationWind),
	}

	for _, inOut := range device.Channels[channelId].Outputs {
		ws.updateUnitFromOutDatapoint(inOut)
	}

	return &ws
}
