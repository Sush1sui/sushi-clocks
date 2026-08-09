<script lang="ts">
	import type { Component } from 'svelte';

	let {
		icon: Icon,
		title,
		description,
		class: className = ''
	}: {
		icon: Component;
		title: string;
		description: string;
		class?: string;
	} = $props();
</script>

<div class="p-3.5 rounded-lg bg-[var(--surface)] border border-[var(--border)] hover:border-[#f97040]/30 transition-colors shadow-sm {className}">
	<div class="w-7 h-7 rounded-md bg-[var(--surface-raised)] border border-[var(--border)] flex items-center justify-center mb-2.5 text-[#f97040]">
		<Icon class="w-3.5 h-3.5" />
	</div>
	<div class="text-sm font-semibold text-[var(--text-main)]">{title}</div>
	<div class="text-xs text-[var(--text-sub)] mt-1">{description}</div>
</div>
