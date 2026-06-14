// Package config holds PitMate's runtime configuration: which network address
// to listen on, which game adapter to use, and session-level options. Keeping
// this in one place means the rest of the code never hardcodes ports or IPs.
//
// Session 1 scaffolding: the struct and defaults are defined; loading from a
// file/flags/env is added later.
package config

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
}

// Default returns sensible starting values. These are placeholders for Session 1.
func Default() Config {
	return Config{
		BindAddress:  "0.0.0.0",
		Port:         8080,
		AdapterID:    "lmu",
		UpdateHz:     10,
		EventLogSize: 100,
	}
}
