<script>
  import { onMount } from "svelte";
  import {
    deleteBookmark,
    getErrorMessage,
    listBookmarks,
    listQueries,
    listServers,
    runQuery,
    withAuthorizedRequest
  } from "../api.js";
  import BookmarksSection from "../components/BookmarksSection.svelte";
  import QueriesSection from "../components/QueriesSection.svelte";
  import ServersSection from "../components/ServersSection.svelte";
  import { navigate } from "../routing.js";

  let servers = [];
  let serversError = "";
  let isServersLoading = true;

  let bookmarks = [];
  let bookmarksError = "";
  let isBookmarksLoading = true;
  let runningBookmarkID = "";
  let deletingBookmarkID = "";

  let queries = [];
  let queriesError = "";
  let isQueriesLoading = true;

  const panelClass =
    "rounded-xl border border-zinc-700/90 bg-zinc-900/90 shadow-[0_18px_42px_rgb(0_0_0_/_0.28)] backdrop-blur-xl";
  const chipClass = "rounded-lg border border-zinc-700/80 bg-zinc-800/90";

  function formatMetric(value, label) {
    return `${value} ${label}`;
  }

  function getFieldCount() {
    return servers.reduce((total, server) => {
      return total + (server.Tables ?? []).reduce((fieldsTotal, table) => fieldsTotal + (table.fields?.length ?? 0), 0);
    }, 0);
  }

  function getStatusText(loading, error, count, noun) {
    if (loading) {
      return `Loading ${noun}`;
    }

    if (error) {
      return error;
    }

    if (count === 0) {
      return `No ${noun} available`;
    }

    return formatMetric(count, noun);
  }

  function failAllResources(message) {
    serversError = message;
    bookmarksError = message;
    queriesError = message;

    isServersLoading = false;
    isBookmarksLoading = false;
    isQueriesLoading = false;
  }

  function applyResult(result, onSuccess, onError, onFinally, fallbackMessage) {
    if (result.status === "fulfilled") {
      onSuccess(result.value);
    } else {
      onError(getErrorMessage(result.reason, fallbackMessage));
    }

    onFinally();
  }

  async function loadDashboard() {
    let results;

    try {
      results = await withAuthorizedRequest(async (token) =>
        Promise.allSettled([listServers(token), listBookmarks(token), listQueries(token)])
      );
    } catch (error) {
      failAllResources(getErrorMessage(error, "Failed to load dashboard"));
      return;
    }

    if (!results) {
      return;
    }

    applyResult(
      results[0],
      (value) => {
        servers = value.targets ?? [];
        serversError = "";
      },
      (message) => {
        serversError = message;
      },
      () => {
        isServersLoading = false;
      },
      "Failed to load servers"
    );
    applyResult(
      results[1],
      (value) => {
        bookmarks = value.bookmarks ?? [];
        bookmarksError = "";
      },
      (message) => {
        bookmarksError = message;
      },
      () => {
        isBookmarksLoading = false;
      },
      "Failed to load bookmarks"
    );
    applyResult(
      results[2],
      (value) => {
        queries = value.queries ?? [];
        queriesError = "";
      },
      (message) => {
        queriesError = message;
      },
      () => {
        isQueriesLoading = false;
      },
      "Failed to load queries"
    );
  }

  async function runBookmark(bookmark) {
    runningBookmarkID = bookmark.id;

    try {
      const result = await withAuthorizedRequest((token) => runQuery(token, bookmark.target_id, bookmark.query));
      if (!result) {
        return;
      }

      bookmarksError = "";
      navigate(`/servers/${bookmark.target_id}/${result.query_id}`);
    } catch (error) {
      bookmarksError = getErrorMessage(error, "Failed to run bookmark");
    } finally {
      runningBookmarkID = "";
    }
  }

  async function removeBookmark(bookmark) {
    deletingBookmarkID = bookmark.id;

    try {
      const result = await withAuthorizedRequest((token) => deleteBookmark(token, bookmark.id));
      if (!result && result !== undefined) {
        return;
      }

      bookmarks = bookmarks.filter((item) => item.id !== bookmark.id);
      bookmarksError = "";
    } catch (error) {
      bookmarksError = getErrorMessage(error, "Failed to delete bookmark");
    } finally {
      deletingBookmarkID = "";
    }
  }

  onMount(loadDashboard);
