// Package server exposes telemetry to the browser. It runs an HTTP server that
// (1) serves the built Svelte frontend as static files and (2) upgrades a
// WebSocket connection over which it pushes telemetry.Frame values encoded as
// JSON. The server is fully game-agnostic: it only ever sees telemetry.Frame
// and never imports any adapter package.
//
// Session 1 scaffolding: this defines the server shape. The HTTP routes,
// WebSocket upgrade, and broadcast loop are implemented in a later session.
package server

import "github.com/stevirn/PitMate/telemetry"

// Server holds the state needed to serve the UI and stream telemetry.
type Server struct {
	// addr is the "host:port" the server listens on.
	addr string
}

// New creates a server that will listen on the given "host:port" address.
func New(addr string) *Server {
	return &Server{addr: addr}
}

// Broadcast sends one frame to every connected browser client as JSON.
//
// TODO: maintain a set of WebSocket clients and write the JSON-encoded frame
// to each, dropping clients that error.
func (s *Server) Broadcast(frame telemetry.Frame) {
	// TODO: implement JSON encode + fan-out to connected clients.
}

// ListenAndServe starts the HTTP/WebSocket server and the static file handler.
//
// TODO: register the static file handler for the built Svelte app and the
// /ws WebSocket endpoint, then serve.
func (s *Server) ListenAndServe() error {
	// TODO: implement.
	return nil
}
