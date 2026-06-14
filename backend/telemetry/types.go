// Package telemetry defines PitMate's game-agnostic data model.
//
// This is the most important file in the project. Every adapter (LMU today,
// other games later) must translate its game's native data into the structs
// defined here, and the WebSocket server always sends these structs — never
// anything game-specific. If a field does not exist here, the frontend can
// never see it; if it does exist here, every game must be able to fill it
// (or leave it at its zero value when unavailable).
//
// Design rules:
//   - Plain English comment on every field. A non-coder should be able to read
//     this file and understand what data PitMate works with.
//   - Units are stated explicitly in comments (km/h, °C, litres, seconds, etc.)
//     and kept consistent everywhere. Adapters convert into these units.
//   - Fields LMU may not expose yet are kept, but marked:
//     // TODO: verify LMU availability
//   - Where a value can be genuinely "unknown", prefer a pointer or a sentinel
//     and document it, so the UI can distinguish "zero" from "no data".
//
// All times are seconds (float64) unless the field name says otherwise.
// All temperatures are degrees Celsius. All speeds are km/h.
package telemetry

// Frame is the single top-level object pushed over the WebSocket on every
// update. One Frame is a complete snapshot of the race world at one instant.
// The frontend re-renders from whatever the latest Frame contains, so a Frame
// should always be internally consistent (don't half-fill it).
type Frame struct {
	// Unix time in milliseconds when this snapshot was produced by the server.
	// Used by the UI to detect stale data and to order/animate updates.
	Timestamp int64 `json:"timestamp"`

	// Monotonically increasing counter, one per Frame sent this session.
	// Lets the UI notice dropped or out-of-order frames.
	Sequence uint64 `json:"sequence"`

	// Which adapter produced this data and whether it is currently connected
	// to a running game. The UI shows a clear "no data" state when not connected.
	Source SourceInfo `json:"source"`

	// High-level facts about the current session (race type, track, weather).
	Session SessionInfo `json:"session"`

	// Everything about the player's own car — identity, lap, fuel, tires, etc.
	// This is the car the strategist is supporting.
	Player Car `json:"player"`

	// Every other car in the session, including the player is optional but by
	// convention excluded here (the player lives in Player). Order is not
	// guaranteed; sort in the UI by Position when needed.
	Competitors []Car `json:"competitors"`

	// Current flag / safety-car state for the whole session.
	RaceControl RaceControl `json:"raceControl"`

	// Notable things that have happened (contacts, pit stops, penalties).
	// This is a rolling log; the adapter/server decides how many to retain.
	Events []Event `json:"events"`
}

// SourceInfo describes where the data came from. It exists so the frontend is
// always honest about what it is showing: live game, replay, or nothing.
type SourceInfo struct {
	// Human-readable adapter/game name, e.g. "Le Mans Ultimate".
	Game string `json:"game"`

	// Short adapter identifier, e.g. "lmu". Lets the UI enable game-specific
	// presentation without the server knowing game details.
	AdapterID string `json:"adapterId"`

	// True when the adapter is actually reading live data from a running game.
	// False means the UI should show a disconnected / waiting state.
	Connected bool `json:"connected"`

	// Adapter/protocol version string, useful for debugging mismatches.
	Version string `json:"version"`
}

// -----------------------------------------------------------------------------
// Session
// -----------------------------------------------------------------------------

// SessionType is the kind of on-track session currently running.
type SessionType string

const (
	SessionUnknown    SessionType = "unknown"
	SessionPractice   SessionType = "practice"
	SessionQualifying SessionType = "qualifying"
	SessionRace       SessionType = "race"
	SessionHotlap     SessionType = "hotlap"
)

// SessionInfo holds facts about the session as a whole — the stuff that is the
// same for every car on track.
type SessionInfo struct {
	// Practice / qualifying / race / etc.
	Type SessionType `json:"type"`

	// Track name as reported by the game, e.g. "Le Mans 24h".
	TrackName string `json:"trackName"`

	// Track configuration/layout name when a track has variants.
	TrackConfig string `json:"trackConfig"`

	// Full lap length in metres. Used for fuel-per-distance and pace math.
	TrackLengthM float64 `json:"trackLengthM"`

	// Seconds of green-flag running elapsed in this session.
	ElapsedSeconds float64 `json:"elapsedSeconds"`

	// Seconds remaining if the session is time-limited. Negative or zero means
	// "not time-limited" — check IsTimed.
	RemainingSeconds float64 `json:"remainingSeconds"`

	// True if the session ends on a clock, false if it ends on a lap count.
	IsTimed bool `json:"isTimed"`

	// Total laps if the session is lap-limited (0 when time-limited).
	TotalLaps int `json:"totalLaps"`

	// Weather and surface conditions for the whole track.
	Conditions Conditions `json:"conditions"`
}

