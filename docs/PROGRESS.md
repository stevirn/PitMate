# PitMate — Progress & Session Log

A running handoff log so work can continue on any machine. Update it at the end
of each session. For *how the code works*, see `architecture.md`,
`code-explainer.md`, and `functions-workflow.md`; this file is the *narrative of
what's been done, why, and what's next*.

## How to resume on another computer

```bash
git clone git@github.com:stevirn/PitMate.git
cd PitMate
# 1. build the UI (needs Node.js)
cd frontend && npm install && npm run build && cd ..
# 2. run the backend serving the UI, with synthetic data
cd backend && go run . -static ../frontend/dist -mock
# open http://localhost:8080
```

- Requires **Go** (developed on 1.26; `go.mod` declares 1.25) and **Node.js**
  (for the frontend build only).
- The LMU adapter only reads real data on **Windows** with LMU + the rFactor2
  Shared Memory Map Plugin running. Everywhere else it reports "not connected" —
  use `-mock` for development.
- Frontend hot-reload dev: run the backend (step 2), then in another terminal
  `cd frontend && npm run dev` and open http://localhost:5173 (it proxies `/ws`).
- Verify a checkout: from `backend/`, `go test ./...` and
  `GOOS=windows GOARCH=amd64 go build ./...`; from `frontend/`, `npm run build`.

## Current status (as of Session 4)

Working end to end: game → adapter → server → **browser cockpit**.

```
[LMU] -> rF2 Shared Memory Plugin -> [LMU adapter] -> telemetry.Frame
      -> [Go server: WebSocket + static files] -> [Svelte cockpit UI]
```

- ✅ Repo scaffold, folder structure, docs, Go module `github.com/stevirn/PitMate`
- ✅ Game-agnostic data model — `backend/telemetry/types.go` (the central file)
- ✅ WebSocket broadcast loop — `backend/server/` (hub fan-out, debug page)
- ✅ LMU shared-memory adapter — `backend/adapters/lmu/` (tested + Windows-compiled)
- ✅ Svelte UI — `frontend/` (shell + WS store + Live Data tab; partial Car/
  Settings; placeholders for Strategy/Coaching/Driver Vs.). Builds under
  Svelte 5 / Vite 8; served end-to-end by the Go binary.
- ⬜ Live on-PC validation of the LMU adapter — not done (see caveat below)
- ⬜ UI not yet visually confirmed in a browser (build + serving verified)

## Session history

### Session 1 — Scaffold, data model, docs
- Initialised git + full folder skeleton; module `github.com/stevirn/PitMate`.
- Wrote `telemetry/types.go` — the game-agnostic `Frame` and all sub-structs
  (session, car, timing, energy/hybrid, tires, speed, systems, damage, race
  state, flags, events). Fields rF2 may not expose are marked
  `// TODO: verify LMU availability`.
- Wrote `architecture.md`, `code-explainer.md` (for non-coders),
  `functions-workflow.md` (UI spec), `README.md`. Compilable stubs everywhere.

### Session 2 — WebSocket broadcast loop
- `server/hub.go`: classic hub pattern — one owner goroutine (lock-free),
  per-client send/receive goroutines, **encode-once** JSON fan-out, slow-client
  drop, ping/pong keepalive.
- `server/websocket.go`: `Server.New(addr, staticDir)`; `Broadcast` **stamps
  `Timestamp` + `Sequence`** (single source of truth) then encodes once; `/ws`
  upgrade; static file serving with a built-in **debug page** fallback; graceful
  context shutdown.
- `main.go`: flags (`-mock/-bind/-port/-hz/-static`), produce→broadcast loop at
  `UpdateHz`, signal shutdown, `source` interface (adapter or mock).
- `mock.go`: synthetic moving telemetry for testing without the game.
- Tests (race-clean): broadcast reaches client w/ stamped sequence; slow client
  dropped; mock source moves & JSON-encodes.
- Dependency added: `github.com/gorilla/websocket`.

### Session 3 — LMU shared-memory adapter
- Confirmed via research that LMU uses the **rFactor2 Shared Memory Map Plugin**
  (same as rF2). Transcribed the buffer layout from the plugin header and
  cross-checked against the reference reader `pyRfactor2SharedMemory`.
