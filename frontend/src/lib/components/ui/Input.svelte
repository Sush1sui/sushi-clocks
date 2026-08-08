<script lang="ts">
	import { Eye, EyeOff } from '@lucide/svelte';
	import type { HTMLInputAttributes } from 'svelte/elements';

	let {
		id,
		label,
		type = 'text',
		value = $bindable(''),
		placeholder = '',
		required = false,
		class: className = '',
		...restProps
	}: {
		id: string;
		label?: string;
		type?: string;
		value?: string;
		placeholder?: string;
		required?: boolean;
		class?: string;
	} & HTMLInputAttributes = $props();

	let showPassword = $state(false);
	const isPasswordField = type === 'password';
</script>

<div class="space-y-1.5 {className}">
	{#if label}
		<label for={id} class="block text-xs font-mono uppercase tracking-wider text-[var(--text-sub)]">
			{label}
		</label>
	{/if}
	<div class="relative">
		<input
			{id}
			type={isPasswordField ? (showPassword ? 'text' : 'password') : type}
			bind:value
			{required}
			{placeholder}
			class="w-full h-11 px-3.5 rounded-xl bg-[var(--surface-raised)] border border-[var(--border)] text-[var(--text-main)] placeholder-[var(--text-mute)] text-sm focus:outline-none focus:border-[#f97040] transition-colors {isPasswordField ? 'pr-11' : ''}"
			{...restProps}
		/>
		{#if isPasswordField}
			<button
				type="button"
				onclick={() => (showPassword = !showPassword)}
				class="absolute right-3.5 top-1/2 -translate-y-1/2 text-[var(--text-mute)] hover:text-[var(--text-main)] transition-colors p-1"
				aria-label={showPassword ? 'Hide password' : 'Show password'}
			>
				{#if showPassword}
					<EyeOff class="w-4 h-4" />
				{:else}
					<Eye class="w-4 h-4" />
				{/if}
			</button>
		{/if}
	</div>
</div>
