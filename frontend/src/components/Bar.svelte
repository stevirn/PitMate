<!-- Bar: a horizontal fill bar for a 0..1 value (fuel, tire wear, charge). -->
<script>
  export let value = 0; // 0..1
  export let label = '';
  export let text = ''; // optional right-aligned readout
  // tone: '' | 'ok' | 'warn' | 'danger' | 'accent'
  export let tone = 'accent';

  $: width = Math.max(0, Math.min(1, value || 0)) * 100;
</script>

<div class="bar-row">
  {#if label}<span class="label">{label}</span>{/if}
  <div class="track">
    <div class="fill {tone}" style="width:{width}%"></div>
  </div>
  {#if text}<span class="text">{text}</span>{/if}
</div>

<style>
  .bar-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .label {
    width: 34px;
    font-size: 11px;
    color: var(--dim);
    flex: none;
  }
  .track {
    flex: 1;
    height: 10px;
    background: var(--panel-2);
    border: 1px solid var(--line);
    border-radius: 3px;
    overflow: hidden;
  }
  .fill {
    height: 100%;
    transition: width 0.2s linear;
  }
  .text {
    width: 52px;
    text-align: right;
    font-size: 12px;
    font-variant-numeric: tabular-nums;
    flex: none;
  }
  .ok {
    background: var(--ok);
  }
  .warn {
    background: var(--warn);
  }
  .danger {
    background: var(--danger);
  }
  .accent {
    background: var(--accent);
  }
</style>
