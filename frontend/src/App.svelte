<script>
  import { onMount } from "svelte";
  import { consumeTokenFromURL, getErrorMessage, getProfile, withAuthorizedRequest } from "./api.js";
  import HeaderBar from "./components/HeaderBar.svelte";
  import AdminRequestDetailsPage from "./pages/AdminRequestDetailsPage.svelte";
  import AdminRequestsPage from "./pages/AdminRequestsPage.svelte";
  import DashboardPage from "./pages/DashboardPage.svelte";
  import NotFoundPage from "./pages/NotFoundPage.svelte";
  import QueryResultPage from "./pages/QueryResultPage.svelte";
  import ServerPage from "./pages/ServerPage.svelte";
  import { getCurrentRoute, matchRoute } from "./routing.js";

  let profile = {};
  let profileError = "";
  let isProfileLoading = true;

  let route = {
    pathname: "/",
    searchParams: new URLSearchParams()
  };

  function normalizeProfile(payload) {
    const source = payload && typeof payload === "object" ? payload : {};
    const id = source.id ?? source.ID ?? "";
    const username = source.username ?? source.Username ?? id;
    const role = source.role ?? source.Role ?? "";

    return {
      id,
      username,
      role
    };
  }

  function syncRoute() {
    route = getCurrentRoute();
  }

  async function loadProfile() {
    try {
      const result = await withAuthorizedRequest((token) => getProfile(token));
      if (!result) {
        return;
      }

      profile = normalizeProfile(result);
      profileError = "";
    } catch (error) {
      profileError = getErrorMessage(error, "Failed to load profile");
    } finally {
      isProfileLoading = false;
    }
  }

  $: currentPage = Math.max(1, Number.parseInt(route.searchParams.get("page") || "1", 10) || 1);
  $: adminRequestParams = matchRoute(route.pathname, "/admin/requests/:requestID");
  $: serverResultParams = matchRoute(route.pathname, "/servers/:serverID/:queryID");
  $: queryResultParams = matchRoute(route.pathname, "/queries/:queryID");
  $: serverParams = matchRoute(route.pathname, "/servers/:serverID");

  onMount(() => {
    consumeTokenFromURL();
    syncRoute();
    loadProfile();

    const handlePopState = () => {
      syncRoute();
    };

    window.addEventListener("popstate", handlePopState);

    return () => {
      window.removeEventListener("popstate", handlePopState);
    };
  });
</script>

<div class="relative min-h-screen overflow-hidden bg-zinc-950 text-zinc-100">
  <div
    class="pointer-events-none fixed inset-0 bg-[radial-gradient(circle_at_10%_12%,rgba(113,113,122,0.16),transparent_26%),radial-gradient(circle_at_86%_18%,rgba(24,24,27,0.22),transparent_28%),linear-gradient(180deg,rgba(39,39,42,1)_0%,rgba(24,24,27,1)_100%)]"
  ></div>

  <div class="relative z-10 mx-auto flex min-h-screen w-full flex-col gap-3 px-2.5 py-3 md:px-3 md:py-2">
    <HeaderBar
      loading={isProfileLoading}
      error={profileError}
      {profile}
      isAdminRequestsPage={route.pathname.startsWith("/admin/requests")}
    />

    <main class="flex flex-col gap-3">
      {#if adminRequestParams}
        <AdminRequestDetailsPage requestID={adminRequestParams.requestID} />
      {:else if route.pathname === "/admin/requests"}
        <AdminRequestsPage page={currentPage} />
      {:else if serverResultParams}
        <ServerPage
          serverID={serverResultParams.serverID}
          queryID={serverResultParams.queryID}
          initialQuery={route.searchParams.get("query") || ""}
          autoRun={route.searchParams.get("autorun") === "1"}
        />
      {:else if serverParams}
        <ServerPage
          serverID={serverParams.serverID}
          initialQuery={route.searchParams.get("query") || ""}
          autoRun={route.searchParams.get("autorun") === "1"}
        />
      {:else if queryResultParams}
        <QueryResultPage queryID={queryResultParams.queryID} />
      {:else if route.pathname === "/"}
        <DashboardPage />
      {:else}
        <NotFoundPage pathname={route.pathname} />
      {/if}
    </main>
  </div>
</div>
