// This file mirrors the in-memory layout that the rFactor2 Shared Memory Map
// Plugin (by The Iron Wolf) writes, and that Le Mans Ultimate uses unchanged.
// LMU is built on the rFactor 2 engine and reads/writes the SAME plugin, so the
// memory layout below is the rF2 layout.
//
// The structs are a FIELD-FOR-FIELD, byte-exact translation of the plugin's C++
// headers, cross-checked against the reference Python reader
// (TonyWhitley/pyRfactor2SharedMemory). They MUST stay byte-exact: the Windows
// reader casts raw mapped memory straight onto these structs (see
// reader_windows.go), so any wrong field type, order, or size silently corrupts
// every value after it.
//
// Important layout facts baked in here:
//   - Target is 64-bit Windows (the plugin DLL is 64-bit), amd64 Go.
//   - C++ `long`/`unsigned long` are 32-bit on Windows (LLP64) -> int32/uint32.
//   - C++ `bool` and `unsigned char` are 1 byte -> uint8; `signed char` -> int8.
//   - **The plugin compiles these structs under `#pragma pack(push, 4)`** — 4-byte
//     packing. That matters enormously: under pack(4) a `double` is aligned to 4
//     bytes, but Go ALWAYS aligns float64 to 8 bytes and offers no struct packing.
//     So we cannot use float64 here — instead every 8-byte double is represented
//     as rf2f64 ([2]uint32), which has 4-byte alignment and therefore reproduces
//     the pack(4) layout exactly. Read its value with .val(). (4-byte and smaller
//     fields already align identically in Go and C under pack(4).)
//   - The size/offset guard tests in layout_test.go assert the resulting layout
//     matches the pack(4) sizes; the live game's buffer size is the final proof.
//   - Each mapped buffer is prefixed by a version-tear block
//     (mVersionUpdateBegin/End) followed by mBytesUpdatedHint, then the payload.
//     Sources differ on whether these live in a base struct; on the wire they
//     are simply the first fields, so we inline them.
//
// Tire/carcass temperatures are in KELVIN here (the plugin's unit); the mapping
// layer converts to Celsius. See mapping.go.
package lmu

import "math"

// rf2f64 is an 8-byte IEEE-754 double stored as two little-endian uint32 halves.
// Using [2]uint32 (4-byte alignment) instead of float64 (8-byte alignment) is
// what makes our Go structs match the plugin's #pragma pack(4) layout. Call
// val() to get the float64 value.
type rf2f64 [2]uint32

// val decodes the double. x86/amd64 is little-endian, so [0] is the low word.
func (d rf2f64) val() float64 {
	return math.Float64frombits(uint64(d[0]) | uint64(d[1])<<32)
}

// Names of the plugin's memory-mapped files. The plugin creates them in the
// per-session Local namespace, so PitMate must run in the same Windows session
// as the game (which it does — it runs on the gaming PC).
const (
	mmTelemetryName = "$rFactor2SMMP_Telemetry$"
	mmScoringName   = "$rFactor2SMMP_Scoring$"
)

// maxMappedVehicles is the fixed size of the vehicle arrays in both buffers.
const maxMappedVehicles = 128

// rf2Vec3 is the plugin's 3D vector (world or local coordinates), 3 doubles.
type rf2Vec3 struct {
	x, y, z rf2f64
}

// rf2Wheel is per-wheel telemetry. Order: FL, FR, RL, RR within mWheels.
type rf2Wheel struct {
	mSuspensionDeflection rf2f64 // meters
	mRideHeight           rf2f64 // meters
	mSuspForce            rf2f64 // Newtons
	mBrakeTemp            rf2f64 // Celsius
	mBrakePressure        rf2f64 // 0.0-1.0

	mRotation              rf2f64 // radians/sec
	mLateralPatchVel       rf2f64
	mLongitudinalPatchVel  rf2f64
	mLateralGroundVel      rf2f64
	mLongitudinalGroundVel rf2f64
	mCamber                rf2f64 // radians
	mLateralForce          rf2f64 // Newtons
	mLongitudinalForce     rf2f64 // Newtons
	mTireLoad              rf2f64 // Newtons

	mGripFract   rf2f64    // fraction of contact patch sliding
	mPressure    rf2f64    // kPa
	mTemperature [3]rf2f64 // KELVIN, left/center/right
	mWear        rf2f64    // 0.0-1.0, fraction of maximum (1.0 = new)

	mTerrainName             [16]byte
	mSurfaceType             uint8
	mFlat                    uint8 // bool
	mDetached                uint8 // bool
	mStaticUndeflectedRadius uint8 // centimeters

	mVerticalTireDeflection rf2f64
	mWheelYLocation         rf2f64
	mToe                    rf2f64

	mTireCarcassTemperature    rf2f64    // KELVIN
	mTireInnerLayerTemperature [3]rf2f64 // KELVIN

	mExpansion [24]uint8
}

