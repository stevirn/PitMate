// This file translates the raw rF2 shared-memory structs into PitMate's
// game-agnostic telemetry.Frame. It is the LMU-specific "translation" half of
// the adapter and is deliberately PURE: given two raw buffers it returns a
// Frame, with no I/O and no OS calls. That means it builds and is fully unit
// tested on any platform (see mapping_test.go), which matters because the rest
// of the adapter only runs on Windows.
//
// Where rF2 simply does not expose something PitMate's model has a field for,
// the field is left at its zero value and marked with a TODO. Several mappings
// involve rF2 enum values whose exact meaning should be confirmed against a live
// LMU session; those are marked too.
package lmu

import (
	"bytes"
	"math"
	"strconv"

	"github.com/stevirn/PitMate/telemetry"
)

const kelvinToCelsius = 273.15

// mapFrame builds a telemetry.Frame from the raw telemetry and scoring buffers.
func mapFrame(tel *rf2Telemetry, sc *rf2Scoring) telemetry.Frame {
	info := &sc.mScoringInfo
	trackLen := info.mLapDist.val()

	// Index telemetry vehicles by slot ID so we can attach physics data to the
	// matching scoring entry (the player, primarily).
	telByID := make(map[int32]*rf2VehicleTelemetry)
	for i := 0; i < int(tel.mNumVehicles) && i < maxMappedVehicles; i++ {
		v := &tel.mVehicles[i]
		telByID[v.mID] = v
	}

	// Precompute place->vehicle and class ordering so we can derive class
	// positions and the gap to the car behind (rF2 only gives gap to the car
	// ahead directly).
	n := int(info.mNumVehicles)
	if n > maxMappedVehicles {
		n = maxMappedVehicles
	}
	byPlace := make(map[uint8]*rf2VehicleScoring, n)
	for i := 0; i < n; i++ {
		byPlace[sc.mVehicles[i].mPlace] = &sc.mVehicles[i]
	}

	var player telemetry.Car
	var competitors []telemetry.Car
	havePlayer := false

	for i := 0; i < n; i++ {
		vs := &sc.mVehicles[i]
		car := mapCar(vs, telByID[vs.mID], info, trackLen)

		// Class position: rank within the same class by overall place.
		car.Race.PositionInClass = classPosition(sc, n, vs.mVehicleClass[:], vs.mPlace)

		// Gap to the car behind = that car's gap-to-next (it trails us).
		if behind, ok := byPlace[vs.mPlace+1]; ok {
			car.Race.GapBehindSeconds = behind.mTimeBehindNext.val()
		}

		if vs.mIsPlayer != 0 {
			player = car
			havePlayer = true
		} else {
			competitors = append(competitors, car)
		}
	}

	frame := telemetry.Frame{
		// Timestamp and Sequence are stamped by the server on Broadcast.
		Source: telemetry.SourceInfo{
			Game:      "Le Mans Ultimate",
			AdapterID: "lmu",
			Connected: true,
		},
		Session:     mapSession(info),
		Competitors: competitors,
		RaceControl: mapRaceControl(info),
		Events:      nil, // TODO: derive events (contacts, pit stops) from frame-to-frame changes
	}
	if havePlayer {
		frame.Player = player
	}
	return frame
}

// mapSession fills session-wide info from rF2 scoring info.
func mapSession(info *rf2ScoringInfo) telemetry.SessionInfo {
	endET := info.mEndET.val()
	timed := endET > 0
	remaining := 0.0
	if timed {
		remaining = endET - info.mCurrentET.val()
	}
	totalLaps := 0
	if !timed && info.mMaxLaps > 0 && info.mMaxLaps < 100000 {
		totalLaps = int(info.mMaxLaps)
	}

	return telemetry.SessionInfo{
		Type:             mapSessionType(info.mSession),
		TrackName:        cstr(info.mTrackName[:]),
		TrackLengthM:     info.mLapDist.val(),
		ElapsedSeconds:   info.mCurrentET.val(),
		RemainingSeconds: remaining,
		IsTimed:          timed,
		TotalLaps:        totalLaps,
		Conditions: telemetry.Conditions{
			AirTempC:         info.mAmbientTemp.val(),
			TrackTempC:       info.mTrackTemp.val(),
			TrackWetness:     info.mAvgPathWetness.val(),
			RainIntensity:    info.mRaining.val(),
			WindSpeedKph:     vec3Magnitude(info.mWind) * 3.6,
			WindDirectionDeg: windDirectionDeg(info.mWind),
		},
	}
}

