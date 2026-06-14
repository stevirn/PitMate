# PitMate

PitMate is a real-time strategist cockpit for sim racing — a dedicated interface
for the person *not* driving. It gives the strategist race awareness, driver
coaching data, and strategy execution tools. The first supported game is
**Le Mans Ultimate (LMU)**, with multi-game support designed in from the start.

## Stack

| Layer | Technology |
|---|---|
| Backend | Go |
| Frontend | Svelte |
| Transport | WebSocket (JSON) |
| Docs | Markdown (inside repo) |
| Version control | Git → GitHub |

## Architecture (summary)

PitMate has three layers. A **Bridge Adapter** is the only code that understands
a specific game — it reads the game's data (LMU shared memory on Windows) and
translates it into one game-agnostic data model. A **Go server** receives that
normalized model, serves it to the browser over a WebSocket as JSON, and also
serves the Svelte UI as static files. The **Svelte frontend** runs in any
browser on any machine on the LAN. Because only the adapter knows about the
game, adding a new game later means writing a new adapter and nothing else.
One Go binary runs on the gaming PC; the strategist just opens a browser and
points it at the gaming PC's LAN IP — no install on their machine.

See [`docs/architecture.md`](docs/architecture.md) for the full explanation and
diagrams, and [`backend/telemetry/types.go`](backend/telemetry/types.go) for the
central game-agnostic data model.

## How to run

Not runnable yet — this is early scaffolding. See
[`docs/architecture.md`](docs/architecture.md).

## Project status

**Early scaffolding.** The repository structure, the game-agnostic data model,
and the documentation are in place. No UI and no real game-reading logic exist
yet.
