package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stevirn/PitMate/telemetry"
)

// TestBroadcastReachesClient spins up the real server on an httptest listener,
// connects a WebSocket client, broadcasts a frame, and verifies the client
// receives it with the server-stamped Timestamp and Sequence.
func TestBroadcastReachesClient(t *testing.T) {
	s := New("", "") // addr unused; httptest provides the listener

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.hub.run(ctx)

	ts := httptest.NewServer(http.HandlerFunc(s.handleWS))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Give the hub a moment to register the new client before broadcasting.
	waitForClients(t, s, 1)

	s.Broadcast(telemetry.Frame{
		Source: telemetry.SourceInfo{Game: "Test", AdapterID: "test", Connected: true},
	})

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var got telemetry.Frame
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Source.Game != "Test" {
		t.Errorf("Source.Game = %q, want %q", got.Source.Game, "Test")
	}
	if got.Sequence != 1 {
		t.Errorf("Sequence = %d, want 1 (server should stamp it)", got.Sequence)
	}
	if got.Timestamp == 0 {
		t.Error("Timestamp = 0, want server-stamped non-zero value")
	}
}

// TestSlowClientGetsDropped verifies that a client whose buffer fills up is
// removed rather than blocking the broadcast for everyone.
func TestSlowClientGetsDropped(t *testing.T) {
	s := New("", "")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.hub.run(ctx)

	// A client that never drains its send channel.
	stuck := &client{hub: s.hub, conn: nil, send: make(chan []byte, sendBuffer)}
	s.hub.register <- stuck
	waitForClients(t, s, 1)

	// Broadcast more frames than the buffer can hold; the hub must drop the
	// client instead of blocking.
	for i := 0; i < sendBuffer+5; i++ {
		s.Broadcast(telemetry.Frame{})
		time.Sleep(2 * time.Millisecond)
	}

	waitForClients(t, s, 0)
}

// waitForClients polls the hub's race-free clientCount until it reaches want,
// or fails the test after a timeout.
func waitForClients(t *testing.T, s *Server, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s.hub.clientCount() == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("client count did not reach %d (got %d)", want, s.hub.clientCount())
}
