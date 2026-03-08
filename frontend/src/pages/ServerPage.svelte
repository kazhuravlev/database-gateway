<script>
  import {
    addBookmark,
    deleteBookmark,
    getErrorMessage,
    getQueryResultsExportLink,
    getQueryResults,
    getServer,
    listBookmarks,
    runQuery,
    withAuthorizedRequest
  } from "../api.js";
  import BookmarkList from "../components/BookmarkList.svelte";
  import QueryResultsTable from "../components/QueryResultsTable.svelte";
  import { navigate } from "../routing.js";

  let { serverID, queryID = "", initialQuery = "", autoRun = false } = $props();

  let server = $state(null);
  let bookmarks = $state([]);
  let queryText = $state("");
  let bookmarkTitle = $state("");
  let queryResult = $state(null);

  let isLoading = $state(true);
  let isRunning = $state(false);
  let isSavingBookmark = $state(false);
  let runningBookmarkID = $state("");
  let isDeletingBookmarkID = $state("");
  let exportFormatInProgress = $state("");

  let loadError = $state("");
  let queryError = $state("");
  let bookmarkError = $state("");
  let loadedRouteKey = $state("");
  let initialRunKey = $state("");

  const panelClass =
    "rounded-xl border border-zinc-700/90 bg-zinc-900/90 shadow-[0_18px_42px_rgb(0_0_0_/_0.28)] backdrop-blur-xl";
  const chipClass = "rounded-lg border border-zinc-700/80 bg-zinc-800/90";
  const buttonClass =
    "inline-flex min-h-8 items-center justify-center gap-2 rounded-lg border border-lime-300 bg-lime-200 px-2.5 text-[13px] font-semibold leading-none text-zinc-900 transition-colors hover:border-lime-300 hover:bg-lime-300 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-lime-200 disabled:cursor-not-allowed disabled:opacity-60";
  const inputClass =
    "w-full rounded-md border border-zinc-600 bg-zinc-800 px-2 py-1.5 text-zinc-100 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-lime-200";

  async function loadBookmarks() {
    const result = await withAuthorizedRequest((token) => listBookmarks(token, serverID));
    if (!result) {
      return false;
    }

    bookmarks = result.bookmarks ?? [];
    return true;
  }

  async function loadServerPage() {
    isLoading = true;
    loadError = "";

    try {
      const result = await withAuthorizedRequest((token) =>
        Promise.all([getServer(token, serverID), listBookmarks(token, serverID)])
      );
      if (!result) {
        return;
      }

      const [serverResponse, bookmarksResponse] = result;
      server = serverResponse.target ?? null;
      bookmarks = bookmarksResponse.bookmarks ?? [];
      if (!server) {
        loadError = `Server ${serverID} was not found`;
        return;
      }

      if (queryID) {
        const storedResult = await withAuthorizedRequest((token) => getQueryResults(token, queryID));
        if (!storedResult) {
          return;
        }

        queryResult = storedResult;
        queryText = storedResult.query;
      } else {
        queryResult = null;
        queryText = initialQuery || "";
      }
    } catch (error) {
      loadError = getErrorMessage(error, "Failed to load server page");
    } finally {
      isLoading = false;
    }
  }

  function buildPreviewQuery(table) {
    const fields = (table.fields ?? []).join(", ");
    return `select ${fields} from ${table.table} limit 10`;
  }

  function buildCountQuery(table) {
    return `select count(*) from ${table.table}`;
  }

  async function executeQuery(nextQuery = queryText) {
    const preparedQuery = nextQuery.trim();

    if (!server || !preparedQuery) {
      queryError = "Query is required";
      return;
    }

    isRunning = true;
    queryError = "";
    queryText = preparedQuery;

    try {
      const result = await withAuthorizedRequest(async (token) => {
        return runQuery(token, server.ID, preparedQuery);
      });
      if (!result) {
        return;
      }

      navigate(`/servers/${server.ID}/${result.query_id}`);
    } catch (error) {
      queryError = getErrorMessage(error, "Failed to run query");
    } finally {
      isRunning = false;
    }
  }

  async function saveBookmark() {
    if (!server || !queryText.trim()) {
      bookmarkError = "Run or enter a query before saving a bookmark";
      return;
    }

    isSavingBookmark = true;
    bookmarkError = "";

    try {
      await withAuthorizedRequest((token) =>
        addBookmark(token, server.ID, bookmarkTitle.trim(), queryText.trim())
      );
      bookmarkTitle = "";
      await loadBookmarks();
    } catch (error) {
      bookmarkError = getErrorMessage(error, "Failed to save bookmark");
    } finally {
      isSavingBookmark = false;
    }
  }

  async function removeBookmark(bookmarkID) {
    isDeletingBookmarkID = bookmarkID;
    bookmarkError = "";

    try {
      await withAuthorizedRequest((token) => deleteBookmark(token, bookmarkID));
      await loadBookmarks();
    } catch (error) {
      bookmarkError = getErrorMessage(error, "Failed to delete bookmark");
    } finally {
      isDeletingBookmarkID = "";
    }
  }

  async function runBookmark(bookmark) {
    runningBookmarkID = bookmark.id;

    try {
      await executeQuery(bookmark.query);
    } finally {
      runningBookmarkID = "";
    }
  }

  function handleQueryKeydown(event) {
    if (event.key === "Enter" && event.shiftKey) {
      event.preventDefault();
      executeQuery();
    }
  }

  async function downloadResult(format) {
    if (!queryResult?.id || exportFormatInProgress) {
      return;
    }

    exportFormatInProgress = format;
    queryError = "";

    try {
      const result = await withAuthorizedRequest((token) =>
        getQueryResultsExportLink(token, queryResult.id, format)
      );
      if (!result?.url) {
        return;
      }

      window.location.assign(result.url);
    } catch (error) {
      queryError = getErrorMessage(error, `Failed to prepare ${format.toUpperCase()} export`);
    } finally {
      exportFormatInProgress = "";
    }
  }

  $effect(() => {
    const nextRouteKey = `${serverID}:${queryID}:${initialQuery}:${autoRun}`;

    if (serverID && nextRouteKey !== loadedRouteKey) {
      loadedRouteKey = nextRouteKey;
      server = null;
      bookmarks = [];
      queryText = queryID ? "" : initialQuery || "";
      bookmarkTitle = "";
      queryResult = null;
      queryError = "";
      bookmarkError = "";
      initialRunKey = "";
      loadServerPage();
    }
  });

  $effect(() => {
    const nextKey = `${serverID}:${queryID}:${initialQuery}:${autoRun}`;

    if (!queryID && server && autoRun && initialQuery.trim() && initialRunKey !== nextKey && !isRunning) {
      initialRunKey = nextKey;
      executeQuery(initialQuery);
    }
  });