// rf2VehicleTelemetry is the full physics/telemetry for one car.
type rf2VehicleTelemetry struct {
	// Time
	mID          int32 // slot ID (matches scoring mID)
	mDeltaTime   rf2f64
	mElapsedTime rf2f64
	mLapNumber   int32
	mLapStartET  rf2f64
	mVehicleName [64]byte
	mTrackName   [64]byte

	// Position and derivatives
	mPos        rf2Vec3
	mLocalVel   rf2Vec3 // meters/sec, local coords
	mLocalAccel rf2Vec3

	// Orientation and derivatives
	mOri           [3]rf2Vec3
	mLocalRot      rf2Vec3
	mLocalRotAccel rf2Vec3

	// Vehicle status
	mGear            int32 // -1=reverse, 0=neutral, 1+=forward
	mEngineRPM       rf2f64
	mEngineWaterTemp rf2f64 // Celsius
	mEngineOilTemp   rf2f64 // Celsius
	mClutchRPM       rf2f64

	// Driver input (unfiltered)
	mUnfilteredThrottle rf2f64
	mUnfilteredBrake    rf2f64
	mUnfilteredSteering rf2f64
	mUnfilteredClutch   rf2f64

	// Driver input (filtered)
	mFilteredThrottle rf2f64
	mFilteredBrake    rf2f64
	mFilteredSteering rf2f64
	mFilteredClutch   rf2f64

	// Misc
	mSteeringShaftTorque rf2f64
	mFront3rdDeflection  rf2f64
	mRear3rdDeflection   rf2f64

	// Aerodynamics
	mFrontWingHeight rf2f64
	mFrontRideHeight rf2f64
	mRearRideHeight  rf2f64
	mDrag            rf2f64
	mFrontDownforce  rf2f64
	mRearDownforce   rf2f64

	// State/damage
	mFuel                rf2f64 // liters
	mEngineMaxRPM        rf2f64
	mScheduledStops      uint8
	mOverheating         uint8    // bool
	mDetached            uint8    // bool
	mHeadlights          uint8    // bool
	mDentSeverity        [8]uint8 // 0=none,1=some,2=more
	mLastImpactET        rf2f64
	mLastImpactMagnitude rf2f64
	mLastImpactPos       rf2Vec3

	// Expanded
	mEngineTorque           rf2f64
	mCurrentSector          int32 // zero-based, pitlane in sign bit
	mSpeedLimiter           uint8
	mMaxGears               uint8
	mFrontTireCompoundIndex uint8
	mRearTireCompoundIndex  uint8
	mFuelCapacity           rf2f64 // liters
	mFrontFlapActivated     uint8
	mRearFlapActivated      uint8
	mRearFlapLegalStatus    uint8
	mIgnitionStarter        uint8

	mFrontTireCompoundName [18]byte
	mRearTireCompoundName  [18]byte

	mSpeedLimiterAvailable    uint8
	mAntiStallActivated       uint8
	mUnused                   [2]uint8
	mVisualSteeringWheelRange float32

	mRearBrakeBias              rf2f64 // fraction of brakes on rear
	mTurboBoostPressure         rf2f64
	mPhysicsToGraphicsOffset    [3]float32
	mPhysicalSteeringWheelRange float32

	mBatteryChargeFraction rf2f64 // 0.0-1.0

	// Electric boost motor
	mElectricBoostMotorTorque      rf2f64
	mElectricBoostMotorRPM         rf2f64
	mElectricBoostMotorTemperature rf2f64
	mElectricBoostWaterTemperature rf2f64
	mElectricBoostMotorState       uint8 // 0=unavailable,1=inactive,2=propulsion,3=regen

	mExpansion [111]uint8

	// Kept last in the C++ struct to ease future changes.
	mWheels [4]rf2Wheel
}

