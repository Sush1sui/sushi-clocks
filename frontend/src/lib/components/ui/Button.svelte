<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	let {
		variant = 'primary',
		type = 'button',
		disabled = false,
		loading = false,
		class: className = '',
		children,
		...restProps
	}: {
		variant?: 'primary' | 'secondary' | 'ghost' | 'outline';
		type?: 'button' | 'submit' | 'reset';
		disabled?: boolean;
		loading?: boolean;
		class?: string;
		children?: Snippet;
	} & HTMLButtonAttributes = $props();

	const variantStyles = {
		primary: 'bg-[#f97040] hover:bg-[#e0582a] active:bg-[#cf332b] text-[var(--text-main)] font-semibold shadow-sm',
		secondary: 'bg-[var(--surface-raised)] hover:bg-[var(--surface-hover)] text-[var(--text-main)] border border-[var(--border)]',
		outline: 'bg-transparent border border-[#f97040]/40 text-[#f97040] hover:bg-[#f97040]/10',
		ghost: 'bg-transparent text-[var(--text-sub)] hover:text-[var(--text-main)] hover:bg-[var(--surface-hover)]'
	};
</script>

<button
	{type}
	disabled={disabled || loading}
	class="inline-flex items-center justify-center gap-2 px-4 py-2.5 rounded-xl font-display text-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer {variantStyles[variant]} {className}"
	{...restProps}
>
	{#if loading}
		<div class="w-4 h-4 border-2 border-current/30 border-t-current rounded-full animate-spin"></div>
	{/if}
	{#if children}
		{@render children()}
	{/if}
</button>
