// Command PitMate is the single binary that runs on the gaming PC. It will:
//  1. start a game adapter (LMU today) that reads telemetry,
//  2. normalize that data into telemetry.Frame structs,
//  3. serve those frames to the browser UI over a WebSocket, and
//  4. serve the built Svelte frontend as static files.
//
// Session 1 scaffolding: this is intentionally a stub. The real wiring of
// adapter -> server is added in a later session.
package main

import "fmt"

func main() {
	// TODO: load config, start the selected adapter, start the WebSocket +
	// static file server, and pump telemetry.Frame values to connected clients.
	fmt.Println("PitMate — early scaffolding. Nothing to run yet. See docs/architecture.md")
}
