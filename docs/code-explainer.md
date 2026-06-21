# PitMate Code Explainer (for non-coders)

This document explains, in plain language, what each part of the PitMate code
does and what the technical words mean. It is updated whenever a new file or
concept is introduced. If you are not a programmer, start here.

## Key concepts

**Go** — the programming language used for the part of PitMate that runs on the
gaming PC. It compiles into a single program file (a "binary") that you can just
run, with nothing else to install.

**Svelte** — the toolkit used to build the visual interface (the "frontend")
that the strategist looks at in a web browser.

**A struct** — in Go, a "struct" is just a labelled container for related pieces
of information. For example, a "tire" struct might hold temperature, pressure,
and wear all together. PitMate describes the whole race as a set of nested
structs. Think of it like a form with named boxes to fill in.

**A WebSocket** — a normal web page asks the server for information once and then
sits still. A WebSocket is instead an open, two-way phone line between the server
and the browser that stays connected, so the server can keep pushing fresh data
many times per second. PitMate uses it to stream live telemetry to the cockpit.

**JSON** — a simple text format for sending structured data. PitMate's Go structs
are converted to JSON to travel over the WebSocket, and the browser converts them
back into something it can display.

**Shared memory** — a block of memory that a running game (Le Mans Ultimate)
publishes so other programs on the same computer can read its live data. This is
how PitMate finds out what is happening in the race.

**An adapter** — the one piece of PitMate that understands a specific game. It
reads that game's shared memory and translates it into PitMate's own standard
format. See `docs/architecture.md` for why this matters.

**The broadcast loop** — the heartbeat of the live data. Many times per second,
the server takes the latest race snapshot, converts it to JSON once, and sends
that same copy to every connected browser at the same time. It uses the well-known
"hub" pattern: a single manager (the hub) keeps the list of connected browsers
and hands each new snapshot to all of them.

**A goroutine** — Go's name for a lightweight task that runs at the same time as
others. PitMate gives each connected browser its own goroutines (one for sending,
one for listening) so a slow browser can't hold up the rest.

**Ping / pong** — tiny "are you still there?" messages the server sends each
browser. If a browser stops answering (e.g. the laptop was closed), the server
notices and cleans up that dead connection.

## What each file does

### Backend (the Go program on the gaming PC)

- **`backend/main.go`** — the starting point of the program. It reads the
  settings and command-line flags, picks a data source (the real LMU adapter, or
  the mock generator when `-mock` is given), starts the server, and runs the
  broadcast loop: read a snapshot at the chosen rate and send it to all browsers.
  It also handles Ctrl+C for a clean shutdown.

- **`backend/mock.go`** — a synthetic data generator used for testing. With the
  `-mock` flag it produces plausible, continuously moving values (lap progress,
  draining fuel, oscillating tire temps) so the whole pipeline can be exercised
  without the game running. It is a test tool, not a game adapter.

- **`backend/telemetry/types.go`** — the most important file in the project. It
  defines all the "boxes on the form": every piece of race information PitMate
  can hold (session, car, tires, fuel, position, flags, and so on), written in a
  way that does not depend on any particular game. Every game adapter must fill
  in this same form, and the browser can only ever show what is on it.

- **`backend/config/config.go`** — the settings: which network address and port
  to use, which game to read, how many updates per second to send, where the
  built frontend lives, and whether to use mock data. Keeping these in one place
  means they are never scattered through the code.