// Conditions describes weather and track surface state.
type Conditions struct {
	// Air temperature in °C.
	AirTempC float64 `json:"airTempC"`

	// Track surface temperature in °C.
	TrackTempC float64 `json:"trackTempC"`

	// 0.0 = bone dry, 1.0 = fully wet. A normalized wetness figure so the UI
	// can show one number regardless of how a game models rain.
	TrackWetness float64 `json:"trackWetness"`

	// 0.0 = clear, 1.0 = heaviest rain currently falling.
	RainIntensity float64 `json:"rainIntensity"`

	// Wind speed in km/h. // TODO: verify LMU availability
	WindSpeedKph float64 `json:"windSpeedKph"`

	// Wind direction in degrees (0–360, 0 = North). // TODO: verify LMU availability
	WindDirectionDeg float64 `json:"windDirectionDeg"`

	// Short text forecast if the game provides one, else empty.
	// TODO: verify LMU availability
	Forecast string `json:"forecast"`
}

// -----------------------------------------------------------------------------
// Car — the central per-vehicle struct (used for player and competitors)
// -----------------------------------------------------------------------------

// Car is everything PitMate knows about one vehicle. The player's car fills in
// every section; competitor cars typically fill only the fields visible from
// the outside (identity, position, gaps, last lap, pit status) and leave the
// rest at their zero values.
type Car struct {
	// Who/what this car is.
	Identity Identity `json:"identity"`

	// Lap counts and timing.
	Timing Timing `json:"timing"`

	// Fuel and hybrid/EV energy state.
	Energy Energy `json:"energy"`

	// All four tires.
	Tires Tires `json:"tires"`

	// Current/peak speeds.
	Speed Speed `json:"speed"`

	// Adjustable and monitored car systems (brake bias, TC, temps, etc.).
	Systems Systems `json:"systems"`

	// Damage state by zone and its performance effect.
	Damage Damage `json:"damage"`

	// Where this car sits in the race and gaps to others.
	Race RaceState `json:"race"`
}

// Identity is who the car and driver are. These rarely change during a session.
type Identity struct {
	// Stable unique ID for this car within the session, used to match a car
	// across frames even if positions change. Adapter-defined but consistent.
	CarID string `json:"carId"`

	// Visible car number, e.g. "7".
	CarNumber string `json:"carNumber"`

	// Class name, e.g. "Hypercar", "LMGT3".
	CarClass string `json:"carClass"`

	// Manufacturer/model, e.g. "Toyota GR010".
	CarModel string `json:"carModel"`

	// Current driver's name (can change in driver-swap races).
	DriverName string `json:"driverName"`

	// Team name if provided. // TODO: verify LMU availability
	TeamName string `json:"teamName"`

	// True if this struct represents the player's own car.
	IsPlayer bool `json:"isPlayer"`
}

// -----------------------------------------------------------------------------
// Timing
// -----------------------------------------------------------------------------

// Timing covers lap counts and all lap/sector times. All times are seconds.
type Timing struct {
	// Laps completed by this car so far.
	LapsCompleted int `json:"lapsCompleted"`

	// Distance around the current lap as a fraction 0.0–1.0. Used to place the
	// car on the track-position circle and to align mini-sectors.
	LapFraction float64 `json:"lapFraction"`

	// Time elapsed on the current (in-progress) lap, seconds.
	CurrentLapSeconds float64 `json:"currentLapSeconds"`

	// The most recently completed lap time, seconds (0 if none yet).
	LastLapSeconds float64 `json:"lastLapSeconds"`

	// Best lap time this session, seconds (0 if none yet).
	BestLapSeconds float64 `json:"bestLapSeconds"`

	// Predicted/estimated current lap time if the game offers one, seconds.
	// TODO: verify LMU availability
	EstimatedLapSeconds float64 `json:"estimatedLapSeconds"`

	// Sector times for the last completed lap, seconds. Usually length 3.
	LastSectors []float64 `json:"lastSectors"`

	// Sector times of the best lap, seconds. Usually length 3.
	BestSectors []float64 `json:"bestSectors"`

	// The best each individual sector has ever been this session ("ideal"/
	// theoretical-best components), seconds. Usually length 3.
	OptimalSectors []float64 `json:"optimalSectors"`

	// Finer-grained timing splits within the lap. Length is track-dependent.
	// Enables the driver-coaching "where am I gaining/losing time" view.
	// TODO: verify LMU availability
	MiniSectors []MiniSector `json:"miniSectors"`
}

