// Package lmu is the Le Mans Ultimate adapter. It is the ONLY part of PitMate
// that knows anything about LMU. Its job is to read LMU's shared memory (written
// by the rFactor2 Shared Memory Map Plugin, on Windows) and translate that
// game-specific data into the game-agnostic telemetry.Frame defined in
// backend/telemetry/types.go.
//
// Everything above this layer (the WebSocket server and the Svelte frontend)
// only ever sees telemetry.Frame, so swapping or adding a game means writing a
// new adapter package like this one — and nothing else changes.
//
// Structure of this package:
//   - rf2_structs.go  : byte-exact mirror of the plugin's shared-memory layout
//   - reader_windows.go / reader_other.go : platform-specific memory access
//   - mapping.go      : pure translation of rF2 structs -> telemetry.Frame
//   - adapter.go      : this file — ties the reader and mapper together
//
// Only the reader is OS-specific. On non-Windows builds the reader is a stub, so
// the adapter compiles everywhere and simply reports "not connected".
package lmu

import "github.com/stevirn/PitMate/telemetry"

// reader abstracts platform-specific access to the plugin's shared memory. It
// returns raw rF2 buffers; the mapping layer turns them into a Frame.
type reader interface {
	// read returns the latest telemetry and scoring buffers. ok is false when no
	// consistent data is currently available (game not running, or a torn read).
	read() (tel rf2Telemetry, sc rf2Scoring, ok bool)
	// close releases any OS resources held by the reader.
	close() error
}

// Adapter reads Le Mans Ultimate telemetry and produces telemetry.Frame values.
type Adapter struct {
	rd        reader
	connected bool
}

// New creates an LMU adapter. It does not open shared memory yet.
func New() *Adapter { return &Adapter{} }

// ID is the short adapter identifier carried in every frame's SourceInfo.
func (a *Adapter) ID() string { return "lmu" }

// Name is the human-readable game name shown in the UI.
func (a *Adapter) Name() string { return "Le Mans Ultimate" }

// Connect prepares the platform reader. On Windows this succeeds immediately and
// the shared memory is opened lazily on the first Read once the game is running.
// On other operating systems it returns an error explaining LMU is Windows-only;
// the adapter then simply reports "not connected" on every Read.
func (a *Adapter) Connect() error {
	rd, err := newReader()
	if err != nil {
		return err
	}
	a.rd = rd
	return nil
}

// Read produces the latest snapshot as a game-agnostic Frame. If the game is not
// running, the reader is unavailable, or the data is momentarily inconsistent,
// it returns a frame marked disconnected so the UI shows a clear "no data" state.
func (a *Adapter) Read() telemetry.Frame {
	if a.rd == nil {
		a.connected = false
		return a.disconnectedFrame()
	}
	tel, sc, ok := a.rd.read()
	if !ok {
		a.connected = false
		return a.disconnectedFrame()
	}
	a.connected = true
	return mapFrame(&tel, &sc)
}

// Close releases the reader's resources.
func (a *Adapter) Close() error {
	if a.rd == nil {
		return nil
	}
	return a.rd.close()
}

// disconnectedFrame is the empty frame sent when there is no live data.
func (a *Adapter) disconnectedFrame() telemetry.Frame {
	return telemetry.Frame{
		Source: telemetry.SourceInfo{
			Game:      a.Name(),
			AdapterID: a.ID(),
			Connected: false,
		},
	}
}
