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
//   - Fields use natural alignment (MSVC default pack 8). Go's struct layout on
//     amd64 inserts the same padding, so the casts line up. The size/offset
//     guard tests in layout_test.go assert this stays true.
//   - Each mapped buffer is prefixed by a version-tear block
//     (mVersionUpdateBegin/End) followed by mBytesUpdatedHint, then the payload.
//     Sources differ on whether these live in a base struct; on the wire they
//     are simply the first fields, so we inline them.
//
// Tire/carcass temperatures are in KELVIN here (the plugin's unit); the mapping
// layer converts to Celsius. See mapping.go.
package lmu

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
	x, y, z float64
}

// rf2Wheel is per-wheel telemetry. Order: FL, FR, RL, RR within mWheels.
type rf2Wheel struct {
	mSuspensionDeflection float64 // meters
	mRideHeight           float64 // meters
	mSuspForce            float64 // Newtons
	mBrakeTemp            float64 // Celsius
	mBrakePressure        float64 // 0.0-1.0

	mRotation              float64 // radians/sec
	mLateralPatchVel       float64
	mLongitudinalPatchVel  float64
	mLateralGroundVel      float64
	mLongitudinalGroundVel float64
	mCamber                float64 // radians
	mLateralForce          float64 // Newtons
	mLongitudinalForce     float64 // Newtons
	mTireLoad              float64 // Newtons

	mGripFract   float64    // fraction of contact patch sliding
	mPressure    float64    // kPa
	mTemperature [3]float64 // KELVIN, left/center/right
	mWear        float64    // 0.0-1.0, fraction of maximum (1.0 = new)

	mTerrainName             [16]byte
	mSurfaceType             uint8
	mFlat                    uint8 // bool
	mDetached                uint8 // bool
	mStaticUndeflectedRadius uint8 // centimeters

	mVerticalTireDeflection float64
	mWheelYLocation         float64
	mToe                    float64

	mTireCarcassTemperature    float64    // KELVIN
	mTireInnerLayerTemperature [3]float64 // KELVIN

	mExpansion [24]uint8
}

// rf2VehicleTelemetry is the full physics/telemetry for one car.
type rf2VehicleTelemetry struct {
	// Time
	mID          int32 // slot ID (matches scoring mID)
	mDeltaTime   float64
	mElapsedTime float64
	mLapNumber   int32
	mLapStartET  float64
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
	mEngineRPM       float64
	mEngineWaterTemp float64 // Celsius
	mEngineOilTemp   float64 // Celsius
	mClutchRPM       float64

	// Driver input (unfiltered)
	mUnfilteredThrottle float64
	mUnfilteredBrake    float64
	mUnfilteredSteering float64
	mUnfilteredClutch   float64

	// Driver input (filtered)
	mFilteredThrottle float64
	mFilteredBrake    float64
	mFilteredSteering float64
	mFilteredClutch   float64

	// Misc
	mSteeringShaftTorque float64
	mFront3rdDeflection  float64
	mRear3rdDeflection   float64

	// Aerodynamics
	mFrontWingHeight float64
	mFrontRideHeight float64
	mRearRideHeight  float64
	mDrag            float64
	mFrontDownforce  float64
	mRearDownforce   float64

	// State/damage
	mFuel                float64 // liters
	mEngineMaxRPM        float64
	mScheduledStops      uint8
	mOverheating         uint8    // bool
	mDetached            uint8    // bool
	mHeadlights          uint8    // bool
	mDentSeverity        [8]uint8 // 0=none,1=some,2=more
	mLastImpactET        float64
	mLastImpactMagnitude float64
	mLastImpactPos       rf2Vec3

	// Expanded
	mEngineTorque           float64
	mCurrentSector          int32 // zero-based, pitlane in sign bit
	mSpeedLimiter           uint8
	mMaxGears               uint8
	mFrontTireCompoundIndex uint8
	mRearTireCompoundIndex  uint8
	mFuelCapacity           float64 // liters
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

	mRearBrakeBias              float64 // fraction of brakes on rear
	mTurboBoostPressure         float64
	mPhysicsToGraphicsOffset    [3]float32
	mPhysicalSteeringWheelRange float32

	mBatteryChargeFraction float64 // 0.0-1.0

	// Electric boost motor
	mElectricBoostMotorTorque      float64
	mElectricBoostMotorRPM         float64
	mElectricBoostMotorTemperature float64
	mElectricBoostWaterTemperature float64
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
	mCurrentET           float64
	mEndET               float64 // <=0 if not time-limited
	mMaxLaps             int32
	mLapDist             float64 // track length in meters
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
	mDarkCloud           float64
	mRaining             float64 // 0.0-1.0
	mAmbientTemp         float64 // Celsius
	mTrackTemp           float64 // Celsius
	mWind                rf2Vec3 // m/s
	mMinPathWetness      float64
	mMaxPathWetness      float64
	mGameMode            uint8
	mIsPasswordProtected uint8 // bool
	mServerPort          uint16
	mServerPublicIP      uint32
	mMaxPlayers          int32
	mServerName          [32]byte
	mStartET             float32
	mAvgPathWetness      float64 // 0.0-1.0
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
	mLapDist          float64 // distance along current lap, meters
	mPathLateral      float64
	mTrackEdge        float64
	mBestSector1      float64
	mBestSector2      float64 // cumulative through sector 2
	mBestLapTime      float64
	mLastSector1      float64
	mLastSector2      float64 // cumulative through sector 2
	mLastLapTime      float64
	mCurSector1       float64
	mCurSector2       float64
	mNumPitstops      int16
	mNumPenalties     int16
	mIsPlayer         uint8 // bool
	mControl          int8
	mInPits           uint8 // bool
	mPlace            uint8 // overall position, 1 = leader
	mVehicleClass     [32]byte
	mTimeBehindNext   float64 // gap to car ahead, seconds
	mLapsBehindNext   int32
	mTimeBehindLeader float64 // gap to leader, seconds
	mLapsBehindLeader int32
	mLapStartET       float64
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
	mTimeIntoLap      float64
	mEstimatedLapTime float64
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