// MiniSector is one small timed segment of a lap, finer than a sector.
type MiniSector struct {
	// Index of this mini-sector within the lap (0-based).
	Index int `json:"index"`

	// Time spent in this segment on the current/last lap, seconds.
	TimeSeconds float64 `json:"timeSeconds"`

	// Difference vs the driver's best for this segment, seconds.
	// Negative = faster than personal best here.
	DeltaSeconds float64 `json:"deltaSeconds"`
}

// -----------------------------------------------------------------------------
// Fuel & energy
// -----------------------------------------------------------------------------

// Energy covers both liquid fuel and electric/hybrid energy, since modern
// endurance cars use both. Fields that don't apply to a given car stay zero.
type Energy struct {
	// Fuel currently in the tank, litres.
	FuelLitres float64 `json:"fuelLitres"`

	// Maximum tank capacity, litres.
	FuelCapacityLitres float64 `json:"fuelCapacityLitres"`

	// Fuel used on the last completed lap, litres.
	FuelPerLastLap float64 `json:"fuelPerLastLap"`

	// Rolling-average fuel used per lap, litres. Smoother than per-lap for
	// strategy math.
	FuelPerLapAvg float64 `json:"fuelPerLapAvg"`

	// Estimated laps of fuel remaining at the current usage rate.
	FuelLapsRemaining float64 `json:"fuelLapsRemaining"`

	// True if this car has a hybrid/EV system worth displaying.
	HasHybrid bool `json:"hasHybrid"`

	// Stored electrical energy as a fraction 0.0–1.0 of capacity.
	// TODO: verify LMU availability
	HybridChargeFraction float64 `json:"hybridChargeFraction"`

	// Stored electrical energy in megajoules, if the game reports absolute MJ.
	// TODO: verify LMU availability
	HybridStoredMJ float64 `json:"hybridStoredMJ"`

	// Current deployment mode/map name, e.g. "Hotlap", "Attack", "Build".
	// TODO: verify LMU availability
	HybridMode string `json:"hybridMode"`

	// Electrical energy deployed so far this lap, megajoules.
	// TODO: verify LMU availability
	HybridDeployedThisLapMJ float64 `json:"hybridDeployedThisLapMJ"`
}

// -----------------------------------------------------------------------------
// Tires
// -----------------------------------------------------------------------------

// Tires groups all four corners plus the fitted compound.
type Tires struct {
	// Compound name, e.g. "Soft", "Medium", "Wet".
	Compound string `json:"compound"`

	// The four corners. See TireCorner.
	FrontLeft  TireCorner `json:"frontLeft"`
	FrontRight TireCorner `json:"frontRight"`
	RearLeft   TireCorner `json:"rearLeft"`
	RearRight  TireCorner `json:"rearRight"`
}

// TireCorner is the full state of one tire.
type TireCorner struct {
	// Surface temperature in °C (single representative value).
	TempC float64 `json:"tempC"`

	// Inner / middle / outer surface temps in °C, if the game splits them.
	// Order: [inner, middle, outer]. Empty if unavailable.
	// TODO: verify LMU availability
	TempBandsC []float64 `json:"tempBandsC"`

	// Core/carcass temperature in °C. // TODO: verify LMU availability
	CoreTempC float64 `json:"coreTempC"`

	// Hot pressure in kPa.
	PressureKpa float64 `json:"pressureKpa"`

	// Remaining tread as a fraction 1.0 = new, 0.0 = fully worn.
	WearFraction float64 `json:"wearFraction"`

	// True if the tire is currently locked/sliding badly (flat-spotting risk).
	// TODO: verify LMU availability
	Locking bool `json:"locking"`

	// Brake temperature at this corner in °C. Lives here because it's per-corner.
	BrakeTempC float64 `json:"brakeTempC"`
}

