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

## What each file does

### Backend (the Go program on the gaming PC)

- **`backend/main.go`** — the starting point of the program. Eventually it will
  start the adapter, start the server, and connect them. Right now it is a stub
  that just prints a message.

- **`backend/telemetry/types.go`** — the most important file in the project. It
  defines all the "boxes on the form": every piece of race information PitMate
  can hold (session, car, tires, fuel, position, flags, and so on), written in a
  way that does not depend on any particular game. Every game adapter must fill
  in this same form, and the browser can only ever show what is on it.

- **`backend/config/config.go`** — the settings: which network address and port
  to use, which game to read, how many updates per second to send. Keeping these
  in one place means they are never scattered through the code.

- **`backend/adapters/lmu/adapter.go`** — the Le Mans Ultimate translator. The
  only file that knows anything about LMU. It reads LMU's shared memory and fills
  in the standard form from `types.go`. Currently a stub.

- **`backend/server/websocket.go`** — the server. It hands the cockpit web page
  to browsers and streams the live data to them over a WebSocket. It never knows
  which game is running; it only handles the standard form. Currently a stub.

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

Early scaffolding. The structure and the standard data form exist; the parts
that read the game and draw the cockpit are still to be built.
