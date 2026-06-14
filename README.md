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

The backend builds and runs today. The LMU adapter is still a stub, so use the
`-mock` flag to stream synthetic data through the full pipeline:

```bash
cd backend
go run . -mock
```

Then open <http://localhost:8080> in a browser. With no built Svelte frontend
present yet, the server serves a built-in debug page that connects to the
WebSocket and shows each incoming frame as live JSON.

Useful flags (run `go run . -h` for all):

| Flag | Default | Meaning |
|---|---|---|
| `-mock` | off | stream synthetic data instead of the (stubbed) adapter |
| `-bind` | `0.0.0.0` | address to bind (`0.0.0.0` = reachable on the LAN) |
| `-port` | `8080` | TCP port |
| `-hz` | `10` | telemetry frames broadcast per second |
| `-static` | _(empty)_ | directory of built Svelte files (empty = debug page) |

See [`docs/architecture.md`](docs/architecture.md) for how the layers connect.

## Project status

**Backend functional; UI pending.** In place: the repository structure, the
game-agnostic data model, the documentation, the **WebSocket broadcast loop**,
and the **LMU shared-memory adapter** that reads Le Mans Ultimate via the
rFactor2 Shared Memory Map Plugin on Windows. The adapter is covered by tests and
compiles for Windows, but still needs a validation pass against a live game (see
[`docs/architecture.md`](docs/architecture.md)). On non-Windows machines use
`-mock`. Still to come: the Svelte cockpit UI.

> The LMU adapter is Windows-only at runtime (LMU and the plugin are Windows).
> The rest of PitMate builds and runs on any OS; on Linux/macOS the adapter
> reports "not connected", so develop with `go run . -mock`.