- **`backend/adapters/lmu/`** — the Le Mans Ultimate translator: the only part
  of PitMate that knows anything about LMU. LMU publishes its live data through
  the *rFactor2 Shared Memory Map Plugin* (a free add-on installed into the
  game), which writes the race state into a block of shared memory. This folder
  reads that block and fills in the standard form from `types.go`. It is split
  into several files:
  - **`adapter.go`** — ties everything together: on each tick it asks the
    platform reader for the raw data and the mapper to translate it.
  - **`rf2_structs.go`** — a byte-for-byte description, in Go, of exactly how the
    plugin lays out its data in memory. This has to match the plugin precisely or
    every value would be misread.
  - **`reader_windows.go`** — the Windows-only code that opens the shared memory
    and copies out a clean snapshot. (Reading live shared memory needs a tiny
    "did the game change it mid-read?" check, which this handles.)
  - **`reader_other.go`** — a stub for Mac/Linux (where LMU does not run), so the
    project still builds and the adapter simply reports "not connected". This is
    why development works on a non-Windows machine using `-mock`.
  - **`mapping.go`** — the translation itself: raw plugin numbers in, PitMate's
    standard form out (including unit conversions, e.g. tire temperatures from
    Kelvin to Celsius). This is plain logic with no Windows dependency, so it is
    fully tested everywhere.
  - **`restapi.go`** — a second data source. Some things LMU does NOT put in
    shared memory (notably **virtual energy**, its per-stint energy budget) are
    available from a small web server LMU runs on the gaming PC (a "REST API").
    This file quietly polls that web server in the background and folds the
    result into the same standard form, so the rest of PitMate doesn't care that
    it came from a different place. If the web server isn't reachable, PitMate
    just carries on with the shared-memory data. (Being plain web requests, this
    is testable on any machine.)
  - **`layout_test.go` / `mapping_test.go`** — automated checks that the memory
    description has the right size/shape and that the translation produces the
    expected values.

- **`backend/server/websocket.go`** — the server's public face. It hands the
  cockpit web page (or a built-in debug page) to browsers, accepts WebSocket
  connections, and stamps each outgoing snapshot with a time and a sequence
  number before sending. It never knows which game is running; it only handles
  the standard form.

- **`backend/server/hub.go`** — the broadcast loop itself: the hub that keeps the
  list of connected browsers and fans each snapshot out to all of them, the
  per-browser send/receive goroutines, the slow-browser drop logic, and the
  ping/pong keepalive.

- **`backend/server/server_test.go`** — automated tests proving a broadcast
  actually reaches a connected browser and that a stuck browser gets dropped
  instead of jamming everyone else.

### Frontend (the cockpit web page)

The frontend is a **Svelte** app, built by **Vite** into plain files the Go
server hands to the browser. "Svelte" is a toolkit that turns `.svelte` files
(HTML + a little JavaScript + styles, all in one file) into a fast web page;
"Vite" is the tool that bundles it. You build it once with `npm run build`.

- **`frontend/index.html` / `src/main.js`** — the page shell and the line that
  starts the app.
- **`frontend/src/app.css`** — global styling (the dark, monospace, readable
  look). Colors are defined once as variables so they're easy to change.
- **`frontend/src/lib/telemetry.js`** — the data pipe. It opens the WebSocket to
  the server and publishes each incoming frame as a "store" — a value that any
  part of the UI can read and that updates the screen automatically when new data
  arrives. It also tracks the connection and reconnects if it drops.
- **`frontend/src/lib/format.js`** — small helpers that turn raw numbers into
  readable text (lap times like `1:23.456`, gaps like `+2.1`, "—" for missing
  data).
- **`frontend/src/App.svelte`** — the top-level page: the header (title,
  connection light, tab buttons), the status/flag banner, and whichever tab is
  open.
- **`frontend/src/tabs/`** — one file per tab. `LiveData.svelte` is fully built;
  `CarManagement.svelte` and `Settings.svelte` are partial; `StrategyCalls`,
  `DriverCoaching`, and `DriverVs` are placeholders describing what's coming.
- **`frontend/src/components/`** — reusable visual pieces shared by the tabs:
  `Panel` (a titled box), `Stat` (a labelled value), `Bar` (a fill bar for fuel/
  wear/charge), `TireGrid` (the four tires), `Banner` (the alert strip), and
  `Placeholder` (the "not built yet" panel).
- **`frontend/package.json` / `vite.config.js`** — the frontend's dependencies,
  build commands, and dev-server settings.

## Current status

PitMate now works end to end: the data model, the **broadcast loop**, the **LMU
shared-memory adapter**, and a **Svelte cockpit UI** all exist. On Windows with
LMU and the shared-memory plugin running it reads real race data; on any machine
you can run the backend with `-mock` and watch synthetic data fill the cockpit in
a browser. The Live Data tab is fully built; Car Management and Settings are
partial; Strategy, Coaching, and Driver Vs. are placeholders. Still pending:
validating the adapter against a live game (see `docs/architecture.md`), and
fleshing out the remaining tabs.
