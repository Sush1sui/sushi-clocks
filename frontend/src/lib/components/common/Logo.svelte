<script lang="ts">
	import sushiLogo from '$lib/assets/sushi_logo_without_bg.png';

	let {
		size = 'md',
		showSubtitle = true,
		class: className = ''
	}: {
		size?: 'sm' | 'md' | 'lg';
		showSubtitle?: boolean;
		class?: string;
	} = $props();

	const sizeClasses = {
		sm: { icon: 'w-7 h-7 rounded-lg p-0.5', text: 'text-base', sub: 'text-[9px]' },
		md: { icon: 'w-10 h-10 rounded-xl p-1', text: 'text-xl', sub: 'text-[10px]' },
		lg: { icon: 'w-12 h-12 rounded-2xl p-1.5', text: 'text-2xl', sub: 'text-xs' }
	};
</script>

<div class="flex items-center gap-3 {className}">
	<div class="{sizeClasses[size].icon} bg-[#f97040] flex items-center justify-center shadow-sm shrink-0">
		<img src={sushiLogo} alt="Sushi Clocks Mascot" class="w-full h-full object-contain select-none" />
	</div>
	<div>
		<div class="font-display font-bold tracking-tight {sizeClasses[size].text} text-[var(--text-main)] leading-none">
			sushi<span class="text-[#f97040]">.clocks</span>
		</div>
		{#if showSubtitle}
			<div class="{sizeClasses[size].sub} font-mono uppercase tracking-widest text-[var(--text-sub)] mt-0.5">
				Time & Payroll
			</div>
		{/if}
	</div>
</div>
