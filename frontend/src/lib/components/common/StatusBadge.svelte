<script lang="ts">
	let {
		label,
		dotColor = 'emerald',
		class: className = ''
	}: {
		label: string;
		dotColor?: 'emerald' | 'amber' | 'blue';
		class?: string;
	} = $props();

	const dotClasses = {
		emerald: 'bg-emerald-500 shadow-emerald-500/50',
		amber: 'bg-[#f97040] shadow-[#f97040]/50',
		blue: 'bg-blue-500 shadow-blue-500/50'
	};
</script>

<div class="inline-flex items-center gap-2 px-3 py-1.5 rounded-lg bg-[var(--surface)] border border-[var(--border)] text-xs font-mono text-[var(--text-sub)] backdrop-blur-sm shadow-sm {className}">
	<span class="w-2 h-2 rounded-full {dotClasses[dotColor]} animate-pulse"></span>
	<span>{label}</span>
</div>
