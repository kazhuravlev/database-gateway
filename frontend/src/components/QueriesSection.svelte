<script>
  import { appHref } from "../routing.js";

  let { loading, error, queries } = $props();

  const panelClass =
    "rounded-xl border border-zinc-700/90 bg-zinc-900/90 p-3 shadow-[0_18px_42px_rgb(0_0_0_/_0.28)] backdrop-blur-xl";
  const chipClass = "rounded-lg border border-zinc-700/80 bg-zinc-800/90";
  const buttonClass =
    "inline-flex min-h-8 items-center justify-center gap-2 rounded-lg border border-lime-300 bg-lime-200 px-2.5 text-[13px] font-semibold leading-none text-zinc-900 no-underline transition-colors hover:border-lime-300 hover:bg-lime-300 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-lime-200";

  function formatTimestamp(value) {
    if (!value) {
      return "Unknown time";
    }

    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return value;
    }

    return new Intl.DateTimeFormat(undefined, {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit"
    }).format(date);
  }
</script>

<section class={`${panelClass} flex flex-col gap-3`}>
  <div class="flex items-start justify-between gap-3">
    <div>
      <div class="text-[11px] font-bold uppercase tracking-[0.16em] text-lime-200">Activity</div>
      <h2 class="text-xl font-bold tracking-[-0.02em] text-zinc-100">Recent queries</h2>
      <p class="mt-1 text-sm leading-6 text-zinc-400">
        Jump back into the latest executions and inspect results without repeating the request.
      </p>
    </div>
    <div class="inline-flex min-h-7 min-w-7 items-center justify-center rounded-full border border-zinc-600 bg-zinc-800 px-2.5 text-xs font-bold text-zinc-300">
      {queries.length}
    </div>
  </div>

  {#if loading}
    <div class="flex min-h-24 items-center justify-center rounded-lg border border-dashed border-zinc-600 bg-zinc-800/70 p-3 text-center text-sm text-zinc-400">
      Loading queries...
    </div>
  {:else if error}
    <div class="flex min-h-24 items-center justify-center rounded-lg border border-dashed border-red-500/70 bg-red-950/30 p-3 text-center text-sm text-red-300">
      {error}
    </div>
  {:else if queries.length === 0}
    <div class="flex min-h-24 items-center justify-center rounded-lg border border-dashed border-zinc-600 bg-zinc-800/70 p-3 text-center text-sm text-zinc-400">
      No queries found.
    </div>
  {:else}
    <div class="grid gap-2.5">
      {#each queries as query}
        <article class={`${chipClass} flex flex-col gap-2.5 p-2.5`}>
          <div class="flex flex-col items-start justify-between gap-2 sm:flex-row">
            <div class="min-w-0">
              <div class="text-xs font-bold text-zinc-300">{formatTimestamp(query.created_at)}</div>
              <div class="break-words text-xs leading-5 text-zinc-400">{query.target_id}</div>
            </div>
            <a class={`${buttonClass} min-w-[72px]`} href={appHref(`/servers/${query.target_id}/${query.id}`)}>
              View
            </a>
          </div>
          <div class="break-words whitespace-pre-wrap text-[13px] leading-5 text-zinc-300">
            {query.query}
          </div>
        </article>
      {/each}
    </div>
  {/if}
</section>
