package lmu

import (
	"testing"
	"unsafe"
)

// These tests guard the byte-exact layout of the rF2 structs. The expected
// numbers were computed by hand from the plugin's field list (Windows amd64,
// natural alignment, pack 8). They run on any amd64 platform because Go's struct
// layout depends on the architecture, not the OS — so a transcription mistake
// (e.g. int64 where the plugin uses a 32-bit long, or a missing field) is caught
// here even though the real shared memory only exists on Windows.
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
		{"rf2Wheel", unsafe.Sizeof(rf2Wheel{}), 264},
		{"rf2VehicleTelemetry", unsafe.Sizeof(rf2VehicleTelemetry{}), 1920},
		{"rf2ScoringInfo", unsafe.Sizeof(rf2ScoringInfo{}), 560},
		{"rf2VehicleScoring", unsafe.Sizeof(rf2VehicleScoring{}), 592},
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
		{"VehicleTelemetry.mFuel", unsafe.Offsetof(vt.mFuel), 536},
		{"VehicleTelemetry.mWheels", unsafe.Offsetof(vt.mWheels), 864},
		{"VehicleScoring.mPos", unsafe.Offsetof(vs.mPos), 272},
		{"Telemetry.mVehicles", unsafe.Offsetof(tel.mVehicles), 16},
		{"Scoring.mScoringInfo", unsafe.Offsetof(sc.mScoringInfo), 16},
		{"Scoring.mVehicles", unsafe.Offsetof(sc.mVehicles), 576},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("offsetof(%s) = %d, want %d", c.name, c.got, c.want)
		}
	}
}
