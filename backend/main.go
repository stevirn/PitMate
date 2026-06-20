// Command PitMate is the single binary that runs on the gaming PC. It:
//  1. starts a game adapter (LMU today) that reads telemetry,
//  2. normalizes that data into telemetry.Frame structs,
//  3. serves those frames to the browser UI over a WebSocket, and
//  4. serves the built Svelte frontend as static files.
//
// Session 2 wires the produce->broadcast loop end to end. The LMU adapter is
// still a stub (it returns a disconnected frame), so use the -mock flag to see
// synthetic data flow through the pipeline and into a browser.
package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stevirn/PitMate/adapters/lmu"
	"github.com/stevirn/PitMate/config"
	"github.com/stevirn/PitMate/server"
	"github.com/stevirn/PitMate/telemetry"
)

// source is anything that can produce a telemetry frame on demand. Both the LMU
// adapter and the mock generator satisfy it, so the broadcast loop doesn't care
// which one it's pumping.
type source interface {
	Read() telemetry.Frame
}

func main() {
	cfg := config.Default()

	// Command-line flags override the defaults.
	flag.StringVar(&cfg.BindAddress, "bind", cfg.BindAddress, "address to bind (0.0.0.0 = all interfaces)")
	flag.IntVar(&cfg.Port, "port", cfg.Port, "TCP port to listen on")
	flag.StringVar(&cfg.StaticDir, "static", cfg.StaticDir, "directory of built Svelte files (empty = debug page)")
	flag.IntVar(&cfg.UpdateHz, "hz", cfg.UpdateHz, "telemetry frames per second to broadcast")
	flag.BoolVar(&cfg.MockData, "mock", cfg.MockData, "stream synthetic data instead of the real adapter")
	var dump bool
	flag.BoolVar(&dump, "dump", false, "print a one-second summary of the telemetry to the console (for validation)")
	var lmuDebug bool
	flag.BoolVar(&lmuDebug, "lmudebug", false, "log raw LMU enum fields (pit state, flags, game phase) for the player (for diagnosing mappings)")
	flag.Parse()

	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	// Pick the data source: synthetic generator or the (stubbed) LMU adapter.
	var src source
	if cfg.MockData {
		log.Print("PitMate: using MOCK data source")
		src = newMockSource()
	} else {
		log.Print("PitMate: using LMU adapter (reads LMU shared memory on Windows; reports disconnected elsewhere or until the game is running)")
		a := lmu.New()
		a.Debug = lmuDebug
		if err := a.Connect(); err != nil {
			log.Printf("PitMate: adapter connect failed: %v", err)
		}
		src = a
	}

	srv := server.New(cfg.Addr(), cfg.StaticDir)

	// ctx is cancelled on Ctrl+C / SIGTERM, which unwinds both the server and
	// the broadcast loop for a clean shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Run the HTTP/WebSocket server in the background.
	go func() {
		if err := srv.ListenAndServe(ctx); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}()

	// The broadcast loop: read a frame at the configured rate and push it out.
	broadcastLoop(ctx, srv, src, cfg.UpdateHz, dump)

	// Release any resources the data source holds (e.g. the LMU adapter's
	// shared-memory handles on Windows).
	if c, ok := src.(io.Closer); ok {
		if err := c.Close(); err != nil {
			log.Printf("PitMate: error closing data source: %v", err)
		}
	}

	log.Print("PitMate: shut down cleanly")
}

// broadcastLoop reads one frame from src every 1/hz seconds and broadcasts it
// to all connected browsers, until ctx is cancelled. When dump is true it also
// prints a summary of the latest frame to the console once per second.
func broadcastLoop(ctx context.Context, srv *server.Server, src source, hz int, dump bool) {
	interval := time.Second / time.Duration(hz)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastDump time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			frame := src.Read()
			srv.Broadcast(frame)
			if dump && now.Sub(lastDump) >= time.Second {
				lastDump = now
				dumpFrame(frame)
			}
		}
	}
}
