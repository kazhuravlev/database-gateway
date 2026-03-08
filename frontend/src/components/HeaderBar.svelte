<script>
  import { clearStoredToken } from "../api.js";
  import { appHref } from "../routing.js";

  let { loading, error, profile, isAdminRequestsPage = false } = $props();

  const shellClass =
    "rounded-xl border border-zinc-700/90 bg-zinc-900/90 px-3 py-2.5 shadow-[0_18px_42px_rgb(0_0_0_/_0.28)] backdrop-blur-xl";
  const navLinkClass =
    "inline-flex min-h-8 items-center justify-center gap-2 rounded-lg border px-2.5 text-[13px] font-semibold leading-none transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-lime-200";
  const activeLinkClass = "border-lime-300 bg-lime-300 text-zinc-950";
  const inactiveLinkClass =
    "border-zinc-700 bg-zinc-800/90 text-zinc-100 hover:border-zinc-500 hover:bg-zinc-700/90";
  const profileUsername = $derived(profile?.username || profile?.id || "Unknown user");
  const profileRole = $derived(profile?.role || "");
  const isAdmin = $derived(profileRole === "admin");
</script>

<div class={`${shellClass} mb-3 flex flex-col gap-3 md:flex-row md:items-center md:justify-between`}>
  <div class="flex min-w-0 items-center gap-3">
    <a
      class="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-zinc-600 bg-zinc-800 text-xs font-extrabold tracking-[0.08em] text-zinc-100 no-underline"
      href={appHref("/")}
    >
      DG
    </a>
    <div class="min-w-0">
      <div class="text-sm font-bold tracking-[-0.01em] text-zinc-100">Database Gateway</div>
      <div class="mt-0.5 truncate text-xs leading-5 text-zinc-400">
        {#if loading}
          Loading session
        {:else if error}
          {error}
        {:else}
          <span class="truncate">{profileUsername}</span>
          {#if profileRole}
            <span class="mx-1.5 text-zinc-600">·</span>
            <span class="inline-flex items-center rounded-full border border-zinc-700 bg-zinc-800 px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-[0.12em] text-zinc-300">
              {profileRole}
            </span>
          {/if}
        {/if}
      </div>
    </div>
  </div>

  <div class="flex flex-wrap items-center justify-stretch gap-2 md:justify-end">
    <a
      class={`${navLinkClass} ${isAdminRequestsPage === false ? activeLinkClass : inactiveLinkClass} flex-1 no-underline md:flex-none`}
      href={appHref("/")}
    >
      Dashboard
    </a>
    {#if isAdmin}
      <a
        class={`${navLinkClass} ${isAdminRequestsPage ? activeLinkClass : inactiveLinkClass} flex-1 no-underline md:flex-none`}
        href={appHref("/admin/requests")}
      >
        Admin requests
      </a>
    {/if}
    <a
      class={`${navLinkClass} ${inactiveLinkClass} flex-1 no-underline md:flex-none`}
      href="/logout"
      onclick={clearStoredToken}
    >
      Logout
    </a>
  </div>
</div>