// -----------------------------------------------------------------------------
// Speed
// -----------------------------------------------------------------------------

// Speed holds current and notable peak speeds, km/h.
type Speed struct {
	// Instantaneous ground speed, km/h.
	CurrentKph float64 `json:"currentKph"`

	// Highest speed reached this session, km/h.
	TopKph float64 `json:"topKph"`

	// Speed recorded at the speed trap on the last pass, km/h.
	// TODO: verify LMU availability
	TrapKph float64 `json:"trapKph"`

	// Engine RPM. // TODO: verify LMU availability
	RPM float64 `json:"rpm"`

	// Current gear (0 = neutral, -1 = reverse). // TODO: verify LMU availability
	Gear int `json:"gear"`
}

// -----------------------------------------------------------------------------
// Car systems
// -----------------------------------------------------------------------------

// Systems holds adjustable driver controls and monitored mechanical temps.
type Systems struct {
	// Brake bias toward the front as a percentage, e.g. 56.5 means 56.5% front.
	BrakeBiasFrontPct float64 `json:"brakeBiasFrontPct"`

	// Traction control level/setting (game-defined scale).
	TractionControl int `json:"tractionControl"`

	// Secondary traction-control "cut" setting if the car has one.
	// TODO: verify LMU availability
	TractionControlCut int `json:"tractionControlCut"`

	// ABS level/setting (game-defined scale).
	ABS int `json:"abs"`

	// Engine/power map setting. // TODO: verify LMU availability
	EngineMap int `json:"engineMap"`

	// Front anti-roll bar setting (game-defined scale). // TODO: verify LMU availability
	ARBFront int `json:"arbFront"`

	// Rear anti-roll bar setting (game-defined scale). // TODO: verify LMU availability
	ARBRear int `json:"arbRear"`

	// Engine oil temperature in °C.
	OilTempC float64 `json:"oilTempC"`

	// Engine coolant/water temperature in °C.
	WaterTempC float64 `json:"waterTempC"`

	// Whether the headlights are currently on. // TODO: verify LMU availability
	LightsOn bool `json:"lightsOn"`

	// Whether the pit speed limiter is engaged. // TODO: verify LMU availability
	PitLimiterOn bool `json:"pitLimiterOn"`
}

// -----------------------------------------------------------------------------
// Damage
// -----------------------------------------------------------------------------

// Damage describes physical damage and its effect on performance. Damage levels
// are fractions 0.0 = pristine, 1.0 = destroyed, so the UI can show one scale
// regardless of how a game models damage internally.
type Damage struct {
	// Aggregate body/aero damage per zone, each 0.0–1.0.
	FrontWing float64 `json:"frontWing"`
	RearWing  float64 `json:"rearWing"`
	Floor     float64 `json:"floor"`     // TODO: verify LMU availability
	Diffuser  float64 `json:"diffuser"`  // TODO: verify LMU availability
	LeftSide  float64 `json:"leftSide"`  // TODO: verify LMU availability
	RightSide float64 `json:"rightSide"` // TODO: verify LMU availability

	// Mechanical damage zones, each 0.0–1.0. // TODO: verify LMU availability
	Engine     float64 `json:"engine"`
	Gearbox    float64 `json:"gearbox"`
	Suspension float64 `json:"suspension"`

	// --- Derived performance effects (so the strategist sees consequences) ---

	// Estimated downforce lost vs an undamaged car, fraction 0.0–1.0.
	// TODO: verify LMU availability
	DownforceLoss float64 `json:"downforceLoss"`

	// Estimated overall aero efficiency lost, fraction 0.0–1.0.
	// TODO: verify LMU availability
	AeroEfficiencyLoss float64 `json:"aeroEfficiencyLoss"`

	// Estimated top-speed reduction from damage, km/h.
	// TODO: verify LMU availability
	TopSpeedDeltaKph float64 `json:"topSpeedDeltaKph"`
}

// -----------------------------------------------------------------------------
// Race state, position and gaps
// -----------------------------------------------------------------------------

// PitStatus is where a car is in the pit cycle.
type PitStatus string

