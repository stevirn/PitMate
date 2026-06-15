package lmu

import (
	"math"
	"testing"

	"github.com/stevirn/PitMate/telemetry"
)

// setStr copies a Go string into a fixed C char array slice, leaving the rest as
// NUL bytes (the array starts zeroed in a fresh struct).
func setStr(dst []byte, s string) { copy(dst, s) }

// d wraps a float64 as the packed rf2f64 wire type, for building fixtures.
func d(f float64) rf2f64 {
	bits := math.Float64bits(f)
	return rf2f64{uint32(bits), uint32(bits >> 32)}
}

func approx(t *testing.T, label string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("%s = %v, want %v", label, got, want)
	}
}

// buildFixture creates a small but realistic 3-car race snapshot:
//
//	place 1: leader   (Hypercar)
//	place 2: player   (Hypercar)  <- full telemetry attached
//	place 3: GT car   (LMGT3)
func buildFixture() (*rf2Telemetry, *rf2Scoring) {
	sc := &rf2Scoring{}
	info := &sc.mScoringInfo
	setStr(info.mTrackName[:], "Le Mans 24h")
	info.mSession = 10 // race
	info.mCurrentET = d(3600)
	info.mEndET = d(86400)
	info.mLapDist = d(13626) // track length, meters
	info.mNumVehicles = 3
	info.mAmbientTemp = d(22)
	info.mTrackTemp = d(30)
	info.mRaining = d(0)
	info.mAvgPathWetness = d(0)
	info.mGamePhase = 5 // green

	// place 1 — leader
	v0 := &sc.mVehicles[0]
	v0.mID = 11
	v0.mPlace = 1
	setStr(v0.mDriverName[:], "Leader")
	setStr(v0.mVehicleClass[:], "Hypercar")

	// place 2 — player
	v1 := &sc.mVehicles[1]
	v1.mID = 10
	v1.mPlace = 2
	v1.mIsPlayer = 1
	setStr(v1.mDriverName[:], "Me")
	setStr(v1.mVehicleName[:], "Toyota GR010")
	setStr(v1.mVehicleClass[:], "Hypercar")
	v1.mTotalLaps = 5
	v1.mLapDist = d(6813) // half the lap -> LapFraction 0.5
	v1.mTimeIntoLap = d(100)
	v1.mLastLapTime = d(210)
	v1.mLastSector1 = d(62)
	v1.mLastSector2 = d(140) // cumulative -> sectors [62, 78, 70]
	v1.mBestLapTime = d(208)
	v1.mTimeBehindLeader = d(4.5)
	v1.mTimeBehindNext = d(2.1)
	v1.mNumPitstops = 1

	// place 3 — GT car
	v2 := &sc.mVehicles[2]
	v2.mID = 12
	v2.mPlace = 3
	setStr(v2.mDriverName[:], "GT Driver")
	setStr(v2.mVehicleClass[:], "LMGT3")
	v2.mTimeBehindNext = d(1.4) // trails the player by 1.4s

	// Telemetry: only the player's car has physics data.
	tel := &rf2Telemetry{}
	tel.mNumVehicles = 1
	pt := &tel.mVehicles[0]
	pt.mID = 10
	pt.mFuel = d(50)
	pt.mFuelCapacity = d(80)
	pt.mGear = 4
	pt.mEngineRPM = d(7000)
	pt.mLocalVel = rf2Vec3{x: d(0), y: d(0), z: d(-55.5)} // 55.5 m/s -> 199.8 km/h
	pt.mRearBrakeBias = d(0.45)                           // -> 55% front
	pt.mEngineOilTemp = d(105)
	pt.mEngineWaterTemp = d(88)
	pt.mHeadlights = 1
	pt.mElectricBoostMotorState = 2 // deploy
	pt.mBatteryChargeFraction = d(0.6)
	setStr(pt.mFrontTireCompoundName[:], "Medium")
	fl := &pt.mWheels[0]
	fl.mTemperature = [3]rf2f64{d(350), d(360), d(370)} // Kelvin, avg 360 -> 86.85 C
	fl.mPressure = d(165)
	fl.mWear = d(0.9)
	fl.mBrakeTemp = d(300) // already Celsius
	fl.mTireCarcassTemperature = d(380)

	return tel, sc
}