</script>

<svelte:head>
  <title>Database Gateway</title>
</svelte:head>

<div class="flex flex-col gap-3.5">
  <section class={`${panelClass} grid gap-3.5 p-3.5 xl:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)]`}>
    <div class="flex min-h-full flex-col justify-between">
      <div>
        <div class="text-[11px] font-bold uppercase tracking-[0.16em] text-lime-200">Workspace overview</div>
        <h1 class="mt-1.5 max-w-[12ch] text-[clamp(28px,4vw,38px)] font-extrabold leading-[1.05] tracking-[-0.04em] text-zinc-100">
          Operate approved data access from one place.
        </h1>
        <p class="mt-2.5 max-w-[62ch] text-[15px] leading-7 text-zinc-300">
          Review available targets, rerun trusted bookmarks, and reopen recent query results without jumping between
          views.
        </p>
      </div>
    </div>

    <div class="grid gap-2.5 sm:grid-cols-2">
      <div class={`${chipClass} p-2.5`}>
        <div class="text-xs font-bold uppercase tracking-[0.08em] text-zinc-400">Servers</div>
        <div class="mt-1.5 text-2xl font-extrabold leading-none tracking-[-0.04em] text-zinc-100">
          {servers.length}
        </div>
        <div class="mt-1.5 text-[13px] leading-5 text-zinc-400">
          {getStatusText(isServersLoading, serversError, servers.length, "servers")}
        </div>
      </div>
      <div class={`${chipClass} p-2.5`}>
        <div class="text-xs font-bold uppercase tracking-[0.08em] text-zinc-400">Bookmarks</div>
        <div class="mt-1.5 text-2xl font-extrabold leading-none tracking-[-0.04em] text-zinc-100">
          {bookmarks.length}
        </div>
        <div class="mt-1.5 text-[13px] leading-5 text-zinc-400">
          {getStatusText(isBookmarksLoading, bookmarksError, bookmarks.length, "bookmarks")}
        </div>
      </div>
      <div class={`${chipClass} p-2.5`}>
        <div class="text-xs font-bold uppercase tracking-[0.08em] text-zinc-400">Schema fields</div>
        <div class="mt-1.5 text-2xl font-extrabold leading-none tracking-[-0.04em] text-zinc-100">
          {getFieldCount()}
        </div>
        <div class="mt-1.5 text-[13px] leading-5 text-zinc-400">Across {servers.length} configured targets</div>
      </div>
      <div class={`${chipClass} p-2.5`}>
        <div class="text-xs font-bold uppercase tracking-[0.08em] text-zinc-400">Recent queries</div>
        <div class="mt-1.5 text-2xl font-extrabold leading-none tracking-[-0.04em] text-zinc-100">
          {queries.length}
        </div>
        <div class="mt-1.5 text-[13px] leading-5 text-zinc-400">
          {getStatusText(isQueriesLoading, queriesError, queries.length, "recent queries")}
        </div>
      </div>
    </div>
  </section>

  <div class="grid items-start gap-3.5 xl:grid-cols-[minmax(0,1.75fr)_minmax(320px,0.9fr)]">
    <div class="grid min-w-0 gap-3.5 xl:grid-cols-[minmax(0,1.2fr)_minmax(0,0.9fr)]">
      <div class="min-w-0">
        <ServersSection loading={isServersLoading} error={serversError} {servers} />
      </div>
      <div class="min-w-0">
        <BookmarksSection
          loading={isBookmarksLoading}
          error={bookmarksError}
          {bookmarks}
          {runningBookmarkID}
          {deletingBookmarkID}
          onRun={runBookmark}
          onDelete={removeBookmark}
        />
      </div>
    </div>

    <aside class="flex min-w-0 flex-col gap-3.5">
      <QueriesSection loading={isQueriesLoading} error={queriesError} {queries} />
    </aside>
  </div>
</div>
