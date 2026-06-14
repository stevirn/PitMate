// This file implements the broadcast loop — the heart of how PitMate gets live
// data to the browser.
//
// The design is the classic "hub" pattern (the same one in the gorilla/websocket
// chat example, https://github.com/gorilla/websocket/tree/master/examples/chat):
//
//   - A single hub goroutine owns the set of connected clients. Because only one
//     goroutine touches that set, we never need locks around it.
//   - Each browser connection is a *client with its own buffered send channel and
//     its own goroutines for reading and writing.
//   - To broadcast a frame, we JSON-encode it ONCE and hand the same bytes to
//     every client. Encoding once (not per client) is what keeps this cheap even
//     with many connected browsers.
//   - If a client is too slow to keep up (its buffer fills), we drop it rather
//     than letting one slow browser stall everyone else. Telemetry is a constant
//     stream where only the latest frame matters, so a dropped client simply
//     reconnects and resyncs.
package server

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
)

// Timing constants for the WebSocket keepalive. These detect and clean up
// connections that have silently died (e.g. the strategist closed the laptop
// lid) so they don't linger as half-open connections.
const (
	// writeWait is the maximum time allowed to write one message to a client.
	writeWait = 10 * time.Second

	// pongWait is how long we wait for a pong reply before assuming the client
	// is gone.
	pongWait = 60 * time.Second

	// pingPeriod is how often we send a ping. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// sendBuffer is how many frames we'll queue for a client before considering
	// it too slow and dropping it.
	sendBuffer = 8
)

// hub owns the set of connected clients and fans out frames to them. All access
// to the clients map happens inside the run loop, so no mutex is needed.
type hub struct {
	// clients is the set of currently connected browsers.
	clients map[*client]bool

	// broadcast carries already-JSON-encoded frames to be sent to every client.
	broadcast chan []byte

	// register and unregister add/remove clients from the set.
	register   chan *client
	unregister chan *client

	// count lets other goroutines ask the run loop how many clients are
	// connected without touching the clients map directly (which would race).
	count chan chan int
}

// newHub creates an empty hub. Call run in its own goroutine to start it.
func newHub() *hub {
	return &hub{
		clients:    make(map[*client]bool),
		broadcast:  make(chan []byte, 1),
		register:   make(chan *client),
		unregister: make(chan *client),
		count:      make(chan chan int),
	}
}

// clientCount returns the number of currently connected clients. It is safe to
// call from any goroutine because the read happens inside the run loop.
func (h *hub) clientCount() int {
	reply := make(chan int)
	h.count <- reply
	return <-reply
}

// run is the hub's single goroutine. It is the only place the clients map is
// read or written, which is what makes the whole hub lock-free. It exits when
// ctx is cancelled (i.e. on server shutdown), closing every client as it goes.
func (h *hub) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			for c := range h.clients {
				close(c.send)
				delete(h.clients, c)
			}
			return

		case reply := <-h.count:
			reply <- len(h.clients)

		case c := <-h.register:
			h.clients[c] = true

		case c := <-h.unregister:
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}

		case msg := <-h.broadcast:
			for c := range h.clients {
				select {
				case c.send <- msg:
					// queued for this client
				default:
					// The client's buffer is full — it's too slow. Drop it; it
					// will reconnect and resync from the next frame.
					delete(h.clients, c)
					close(c.send)
				}
			}
		}
	}
}

// client is one connected browser. It pairs the WebSocket connection with a
// buffered channel of outgoing messages.
type client struct {
	hub  *hub
	conn *websocket.Conn

	// send holds frames waiting to be written to this client's socket.
	send chan []byte
}

// writePump is the only goroutine that writes to this client's socket (gorilla
// requires at most one concurrent writer). It drains the send channel and also
// emits periodic pings to keep the connection alive.
func (c *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed our channel — tell the client we're done.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump is the only goroutine that reads from this client's socket. PitMate
// never expects meaningful data FROM the browser (telemetry flows one way), so
// this exists mainly to process pong replies and to notice when the connection
// closes. When it returns, the client is unregistered.
func (c *client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512) // we expect nothing large from the browser
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}