// rf2Telemetry is the whole telemetry buffer: version block, hint, count, cars.
type rf2Telemetry struct {
	mVersionUpdateBegin uint32
	mVersionUpdateEnd   uint32
	mBytesUpdatedHint   int32
	mNumVehicles        int32
	mVehicles           [maxMappedVehicles]rf2VehicleTelemetry
}

// rf2ScoringInfo is session-wide scoring (track, session, weather, flags).
type rf2ScoringInfo struct {
	mTrackName           [64]byte
	mSession             int32 // 0=testday,1-4=practice,5-8=qual,9=warmup,10-13=race
	mCurrentET           rf2f64
	mEndET               rf2f64 // <=0 if not time-limited
	mMaxLaps             int32
	mLapDist             rf2f64  // track length in meters
	pointer1             [8]byte // 64-bit pointer padding in the C++ struct
	mNumVehicles         int32
	mGamePhase           uint8 // see mapping.go for values
	mYellowFlagState     int8
	mSectorFlag          [3]int8
	mStartLight          uint8
	mNumRedLights        uint8
	mInRealtime          uint8 // bool
	mPlayerName          [32]byte
	mPlrFileName         [64]byte
	mDarkCloud           rf2f64
	mRaining             rf2f64  // 0.0-1.0
	mAmbientTemp         rf2f64  // Celsius
	mTrackTemp           rf2f64  // Celsius
	mWind                rf2Vec3 // m/s
	mMinPathWetness      rf2f64
	mMaxPathWetness      rf2f64
	mGameMode            uint8
	mIsPasswordProtected uint8 // bool
	mServerPort          uint16
	mServerPublicIP      uint32
	mMaxPlayers          int32
	mServerName          [32]byte
	mStartET             float32
	mAvgPathWetness      rf2f64 // 0.0-1.0
	mExpansion           [200]uint8
	pointer2             [8]byte
}

// rf2VehicleScoring is per-car scoring (position, gaps, timing, pit state).
type rf2VehicleScoring struct {
	mID               int32
	mDriverName       [32]byte
	mVehicleName      [64]byte
	mTotalLaps        int16 // laps completed
	mSector           int8
	mFinishStatus     int8
	mLapDist          rf2f64 // distance along current lap, meters
	mPathLateral      rf2f64
	mTrackEdge        rf2f64
	mBestSector1      rf2f64
	mBestSector2      rf2f64 // cumulative through sector 2
	mBestLapTime      rf2f64
	mLastSector1      rf2f64
	mLastSector2      rf2f64 // cumulative through sector 2
	mLastLapTime      rf2f64
	mCurSector1       rf2f64
	mCurSector2       rf2f64
	mNumPitstops      int16
	mNumPenalties     int16
	mIsPlayer         uint8 // bool
	mControl          int8
	mInPits           uint8 // bool
	mPlace            uint8 // overall position, 1 = leader
	mVehicleClass     [32]byte
	mTimeBehindNext   rf2f64 // gap to car ahead, seconds
	mLapsBehindNext   int32
	mTimeBehindLeader rf2f64 // gap to leader, seconds
	mLapsBehindLeader int32
	mLapStartET       rf2f64
	mPos              rf2Vec3
	mLocalVel         rf2Vec3
	mLocalAccel       rf2Vec3
	mOri              [3]rf2Vec3
	mLocalRot         rf2Vec3
	mLocalRotAccel    rf2Vec3
	mHeadlights       uint8
	mPitState         uint8 // 0=none,1=request,2=entering,3=stopped,4=exiting
	mServerScored     uint8
	mIndividualPhase  uint8
	mQualification    int32
	mTimeIntoLap      rf2f64
	mEstimatedLapTime rf2f64
	mPitGroup         [24]byte
	mFlag             uint8 // per-car flag
	mUnderYellow      uint8 // bool
	mCountLapFlag     uint8
	mInGarageStall    uint8 // bool
	mUpgradePack      [16]uint8
	mPitLapDist       float32
	mBestLapSector1   float32
	mBestLapSector2   float32
	mExpansion        [48]uint8
}

// rf2Scoring is the whole scoring buffer: version block, hint, info, cars.
type rf2Scoring struct {
	mVersionUpdateBegin uint32
	mVersionUpdateEnd   uint32
	mBytesUpdatedHint   int32
	mScoringInfo        rf2ScoringInfo
	mVehicles           [maxMappedVehicles]rf2VehicleScoring
}