// mapSessionType maps rF2's numeric session id to PitMate's session type.
// rF2 ids: 0=test day, 1-4=practice, 5-8=qualifying, 9=warmup, 10-13=race.
func mapSessionType(s int32) telemetry.SessionType {
	switch {
	case s == 0 || (s >= 1 && s <= 4) || s == 9:
		return telemetry.SessionPractice
	case s >= 5 && s <= 8:
		return telemetry.SessionQualifying
	case s >= 10 && s <= 13:
		return telemetry.SessionRace
	default:
		return telemetry.SessionUnknown
	}
}

// mapCar builds one Car. vs (scoring) is always present; vt (telemetry/physics)
// is present for the player and any nearby car the plugin reports, otherwise nil.
func mapCar(vs *rf2VehicleScoring, vt *rf2VehicleTelemetry, info *rf2ScoringInfo, trackLen float64) telemetry.Car {
	car := telemetry.Car{
		Identity: telemetry.Identity{
			CarID:      strconv.Itoa(int(vs.mID)),
			CarClass:   cstr(vs.mVehicleClass[:]),
			CarModel:   cstr(vs.mVehicleName[:]),
			DriverName: cstr(vs.mDriverName[:]),
			IsPlayer:   vs.mIsPlayer != 0,
			// CarNumber: not exposed by rF2 shared memory.
			// TODO: derive car number (e.g. parse from vehicle name/livery).
		},
		Timing: mapTiming(vs, trackLen),
		Race:   mapRaceState(vs),
	}

	// Physics-derived sections require the telemetry buffer.
	if vt != nil {
		car.Energy = mapEnergy(vt)
		car.Tires = mapTires(vt)
		car.Speed = mapSpeed(vt)
		car.Systems = mapSystems(vt)
		car.Damage = mapDamage(vt)
	}
	return car
}

// mapTiming fills lap counts and lap/sector times from scoring.
func mapTiming(vs *rf2VehicleScoring, trackLen float64) telemetry.Timing {
	t := telemetry.Timing{
		LapsCompleted:       int(vs.mTotalLaps),
		CurrentLapSeconds:   vs.mTimeIntoLap.val(),
		EstimatedLapSeconds: vs.mEstimatedLapTime.val(),
	}
	if trackLen > 0 {
		t.LapFraction = clamp01(vs.mLapDist.val() / trackLen)
	}
	if last := vs.mLastLapTime.val(); last > 0 {
		t.LastLapSeconds = last
		t.LastSectors = splitSectors(vs.mLastSector1.val(), vs.mLastSector2.val(), last)
	}
	if best := vs.mBestLapTime.val(); best > 0 {
		t.BestLapSeconds = best
		t.BestSectors = splitSectors(vs.mBestSector1.val(), vs.mBestSector2.val(), best)
	}
	// MiniSectors: not exposed by rF2 shared memory. // TODO: verify LMU availability
	return t
}

// splitSectors converts rF2's cumulative sector splits (s1, s1+s2, lap) into
// per-sector durations [s1, s2, s3]. Returns nil if the data looks incomplete.
func splitSectors(cumS1, cumS2, lap float64) []float64 {
	if cumS1 <= 0 || cumS2 <= cumS1 || lap <= cumS2 {
		return nil
	}
	return []float64{cumS1, cumS2 - cumS1, lap - cumS2}
}