func TestMapFrame(t *testing.T) {
	tel, sc := buildFixture()
	f := mapFrame(tel, sc)

	// Source / session
	if !f.Source.Connected || f.Source.AdapterID != "lmu" {
		t.Errorf("source = %+v", f.Source)
	}
	if f.Session.Type != telemetry.SessionRace {
		t.Errorf("session type = %v, want race", f.Session.Type)
	}
	if f.Session.TrackName != "Le Mans 24h" {
		t.Errorf("track = %q", f.Session.TrackName)
	}
	if !f.Session.IsTimed {
		t.Error("expected timed session")
	}
	approx(t, "remaining", f.Session.RemainingSeconds, 82800)
	approx(t, "airTemp", f.Session.Conditions.AirTempC, 22)

	// Player identity & race position
	p := f.Player
	if p.Identity.DriverName != "Me" || !p.Identity.IsPlayer {
		t.Errorf("player identity = %+v", p.Identity)
	}
	if p.Race.PositionOverall != 2 || p.Race.PositionInClass != 2 {
		t.Errorf("player pos overall=%d inclass=%d, want 2/2", p.Race.PositionOverall, p.Race.PositionInClass)
	}
	approx(t, "gapToLeader", p.Race.GapToLeaderSeconds, 4.5)
	approx(t, "gapAhead", p.Race.GapAheadSeconds, 2.1)
	approx(t, "gapBehind", p.Race.GapBehindSeconds, 1.4)
	if p.Race.PitStopCount != 1 {
		t.Errorf("pit stops = %d, want 1", p.Race.PitStopCount)
	}

	// Timing
	if p.Timing.LapsCompleted != 5 {
		t.Errorf("laps = %d, want 5", p.Timing.LapsCompleted)
	}
	approx(t, "lapFraction", p.Timing.LapFraction, 0.5)
	approx(t, "lastLap", p.Timing.LastLapSeconds, 210)
	if len(p.Timing.LastSectors) != 3 {
		t.Fatalf("last sectors = %v", p.Timing.LastSectors)
	}
	approx(t, "s1", p.Timing.LastSectors[0], 62)
	approx(t, "s2", p.Timing.LastSectors[1], 78)
	approx(t, "s3", p.Timing.LastSectors[2], 70)

	// Energy
	approx(t, "fuel", p.Energy.FuelLitres, 50)
	if !p.Energy.HasHybrid || p.Energy.HybridMode != "deploy" {
		t.Errorf("hybrid = %+v", p.Energy)
	}
	approx(t, "battery", p.Energy.HybridChargeFraction, 0.6)

	// Tires (Kelvin -> Celsius conversion)
	if p.Tires.Compound != "Medium" {
		t.Errorf("compound = %q", p.Tires.Compound)
	}
	approx(t, "FL tempC", p.Tires.FrontLeft.TempC, 86.85)
	approx(t, "FL coreC", p.Tires.FrontLeft.CoreTempC, 106.85)
	approx(t, "FL pressure", p.Tires.FrontLeft.PressureKpa, 165)
	approx(t, "FL wear", p.Tires.FrontLeft.WearFraction, 0.9)
	approx(t, "FL brake", p.Tires.FrontLeft.BrakeTempC, 300)

	// Speed & systems
	approx(t, "speed", p.Speed.CurrentKph, 199.8)
	if p.Speed.Gear != 4 {
		t.Errorf("gear = %d", p.Speed.Gear)
	}
	approx(t, "brakeBiasFront", p.Systems.BrakeBiasFrontPct, 55)
	approx(t, "oilTemp", p.Systems.OilTempC, 105)
	if !p.Systems.LightsOn {
		t.Error("expected lights on")
	}

	// Competitors and class position of the GT car
	if len(f.Competitors) != 2 {
		t.Fatalf("competitors = %d, want 2", len(f.Competitors))
	}
	var gt *telemetry.Car
	for i := range f.Competitors {
		if f.Competitors[i].Identity.CarID == "12" {
			gt = &f.Competitors[i]
		}
	}
	if gt == nil {
		t.Fatal("GT car (id 12) not found among competitors")
	}
	if gt.Race.PositionInClass != 1 {
		t.Errorf("GT class position = %d, want 1 (only car in LMGT3)", gt.Race.PositionInClass)
	}

	// Race control
	if f.RaceControl.CurrentFlag != telemetry.FlagGreen {
		t.Errorf("flag = %v, want green", f.RaceControl.CurrentFlag)
	}
}

func TestMapFrameDisconnectedSafety(t *testing.T) {
	// An all-zero scoring/telemetry pair (no vehicles) must not panic and must
	// produce an empty-but-valid frame.
	f := mapFrame(&rf2Telemetry{}, &rf2Scoring{})
	if len(f.Competitors) != 0 {
		t.Errorf("expected no competitors, got %d", len(f.Competitors))
	}
}
