<script>
  let { table } = $props();
</script>

<div class="w-full">
  <div class="w-full overflow-x-auto rounded-xl border border-zinc-700 bg-zinc-900/90">
    <table class="min-w-full border-collapse">
      <thead class="sticky top-0 z-10 bg-zinc-950">
        <tr>
          {#each table.headers ?? [] as header}
            <th
              scope="col"
              class="whitespace-nowrap border-b border-zinc-700 px-3 py-2 text-left text-[11px] font-semibold uppercase tracking-[0.04em] text-zinc-400"
            >
              {header}
            </th>
          {/each}
        </tr>
      </thead>
      <tbody>
        {#each table.rows ?? [] as row, rowIndex}
          <tr class={rowIndex % 2 === 0 ? "bg-zinc-900/80" : "bg-zinc-800/90"}>
            {#each row as cell}
              <td
                title={cell}
                class="whitespace-nowrap border-b border-zinc-800 px-3 py-1.5 text-sm text-zinc-100 transition-colors hover:bg-zinc-700/70"
              >
                {cell}
              </td>
            {/each}
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>
