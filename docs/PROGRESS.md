# PitMate — Progress & Session Log

A running handoff log so work can continue on any machine. Update it at the end
of each session. For *how the code works*, see `architecture.md`,
`code-explainer.md`, and `functions-workflow.md`; this file is the *narrative of
what's been done, why, and what's next*.

## How to resume on another computer

```bash
git clone git@github.com:stevirn/PitMate.git
cd PitMate/backend
go run . -mock          # streams synthetic data; open http://localhost:8080
```

- Requires Go (developed on 1.26; `go.mod` declares 1.25). `go mod download` pulls deps.
- The LMU adapter only reads real data on **Windows** with LMU + the rFactor2
  Shared Memory Map Plugin running. Everywhere else it reports "not connected" —
  use `-mock` for development.
- Verify a checkout: from `backend/`, `go test ./...` and
  `GOOS=windows GOARCH=amd64 go build ./...`.

## Current status (as of Session 3)

Backend is functional end to end; **no UI yet**.

```
[LMU] -> rF2 Shared Memory Plugin -> [LMU adapter] -> telemetry.Frame
      -> [Go server: WebSocket + static files] -> [browser]
```

- ✅ Repo scaffold, folder structure, docs, Go module `github.com/stevirn/PitMate`
- ✅ Game-agnostic data model — `backend/telemetry/types.go` (the central file)
- ✅ WebSocket broadcast loop — `backend/server/` (hub fan-out, debug page)
- ✅ LMU shared-memory adapter — `backend/adapters/lmu/` (tested + Windows-compiled)
- ⬜ Svelte cockpit UI — not started (placeholder only)
- ⬜ Live on-PC validation of the LMU adapter — not done (see caveat below)

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

## Key decisions (don't silently reverse)

- **Server stamps `Timestamp` + `Sequence`** on `Broadcast` (not the adapter), so
  ordering has one source of truth regardless of data source.
- **Slow clients are dropped, frames are disposable** — telemetry is a stream
  where only the latest frame matters; a dropped browser just reconnects.
- **Adapter split**: `rf2_structs` (layout) / `reader_*` (OS) / `mapping` (pure
  logic). Only the reader is OS-specific; the heavy logic is testable anywhere.
- **`gorilla/websocket`** chosen for the canonical, well-documented hub pattern.

## Open items / next steps

1. **Validate the LMU adapter against a live session** (Windows + LMU + plugin).
   Compare the `/ws` debug-page JSON to in-game values. Confirm/adjust the rF2
   enum mappings flagged in `mapping.go`: session type (`mSession`), flags/safety
   car (`mGamePhase`, `mYellowFlagState`), pit state (`mPitState`). Re-verify the
   tire-wear direction (1.0 = new assumed) and `mAvgPathWetness` as the wetness
   source.
2. **Build the Svelte UI** (`frontend/`) — start with the Live Data tab consuming
   `/ws`. Then wire `-static frontend/dist` to serve the built app.
3. **Stateful adapter enrichments** (currently `TODO`/zero): fuel-per-lap from
   frame deltas, session top speed, event log (contacts/pit stops) from
   frame-to-frame changes.
4. **Fields rF2 doesn't expose**: car number, TC/ABS/ARB/engine-map, detailed
   damage zones — may need the plugin's PitInfo/Extended buffers later.

## Known caveats

- The Windows reader has not been run against the game; it's verified only by
  tests + cross-compile. See the validation note in `architecture.md`.
- `GOOS=windows go vet` reports one intentional `unsafe.Pointer` finding (the
  `MapViewOfFile` mmap boundary). `go vet -unsafeptr=false` is otherwise clean.
