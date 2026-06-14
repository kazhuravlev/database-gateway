<script>
	import QueryResultsTable from "./QueryResultsTable.svelte";
	import Chip from "./Chip.svelte";

	let {
		title,
		subtitle = "",
		backHref,
		backLabel,
		query,
		createdAt,
		targetID = "",
		userID = "",
		requestID = "",
		table = {headers: [], rows: []},
		meta = null,
		status = "",
		error = ""
	} = $props();
	console.log(meta)
	const panelClass =
		"rounded-xl border border-zinc-700/90 bg-zinc-900/90 shadow-[0_18px_42px_rgb(0_0_0_/_0.28)] backdrop-blur-xl";
	const chipClass = "rounded-md border border-zinc-700/80 bg-zinc-800/90";
	const buttonClass =
		"inline-flex min-h-8 items-center justify-center gap-2 rounded-lg border border-lime-300 bg-lime-200 px-2.5 text-[13px] font-semibold leading-none text-zinc-900 no-underline transition-colors hover:border-lime-300 hover:bg-lime-300 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-lime-200";
</script>

<div class="grid w-full gap-3">
	<section class={`${panelClass} p-5 md:p-6`}>
		<div class="flex flex-col gap-4">
			<div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
				<div class="min-w-0">
					<div class="text-lg font-semibold text-zinc-100">{title}</div>
					{#if subtitle}
						<p class="mt-1 text-sm leading-6 text-zinc-400">{subtitle}</p>
					{/if}
				</div>
				<a class={`${buttonClass} justify-center`} href={backHref}>{backLabel}</a>
			</div>

			<div class="grid gap-2 md:grid-cols-4 xl:grid-cols-6">
				<Chip title="ID" value={requestID}/>
				<Chip title="Status" value={status} color={status === "failed" ? "red" : "green"}/>
				<Chip title="Created" value={createdAt}/>
				<Chip title="Target" value={targetID}/>
				<Chip title="User" value={userID}/>
				<Chip title="Rows" value={meta.rows_count}/>
				<Chip title="Columns" value={meta.columns_count}/>
				<Chip title="Vectors" value={meta.vectors_count}/>
				{#each meta.trace.records as record}
					<Chip title="{record.name} (ms)" value={record.duration/1000000}/>
				{/each}
			</div>

			<div>
				<Chip title="Query" value={query} code={true}/>
			</div>

			{#if error}
				<div class="rounded-xl border border-red-500/70 bg-red-950/30 p-4 text-sm leading-6 text-red-200">
					{error}
				</div>
			{/if}
		</div>
	</section>

	{#if !error}
		<section class={`${panelClass} p-3`}>
			<QueryResultsTable {table}/>
		</section>
	{/if}
</div>
