// telemetry.js — the single source of live data for the whole UI.
//
// It opens a WebSocket to the Go server, parses each telemetry.Frame (JSON), and
// exposes it through Svelte stores. Any component can read the latest frame by
// importing `frame` and using `$frame` — Svelte re-renders it automatically when
// a new frame arrives.
//
// Connection handling:
//   - status: 'connecting' | 'open' | 'closed' (the socket state)
//   - frame:  the latest telemetry.Frame object, or null before the first message
//   - connected: derived; true only when we have a frame AND the adapter reports
//     a live game (frame.source.connected). This is what the UI uses to tell
//     "PitMate is up" from "the game is actually running".
//   - framesReceived: a simple counter, handy for the Settings/diagnostics view.
// If the socket drops, it auto-reconnects every second.

import { writable, derived } from 'svelte/store';

export const status = writable('connecting');
export const frame = writable(null);
export const framesReceived = writable(0);

// connected = the game is actually feeding data (not just "socket is open").
export const connected = derived(frame, ($frame) =>
  Boolean($frame && $frame.source && $frame.source.connected)
);

let ws = null;
let reconnectTimer = null;

function socketURL() {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${proto}//${location.host}/ws`;
}

function connect() {
  status.set('connecting');
  ws = new WebSocket(socketURL());

  ws.onopen = () => status.set('open');

  ws.onmessage = (event) => {
    try {
      frame.set(JSON.parse(event.data));
      framesReceived.update((n) => n + 1);
    } catch {
      // Ignore malformed frames rather than tearing down the connection.
    }
  };

  ws.onclose = () => {
    status.set('closed');
    scheduleReconnect();
  };

  // An error is always followed by a close; let onclose handle reconnection.
  ws.onerror = () => ws && ws.close();
}

function scheduleReconnect() {
  clearTimeout(reconnectTimer);
  reconnectTimer = setTimeout(connect, 1000);
}

// startTelemetry opens the connection once. Call it from the app's onMount.
export function startTelemetry() {
  if (!ws) connect();
}

// reconnect forces an immediate reconnect (used by the Settings tab).
export function reconnect() {
  clearTimeout(reconnectTimer);
  if (ws) {
    ws.onclose = null; // avoid double reconnect from the old socket
    ws.close();
  }
  ws = null;
  connect();
}
