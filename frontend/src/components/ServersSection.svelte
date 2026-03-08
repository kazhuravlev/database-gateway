<script>
  import { appHref } from "../routing.js";

  let { loading, error, servers } = $props();

  const panelClass =
    "rounded-xl border border-zinc-700/90 bg-zinc-900/90 p-3 shadow-[0_18px_42px_rgb(0_0_0_/_0.28)] backdrop-blur-xl";
  const chipClass = "rounded-lg border border-zinc-700/80 bg-zinc-800/90";
</script>

<section class={`${panelClass} flex flex-col gap-3`}>
  <div class="flex items-start justify-between gap-3">
    <div>
      <div class="text-[11px] font-bold uppercase tracking-[0.16em] text-lime-200">Targets</div>
      <h2 class="text-xl font-bold tracking-[-0.02em] text-zinc-100">Available servers</h2>
      <p class="mt-1 text-sm leading-6 text-zinc-400">
        Browse approved data sources and inspect the tables exposed for safe querying.
      </p>
    </div>
    <div class="inline-flex min-h-7 min-w-7 items-center justify-center rounded-full border border-zinc-600 bg-zinc-800 px-2.5 text-xs font-bold text-zinc-300">
      {servers.length}
    </div>
  </div>

  {#if loading}
    <div class="flex min-h-24 items-center justify-center rounded-lg border border-dashed border-zinc-600 bg-zinc-800/70 p-3 text-center text-sm text-zinc-400">
      Loading servers...
    </div>
  {:else if error}
    <div class="flex min-h-24 items-center justify-center rounded-lg border border-dashed border-red-500/70 bg-red-950/30 p-3 text-center text-sm text-red-300">
      {error}
    </div>
  {:else if servers.length === 0}
    <div class="flex min-h-24 items-center justify-center rounded-lg border border-dashed border-zinc-600 bg-zinc-800/70 p-3 text-center text-sm text-zinc-400">
      No servers found.
    </div>
  {:else}
    <div class="grid gap-2.5">
      {#each servers as server}
        <a class="block text-inherit no-underline" href={appHref(`/servers/${server.ID}`)}>
          <article class={`${chipClass} flex flex-col gap-3 p-3 transition-colors hover:border-zinc-500 hover:bg-zinc-800`}>
            <div class="flex flex-col items-start justify-between gap-3 md:flex-row">
              <div class="flex min-w-0 gap-2.5">
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <div class="break-words text-base font-bold tracking-[-0.02em] text-zinc-100">{server.ID}</div>
                    <div class="text-[11px] font-bold uppercase tracking-[0.16em] text-zinc-300">{server.Type}</div>
                  </div>
                  <div class="mt-1 text-[13px] leading-5 text-zinc-400">
                    {server.Description || "No description provided."}
                  </div>
                </div>
              </div>

              {#if (server.Tags ?? []).length > 0}
                <div class="flex flex-wrap justify-start gap-1.5 md:justify-end">
                  {#each server.Tags ?? [] as tag}
                    <div class="inline-flex items-center rounded-full border border-zinc-600 bg-zinc-800 px-2 py-0.5 text-[11px] font-bold text-zinc-300">
                      {tag.Name}
                    </div>
                  {/each}
                </div>
              {/if}
            </div>

            <div class="flex flex-wrap gap-2">
              <div class="inline-flex items-center rounded-full bg-zinc-800/80 px-2 py-0.5 text-[11px] font-bold text-zinc-300 gap-1 ring-1 ring-inset ring-zinc-700/70">
                <span class="text-xs font-semibold text-zinc-400">Tables</span>
                <span class="text-xs font-bold text-zinc-100">{server.Tables?.length ?? 0}</span>
              </div>
              <div class="inline-flex items-center rounded-full bg-zinc-800/80 px-2 py-0.5 text-[11px] font-bold text-zinc-300 gap-1 ring-1 ring-inset ring-zinc-700/70">
                <span class="text-xs font-semibold text-zinc-400">Fields</span>
                <span class="text-xs font-bold text-zinc-100">
                  {(server.Tables ?? []).reduce((total, table) => total + (table.fields?.length ?? 0), 0)}
                </span>
              </div>
            </div>

            {#if (server.Tables ?? []).length > 0}
              <div class="mt-1 border-t border-zinc-700/70 pt-2">
                <div class="mb-2 text-[11px] font-semibold uppercase tracking-[0.16em] text-zinc-500">
                  Exposed tables
                </div>
                <div class="flex flex-col divide-y divide-zinc-800/90">
                  {#each server.Tables ?? [] as table}
                    <div class="py-2 first:pt-0 last:pb-0">
                      <div class="text-[13px] font-bold text-zinc-100">{table.table}</div>
                      <div class="mt-1 break-words text-[13px] leading-5 text-zinc-400">
                        {#if (table.fields ?? []).length > 0}
                          {table.fields.join(", ")}
                        {:else}
                          No exposed fields
                        {/if}
                      </div>
                    </div>
                  {/each}
                </div>
              </div>
            {/if}
          </article>
        </a>
      {/each}
    </div>
  {/if}
</section>
