<script>
  import QueryResultsTable from "./QueryResultsTable.svelte";

  let {
    title,
    subtitle = "",
    backHref,
    backLabel,
    query,
    createdAt,
    targetID = "",
    userID = "",
    requestID = "",
    queryID = "",
    table,
    meta = null
  } = $props();

  const panelClass =
    "rounded-xl border border-zinc-700/90 bg-zinc-900/90 shadow-[0_18px_42px_rgb(0_0_0_/_0.28)] backdrop-blur-xl";
  const chipClass = "rounded-lg border border-zinc-700/80 bg-zinc-800/90";
  const buttonClass =
    "inline-flex min-h-8 items-center justify-center gap-2 rounded-lg border border-lime-300 bg-lime-200 px-2.5 text-[13px] font-semibold leading-none text-zinc-900 no-underline transition-colors hover:border-lime-300 hover:bg-lime-300 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-lime-200";
</script>

<div class="grid w-full gap-3">
  <section class={`${panelClass} p-5 md:p-6`}>
    <div class="flex flex-col gap-4">
      <div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div class="min-w-0">
          <div class="text-lg font-semibold text-zinc-100">{title}</div>
          {#if subtitle}
            <p class="mt-1 text-sm leading-6 text-zinc-400">{subtitle}</p>
          {/if}
        </div>
        <a class={`${buttonClass} justify-center`} href={backHref}>{backLabel}</a>
      </div>

      <div class="grid gap-2 md:grid-cols-2 xl:grid-cols-4">
        {#if createdAt}
          <div class={`${chipClass} p-3`}>
            <div class="text-[11px] font-semibold uppercase tracking-[0.16em] text-zinc-400">Created</div>
            <div class="mt-1 break-words text-sm text-zinc-100">{createdAt}</div>
          </div>
        {/if}
        {#if targetID}
          <div class={`${chipClass} p-3`}>
            <div class="text-[11px] font-semibold uppercase tracking-[0.16em] text-zinc-400">Target</div>
            <div class="mt-1 break-all font-mono text-sm text-zinc-100">{targetID}</div>
          </div>
        {/if}
        {#if userID}
          <div class={`${chipClass} p-3`}>
            <div class="text-[11px] font-semibold uppercase tracking-[0.16em] text-zinc-400">User</div>
            <div class="mt-1 break-all font-mono text-sm text-zinc-100">{userID}</div>
          </div>
        {/if}
        {#if requestID || queryID}
          <div class={`${chipClass} p-3`}>
            <div class="text-[11px] font-semibold uppercase tracking-[0.16em] text-zinc-400">
              {requestID ? "Request" : "Query"}
            </div>
            <div class="mt-1 break-all font-mono text-sm text-zinc-100">
              {requestID || queryID}
            </div>
          </div>
        {/if}
        <div class={`${chipClass} p-3`}>
          <div class="text-[11px] font-semibold uppercase tracking-[0.16em] text-zinc-400">Rows</div>
          <div class="mt-1 text-sm text-zinc-100">{table.rows?.length ?? 0}</div>
        </div>
        <div class={`${chipClass} p-3`}>
          <div class="text-[11px] font-semibold uppercase tracking-[0.16em] text-zinc-400">Columns</div>
          <div class="mt-1 text-sm text-zinc-100">
            {meta?.columns_count > 0 ? meta.columns_count : table.headers?.length ?? 0}
          </div>
        </div>
        {#if meta}
          <div class={`${chipClass} p-3`}>
            <div class="text-[11px] font-semibold uppercase tracking-[0.16em] text-zinc-400">Execution</div>
            <div class="mt-1 text-sm text-zinc-100">{meta.execution_time_ms} ms</div>
          </div>
          <div class={`${chipClass} p-3`}>
            <div class="text-[11px] font-semibold uppercase tracking-[0.16em] text-zinc-400">Parsing</div>
            <div class="mt-1 text-sm text-zinc-100">{meta.parsing_time_ms} ms</div>
          </div>
          <div class={`${chipClass} p-3`}>
            <div class="text-[11px] font-semibold uppercase tracking-[0.16em] text-zinc-400">Network</div>
            <div class="mt-1 text-sm text-zinc-100">{meta.network_round_trip_ms} ms</div>
          </div>
        {/if}
      </div>

      <div>
        <div class="text-[11px] font-semibold uppercase tracking-[0.16em] text-zinc-400">SQL</div>
        <div class={`${chipClass} mt-2 break-words whitespace-pre-wrap p-4 text-sm leading-6 text-zinc-100`}>
          {query}
        </div>
      </div>
    </div>
  </section>

  <section class={`${panelClass} p-3`}>
    <QueryResultsTable {table} />
  </section>
</div>
