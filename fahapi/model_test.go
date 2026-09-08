package fahapi

import "testing"

// Every unit type reacts to its own pairing IDs. The table drives one datapoint
// into a fresh unit and checks what it made of it.
func TestUpdateUnitFromOutDatapoint(t *testing.T) {
	quietApi(t)

	tests := []struct {
		name        string
		unit        Unit
		out         *InOutPut
		wantChanged bool
		check       func(t *testing.T, u Unit)
	}{
		{
			name: "switch sensor on",
			unit: &SwitchSensorUnit{}, out: out(0x0001, "1"), wantChanged: true,
			check: func(t *testing.T, u Unit) {
				s := u.(*SwitchSensorUnit)
				if !s.On || !s.OnSet {
					t.Errorf("On=%v OnSet=%v, want both true", s.On, s.OnSet)
				}
			},
		},
		{
			name: "switch sensor unhandled pairing id",
			unit: &SwitchSensorUnit{}, out: out(0x0004, "1"), wantChanged: false,
		},
		{
			name: "switch actuator forced",
			unit: &SwitchActuatorUnit{}, out: out(0x0101, "2"), wantChanged: true,
			check: func(t *testing.T, u Unit) {
				if s := u.(*SwitchActuatorUnit); !s.Force || !s.ForceSet {
					t.Errorf("Force=%v ForceSet=%v, want both true", s.Force, s.ForceSet)
				}
			},
		},
		{
			name: "dimming actuator value",
			unit: &DimmingActuatorUnit{}, out: out(0x0110, "42"), wantChanged: true,
			check: func(t *testing.T, u Unit) {
				if d := u.(*DimmingActuatorUnit); d.DimmingValue != 42 {
					t.Errorf("DimmingValue=%d, want 42", d.DimmingValue)
				}
			},
		},
		{
			name: "window sensor open",
			unit: &WindowDoorSensorUnit{}, out: out(0x0035, "1"), wantChanged: true,
			check: func(t *testing.T, u Unit) {
				if w := u.(*WindowDoorSensorUnit); !w.Open || !w.OpenSet {
					t.Errorf("Open=%v OpenSet=%v, want both true", w.Open, w.OpenSet)
				}
			},
		},
		{
			name: "rtc measured temperature",
			unit: &RoomTemperatureControllerUnit{}, out: out(0x0130, "21.5"), wantChanged: true,
			check: func(t *testing.T, u Unit) {
				if r := u.(*RoomTemperatureControllerUnit); r.ActualDegree != 21.5 {
					t.Errorf("ActualDegree=%v, want 21.5", r.ActualDegree)
				}
			},
		},
		{
			name: "rtc device error is reported, not fatal",
			unit: &RoomTemperatureControllerUnit{}, out: out(0x0111, "17"), wantChanged: false,
		},
		{
			name: "weather temperature",
			unit: &WeatherStationTemperatureUnit{}, out: out(0x0400, "18.50"), wantChanged: true,
			check: func(t *testing.T, u Unit) {
				if w := u.(*WeatherStationTemperatureUnit); w.Temperature != 18.5 {
					t.Errorf("Temperature=%v, want 18.5", w.Temperature)
				}
			},
		},
		{
			name: "weather wind speed",
			unit: &WeatherStationWindUnit{}, out: out(0x0404, "3.25"), wantChanged: true,
			check: func(t *testing.T, u Unit) {
				if w := u.(*WeatherStationWindUnit); w.Wind != 3.25 {
					t.Errorf("Wind=%v, want 3.25", w.Wind)
				}
			},
		},
		{
			name: "weather rain alarm",
			unit: &WeatherStationRainUnit{}, out: out(0x0027, "1"), wantChanged: true,
			check: func(t *testing.T, u Unit) {
				if w := u.(*WeatherStationRainUnit); !w.RainAlarm {
					t.Error("RainAlarm=false, want true")
				}
			},
		},
		{
			name: "weather brightness",
			unit: &WeatherStationBrightnessUnit{}, out: out(0x0403, "1200"), wantChanged: true,
			check: func(t *testing.T, u Unit) {
				if w := u.(*WeatherStationBrightnessUnit); w.Luminance != 1200 {
					t.Errorf("Luminance=%v, want 1200", w.Luminance)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := applyOutput(tc.unit, tc.out); got != tc.wantChanged {
				t.Fatalf("applyOutput = %v, want %v", got, tc.wantChanged)
			}
			if tc.check != nil {
				tc.check(t, tc.unit)
			}
		})
	}
}

// An unchanged value must not be reported as an update, otherwise every
// refresh would look like a real change.
func TestRepeatedValueIsNotAChange(t *testing.T) {
	quietApi(t)

	u := &SwitchSensorUnit{}
	if !applyOutput(u, out(0x0001, "1")) {
		t.Fatal("first update should change the unit")
	}
	u.resetChanged()
	if applyOutput(u, out(0x0001, "1")) {
		t.Error("same value reported as a change")
	}
	if u.OnSet {
		t.Error("OnSet still set after resetChanged")
	}
}

// A malformed value must be rejected. Silently turning it into 0 used to pass
// it on as a genuine measurement.
func TestMalformedValuesAreRejected(t *testing.T) {
	quietApi(t)

	t.Run("temperature", func(t *testing.T) {
		u := &WeatherStationTemperatureUnit{Temperature: 18.5}
		if applyOutput(u, out(0x0400, "")) {
			t.Error("empty value accepted")
		}
		if u.Temperature != 18.5 {
			t.Errorf("Temperature=%v, want the previous 18.5", u.Temperature)
		}
	})

	t.Run("dimming value", func(t *testing.T) {
		u := &DimmingActuatorUnit{DimmingValue: 30}
		if applyOutput(u, out(0x0110, "nonsense")) {
			t.Error("non-numeric value accepted")
		}
		if u.DimmingValue != 30 {
			t.Errorf("DimmingValue=%d, want the previous 30", u.DimmingValue)
		}
	})
}

// applyOutput absorbs the optional fields of the API model so no unit
// implementation has to.
func TestApplyOutputRejectsIncompleteDatapoints(t *testing.T) {
	quietApi(t)

	pairing := 0x0001
	value := "1"

	cases := map[string]*InOutPut{
		"nil datapoint": nil,
		"nil pairingID": {Value: &value},
		"nil value":     {PairingID: &pairing},
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			if applyOutput(&SwitchSensorUnit{}, in) {
				t.Error("incomplete datapoint accepted")
			}
		})
	}

	t.Run("nil unit", func(t *testing.T) {
		if applyOutput(nil, out(0x0001, "1")) {
			t.Error("nil unit accepted")
		}
	})
}

// String must survive a unit whose channel or display name is missing.
func TestStringWithoutChannel(t *testing.T) {
	quietApi(t)

	for _, u := range []Unit{
		&SwitchSensorUnit{}, &SwitchActuatorUnit{}, &DimmingSensorUnit{},
		&DimmingActuatorUnit{}, &WindowDoorSensorUnit{},
		&RoomTemperatureControllerUnit{}, &WeatherStationWindUnit{},
		&WeatherStationRainUnit{}, &WeatherStationTemperatureUnit{},
		&WeatherStationBrightnessUnit{},
	} {
		if u.String() == "" {
			t.Errorf("%T.String() returned empty", u)
		}
	}
}
