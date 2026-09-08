package fahapi

import "testing"

// testSysAP builds a small installation: one switch actuator with a floor and
// room, one channel without a functionID, and one device that has a floor but
// no room.
func testSysAP() *SysAP {
	sysap := &SysAP{
		SysapName: ptr("test"),
		Devices: map[string]*Device{
			"DEV1": {
				DisplayName: ptr("Device One"),
				Floor:       ptr("01"),
				Room:        ptr("11"),
				Channels: map[string]*Channel{
					"ch0000": {
						DisplayName: ptr("Ceiling light"),
						FunctionID:  ptr(string(FID_SWITCH_ACTUATOR)),
						Outputs: map[string]*InOutPut{
							"odp0000": out(0x0100, "1"),
						},
					},
					"ch0001": {
						DisplayName: ptr("no function id"),
						Outputs:     map[string]*InOutPut{},
					},
				},
			},
			"DEV2": {
				DisplayName: ptr("Device Two"),
				Floor:       ptr("01"),
				Channels: map[string]*Channel{
					"ch0000": {
						DisplayName: ptr("Window"),
						FunctionID:  ptr(string(FID_WINDOW_DOOR_SENSOR)),
						Outputs: map[string]*InOutPut{
							"odp0000": out(0x0035, "1"),
						},
					},
				},
			},
		},
	}
	sysap.Floorplan.Floors = map[string]*Floors{
		"01": {Name: ptr("Ground floor"), Rooms: map[string]*Rooms{"11": {Name: ptr("Living room")}}},
	}
	return sysap
}

func hydrateTestSysAP(t *testing.T) {
	t.Helper()
	quietApi(t)
	SysAPConfiguration = testSysAP()
	FreeDevices = SysAPConfiguration.Devices
	hydrateAllDevices(FreeDevices)
}

func TestHydrationBuildsUnits(t *testing.T) {
	hydrateTestSysAP(t)

	if len(UnitMap) != 2 {
		t.Fatalf("hydrated %d units, want 2: %v", len(UnitMap), UnitMap)
	}

	unit := getUnit("DEV1", "ch0000")
	if unit == nil {
		t.Fatal("no unit for DEV1.ch0000")
	}
	sau := CastSAU(unit)
	if sau == nil {
		t.Fatal("DEV1.ch0000 is not a switch actuator")
	}
	if !sau.On {
		t.Error("initial output datapoint was not applied")
	}
	if sau.Floor != "Ground floor" || sau.Room != "Living room" {
		t.Errorf("floor/room = %q/%q, want Ground floor/Living room", sau.Floor, sau.Room)
	}
	if sau.DisplayName() != "Ceiling light" {
		t.Errorf("DisplayName = %q", sau.DisplayName())
	}
}

// A channel without a functionID exists on real hardware and must be skipped
// rather than crash the hydration.
func TestChannelWithoutFunctionIDIsSkipped(t *testing.T) {
	hydrateTestSysAP(t)

	if getUnit("DEV1", "ch0001") != nil {
		t.Error("channel without functionID produced a unit")
	}
}

// A device with a floor but no room used to panic: the early return only
// covers a missing floor.
func TestDeviceWithFloorButNoRoom(t *testing.T) {
	hydrateTestSysAP(t)

	unit := getUnit("DEV2", "ch0000")
	if unit == nil {
		t.Fatal("no unit for DEV2.ch0000")
	}
	data := unit.GetUnitData()
	if data.Floor != "Ground floor" {
		t.Errorf("Floor = %q, want Ground floor", data.Floor)
	}
	if data.Room != "-" {
		t.Errorf("Room = %q, want the placeholder for an unknown room", data.Room)
	}
}

func TestCastToWrongTypeReturnsNil(t *testing.T) {
	hydrateTestSysAP(t)

	if got := CastRTC(getUnit("DEV1", "ch0000")); got != nil {
		t.Errorf("CastRTC on a switch actuator returned %v, want nil", got)
	}
}

// A websocket update must reach the hydrated unit.
func TestWebsocketUpdateReachesUnit(t *testing.T) {
	hydrateTestSysAP(t)

	var reported []string
	wsUpdateUnitCallback = func(keys []string) { reported = append(reported, keys...) }

	var msg WebsocketMessage
	msg.ZeroSysAp.Datapoints = map[string]string{"DEV1/ch0000/odp0000": "0"}
	processWebsocketMessage(msg)

	if sau := CastSAU(getUnit("DEV1", "ch0000")); sau.On {
		t.Error("unit still on after the update said off")
	}
	if len(reported) != 1 || reported[0] != "DEV1.ch0000" {
		t.Errorf("callback got %v, want [DEV1.ch0000]", reported)
	}
}

// A malformed datapoint key must not take the process down.
func TestMalformedDatapointKeyIsSkipped(t *testing.T) {
	hydrateTestSysAP(t)

	var msg WebsocketMessage
	msg.ZeroSysAp.Datapoints = map[string]string{
		"this-is-not-a-valid-key": "1",
		"DEV1/ch0000/odp0000":     "0",
	}
	processWebsocketMessage(msg)

	if sau := CastSAU(getUnit("DEV1", "ch0000")); sau.On {
		t.Error("the valid datapoint next to the malformed one was not applied")
	}
}
