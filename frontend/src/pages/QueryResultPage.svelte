<script>
  import { getErrorMessage, getQueryResults, withAuthorizedRequest } from "../api.js";
  import QueryResultsView from "../components/QueryResultsView.svelte";
  import { appHref } from "../routing.js";

  let { queryID } = $props();

  let result = $state(null);
  let error = $state("");
  let isLoading = $state(true);
  let loadedQueryID = $state("");

  const panelClass =
    "rounded-xl border border-zinc-700/90 bg-zinc-900/90 p-3 text-sm shadow-[0_18px_42px_rgb(0_0_0_/_0.28)] backdrop-blur-xl";

  async function loadQueryResult() {
    isLoading = true;
    error = "";

    try {
      const response = await withAuthorizedRequest((token) => getQueryResults(token, queryID));
      if (!response) {
        return;
      }

      result = response;
    } catch (loadError) {
      error = getErrorMessage(loadError, "Failed to load query result");
    } finally {
      isLoading = false;
    }
  }

  $effect(() => {
    if (queryID && queryID !== loadedQueryID) {
      loadedQueryID = queryID;
      result = null;
      loadQueryResult();
    }
  });
</script>

{#if isLoading}
  <div class={`${panelClass} text-zinc-400`}>Loading query result...</div>
{:else if error}
  <div class={`${panelClass} border-red-500/70 bg-red-950/20 text-red-300`}>{error}</div>
{:else if result}
  <QueryResultsView
    title="Query results"
    subtitle="Review a previously executed query without returning to the server page."
    backHref={appHref("/")}
    backLabel="Back to dashboard"
    query={result.query}
    createdAt={result.created_at}
    queryID={queryID}
    table={result.table}
    meta={result.meta}
  />
{/if}
