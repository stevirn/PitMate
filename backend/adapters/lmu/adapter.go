// Package lmu is the Le Mans Ultimate adapter. It is the ONLY part of PitMate
// that knows anything about LMU. Its job is to read LMU's shared memory (on
// Windows) and translate that game-specific data into the game-agnostic
// telemetry.Frame defined in backend/telemetry/types.go.
//
// Everything above this layer (the WebSocket server and the Svelte frontend)
// only ever sees telemetry.Frame, so swapping or adding a game means writing a
// new adapter package like this one — and nothing else changes.
//
// Session 1 scaffolding: this defines the adapter shape and a stub Read. The
// real shared-memory reading is implemented in a later session.
package lmu

import "github.com/stevirn/PitMate/telemetry"

// Adapter reads Le Mans Ultimate telemetry and produces telemetry.Frame values.
type Adapter struct {
	// connected is true once we've successfully attached to LMU's shared memory.
	connected bool
}

// New creates an LMU adapter. It does not connect yet.
func New() *Adapter {
	return &Adapter{}
}

// ID is the short adapter identifier carried in every frame's SourceInfo.
func (a *Adapter) ID() string { return "lmu" }

// Name is the human-readable game name shown in the UI.
func (a *Adapter) Name() string { return "Le Mans Ultimate" }

// Connect attaches to LMU's shared memory. On non-Windows machines, or when
// LMU is not running, this will fail and the server keeps reporting "not
// connected" until it succeeds.
//
// TODO: open LMU shared-memory map (rF2/LMU plugin) and verify layout version.
func (a *Adapter) Connect() error {
	// TODO: implement shared-memory attach.
	return nil
}

// Read produces the latest snapshot as a game-agnostic Frame. While this is a
// stub it returns a disconnected frame so the rest of the pipeline can be
// developed and tested without the game.
//
// TODO: map LMU shared-memory fields into the telemetry structs.
func (a *Adapter) Read() telemetry.Frame {
	return telemetry.Frame{
		Source: telemetry.SourceInfo{
			Game:      a.Name(),
			AdapterID: a.ID(),
			Connected: a.connected,
		},
	}
}
