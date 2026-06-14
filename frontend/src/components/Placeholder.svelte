<!-- Placeholder: a consistent "not built yet" panel for tabs that are specified
     but not implemented, so the navigation is complete and honest about scope.
     `items` lists what the tab will eventually show (see functions-workflow.md). -->
<script>
  export let title = '';
  export let summary = '';
  export let items = [];
</script>

<div class="ph">
  <h2>{title}</h2>
  <p>{summary}</p>
  {#if items.length}
    <ul>
      {#each items as item}
        <li>{item}</li>
      {/each}
    </ul>
  {/if}
  <p class="tag">Planned — not built yet.</p>
</div>

<style>
  .ph {
    max-width: 640px;
    margin: 20px auto;
    background: var(--panel);
    border: 1px solid var(--line);
    border-radius: 6px;
    padding: 24px;
  }
  h2 {
    font-size: 18px;
    margin-bottom: 8px;
  }
  p {
    color: var(--dim);
    margin: 6px 0;
  }
  ul {
    color: var(--dim);
    margin: 12px 0;
    padding-left: 20px;
  }
  li {
    margin: 4px 0;
  }
  .tag {
    margin-top: 16px;
    color: var(--accent);
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }
</style>
