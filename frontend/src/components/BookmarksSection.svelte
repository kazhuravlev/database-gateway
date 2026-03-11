<script>
  import BookmarkList from "./BookmarkList.svelte";

  let {
    loading,
    error,
    bookmarks,
    runningBookmarkID = "",
    deletingBookmarkID = "",
    onRun = () => {},
    onDelete = () => {}
  } = $props();

  const panelClass =
    "rounded-xl border border-zinc-700/90 bg-zinc-900/90 p-3 shadow-[0_18px_42px_rgb(0_0_0_/_0.28)] backdrop-blur-xl";
</script>

<section class={`${panelClass} flex flex-col gap-3`}>
  <div class="flex items-start justify-between gap-3">
    <div>
      <div class="text-[11px] font-bold uppercase tracking-[0.16em] text-lime-200">Saved work</div>
      <h2 class="text-xl font-bold tracking-[-0.02em] text-zinc-100">Bookmarks</h2>
      <p class="mt-1 text-sm leading-6 text-zinc-400">
        Keep repeatable queries close at hand and rerun them without re-entering SQL.
      </p>
    </div>
    <div class="inline-flex min-h-7 min-w-7 items-center justify-center rounded-full border border-zinc-600 bg-zinc-800 px-2.5 text-xs font-bold text-zinc-300">
      {bookmarks.length}
    </div>
  </div>

  {#if loading}
    <div class="flex min-h-24 items-center justify-center rounded-lg border border-dashed border-zinc-600 bg-zinc-800/70 p-3 text-center text-sm text-zinc-400">
      Loading bookmarks...
    </div>
  {:else if error}
    <div class="flex min-h-24 items-center justify-center rounded-lg border border-dashed border-red-500/70 bg-red-950/30 p-3 text-center text-sm text-red-300">
      {error}
    </div>
  {:else}
    <BookmarkList
      {bookmarks}
      emptyText="No bookmarks found."
      {runningBookmarkID}
      {deletingBookmarkID}
      {onRun}
      {onDelete}
    />
  {/if}
</section>
