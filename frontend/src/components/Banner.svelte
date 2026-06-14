<!-- Banner: a single, prominent strip that surfaces the most important current
     state — connection problems first, then race-control flags / safety car.
     It hides itself when everything is green and running. -->
<script>
  import { status, frame } from '../lib/telemetry.js';
  import { titleCase } from '../lib/format.js';

  // Returns {level, text} for the highest-priority alert, or null to hide.
  function compute($status, $frame) {
    if ($status === 'connecting') return { level: 'info', text: 'Connecting to PitMate…' };
    if ($status === 'closed') return { level: 'danger', text: 'Disconnected from PitMate — retrying…' };
    if (!$frame) return { level: 'info', text: 'Connected. Waiting for data…' };
    if (!$frame.source?.connected) {
      return {
        level: 'warn',
        text: `PitMate is up, but no live game detected — start ${$frame.source?.game || 'the game'}.`,
      };
    }

    const rc = $frame.raceControl || {};
    const sc = rc.safetyCar && rc.safetyCar !== 'none' ? rc.safetyCar : null;
    if (sc) return { level: 'warn', text: `Safety car: ${titleCase(sc)}` };

    const flag = rc.currentFlag;
    if (flag && flag !== 'none' && flag !== 'green') {
      const level = flag === 'red' || flag === 'black' ? 'danger' : 'warn';
      return { level, text: `${titleCase(flag)} flag` };
    }
    return null;
  }

  $: alert = compute($status, $frame);
</script>

{#if alert}
  <div class="banner {alert.level}">{alert.text}</div>
{/if}

<style>
  .banner {
    padding: 8px 16px;
    text-align: center;
    font-weight: 600;
    border-bottom: 1px solid var(--line);
  }
  .info {
    background: #16304a;
    color: var(--accent);
  }
  .warn {
    background: #3a2f12;
    color: var(--warn);
  }
  .danger {
    background: #3a1714;
    color: var(--danger);
  }
</style>