const (
	PitNone     PitStatus = "none"     // out on track, racing
	PitApproach PitStatus = "approach" // on pit entry / pit lane, not stopped
	PitStopped  PitStatus = "stopped"  // stationary in the box being serviced
	PitExit     PitStatus = "exit"     // leaving the pits
)

// RaceState is where a car sits in the race and its gaps to others. Gaps are in
// seconds; positive means the other car is that many seconds away.
type RaceState struct {
	// Overall position across all classes, 1 = leader.
	PositionOverall int `json:"positionOverall"`

	// Position within this car's own class, 1 = class leader.
	PositionInClass int `json:"positionInClass"`

	// Gap to the overall leader, seconds.
	GapToLeaderSeconds float64 `json:"gapToLeaderSeconds"`

	// Gap to the car immediately ahead on track, seconds.
	GapAheadSeconds float64 `json:"gapAheadSeconds"`

	// Gap to the car immediately behind on track, seconds.
	GapBehindSeconds float64 `json:"gapBehindSeconds"`

	// Gap to the car ahead in the same class, seconds. // TODO: verify LMU availability
	GapAheadInClassSeconds float64 `json:"gapAheadInClassSeconds"`

	// Gap to the car behind in the same class, seconds. // TODO: verify LMU availability
	GapBehindInClassSeconds float64 `json:"gapBehindInClassSeconds"`

	// Current pit-cycle state.
	PitStatus PitStatus `json:"pitStatus"`

	// Total pit stops this car has made so far.
	PitStopCount int `json:"pitStopCount"`

	// True if the car is currently in the pit lane (any pit phase).
	InPitLane bool `json:"inPitLane"`

	// True if this car is running but not on the lead lap.
	Lapped bool `json:"lapped"`
}

// -----------------------------------------------------------------------------
// Flags, safety car, and the event log
// -----------------------------------------------------------------------------

// Flag is the current racing flag condition.
type Flag string

const (
	FlagNone         Flag = "none" // green / racing
	FlagGreen        Flag = "green"
	FlagYellow       Flag = "yellow"
	FlagDoubleYellow Flag = "double_yellow"
	FlagBlue         Flag = "blue"  // faster car approaching to lap you
	FlagWhite        Flag = "white" // slow car ahead / last lap (game-dependent)
	FlagRed          Flag = "red"   // session stopped
	FlagChequered    Flag = "chequered"
	FlagBlack        Flag = "black" // penalty / disqualification
)

// SafetyCarState describes any neutralization of the race.
type SafetyCarState string

const (
	SafetyCarNone    SafetyCarState = "none"
	SafetyCarFull    SafetyCarState = "full"    // physical safety car deployed
	SafetyCarVirtual SafetyCarState = "virtual" // VSC / slow zones
	SafetyCarFCY     SafetyCarState = "fcy"     // full-course yellow
)

// RaceControl is the session-wide flag and safety-car state, shown as overlays.
type RaceControl struct {
	// The flag currently shown to the player's car.
	CurrentFlag Flag `json:"currentFlag"`

	// Safety car / full-course-yellow status for the session.
	SafetyCar SafetyCarState `json:"safetyCar"`

	// Free-text message from race control if provided. // TODO: verify LMU availability
	Message string `json:"message"`
}

// EventType classifies an entry in the event log.
type EventType string

const (
	EventContact    EventType = "contact"     // collision / off
	EventPitStop    EventType = "pit_stop"    // a car pitted
	EventPenalty    EventType = "penalty"     // an offense/penalty issued
	EventFlag       EventType = "flag"        // a flag/safety-car change
	EventOvertake   EventType = "overtake"    // position change of note
	EventFastestLap EventType = "fastest_lap" // a new fastest lap
	EventInfo       EventType = "info"        // generic note
)

// Event is one entry in the rolling event timeline shown as a global overlay.
type Event struct {
	// Unix time in milliseconds when the event occurred.
	Timestamp int64 `json:"timestamp"`

	// What kind of event this is.
	Type EventType `json:"type"`

	// CarID of the car this event is about, if any (matches Identity.CarID).
	CarID string `json:"carId"`

	// Human-readable description, e.g. "#7 contact at Mulsanne".
	Message string `json:"message"`

	// Relative importance 0–2 (0 = info, 1 = notable, 2 = critical) so the UI
	// can decide whether to show a banner or just log it.
	Severity int `json:"severity"`
}
