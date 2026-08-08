<script lang="ts">
	import { toggleMode, mode } from 'mode-watcher';
	import { Sun, Moon } from '@lucide/svelte';

	let { class: className = '' }: { class?: string } = $props();
</script>

<button
	type="button"
	onclick={toggleMode}
	class="inline-flex items-center justify-center w-9 h-9 rounded-xl bg-[var(--surface)] hover:bg-[var(--surface-hover)] border border-[var(--border)] text-[var(--text-sub)] hover:text-[var(--text-main)] transition-colors cursor-pointer {className}"
	aria-label="Toggle theme"
	title={mode.current === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
>
	{#if mode.current === 'dark'}
		<Sun class="w-4 h-4 text-[#f97040]" />
	{:else}
		<Moon class="w-4 h-4 text-[#f97040]" />
	{/if}
</button>