</script>

{#if isLoading}
  <div class={`${panelClass} p-3 text-sm text-zinc-400`}>Loading server...</div>
{:else if loadError}
  <div class={`${panelClass} border-red-500/70 bg-red-950/20 p-3 text-sm text-red-300`}>{loadError}</div>
{:else if server}
  <div class="grid w-full gap-3">
    <section class={`${panelClass} w-full p-3`}>
      <div class="flex items-start gap-3">
        <div class="flex min-w-0 w-full flex-col gap-2">
          <div class="flex flex-wrap items-center gap-1.5">
            <div class="text-xs font-semibold uppercase tracking-wide text-zinc-400">{server.Type}</div>
            <div class="rounded bg-zinc-800 px-1.5 py-0.5 font-mono text-xs text-zinc-200 ring-1 ring-inset ring-zinc-700">
              {server.ID}
            </div>
            {#if server.Description}
              <div class="text-xs text-zinc-400">{server.Description}</div>
            {/if}
            <div class="flex flex-wrap gap-1">
              {#each server.Tags ?? [] as tag}
                <div class="inline-flex items-center rounded-full border border-zinc-600 bg-zinc-800 px-2 py-0.5 text-[11px] font-bold text-zinc-300">
                  {tag.Name}
                </div>
              {/each}
            </div>
          </div>

          <div class="flex flex-col gap-1.5">
            {#each server.Tables ?? [] as table}
              <div class={`${chipClass} flex flex-wrap items-center justify-between gap-2 p-2.5`}>
                <div class="flex flex-wrap items-center gap-1.5">
                  <div class="font-semibold text-zinc-100">{table.table}</div>
                  {#each table.fields ?? [] as field, index}
                    <div class="text-xs text-zinc-400">{field}{index < (table.fields?.length ?? 0) - 1 ? "," : ""}</div>
                  {/each}
                </div>
                <div class="flex flex-wrap gap-1">
                  <button
                    type="button"
                    class={`${buttonClass} px-2 py-1 text-xs`}
                    onclick={() => executeQuery(buildPreviewQuery(table))}
                  >
                    first 10
                  </button>
                  <button
                    type="button"
                    class={`${buttonClass} px-2 py-1 text-xs`}
                    onclick={() => executeQuery(buildCountQuery(table))}
                  >
                    count
                  </button>
                </div>
              </div>
            {/each}
          </div>
        </div>
      </div>
    </section>

    <section class={`${panelClass} w-full p-3`}>
      <div class="flex flex-col items-start gap-2">
        <textarea
          bind:value={queryText}
          placeholder="select col1, col2 from some_table limit 1"
          class={`${inputClass} h-24`}
          onkeydown={handleQueryKeydown}
        ></textarea>
        <div class="flex w-full flex-wrap items-start gap-2">
          <button type="button" class={buttonClass} onclick={() => executeQuery()} disabled={isRunning}>
            {isRunning ? "Running..." : "Run (Shift + Enter)"}
          </button>
          <button
            type="button"
            class={buttonClass}
            onclick={() => downloadResult("json")}
            disabled={!queryResult || Boolean(exportFormatInProgress)}
          >
            {exportFormatInProgress === "json" ? "Preparing JSON..." : "Get JSON"}
          </button>
          <button
            type="button"
            class={buttonClass}
            onclick={() => downloadResult("csv")}
            disabled={!queryResult || Boolean(exportFormatInProgress)}
          >
            {exportFormatInProgress === "csv" ? "Preparing CSV..." : "Get CSV"}
          </button>
          <div class="hidden flex-1 md:block"></div>
          <input
            bind:value={bookmarkTitle}
            class={`${inputClass} w-full md:ml-auto md:w-[200px] md:min-w-[200px] md:flex-none`}
            name="title"
            type="text"
            placeholder="Bookmark title"
          />
          <button
            type="button"
            class={`${buttonClass} whitespace-nowrap`}
            onclick={saveBookmark}
            disabled={isSavingBookmark}
          >
            {isSavingBookmark ? "Saving..." : "Save bookmark"}
          </button>
        </div>
        {#if queryError}
          <div class="w-full rounded-md border border-red-500/70 bg-red-950/50 px-2 py-1.5 text-xs text-red-200">
            {queryError}
          </div>
        {/if}
        {#if bookmarkError}
        <div class="mt-2 w-full rounded-md border border-red-500/70 bg-red-950/50 px-2 py-1.5 text-xs text-red-200">
          {bookmarkError}
        </div>
        {/if}
      </div>
    </section>

    <section class={`${panelClass} w-full p-3`}>
      <BookmarkList
        {bookmarks}
        compact={true}
        emptyText="No bookmarks yet"
        {runningBookmarkID}
        deletingBookmarkID={isDeletingBookmarkID}
        onRun={runBookmark}
        onDelete={(bookmark) => removeBookmark(bookmark.id)}
      />
    </section>

    {#if queryResult}
      <section class={`${panelClass} w-full p-3`}>
        <div class="flex flex-col gap-3">
          <div class="grid gap-2 md:grid-cols-2 xl:grid-cols-5">
            <div class={`${chipClass} p-3`}>
              <div class="text-[11px] font-semibold uppercase tracking-[0.16em] text-zinc-400">Created</div>
              <div class="mt-1 break-words text-sm text-zinc-100">{queryResult.created_at}</div>
            </div>
            <div class={`${chipClass} p-3`}>
              <div class="text-[11px] font-semibold uppercase tracking-[0.16em] text-zinc-400">Rows</div>
              <div class="mt-1 text-sm text-zinc-100">{queryResult.table.rows?.length ?? 0}</div>
            </div>
            <div class={`${chipClass} p-3`}>
              <div class="text-[11px] font-semibold uppercase tracking-[0.16em] text-zinc-400">Columns</div>
              <div class="mt-1 text-sm text-zinc-100">
                {queryResult.meta?.columns_count > 0
                  ? queryResult.meta.columns_count
                  : queryResult.table.headers?.length ?? 0}
              </div>
            </div>
            <div class={`${chipClass} p-3`}>
              <div class="text-[11px] font-semibold uppercase tracking-[0.16em] text-zinc-400">Execution</div>
              <div class="mt-1 text-sm text-zinc-100">{queryResult.meta?.execution_time_ms ?? 0} ms</div>
            </div>
            <div class={`${chipClass} p-3`}>
              <div class="text-[11px] font-semibold uppercase tracking-[0.16em] text-zinc-400">Network</div>
              <div class="mt-1 text-sm text-zinc-100">{queryResult.meta?.network_round_trip_ms ?? 0} ms</div>
            </div>
          </div>

          <QueryResultsTable table={queryResult.table} />
        </div>
      </section>
    {/if}
  </div>
{/if}
