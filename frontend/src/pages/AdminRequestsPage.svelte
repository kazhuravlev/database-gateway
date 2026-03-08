<script>
  import { getErrorMessage, getQueryResultsExportLink, listAdminRequests, withAuthorizedRequest } from "../api.js";
  import { appHref } from "../routing.js";

  let { page } = $props();
  let loadedPage = $state(0);

  let requests = $state([]);
  let error = $state("");
  let isLoading = $state(true);
  let hasNext = $state(false);
  let hasPrev = $state(false);
  let exportRequestIDInProgress = $state("");
  let exportFormatInProgress = $state("");

  const panelClass =
    "rounded-xl border border-zinc-700/90 bg-zinc-900/90 shadow-[0_18px_42px_rgb(0_0_0_/_0.28)] backdrop-blur-xl";
  const chipClass = "rounded-lg border border-zinc-700/80 bg-zinc-800/90";
  const buttonClass =
    "inline-flex min-h-8 items-center justify-center gap-2 rounded-lg border border-lime-300 bg-lime-200 px-2.5 text-[13px] font-semibold leading-none text-zinc-900 no-underline transition-colors hover:border-lime-300 hover:bg-lime-300 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-lime-200";

  async function loadAdminRequests() {
    isLoading = true;
    error = "";
    hasPrev = page > 1;

    try {
      const result = await withAuthorizedRequest((token) => listAdminRequests(token, page));
      if (!result) {
        return;
      }

      requests = result.requests ?? [];
      hasNext = Boolean(result.has_next);
      hasPrev = Boolean(result.has_prev);
    } catch (loadError) {
      error = getErrorMessage(loadError, "Failed to load admin requests");
    } finally {
      isLoading = false;
    }
  }

  $effect(() => {
    if (page && page !== loadedPage) {
      loadedPage = page;
      loadAdminRequests();
    }
  });

  async function downloadRequest(requestID, format) {
    if (!requestID || exportRequestIDInProgress || exportFormatInProgress) {
      return;
    }

    exportRequestIDInProgress = requestID;
    exportFormatInProgress = format;
    error = "";

    try {
      const result = await withAuthorizedRequest((token) => getQueryResultsExportLink(token, requestID, format));
      if (!result?.url) {
        return;
      }

      window.open(result.url, "_blank", "noopener");
    } catch (downloadError) {
      error = getErrorMessage(downloadError, `Failed to prepare ${format.toUpperCase()} export`);
    } finally {
      exportRequestIDInProgress = "";
      exportFormatInProgress = "";
    }
  }
</script>

<div class="grid w-full gap-2.5">
  <div class={`${panelClass} flex items-center justify-between gap-2 p-3`}>
    <div class="text-xs text-zinc-400">Page {page}</div>
    <div class="flex items-center gap-2">
      {#if hasPrev}
        <a class={buttonClass} href={appHref("/admin/requests", `?page=${page - 1}`)}>Previous</a>
      {/if}
      {#if hasNext}
        <a class={buttonClass} href={appHref("/admin/requests", `?page=${page + 1}`)}>Next</a>
      {/if}
    </div>
  </div>

  {#if isLoading}
    <div class={`${panelClass} p-3 text-sm text-zinc-400`}>Loading admin requests...</div>
  {:else if error}
    <div class={`${panelClass} border-red-500/70 bg-red-950/20 p-3 text-sm text-red-300`}>{error}</div>
  {:else if requests.length === 0}
    <div class={`${panelClass} p-3 text-sm text-zinc-400`}>No requests yet.</div>
  {:else}
    <div class="grid w-full gap-1.5">
      {#each requests as request}
        <div class={`${panelClass} flex flex-col gap-1.5 p-3`}>
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0 text-[11px] text-zinc-400">
              {request.created_at} | {request.user_id} | {request.target_id}
            </div>
            <div class="flex shrink-0 items-center gap-1">
              <a class={`${buttonClass} px-2 py-1 text-[11px]`} href={appHref(`/admin/requests/${request.id}`)}>
                HTML
              </a>
              <button
                type="button"
                class={`${buttonClass} px-2 py-1 text-[11px] disabled:cursor-not-allowed disabled:opacity-60`}
                onclick={() => downloadRequest(request.id, "json")}
                disabled={Boolean(exportRequestIDInProgress)}
              >
                JSON
              </button>
              <button
                type="button"
                class={`${buttonClass} px-2 py-1 text-[11px] disabled:cursor-not-allowed disabled:opacity-60`}
                onclick={() => downloadRequest(request.id, "csv")}
                disabled={Boolean(exportRequestIDInProgress)}
              >
                CSV
              </button>
            </div>
          </div>
          <div class={`${chipClass} mt-1 break-words p-2 text-xs text-zinc-200`}>
            {request.query}
          </div>
        </div>
      {/each}
    </div>
  {/if}

  <div class={`${panelClass} flex items-center justify-between gap-2 p-3`}>
    <div class="text-xs text-zinc-400">Page {page}</div>
    <div class="flex items-center gap-2">
      {#if hasPrev}
        <a class={buttonClass} href={appHref("/admin/requests", `?page=${page - 1}`)}>Previous</a>
      {/if}
      {#if hasNext}
        <a class={buttonClass} href={appHref("/admin/requests", `?page=${page + 1}`)}>Next</a>
      {/if}
    </div>
  </div>
</div>
