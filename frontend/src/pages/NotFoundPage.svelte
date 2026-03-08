<script>
  import { appHref } from "../routing.js";

  let { pathname = "/" } = $props();

  const panelClass =
    "rounded-xl border border-zinc-700/90 bg-zinc-900/90 shadow-[0_18px_42px_rgb(0_0_0_/_0.28)] backdrop-blur-xl";
  const chipClass = "rounded-lg border border-zinc-700/80 bg-zinc-800/90";
  const buttonClass =
    "inline-flex min-h-8 items-center justify-center gap-2 rounded-lg border border-lime-300 bg-lime-200 px-2.5 text-[13px] font-semibold leading-none text-zinc-900 no-underline transition-colors hover:border-lime-300 hover:bg-lime-300 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-lime-200";
</script>

<div class="flex min-h-[calc(100vh-88px)] w-full items-center justify-center">
  <div class="grid w-full gap-3 lg:grid-cols-[minmax(0,1.7fr)_minmax(280px,0.9fr)]">
    <section class={`${panelClass} p-6 md:p-8`}>
      <div class="flex flex-col gap-8">
        <div class="flex flex-col gap-4">
          <div class="flex items-center gap-3">
            <div class={`${chipClass} px-2 py-1 text-[11px] font-semibold uppercase tracking-[0.18em] text-zinc-300`}>
              Error 404
            </div>
            <div class="text-xs text-zinc-400">Route unavailable</div>
          </div>

          <div class="space-y-3">
            <h1 class="text-3xl font-semibold tracking-[-0.03em] text-zinc-100 md:text-4xl">
              This page does not exist in Database Gateway.
            </h1>
            <p class="max-w-2xl text-sm leading-6 text-zinc-400">
              The address you opened is not mapped to an active view. It may be outdated, incomplete, or outside the
              routes exposed by this deployment.
            </p>
          </div>
        </div>

        <div class={`${chipClass} p-4`}>
          <div class="text-[11px] font-semibold uppercase tracking-[0.18em] text-zinc-400">Requested path</div>
          <div class="mt-2 break-all font-mono text-sm text-zinc-100">{pathname}</div>
        </div>

        <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
          <a class={`${buttonClass} justify-center`} href={appHref("/")}>Open dashboard</a>
          <a class="text-sm font-medium text-zinc-300 no-underline hover:text-white" href="/logout">Sign out</a>
        </div>
      </div>
    </section>

    <aside class={`${panelClass} p-5 md:p-6`}>
      <div class="flex flex-col gap-4">
        <div>
          <div class="text-sm font-semibold text-zinc-100">What to check</div>
          <p class="mt-1 text-sm leading-6 text-zinc-400">
            Start from a known route and navigate from the interface if you are following an older bookmark or manually
            entered URL.
          </p>
        </div>
      </div>
    </aside>
  </div>
</div>
