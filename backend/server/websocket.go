// Package server exposes telemetry to the browser. It runs an HTTP server that
// (1) serves the built Svelte frontend as static files and (2) upgrades a
// WebSocket connection (the /ws endpoint) over which it pushes telemetry.Frame
// values encoded as JSON. The server is fully game-agnostic: it only ever sees
// telemetry.Frame and never imports any adapter package.
//
// The actual fan-out of frames to connected browsers lives in hub.go.
package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stevirn/PitMate/telemetry"
)

// upgrader turns an incoming HTTP request on /ws into a WebSocket connection.
type upgrader = websocket.Upgrader

// Server holds the state needed to serve the UI and stream telemetry.
type Server struct {
	// addr is the "host:port" the server listens on.
	addr string

	// staticDir is the directory of built Svelte files to serve at "/". If it is
	// empty or missing, the server falls back to a small built-in debug page so
	// the data stream can still be inspected in a browser.
	staticDir string

	// hub fans each broadcast frame out to every connected browser.
	hub *hub

	// upgrader configures the HTTP->WebSocket upgrade.
	upgrader upgrader

	// seq is the per-session frame counter stamped onto every outgoing frame.
	seq atomic.Uint64
}

// New creates a server that will listen on the given "host:port" address and
// serve built frontend files from staticDir (may be "" to use the debug page).
func New(addr, staticDir string) *Server {
	return &Server{
		addr:      addr,
		staticDir: staticDir,
		hub:       newHub(),
		upgrader: upgrader{
			// PitMate is a trusted tool on a local network and the page is served
			// from this same server, so we accept any origin rather than rejecting
			// the strategist's browser for being on another machine.
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// Broadcast sends one frame to every connected browser client as JSON. It is
// safe to call from any goroutine. The server stamps the frame's Timestamp and
// Sequence here so there is a single source of truth for ordering, then encodes
// the frame once and hands the bytes to the hub for fan-out.
//
// Frames are disposable: if the hub is momentarily busy, this drops the frame
// rather than blocking the producer, because the next frame will carry the
// latest state anyway.
func (s *Server) Broadcast(frame telemetry.Frame) {
	frame.Timestamp = time.Now().UnixMilli()
	frame.Sequence = s.seq.Add(1)

	data, err := json.Marshal(frame)
	if err != nil {
		log.Printf("server: failed to encode frame: %v", err)
		return
	}

	select {
	case s.hub.broadcast <- data:
	default:
		// Hub not ready to accept right now; skip this frame.
	}
}

// handleWS upgrades an incoming request to a WebSocket and registers it as a
// client. Each client gets its own read and write goroutines.
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("server: websocket upgrade failed: %v", err)
		return
	}

	c := &client{
		hub:  s.hub,
		conn: conn,
		send: make(chan []byte, sendBuffer),
	}
	s.hub.register <- c

	// writePump owns all writes; readPump owns all reads and unregisters on exit.
	go c.writePump()
	go c.readPump()
}

// routes builds the HTTP handler: the /ws WebSocket endpoint plus static file
// serving (or the debug page fallback) for everything else.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWS)

	if s.staticDir != "" {
		if _, err := os.Stat(s.staticDir); err == nil {
			mux.Handle("/", http.FileServer(http.Dir(s.staticDir)))
			return mux
		}
		log.Printf("server: static dir %q not found, serving built-in debug page", s.staticDir)
	}

	// Fallback: a tiny page that connects to /ws and shows incoming frames, so
	// the broadcast loop is verifiable before the Svelte UI exists.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(debugPage))
	})
	return mux
}

// ListenAndServe starts the hub, the HTTP/WebSocket server, and the static file
// handler. It blocks until ctx is cancelled, then shuts the server down
// gracefully. Returns nil on a clean shutdown.
func (s *Server) ListenAndServe(ctx context.Context) error {
	go s.hub.run(ctx)

	httpSrv := &http.Server{
		Addr:    s.addr,
		Handler: s.routes(),
	}

	// Shut the HTTP server down when the context is cancelled.
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpSrv.Shutdown(shutdownCtx)
	}()

	log.Printf("server: listening on http://%s (WebSocket at /ws)", s.addr)
	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// debugPage is a minimal HTML client used when no built Svelte frontend is
// present. It connects to the WebSocket and renders each frame as raw JSON.
const debugPage = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>PitMate — debug stream</title>
<style>
  body { font-family: monospace; background:#111; color:#0f0; margin:0; padding:1rem; }
  h1 { font-size:1rem; color:#fff; }
  #status { color:#ff0; }
  pre { white-space:pre-wrap; word-break:break-all; }
</style>
</head>
<body>
<h1>PitMate debug stream <span id="status">connecting…</span></h1>
<pre id="out">waiting for first frame…</pre>
<script>
  const status = document.getElementById('status');
  const out = document.getElementById('out');
  function connect() {
    const ws = new WebSocket('ws://' + location.host + '/ws');
    ws.onopen = () => { status.textContent = 'connected'; };
    ws.onclose = () => { status.textContent = 'disconnected — retrying…'; setTimeout(connect, 1000); };
    ws.onmessage = (ev) => {
      try { out.textContent = JSON.stringify(JSON.parse(ev.data), null, 2); }
      catch { out.textContent = ev.data; }
    };
  }
  connect();
</script>
</body>
</html>`
