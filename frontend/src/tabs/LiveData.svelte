<!-- Live Data — the strategist's core at-a-glance dashboard during a stint:
     session state, position & timing, fuel/energy, tires, and speed.
     Reads the latest frame straight from the telemetry store. -->
<script>
  import { frame, connected } from '../lib/telemetry.js';
  import Panel from '../components/Panel.svelte';
  import Stat from '../components/Stat.svelte';
  import Bar from '../components/Bar.svelte';
  import TireGrid from '../components/TireGrid.svelte';
  import { lapTime, secs, gap, clock, num, pct, gear, titleCase } from '../lib/format.js';

  $: f = $frame;
  $: session = f?.session;
  $: p = f?.player;
  $: timing = p?.timing;
  $: energy = p?.energy;
  $: speed = p?.speed;
  $: race = p?.race;
  $: cond = session?.conditions;

  // Fuel tone: getting low on laps of fuel is a strategist's headline worry.
  $: fuelTone =
    energy?.fuelLapsRemaining == null
      ? 'accent'
      : energy.fuelLapsRemaining < 2
        ? 'danger'
        : energy.fuelLapsRemaining < 5
          ? 'warn'
          : 'ok';
</script>

{#if !$connected || !p}
  <div class="waiting">Waiting for live telemetry…</div>
{:else}
  <div class="layout">
    <Panel title="Session">
      <div class="stats">
        <Stat label="Track" value={session?.trackName || '—'} />
        <Stat label="Session" value={titleCase(session?.type)} />
        <Stat label={session?.isTimed ? 'Time left' : 'Lap'} value={session?.isTimed ? clock(session?.remainingSeconds) : `${timing?.lapsCompleted ?? '—'}/${session?.totalLaps || '—'}`} />
        <Stat label="Air / Track" value={`${num(cond?.airTempC, 0)}° / ${num(cond?.trackTempC, 0)}°`} />
        <Stat label="Rain" value={pct(cond?.rainIntensity)} tone={cond?.rainIntensity > 0 ? 'warn' : ''} />
        <Stat label="Wetness" value={pct(cond?.trackWetness)} />
      </div>
    </Panel>

    <Panel title="Position & Timing">
      <div class="stats">
        <Stat label="Overall" value={`P${race?.positionOverall ?? '—'}`} big tone="accent" />
        <Stat label="In class" value={`P${race?.positionInClass ?? '—'}`} big />
        <Stat label="Gap leader" value={gap(race?.gapToLeaderSeconds)} unit="s" />
        <Stat label="Gap ahead" value={gap(race?.gapAheadSeconds)} unit="s" />
        <Stat label="Gap behind" value={gap(race?.gapBehindSeconds)} unit="s" />
        <Stat label="Pit stops" value={num(race?.pitStopCount)} />
      </div>
      <div class="stats timing">
        <Stat label="Last lap" value={lapTime(timing?.lastLapSeconds)} />
        <Stat label="Best lap" value={lapTime(timing?.bestLapSeconds)} tone="ok" />
        <Stat label="Current" value={clock(timing?.currentLapSeconds)} />
      </div>
      {#if timing?.lastSectors?.length === 3}
        <div class="sectors">
          {#each timing.lastSectors as s, i}
            <span class="sector"><em>S{i + 1}</em> {secs(s)}</span>
          {/each}
        </div>
      {/if}
    </Panel>

    <Panel title="Fuel & Energy">
      <div class="stats">
        <Stat label="Fuel" value={num(energy?.fuelLitres, 1)} unit="L" big tone={fuelTone} />
        <Stat label="Laps left" value={num(energy?.fuelLapsRemaining, 1)} tone={fuelTone} />
        <Stat label="Per lap" value={num(energy?.fuelPerLapAvg, 2)} unit="L" />
      </div>
      <div class="bars">
        <Bar label="Tank" value={energy?.fuelCapacityLitres ? energy.fuelLitres / energy.fuelCapacityLitres : 0} tone={fuelTone} text={num(energy?.fuelLitres, 0) + 'L'} />
        {#if energy?.hasHybrid}
          <Bar label="Batt" value={energy?.hybridChargeFraction} tone="accent" text={pct(energy?.hybridChargeFraction)} />
        {/if}
      </div>
      {#if energy?.hasHybrid && energy?.hybridMode}
        <div class="note">Hybrid: {titleCase(energy.hybridMode)}</div>
      {/if}
    </Panel>

    <Panel title="Tires{p?.tires?.compound ? ' · ' + p.tires.compound : ''}">
      <TireGrid tires={p?.tires} />
    </Panel>

    <Panel title="Speed">
      <div class="speed">
        <Stat label="Speed" value={num(speed?.currentKph, 0)} unit="km/h" big tone="accent" />
        <Stat label="Gear" value={gear(speed?.gear)} big />
        <Stat label="RPM" value={num(speed?.rpm, 0)} />
        <Stat label="Top" value={num(speed?.topKph, 0)} unit="km/h" />
      </div>
    </Panel>
  </div>
{/if}

<style>
  .layout {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 16px;
    align-items: start;
  }
  .stats {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 12px;
  }
  .stats.timing {
    margin-top: 12px;
  }
  .speed {
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
  .sectors {
    margin-top: 12px;
    display: flex;
    gap: 14px;
    font-variant-numeric: tabular-nums;
  }
  .sector em {
    color: var(--dim);
    font-style: normal;
    margin-right: 4px;
  }
  .note,
  .waiting {
    color: var(--dim);
  }
  .note {
    margin-top: 10px;
    font-size: 12px;
  }
  .waiting {
    padding: 40px;
    text-align: center;
    font-size: 16px;
  }
</style>
