<!-- Stat: one labelled value, optionally large or highlighted by status. -->
<script>
  export let label = '';
  export let value = '—';
  export let unit = '';
  export let big = false;
  // tone: '' | 'ok' | 'warn' | 'danger' | 'accent' — colors the value text.
  export let tone = '';
</script>

<div class="stat" class:big>
  <div class="label">{label}</div>
  <div class="value {tone}">
    {value}{#if unit}<span class="unit">{unit}</span>{/if}
  </div>
</div>

<style>
  .stat {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .label {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--dim);
  }
  .value {
    font-variant-numeric: tabular-nums;
    font-size: 18px;
  }
  .big .value {
    font-size: 30px;
  }
  .unit {
    font-size: 0.6em;
    color: var(--dim);
    margin-left: 3px;
  }
  .ok {
    color: var(--ok);
  }
  .warn {
    color: var(--warn);
  }
  .danger {
    color: var(--danger);
  }
  .accent {
    color: var(--accent);
  }
</style>
