# PitMate Architecture

This document explains how PitMate is put together, in plain language. It is the
first thing to read before touching the code.

## The big picture

PitMate is software for the **strategist** — the person supporting a sim racing
driver, not the one driving. The driver is busy driving; the strategist watches
fuel, tires, weather, gaps to other cars, and calls the pit strategy. PitMate
gives that person a clear cockpit of their own, running in a web browser on a
separate screen or even a separate machine.

To do that, PitMate has to take messy, game-specific data out of the racing game
and turn it into something clean and consistent that a UI can display. It does
this in **three layers**.

## The three layers

```
[ LMU Game ]
     |  shared memory (Windows only)
     v
[ Bridge Adapter ]   <-- swappable per game; the ONLY layer that knows about LMU
     |  normalized Go struct (game-agnostic)
     v
[ Go Server ]        <-- WebSocket server + static file server for the UI
     |  WebSocket (JSON)
     v
[ Svelte Frontend ]  <-- runs in any browser, any OS, any machine on the LAN
```

### Layer 1 — The Bridge Adapter

A racing game exposes its live data in its own particular way. Le Mans Ultimate
publishes telemetry through **shared memory** — a block of memory on the Windows
machine running the game that other programs can read. Every game does this
differently, with different field names, units, and quirks.

The **adapter** is the one and only part of PitMate that understands a specific
game. The LMU adapter (`backend/adapters/lmu/adapter.go`) reads LMU's shared
memory and **translates** it into PitMate's own data model — a set of Go structs
that are the same no matter which game produced them.

Think of the adapter as a translator. The game speaks "LMU"; everything else in
PitMate speaks "PitMate". The adapter is the only bilingual part.

### Layer 2 — The Go Server

The server (`backend/server/`) takes the normalized data from the adapter and
does two jobs:

1. **Serves the UI.** The Svelte frontend is built into plain HTML/JS/CSS files,
   and the Go server hands those files to any browser that connects. This is why
   the strategist needs nothing installed — they just open a browser.
2. **Streams the data.** It opens a **WebSocket** to each connected browser and
   continuously pushes the latest telemetry as JSON.

Crucially, the server is **game-agnostic**. It never imports the LMU adapter and
never sees an LMU-specific field. It only ever handles PitMate's normalized data
model. That keeps it simple and means it never has to change when a new game is
added.

### Layer 3 — The Svelte Frontend

The frontend (`frontend/`) is the actual cockpit the strategist looks at. It
connects to the server's WebSocket, receives the stream of JSON snapshots, and
renders them into tabs and overlays (Live Data, Strategy Calls, Driver Coaching,
and so on). It is just a web app, so it runs in any browser, on any operating
system, on any machine on the same local network.

## What travels over the WebSocket

The server sends a single object called a **Frame**, defined in
`backend/telemetry/types.go`. One Frame is a **complete snapshot of the race at
one instant**: session info, the player's car (fuel, tires, lap times, systems,
damage, position), every competitor, the current flag/safety-car state, and a
rolling log of notable events. The frontend simply re-renders from whatever the
latest Frame says.

`types.go` is the heart of the whole project. Every adapter must produce Frames
in exactly this shape, and the frontend can only ever display what the Frame
contains. Adding a new feature almost always starts there.

## Why the layers are separated

- **One translator, many consumers.** Game weirdness is quarantined inside the
  adapter. The server and UI never have to deal with it.
- **The project never corners itself.** Because the server and UI only know the
  normalized model, they don't care which game is running.
- **Easy to reason about.** Each layer has one job and a clean boundary.

## How a new game is added later

1. Write a new adapter package, e.g. `backend/adapters/acc/adapter.go`, that
   reads that game's data and fills in the same `telemetry.Frame` structs.
2. Select it in configuration (`backend/config/config.go`).

That's it. The Go server and the entire Svelte frontend are untouched, because
they already speak only the game-agnostic model. If a new game exposes data the
model doesn't have yet, the model in `types.go` is extended once and every game
benefits.

## Deployment

The Go program compiles to a **single binary** that runs on the gaming PC. That
one binary serves both the WebSocket and the Svelte files. The strategist opens
a browser on any machine on the same network, types in the gaming PC's LAN IP
address (and port), and the cockpit loads. Nothing is installed on the
strategist's machine.

## Where things live

| Path | Role |
|---|---|
| `backend/telemetry/types.go` | The game-agnostic data model (most important file) |
| `backend/adapters/lmu/` | The LMU translator (only game-aware code) |
| `backend/server/` | WebSocket + static file server (game-agnostic) |
| `backend/config/` | IP, port, adapter selection, session options |
| `backend/main.go` | Wires the adapter to the server |
| `frontend/` | The Svelte strategist cockpit UI |
| `docs/` | This file plus the code explainer and functions/workflow spec |
