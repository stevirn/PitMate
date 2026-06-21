# PitMate Functions & Workflow

A functional spec of the strategist cockpit: what each tab and tool does, what
data it reads from the telemetry model, and why it exists. This grows with the
UI. Each tab below is tagged with its build status.

The data fields referenced below come from `backend/telemetry/types.go`. The UI
lives in `frontend/src/` (tabs in `tabs/`, reusable pieces in `components/`,
the WebSocket data store in `lib/telemetry.js`).

## Main tabs

### 1. Live Data — ✅ implemented
The strategist's at-a-glance dashboard during a stint.
- **Shows:** fuel level and laps remaining, **virtual energy** (LMU, via REST API),
  hybrid/energy state, tire temps / pressures / wear per corner, current/last/best
  lap, sector times, speeds.
- **Reads:** `Player.Energy` (incl. `VirtualEnergyFraction`), `Player.Tires`,
  `Player.Timing`, `Player.Speed`.
- **Why:** the core moment-to-moment picture the strategist watches most.

### 2. Strategy Calls — ⬜ planned (placeholder)
Tools for deciding and executing pit strategy.
- **Shows:** a track-position circle of all cars, weather/conditions, pit
  parameters (fuel to add, tire choice, estimated stop loss).
- **Reads:** `Competitors[].Race`, `Player.Race`, `Session.Conditions`,
  `Player.Energy`, `Player.Timing`.
- **Why:** turns raw data into "when to pit and what to change".

### 3. Driver Coaching — ⬜ planned (placeholder)
Feedback to help the driver improve and to manage the car.
- **Shows:** tire wear by corner over a stint, fuel use by straight/section, the
  driven line and where time is gained/lost (mini-sectors).
- **Reads:** `Player.Tires` (wear), `Player.Energy`, `Player.Timing.MiniSectors`,
  `Player.Timing` sectors.
- **Why:** converts telemetry into actionable coaching.

### 4. Driver Vs. — ⬜ planned (placeholder)
Compare the driver against others or against a reference.
- **Shows:** side-by-side of the player vs a chosen competitor or a stored
  reference lap — lap/sector deltas, where the gap comes from.
- **Reads:** `Player.Timing` vs a selected `Competitors[].Timing` (or a saved
  reference), plus mini-sectors.
- **Why:** finds concrete time to be gained.

### 5. Car Management — 🟡 partial
Monitoring and adjusting the car's systems and condition.
- **Shows:** hybrid/EV state, aero, damage by zone and its effects, oil/water and
  brake temps, ARBs, brake balance, TC/ABS settings.
- **Reads:** `Player.Energy`, `Player.Damage`, `Player.Systems`,
  `Player.Tires` (brake temps).
- **Why:** keep the car healthy and correctly set up through a stint.
- **Built so far:** oil/water temps, brake bias, limiter/lights, hybrid charge,
  coarse front/rear-wing damage. Not shown yet (the LMU adapter doesn't expose
  them): TC/ABS/ARBs/engine map, detailed per-zone aero damage.

### 6. Settings — 🟡 partial
Configuration for PitMate itself.
- **Shows:** UI options, gaming-PC IP/port, session options, data import/export.
- **Reads/writes:** client-side preferences and `backend/config` values.
- **Why:** make PitMate usable on a given network and to a given taste.
- **Built so far:** connection diagnostics (socket/game state, frame count, last
  update, WebSocket URL) and a Reconnect button. UI options and import/export are
  still to come; server network/rate settings are backend command-line flags.

## Global overlays (not tabs)

These appear over whatever tab is open, because they are always relevant.

### Flag / safety-car / crash banners — 🟡 partial
- **Shows:** prominent popup when the flag or safety-car state changes, or when a
  crash/contact is detected.
- **Reads:** `RaceControl.CurrentFlag`, `RaceControl.SafetyCar`, `Events` with
  high severity.
- **Why:** these demand immediate strategist attention regardless of the tab.
- **Built so far:** `components/Banner.svelte` shows connection problems and the
  current flag / safety-car state. Crash/contact popups depend on the event log,
  which the adapter doesn't populate yet.

### Event timeline — ⬜ planned
- **Shows:** a running list of notable events — contacts, offenses/penalties,
  competitor pit stops, fastest laps.
- **Reads:** `Events`.
- **Why:** the strategist needs a memory of what happened and when.
- **Note:** the backend currently sends an empty `Events` list; populating it
  (from frame-to-frame changes) is a backend task before this can be built.

## Status

Live Data is fully built; Car Management, Settings, and the flag banner are
partial; Strategy, Coaching, Driver Vs., and the event timeline are placeholders.
