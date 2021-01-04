package fahapi

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"time"
)

// change has to be bigger than that
//const luminanceLevel = 0.015  // logarithm change of lux
const temperatureLevel = 0.09 // °C

type WeatherStationTemperatureUnit struct {
	UnitData
	Temperature    float64
	TemperatureSet bool
	FreezeAlarm    bool
	FreezeAlarmSet bool
}

const UntTypeWeatherStationTemperature UnitTypeConst = "SeWTemp"

func CastWST(u Unit) *WeatherStationTemperatureUnit {
	if typeSave, ok := u.(*WeatherStationTemperatureUnit); ok {
		return typeSave
	}
	log.Print("CastWST - wrong type\n")
	return nil
}

func (ws *WeatherStationTemperatureUnit) String() string {
	alarm := ""
	if ws.FreezeAlarm {
		alarm = " Frost"
	}
	return fmt.Sprintf("%s Temperature %2.2f °C%s", ws.prtUnitHead(), ws.Temperature, alarm)
}

func (ws *WeatherStationTemperatureUnit) updateUnitFromOutDatapoint(outPut *InOutPut) bool {
	changed := false

	// f 43
	switch *outPut.PairingID {
	case 0x0004: // AL_SCENE_CONTROL (Recall or learn the set value related to encoded scene number)
	case 0x0026: // AL_FROST_ALARM
		freezeAlarm := *outPut.Value != "0"
		if freezeAlarm != ws.FreezeAlarm {
			ws.FreezeAlarm = freezeAlarm
			ws.FreezeAlarmSet = true
			changed = true
		}
	case 0x0400: // AL_OUTDOOR_TEMPERATURE
		temperature, _ := strconv.ParseFloat(*outPut.Value, 64)
		if math.Abs(ws.Temperature-temperature) >= temperatureLevel {
			if temperature == 0.0 && math.Abs(ws.Temperature) > 5.0 {
				log.Printf("Unplausible temp change: from %.2f°C to 0°C. Ignored.", ws.Temperature)
			} else {
				ws.Temperature = temperature
				ws.TemperatureSet = true
				changed = true
			}
		}
	}

	return changed
}

func (ws *WeatherStationTemperatureUnit) resetChanged() {
	ws.TemperatureSet = false
	ws.FreezeAlarmSet = false
}

func weatherStationTemperatureFactory(deviceId string, device *Device, channelId string) Unit {
	floor, room := GetFloorRoom(device, device.Channels[channelId])

	ws := WeatherStationTemperatureUnit{
		UnitData: UnitData{
			SerialNumber: deviceId,
			ChannelId:    channelId,
			Type:         "SeWTemp",
			Device:       device,
			Floor:        floor,
			Room:         room,
			LastUpdate:   time.Now(),
		},
	}

	for _, inOut := range device.Channels[channelId].Outputs {
		ws.updateUnitFromOutDatapoint(inOut)
	}

	return &ws
}
