<script>
	let {
		title = "No title",
		color = "zinc",
		code = false,
		value = "No value"
	} = $props();

	const chipClass = "rounded-md border border-zinc-700/80 bg-zinc-800/90 p-2";
</script>

<div class={`${chipClass}`}>
	<div class="text-sm font-semibold text-zinc-400">{title}</div>
	<div class="mt-1 text-sm text-{color}-300 {code ? 'whitespace-pre-wrap' : ''}">{value}</div>
</div>
