# PitMate Functions & Workflow

A functional spec of the strategist cockpit: what each tab and tool does, what
data it reads from the telemetry model, and why it exists. This grows with the
UI. Nothing here is built yet — this is the plan the UI will be measured against.

The data fields referenced below come from `backend/telemetry/types.go`.

## Main tabs

### 1. Live Data
The strategist's at-a-glance dashboard during a stint.
- **Shows:** fuel level and laps remaining, hybrid/energy state, tire temps /
  pressures / wear per corner, current/last/best lap, sector times, speeds.
- **Reads:** `Player.Energy`, `Player.Tires`, `Player.Timing`, `Player.Speed`.
- **Why:** the core moment-to-moment picture the strategist watches most.

### 2. Strategy Calls
Tools for deciding and executing pit strategy.
- **Shows:** a track-position circle of all cars, weather/conditions, pit
  parameters (fuel to add, tire choice, estimated stop loss).
- **Reads:** `Competitors[].Race`, `Player.Race`, `Session.Conditions`,
  `Player.Energy`, `Player.Timing`.
- **Why:** turns raw data into "when to pit and what to change".

### 3. Driver Coaching
Feedback to help the driver improve and to manage the car.
- **Shows:** tire wear by corner over a stint, fuel use by straight/section, the
  driven line and where time is gained/lost (mini-sectors).
- **Reads:** `Player.Tires` (wear), `Player.Energy`, `Player.Timing.MiniSectors`,
  `Player.Timing` sectors.
- **Why:** converts telemetry into actionable coaching.

### 4. Driver Vs.
Compare the driver against others or against a reference.
- **Shows:** side-by-side of the player vs a chosen competitor or a stored
  reference lap — lap/sector deltas, where the gap comes from.
- **Reads:** `Player.Timing` vs a selected `Competitors[].Timing` (or a saved
  reference), plus mini-sectors.
- **Why:** finds concrete time to be gained.

### 5. Car Management
Monitoring and adjusting the car's systems and condition.
- **Shows:** hybrid/EV state, aero, damage by zone and its effects, oil/water and
  brake temps, ARBs, brake balance, TC/ABS settings.
- **Reads:** `Player.Energy`, `Player.Damage`, `Player.Systems`,
  `Player.Tires` (brake temps).
- **Why:** keep the car healthy and correctly set up through a stint.

### 6. Settings
Configuration for PitMate itself.
- **Shows:** UI options, gaming-PC IP/port, session options, data import/export.
- **Reads/writes:** client-side preferences and `backend/config` values.
- **Why:** make PitMate usable on a given network and to a given taste.

## Global overlays (not tabs)

These appear over whatever tab is open, because they are always relevant.

### Flag / safety-car / crash banners
- **Shows:** prominent popup when the flag or safety-car state changes, or when a
  crash/contact is detected.
- **Reads:** `RaceControl.CurrentFlag`, `RaceControl.SafetyCar`, `Events` with
  high severity.
- **Why:** these demand immediate strategist attention regardless of the tab.

### Event timeline
- **Shows:** a running list of notable events — contacts, offenses/penalties,
  competitor pit stops, fastest laps.
- **Reads:** `Events`.
- **Why:** the strategist needs a memory of what happened and when.

## Status

Specification only. No tab or overlay is implemented yet.
