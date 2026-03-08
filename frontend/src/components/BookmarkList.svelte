<script>
  let {
    bookmarks = [],
    emptyText = "No bookmarks found.",
    compact = false,
    runningBookmarkID = "",
    deletingBookmarkID = "",
    onRun = () => {},
    onDelete = () => {}
  } = $props();

  const chipClass = "rounded-lg border border-zinc-700/80 bg-zinc-800/90";
  const actionButtonClass =
    "inline-flex min-h-8 min-w-20 items-center justify-center gap-2 rounded-lg border border-lime-300 bg-lime-200 px-2.5 text-[13px] font-semibold leading-none text-zinc-900 transition-colors hover:border-lime-300 hover:bg-lime-300 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-lime-200 disabled:cursor-not-allowed disabled:opacity-60";
</script>

<div class="grid gap-2.5">
  {#if bookmarks.length === 0}
    <div class="flex min-h-24 items-center justify-center rounded-lg border border-dashed border-zinc-600 bg-zinc-800/70 p-3 text-center text-sm text-zinc-400">
      {emptyText}
    </div>
  {:else}
    {#each bookmarks as bookmark}
      {#if compact}
        <article class={`${chipClass} flex items-center gap-2.5 p-2.5`}>
          <div class="min-w-0 shrink-0 text-sm font-bold leading-5 text-zinc-100">{bookmark.title}</div>
          <div
            class="min-w-0 flex-1 truncate text-[13px] leading-5 text-zinc-400"
            title={bookmark.query}
          >
            {bookmark.query}
          </div>
          <div class="flex shrink-0 flex-wrap gap-2">
            <button
              type="button"
              class={actionButtonClass}
              onclick={() => onRun(bookmark)}
              disabled={runningBookmarkID === bookmark.id}
            >
              {runningBookmarkID === bookmark.id ? "Running..." : "Run"}
            </button>
            <button
              type="button"
              class={`${actionButtonClass} border-red-400 bg-red-600 text-white hover:border-red-400 hover:bg-red-600/70`}
              onclick={() => onDelete(bookmark)}
              disabled={deletingBookmarkID === bookmark.id}
            >
              {deletingBookmarkID === bookmark.id ? "Deleting..." : "Delete"}
            </button>
          </div>
        </article>
      {:else}
        <article class={`${chipClass} flex flex-col gap-2.5 p-2.5`}>
          <div class="flex flex-col items-start justify-between gap-2 sm:flex-row">
            <div class="min-w-0">
              <div class="text-sm font-bold leading-5 text-zinc-100">{bookmark.title}</div>
              <div class="mt-1 break-words whitespace-pre-wrap text-[13px] leading-5 text-zinc-400">
                {bookmark.query}
              </div>
            </div>
            <div class="shrink-0 rounded-full border border-slate-700/40 bg-slate-900/75 px-2 py-0.5 font-mono text-[11px] font-bold text-zinc-300">
              {bookmark.target_id}
            </div>
          </div>

          <div class="flex flex-wrap gap-2">
            <button
              type="button"
              class={actionButtonClass}
              onclick={() => onRun(bookmark)}
              disabled={runningBookmarkID === bookmark.id}
            >
              {runningBookmarkID === bookmark.id ? "Running..." : "Run"}
            </button>
            <button
              type="button"
              class={`${actionButtonClass} border-red-400 bg-red-600 text-white hover:border-red-400 hover:bg-red-600/70`}
              onclick={() => onDelete(bookmark)}
              disabled={deletingBookmarkID === bookmark.id}
            >
              {deletingBookmarkID === bookmark.id ? "Deleting..." : "Delete"}
            </button>
          </div>
        </article>
      {/if}
    {/each}
  {/if}
</div>
