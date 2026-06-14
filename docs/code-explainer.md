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

- **`backend/adapters/lmu/adapter.go`** — the Le Mans Ultimate translator. The
  only file that knows anything about LMU. It reads LMU's shared memory and fills
  in the standard form from `types.go`. Currently a stub.

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

- **`frontend/src/App.svelte`** — the top-level page of the cockpit. Later it
  will hold the tabs and the warning banners. Currently a placeholder.

- **`frontend/src/tabs/`** — one file per main tab (Live Data, Strategy Calls,
  etc.). Empty for now.

- **`frontend/src/components/`** — reusable visual pieces shared by the tabs
  (gauges, timelines, and so on). Empty for now.

- **`frontend/package.json`** — lists the frontend's settings and build
  commands. Not wired up yet.

## Current status

The structure, the standard data form, and the **broadcast loop** all exist: run
the backend with `-mock` and open a browser to watch live data stream in. Still
to be built: the part that reads the real game (the LMU adapter) and the part
that draws the cockpit (the Svelte UI).
