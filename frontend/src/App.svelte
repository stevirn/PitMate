<!-- App.svelte — the root of the PitMate cockpit. It hosts the header (title,
     connection indicator, tab bar), the global status/flag banner, and renders
     the active tab. Live data flows in through the telemetry store, so tabs read
     it directly; App does not pass it down. -->
<script>
  import { onMount } from 'svelte';
  import { startTelemetry, status, connected } from './lib/telemetry.js';
  import Banner from './components/Banner.svelte';

  import LiveData from './tabs/LiveData.svelte';
  import StrategyCalls from './tabs/StrategyCalls.svelte';
  import DriverCoaching from './tabs/DriverCoaching.svelte';
  import DriverVs from './tabs/DriverVs.svelte';
  import CarManagement from './tabs/CarManagement.svelte';
  import Settings from './tabs/Settings.svelte';

  const tabs = [
    { id: 'live', label: 'Live Data' },
    { id: 'strategy', label: 'Strategy' },
    { id: 'coaching', label: 'Coaching' },
    { id: 'vs', label: 'Driver Vs.' },
    { id: 'car', label: 'Car' },
    { id: 'settings', label: 'Settings' },
  ];
  let active = 'live';

  onMount(startTelemetry);

  // Connection dot: green when the game feeds data, amber when the socket is up
  // but no game, red when the socket is down.
  $: dotTone = $connected ? 'ok' : $status === 'open' ? 'warn' : 'danger';
</script>

<header>
  <div class="brand">
    <span class="dot {dotTone}"></span>
    <strong>PitMate</strong>
  </div>
  <nav>
    {#each tabs as t}
      <button class:active={active === t.id} on:click={() => (active = t.id)}>
        {t.label}
      </button>
    {/each}
  </nav>
</header>

<Banner />

<main>
  {#if active === 'live'}
    <LiveData />
  {:else if active === 'strategy'}
    <StrategyCalls />
  {:else if active === 'coaching'}
    <DriverCoaching />
  {:else if active === 'vs'}
    <DriverVs />
  {:else if active === 'car'}
    <CarManagement />
  {:else if active === 'settings'}
    <Settings />
  {/if}
</main>

<style>
  header {
    display: flex;
    align-items: center;
    gap: 20px;
    padding: 10px 16px;
    background: var(--panel);
    border-bottom: 1px solid var(--line);
    flex-wrap: wrap;
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 16px;
  }
  .dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: var(--dim);
  }
  .dot.ok {
    background: var(--ok);
  }
  .dot.warn {
    background: var(--warn);
  }
  .dot.danger {
    background: var(--danger);
  }
  nav {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
  }
  nav button {
    background: transparent;
    border-color: transparent;
  }
  nav button.active {
    background: var(--panel-2);
    border-color: var(--accent);
    color: var(--accent);
  }
  main {
    flex: 1;
    padding: 16px;
  }
</style>
