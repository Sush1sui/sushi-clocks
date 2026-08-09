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
		variant?: 'primary' | 'secondary' | 'ghost' | 'outline' | 'danger';
		type?: 'button' | 'submit' | 'reset';
		disabled?: boolean;
		loading?: boolean;
		class?: string;
		children?: Snippet;
	} & HTMLButtonAttributes = $props();

	const variantStyles = {
		primary: 'bg-[#f97040] hover:bg-[#e55f32] active:bg-[#cf4f24] text-white font-medium shadow-sm shadow-[#f97040]/15',
		secondary: 'bg-[var(--surface-raised)] hover:bg-[var(--surface-hover)] text-[var(--text-main)] border border-[var(--border)]',
		outline: 'bg-transparent border border-[#f97040]/30 text-[#f97040] hover:bg-[#f97040]/10',
		ghost: 'bg-transparent text-[var(--text-sub)] hover:text-[var(--text-main)] hover:bg-[var(--surface-hover)]',
		danger: 'bg-red-500/10 hover:bg-red-500/20 text-red-400 border border-red-500/30'
	};
</script>

<button
	{type}
	disabled={disabled || loading}
	class="inline-flex items-center justify-center gap-2 px-3.5 py-2 rounded-lg font-sans text-xs sm:text-sm transition-all duration-150 disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer {variantStyles[variant]} {className}"
	{...restProps}
>
	{#if loading}
		<div class="w-3.5 h-3.5 border-2 border-current/30 border-t-current rounded-full animate-spin"></div>
	{/if}
	{#if children}
		{@render children()}
	{/if}
</button>