- `adapters/lmu/rf2_structs.go`: byte-exact Go mirror of the Telemetry + Scoring
  buffers (Windows LLP64 `long`=32-bit, prepended version-tear block, natural
  alignment).
- `adapters/lmu/mapping.go`: **pure** rF2 → `telemetry.Frame` translation
  (incl. Kelvin→Celsius, cumulative→per-sector splits, class positions, gaps,
  flags). Fully unit-tested on any OS.
- `adapters/lmu/reader_windows.go`: opens named mappings (binds
  `OpenFileMappingW` from kernel32 — `x/sys/windows` lacks it, and
  `CreateFileMapping` would wrongly open-or-create), read-only view, snapshot
  copy with begin/end tear retry.
- `adapters/lmu/reader_other.go`: `!windows` stub.
- `adapters/lmu/adapter.go`: reader interface + lazy connect/read/close.
- Tests: `layout_test.go` (size/offset guards), `mapping_test.go` (3-car race
  fixture).
- Dependency added: `golang.org/x/sys`.

### Session 4 — Svelte cockpit UI
- Vite + **Svelte 5** project under `frontend/` (written in classic store/`$:`
  style, which Svelte 5 compiles in legacy mode). Deps: svelte 5, vite 8,
  @sveltejs/vite-plugin-svelte 7. Builds clean on Node 26.
- Post-session fix: `main.js` uses Svelte 5's `mount(App, …)` (Svelte 4's
  `new App(…)` throws at runtime in 5).
- `src/lib/telemetry.js`: WebSocket store (`frame`, `status`, `connected`,
  `framesReceived`), auto-reconnect, `reconnect()`. `src/lib/format.js`:
  lap-time/gap/clock/pct/gear formatters that show "—" for missing data.
- `src/App.svelte`: header (title, connection dot, tab bar) + `Banner` +
  `{#if}` tab switch. Components: `Panel`, `Stat`, `Bar`, `TireGrid`, `Banner`,
  `Placeholder`.
- Tabs: **LiveData** (full: session, position/timing/sectors, fuel+hybrid,
  tires, speed), **CarManagement** (partial), **Settings** (partial: conn
  diagnostics + reconnect), and placeholders for Strategy/Coaching/Driver Vs.
- Banner surfaces connection problems + flag/safety-car state; UI clearly shows
  "no data" when the game isn't feeding.
- README/docs updated with the `npm install && npm run build` +
  `go run . -static ../frontend/dist` flow.
- **Not built/run here** (no Node on the dev box) — see caveat.

### Session 5 — live-validation prep
- Confirmed the Svelte UI works with mock data (user verified in browser).
- Added the `-dump` flag: prints a one-second telemetry summary to the console
  (`backend/dump.go`), so the LMU adapter can be validated against the in-game
  HUD without a browser.
- Wrote `docs/validation.md`: plugin install steps, how to run on the gaming PC,
  and a field-by-field validation checklist (flags the inferred enum mappings).
- Next: actually run it on the Windows gaming PC (plugin not yet installed).

### Session 6 — first live test: pack(4) layout fix
- First on-PC run: `OpenFileMapping` succeeded (plugin loaded!) but `MapViewOfFile`
  failed with **ACCESS_DENIED** — i.e. our struct was bigger than the real buffer.
- Root cause: the plugin compiles its structs under **`#pragma pack(push, 4)`**
  (confirmed in the plugin source; the Python reference reader uses `_pack_ = 4`).
  Go always 8-aligns float64 and has no struct packing, so our structs were too big.
- Fix: every 8-byte double is now `rf2f64` ([2]uint32, 4-byte aligned) with a
  `.val()` accessor, so the Go layout matches pack(4) exactly. Updated
  `rf2_structs.go`, `mapping.go` (`.val()` everywhere), `mapping_test.go` (`d()`
  helper), and `layout_test.go` (pack(4) sizes: vehicleTelemetry 1888,
  vehicleScoring 584, scoringInfo 548, wheel 260).