// mapEnergy fills fuel and hybrid state from telemetry.
func mapEnergy(vt *rf2VehicleTelemetry) telemetry.Energy {
	e := telemetry.Energy{
		FuelLitres:           vt.mFuel.val(),
		FuelCapacityLitres:   vt.mFuelCapacity.val(),
		HasHybrid:            vt.mElectricBoostMotorState != 0, // 0 = unavailable
		HybridChargeFraction: clamp01(vt.mBatteryChargeFraction.val()),
		HybridMode:           hybridModeName(vt.mElectricBoostMotorState),
		// FuelPerLastLap / FuelPerLapAvg / FuelLapsRemaining require tracking
		// fuel across laps; the adapter does not keep history yet.
		// TODO: compute fuel-per-lap from frame-to-frame fuel deltas.
	}
	return e
}

// hybridModeName maps the electric boost motor state to a short label.
func hybridModeName(state uint8) string {
	switch state {
	case 1:
		return "inactive"
	case 2:
		return "deploy"
	case 3:
		return "regen"
	default:
		return ""
	}
}

// mapTires fills all four corners. rF2 wheel order is FL, FR, RL, RR.
func mapTires(vt *rf2VehicleTelemetry) telemetry.Tires {
	return telemetry.Tires{
		Compound:   cstr(vt.mFrontTireCompoundName[:]),
		FrontLeft:  mapTireCorner(&vt.mWheels[0]),
		FrontRight: mapTireCorner(&vt.mWheels[1]),
		RearLeft:   mapTireCorner(&vt.mWheels[2]),
		RearRight:  mapTireCorner(&vt.mWheels[3]),
	}
}

// mapTireCorner converts one rF2 wheel, translating Kelvin temps to Celsius.
func mapTireCorner(w *rf2Wheel) telemetry.TireCorner {
	return telemetry.TireCorner{
		TempC: kToC(avg3(w.mTemperature)),
		TempBandsC: []float64{
			kToC(w.mTemperature[0].val()), // left
			kToC(w.mTemperature[1].val()), // center
			kToC(w.mTemperature[2].val()), // right
		},
		CoreTempC:    kToC(w.mTireCarcassTemperature.val()),
		PressureKpa:  w.mPressure.val(),
		WearFraction: clamp01(w.mWear.val()),
		BrakeTempC:   w.mBrakeTemp.val(), // already Celsius in rF2
		// Locking: not directly exposed. // TODO: verify LMU availability
	}
}

// mapSpeed fills speeds from telemetry. Current speed is the magnitude of the
// local velocity vector.
func mapSpeed(vt *rf2VehicleTelemetry) telemetry.Speed {
	return telemetry.Speed{
		CurrentKph: vec3Magnitude(vt.mLocalVel) * 3.6,
		Gear:       int(vt.mGear),
		RPM:        vt.mEngineRPM.val(),
		// TopKph / TrapKph need history or a speed trap; not tracked yet.
		// TODO: track session top speed in the adapter.
	}
}

// mapSystems fills adjustable/monitored systems from telemetry.
func mapSystems(vt *rf2VehicleTelemetry) telemetry.Systems {
	return telemetry.Systems{
		BrakeBiasFrontPct: (1.0 - vt.mRearBrakeBias.val()) * 100.0,
		OilTempC:          vt.mEngineOilTemp.val(),
		WaterTempC:        vt.mEngineWaterTemp.val(),
		LightsOn:          vt.mHeadlights != 0,
		PitLimiterOn:      vt.mSpeedLimiter != 0,
		// TC / ABS / EngineMap / ARBs are not in the telemetry buffer.
		// TODO: source these from the plugin's PitInfo/Extended buffers.
	}
}

// mapDamage fills what damage info rF2 shared memory exposes. rF2 reports a
// coarse dent severity at 8 body locations (0/1/2); detailed per-zone aero
// damage and performance deltas are not exposed.
func mapDamage(vt *rf2VehicleTelemetry) telemetry.Damage {
	// Represent the worst dent as front/rear wing proxies until a better source
	// is available. Each severity is 0,1,2 -> 0.0,0.5,1.0.
	sev := func(i int) float64 { return float64(vt.mDentSeverity[i]) / 2.0 }
	return telemetry.Damage{
		FrontWing: sev(0),
		RearWing:  sev(4),
		// Detailed zones, downforce/aero loss and top-speed delta are not exposed.
		// TODO: verify LMU availability / derive from dent severity if possible.
	}
}

