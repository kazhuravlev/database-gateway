<script>
  import { getErrorMessage, getQueryResults, withAuthorizedRequest } from "../api.js";
  import QueryResultsView from "../components/QueryResultsView.svelte";
  import { appHref } from "../routing.js";

  let { requestID } = $props();

  let result = $state(null);
  let error = $state("");
  let isLoading = $state(true);
  let loadedRequestID = $state("");

  const panelClass =
    "rounded-xl border border-zinc-700/90 bg-zinc-900/90 p-3 text-sm shadow-[0_18px_42px_rgb(0_0_0_/_0.28)] backdrop-blur-xl";

  async function loadAdminRequestResult() {
    isLoading = true;
    error = "";

    try {
      const response = await withAuthorizedRequest((token) => getQueryResults(token, requestID));
      if (!response) {
        return;
      }

      result = response;
    } catch (loadError) {
      error = getErrorMessage(loadError, "Failed to load request result");
    } finally {
      isLoading = false;
    }
  }

  $effect(() => {
    if (requestID && requestID !== loadedRequestID) {
      loadedRequestID = requestID;
      result = null;
      loadAdminRequestResult();
    }
  });
</script>

{#if isLoading}
  <div class={`${panelClass} text-zinc-400`}>Loading request result...</div>
{:else if error}
  <div class={`${panelClass} border-red-500/70 bg-red-950/20 text-red-300`}>{error}</div>
{:else if result}
  <QueryResultsView
    title="Request details"
    subtitle="Inspect the stored SQL response for an administrative request."
    backHref={appHref("/admin/requests")}
    backLabel="Back to requests"
    query={result.query}
    createdAt={result.created_at}
    targetID={result.target_id}
    userID={result.user_id}
    requestID={result.id}
    table={result.table}
    meta={result.meta}
  />
{/if}
