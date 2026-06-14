<!-- Settings — connection diagnostics and app options. Minimal for now: shows
     the live connection state and lets the strategist force a reconnect. UI
     options and data import/export will grow here later. -->
<script>
  import { status, frame, framesReceived, connected, reconnect } from '../lib/telemetry.js';
  import Panel from '../components/Panel.svelte';
  import Stat from '../components/Stat.svelte';

  $: wsURL = `${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}/ws`;
  $: statusTone = $connected ? 'ok' : $status === 'open' ? 'warn' : 'danger';
  $: lastUpdate = $frame?.timestamp ? new Date($frame.timestamp).toLocaleTimeString() : '—';
</script>

<div class="layout">
  <Panel title="Connection">
    <div class="stats">
      <Stat label="Socket" value={$status} tone={statusTone} />
      <Stat label="Game" value={$connected ? 'live' : 'no data'} tone={statusTone} />
      <Stat label="Source" value={$frame?.source?.game || '—'} />
      <Stat label="Frames" value={String($framesReceived)} />
      <Stat label="Last update" value={lastUpdate} />
      <Stat label="Seq" value={$frame?.sequence != null ? String($frame.sequence) : '—'} />
    </div>
    <div class="row">
      <code>{wsURL}</code>
      <button on:click={reconnect}>Reconnect</button>
    </div>
  </Panel>

  <Panel title="Options">
    <div class="note">
      UI options, session options, and data import/export will live here. The
      server's bind address, port, and update rate are configured on the gaming
      PC (see the backend's command-line flags).
    </div>
  </Panel>
</div>

<style>
  .layout {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 16px;
    align-items: start;
  }
  .stats {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 12px;
  }
  .row {
    margin-top: 14px;
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
  }
  code {
    background: var(--panel-2);
    border: 1px solid var(--line);
    border-radius: 4px;
    padding: 4px 8px;
    color: var(--dim);
  }
  .note {
    color: var(--dim);
    font-size: 13px;
  }
</style>