// mapRaceState fills position, gaps, and pit status from scoring. Gap-behind is
// filled by the caller (it needs the neighbouring car).
func mapRaceState(vs *rf2VehicleScoring) telemetry.RaceState {
	return telemetry.RaceState{
		PositionOverall:    int(vs.mPlace),
		GapToLeaderSeconds: vs.mTimeBehindLeader.val(),
		GapAheadSeconds:    vs.mTimeBehindNext.val(),
		PitStatus:          mapPitStatus(vs.mPitState),
		PitStopCount:       int(vs.mNumPitstops),
		InPitLane:          vs.mInPits != 0,
		Lapped:             vs.mLapsBehindLeader > 0,
	}
}

// mapPitStatus maps rF2's pit state to PitMate's pit status.
// rF2: 0=none,1=request,2=entering,3=stopped,4=exiting.
func mapPitStatus(s uint8) telemetry.PitStatus {
	switch s {
	case 1, 2:
		return telemetry.PitApproach
	case 3:
		return telemetry.PitStopped
	case 4:
		return telemetry.PitExit
	default:
		return telemetry.PitNone
	}
}

// classPosition returns the 1-based position of place within its class.
func classPosition(sc *rf2Scoring, n int, classBytes []byte, place uint8) int {
	class := cstr(classBytes)
	rank := 1
	for i := 0; i < n; i++ {
		other := &sc.mVehicles[i]
		if cstr(other.mVehicleClass[:]) == class && other.mPlace < place {
			rank++
		}
	}
	return rank
}

// mapRaceControl derives the session flag and safety-car state. The rF2 enum
// values below are the commonly-documented ones; confirm against a live LMU
// session.
//
// mGamePhase: 0=garage,1=warmup,2=grid,3=formation,4=countdown,5=green,
// 6=full course yellow,7=red/stopped,8=session over.
func mapRaceControl(info *rf2ScoringInfo) telemetry.RaceControl {
	rc := telemetry.RaceControl{
		CurrentFlag: telemetry.FlagNone,
		SafetyCar:   telemetry.SafetyCarNone,
	}
	switch info.mGamePhase {
	case 5:
		rc.CurrentFlag = telemetry.FlagGreen
	case 6:
		rc.CurrentFlag = telemetry.FlagYellow
		rc.SafetyCar = telemetry.SafetyCarFCY
	case 7:
		rc.CurrentFlag = telemetry.FlagRed
	case 8:
		rc.CurrentFlag = telemetry.FlagChequered
	}
	// A yellow flag state can be active without a full-course phase.
	if rc.CurrentFlag == telemetry.FlagNone && info.mYellowFlagState > 0 {
		rc.CurrentFlag = telemetry.FlagYellow
	}
	return rc
}

// --- small helpers ---

// cstr converts a fixed-size, NUL-terminated C char array to a Go string.
func cstr(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		return string(b[:i])
	}
	return string(b)
}

func kToC(k float64) float64 { return k - kelvinToCelsius }

func avg3(a [3]rf2f64) float64 { return (a[0].val() + a[1].val() + a[2].val()) / 3.0 }

func vec3Magnitude(v rf2Vec3) float64 {
	x, y, z := v.x.val(), v.y.val(), v.z.val()
	return math.Sqrt(x*x + y*y + z*z)
}

// windDirectionDeg returns the wind heading in degrees (0-360, 0 = +Z/North),
// derived from the horizontal components of the wind vector.
func windDirectionDeg(v rf2Vec3) float64 {
	deg := math.Atan2(v.x.val(), v.z.val()) * 180.0 / math.Pi
	if deg < 0 {
		deg += 360
	}
	return deg
}

func clamp01(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}