- Reader hardened: maps the whole object (length 0, no more ACCESS_DENIED),
  VirtualQuery for the real size, bounded copy, and logs buffer-size vs struct-size
  on connect so a residual mismatch is obvious.
- Status: builds/tests/cross-compiles clean. Awaiting the next on-PC run to confirm
  real values now flow (and to run the validation checklist).

### Session 7 — live validation results + fixes
- Live dump validated by user. OK: speed/gear/rpm, fuel, oil/water, tire temps
  (surface), brake bias, lap time, position, gaps, track name, tire wear,
  session type (practice only), pit status, compound. Untested: flags, class
  position (GT3 only), wetness.
- Fixed: dump now prints **sector times** (last + best) and labels tire temps as
  **surface** (+ adds core°C); UI tire grid gained a "surface temp" legend.
- **Virtual energy (VE)** is NOT in the rF2 shared memory (confirmed in plugin
  source — only mFuel + mBatteryChargeFraction). LMU exposes VE via its **REST
  API**: `GET http://localhost:6397/rest/strategy/usage` (per TinyPedal's
  lmu_restapi.py). Plan: add a small LMU REST client as a second data source and
  merge VE into telemetry.Energy. Need the live JSON schema before implementing.

## Key decisions (don't silently reverse)

- **Server stamps `Timestamp` + `Sequence`** on `Broadcast` (not the adapter), so
  ordering has one source of truth regardless of data source.
- **Slow clients are dropped, frames are disposable** — telemetry is a stream
  where only the latest frame matters; a dropped browser just reconnects.
- **Adapter split**: `rf2_structs` (layout) / `reader_*` (OS) / `mapping` (pure
  logic). Only the reader is OS-specific; the heavy logic is testable anywhere.
- **`gorilla/websocket`** chosen for the canonical, well-documented hub pattern.
- **Frontend = Svelte 5 + Vite, classic store/`$:` style** (legacy mode, not
  runes). Tabs read the `telemetry.js` stores directly rather than via props.
  App is mounted with `mount()` (Svelte 5 API).

## Open items / next steps

1. **Eyeball the UI in a browser**: `cd frontend && npm run build`, then
   `cd backend && go run . -static ../frontend/dist -mock` and open
   http://localhost:8080. Build + serving are verified; the visual/runtime check
   in an actual browser is the remaining step.
2. **Validate the LMU adapter against a live session** (Windows + LMU + plugin) —
   **this is the current active task.** Full procedure + checklist in
   `docs/validation.md`. Use the new `-dump` flag (one-second console summary) to
   compare to the in-game HUD. Confirm/adjust the rF2 enum mappings flagged in
   `mapping.go`: session type (`mSession`), flags/safety car (`mGamePhase`,
   `mYellowFlagState`), pit state (`mPitState`). Re-verify tire-wear direction
   (1.0 = new assumed) and `mAvgPathWetness` as the wetness source.
   Plugin not yet installed as of this writing (see validation.md step 1).
3. **Flesh out the placeholder tabs**: Strategy (track-position circle, pit
   params), Driver Coaching, Driver Vs. Components to reuse: `Panel/Stat/Bar`.
4. **Stateful adapter enrichments** (currently `TODO`/zero): fuel-per-lap from
   frame deltas, session top speed, event log (contacts/pit stops) → also unlocks
   the event-timeline overlay and crash banners in the UI.
5. **Fields rF2 doesn't expose**: car number, TC/ABS/ARB/engine-map, detailed
   damage zones — may need the plugin's PitInfo/Extended buffers later.
6. **Optional**: embed `frontend/dist` into the Go binary (`go:embed`) for a true
   single-file deploy, instead of serving from a directory via `-static`.

## Known caveats

- **The UI builds and is served correctly, but hasn't been eyeballed in a
  browser** — compilation (Svelte 5 / Vite 8) and asset serving via the Go
  binary are verified; the actual on-screen render with live mock data is the
  last unconfirmed step.
- The Windows reader has not been run against the game; it's verified only by
  tests + cross-compile. See the validation note in `architecture.md`.
- `GOOS=windows go vet` reports one intentional `unsafe.Pointer` finding (the
  `MapViewOfFile` mmap boundary). `go vet -unsafeptr=false` is otherwise clean.
