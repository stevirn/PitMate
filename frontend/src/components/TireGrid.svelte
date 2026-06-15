<!-- TireGrid: the four tire corners laid out as they sit on the car
     (FL FR / RL RR), each showing temperature, remaining tread, and pressure. -->
<script>
  import Bar from './Bar.svelte';
  import { num, tempClass } from '../lib/format.js';

  export let tires = null;

  // Remaining-tread tone: lots left = ok, getting low = warn, nearly gone = danger.
  function wearTone(w) {
    if (w == null) return 'accent';
    if (w > 0.5) return 'ok';
    if (w > 0.25) return 'warn';
    return 'danger';
  }

  $: corners = tires
    ? [
        { key: 'FL', t: tires.frontLeft },
        { key: 'FR', t: tires.frontRight },
        { key: 'RL', t: tires.rearLeft },
        { key: 'RR', t: tires.rearRight },
      ]
    : [];
</script>

<p class="legend">surface temp · tread remaining · pressure / brake temp</p>
<div class="grid">
  {#each corners as c}
    <div class="corner">
      <div class="top">
        <span class="pos">{c.key}</span>
        <span class="temp {tempClass(c.t?.tempC)}" title="surface temperature">{num(c.t?.tempC, 0)}°</span>
      </div>
      <Bar value={c.t?.wearFraction} tone={wearTone(c.t?.wearFraction)} text={num((c.t?.wearFraction ?? 0) * 100, 0) + '%'} />
      <div class="sub">
        <span>{num(c.t?.pressureKpa, 0)} kPa</span>
        <span>brk {num(c.t?.brakeTempC, 0)}°</span>
      </div>
    </div>
  {/each}
</div>

<style>
  .legend {
    margin: 0 0 8px;
    font-size: 10px;
    color: var(--dim);
  }
  .grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }
  .corner {
    background: var(--panel-2);
    border: 1px solid var(--line);
    border-radius: 5px;
    padding: 8px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .top {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
  }
  .pos {
    color: var(--dim);
    font-size: 12px;
  }
  .temp {
    font-size: 20px;
    font-variant-numeric: tabular-nums;
  }
  .sub {
    display: flex;
    justify-content: space-between;
    font-size: 11px;
    color: var(--dim);
    font-variant-numeric: tabular-nums;
  }
  .cold {
    color: var(--cold);
  }
  .ok {
    color: var(--ok);
  }
  .hot {
    color: var(--danger);
  }
</style>
