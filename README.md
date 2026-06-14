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

**1. Build the UI** (needs Node.js; produces `frontend/dist/`):

```bash
cd frontend
npm install
npm run build
```

**2. Run the backend, serving the built UI:**

```bash
cd ../backend
go run . -static ../frontend/dist -mock
```

Then open <http://localhost:8080>. `-mock` streams synthetic data so you can see
the cockpit working without the game; drop it on a Windows gaming PC with Le Mans
Ultimate + the shared-memory plugin running to read real telemetry. If you skip
the UI build (no `-static`, or the directory is missing), the server falls back
to a built-in debug page that shows each frame as raw JSON.

**Frontend development** (hot reload) runs Vite on its own port and proxies the
WebSocket to the backend — run the backend (step 2, with or without `-static`)
and, in another terminal, `cd frontend && npm run dev`, then open
<http://localhost:5173>.

Useful backend flags (run `go run . -h` for all):

| Flag | Default | Meaning |
|---|---|---|
| `-mock` | off | stream synthetic data instead of the real LMU adapter |
| `-dump` | off | print a one-second telemetry summary to the console (for validation) |
| `-static` | _(empty)_ | directory of built Svelte files (empty = debug page) |
| `-bind` | `0.0.0.0` | address to bind (`0.0.0.0` = reachable on the LAN) |
| `-port` | `8080` | TCP port |
| `-hz` | `10` | telemetry frames broadcast per second |

See [`docs/architecture.md`](docs/architecture.md) for how the layers connect, and
[`docs/validation.md`](docs/validation.md) for validating the LMU adapter against
the running game.

## Project status

**End-to-end working.** In place: the game-agnostic data model, the **WebSocket
broadcast loop**, the **LMU shared-memory adapter** (reads Le Mans Ultimate via
the rFactor2 Shared Memory Map Plugin on Windows), and a **Svelte cockpit UI**.
The UI has a full Live Data tab, a partial Car Management tab, a Settings/
diagnostics tab, and wired-in placeholders for the remaining tabs (Strategy,
Coaching, Driver Vs.).

Pending: validating the LMU adapter against a live game (see
[`docs/architecture.md`](docs/architecture.md)), and fleshing out the remaining
tabs. On non-Windows machines, develop with `-mock`.

> The LMU adapter is Windows-only at runtime (LMU and the plugin are Windows).
> The rest of PitMate builds and runs on any OS; on Linux/macOS the adapter
> reports "not connected", so develop with `go run . -mock`.
