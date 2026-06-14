<!-- Panel: a titled section box. The default slot holds the panel content. -->
<script>
  export let title = '';
</script>

<section class="panel">
  {#if title}<h2>{title}</h2>{/if}
  <div class="body">
    <slot />
  </div>
</section>

<style>
  .panel {
    background: var(--panel);
    border: 1px solid var(--line);
    border-radius: 6px;
    overflow: hidden;
  }
  h2 {
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--dim);
    padding: 8px 12px;
    border-bottom: 1px solid var(--line);
    background: var(--panel-2);
  }
  .body {
    padding: 12px;
  }
</style>
