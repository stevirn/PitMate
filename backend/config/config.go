// Package config holds PitMate's runtime configuration: which network address
// to listen on, which game adapter to use, and session-level options. Keeping
// this in one place means the rest of the code never hardcodes ports or IPs.
//
// Session 1 scaffolding: the struct and defaults are defined; loading from a
// file/flags/env is added later.
package config

import (
	"fmt"
	"net"
	"strconv"
)

// Config is the full set of options PitMate runs with.
type Config struct {
	// Network address the server binds to, e.g. "0.0.0.0" to accept LAN
	// connections from the strategist's machine. "0.0.0.0" listens on all
	// interfaces; "127.0.0.1" would restrict to the local machine only.
	BindAddress string

	// TCP port the HTTP/WebSocket server listens on.
	Port int

	// Which game adapter to start, by short ID, e.g. "lmu".
	AdapterID string

	// How many telemetry frames per second to push to the browser. Lower values
	// reduce CPU/network use; higher values make the UI smoother.
	UpdateHz int

	// How many past events to keep in the rolling event log sent to clients.
	EventLogSize int

	// StaticDir is the directory of built Svelte files to serve. When empty or
	// missing, the server serves a built-in debug page instead.
	StaticDir string

	// MockData, when true, makes the server stream synthetic moving telemetry
	// instead of (the currently stubbed) real adapter output. Useful for
	// developing and testing the pipeline without the game running.
	MockData bool

	// LMURestURL is the base URL of LMU's local REST API, used for data not in
	// shared memory (virtual energy). Empty disables it.
	LMURestURL string
}

// Addr returns the "host:port" string the HTTP/WebSocket server listens on.
func (c Config) Addr() string {
	return net.JoinHostPort(c.BindAddress, strconv.Itoa(c.Port))
}

// Validate checks that the configuration is usable, returning an error
// describing the first problem found.
func (c Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port %d out of range 1-65535", c.Port)
	}
	if c.UpdateHz < 1 {
		return fmt.Errorf("updateHz must be at least 1, got %d", c.UpdateHz)
	}
	return nil
}

// Default returns sensible starting values. These are placeholders for Session 1.
func Default() Config {
	return Config{
		BindAddress:  "0.0.0.0",
		Port:         8080,
		AdapterID:    "lmu",
		UpdateHz:     10,
		EventLogSize: 100,
		StaticDir:    "", // empty -> built-in debug page until a Svelte build exists
		MockData:     false,
		LMURestURL:   "http://localhost:6397", // LMU REST API (virtual energy); empty disables
	}
}
