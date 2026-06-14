<!-- Car Management — monitoring the car's systems and condition. This is a
     partial implementation covering what the LMU adapter exposes today
     (temperatures, brake balance, hybrid, coarse damage). Settings the adapter
     does not yet provide (TC/ABS, ARBs, detailed aero damage) are noted. -->
<script>
  import { frame, connected } from '../lib/telemetry.js';
  import Panel from '../components/Panel.svelte';
  import Stat from '../components/Stat.svelte';
  import Bar from '../components/Bar.svelte';
  import { num, pct, titleCase } from '../lib/format.js';

  $: p = $frame?.player;
  $: sys = p?.systems;
  $: energy = p?.energy;
  $: dmg = p?.damage;

  function tempTone(c, hot) {
    if (c == null) return '';
    return c > hot ? 'danger' : 'ok';
  }
</script>

{#if !$connected || !p}
  <div class="waiting">Waiting for live telemetry…</div>
{:else}
  <div class="layout">
    <Panel title="Temperatures">
      <div class="stats">
        <Stat label="Oil" value={num(sys?.oilTempC, 0)} unit="°C" tone={tempTone(sys?.oilTempC, 130)} />
        <Stat label="Water" value={num(sys?.waterTempC, 0)} unit="°C" tone={tempTone(sys?.waterTempC, 105)} />
      </div>
    </Panel>

    <Panel title="Controls">
      <div class="stats">
        <Stat label="Brake bias" value={num(sys?.brakeBiasFrontPct, 1)} unit="% F" />
        <Stat label="Limiter" value={sys?.pitLimiterOn ? 'ON' : 'off'} tone={sys?.pitLimiterOn ? 'warn' : ''} />
        <Stat label="Lights" value={sys?.lightsOn ? 'ON' : 'off'} />
      </div>
      <div class="note">TC / ABS / ARBs / engine map are not exposed by the LMU shared memory yet.</div>
    </Panel>

    {#if energy?.hasHybrid}
      <Panel title="Hybrid">
        <div class="stats">
          <Stat label="Mode" value={titleCase(energy?.hybridMode)} />
          <Stat label="Charge" value={pct(energy?.hybridChargeFraction)} tone="accent" />
        </div>
        <div class="bars">
          <Bar label="Batt" value={energy?.hybridChargeFraction} tone="accent" text={pct(energy?.hybridChargeFraction)} />
        </div>
      </Panel>
    {/if}

    <Panel title="Damage">
      <div class="bars">
        <Bar label="F wing" value={dmg?.frontWing} tone="danger" text={pct(dmg?.frontWing)} />
        <Bar label="R wing" value={dmg?.rearWing} tone="danger" text={pct(dmg?.rearWing)} />
      </div>
      <div class="note">rF2 reports only coarse damage; detailed aero/zone damage and performance deltas are not yet available.</div>
    </Panel>
  </div>
{/if}

<style>
  .layout {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
    gap: 16px;
    align-items: start;
  }
  .stats {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }
  .bars {
    margin-top: 12px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .note {
    margin-top: 10px;
    font-size: 12px;
    color: var(--dim);
  }
  .waiting {
    padding: 40px;
    text-align: center;
    color: var(--dim);
    font-size: 16px;
  }
</style>
