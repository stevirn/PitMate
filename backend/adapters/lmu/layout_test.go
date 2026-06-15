package lmu

import (
	"testing"
	"unsafe"
)

// These tests guard the byte-exact layout of the rF2 structs. The expected
// numbers were computed by hand from the plugin's field list under its
// #pragma pack(4) — i.e. doubles aligned to 4, not 8 (which is why the structs
// use rf2f64 instead of float64; see rf2_structs.go). They run on any amd64
// platform because Go's struct layout depends on the architecture, not the OS,
// so a transcription/packing mistake is caught here even though the real shared
// memory only exists on Windows. The live game's buffer size is the final proof
// (the Windows reader logs it on connect).
//
// If the game ever shows garbage telemetry, re-verify these against the current
// plugin headers first.

func TestRF2StructSizes(t *testing.T) {
	cases := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"rf2Vec3", unsafe.Sizeof(rf2Vec3{}), 24},
		{"rf2Wheel", unsafe.Sizeof(rf2Wheel{}), 260},
		{"rf2VehicleTelemetry", unsafe.Sizeof(rf2VehicleTelemetry{}), 1888},
		{"rf2ScoringInfo", unsafe.Sizeof(rf2ScoringInfo{}), 548},
		{"rf2VehicleScoring", unsafe.Sizeof(rf2VehicleScoring{}), 584},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("sizeof(%s) = %d, want %d", c.name, c.got, c.want)
		}
	}
}

func TestRF2FieldOffsets(t *testing.T) {
	var vt rf2VehicleTelemetry
	var vs rf2VehicleScoring
	var tel rf2Telemetry
	var sc rf2Scoring

	cases := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"VehicleTelemetry.mFuel", unsafe.Offsetof(vt.mFuel), 524},
		{"VehicleTelemetry.mWheels", unsafe.Offsetof(vt.mWheels), 848},
		{"VehicleScoring.mPos", unsafe.Offsetof(vs.mPos), 264},
		{"Telemetry.mVehicles", unsafe.Offsetof(tel.mVehicles), 16},
		{"Scoring.mScoringInfo", unsafe.Offsetof(sc.mScoringInfo), 12},
		{"Scoring.mVehicles", unsafe.Offsetof(sc.mVehicles), 560},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("offsetof(%s) = %d, want %d", c.name, c.got, c.want)
		}
	}
}
