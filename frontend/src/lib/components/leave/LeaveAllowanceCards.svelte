<script lang="ts">
	import { onMount } from 'svelte';
	import { CalendarCheck, Send, Info, Repeat, ShieldAlert, Sparkles, Sliders } from '@lucide/svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import { getLeaveBalances, type LeaveBalance, type LeavePeriod } from '$lib/api/leave';

	let {
		companyId,
		initialBalances,
		initialPeriod,
		onRequestLeave,
		isAdmin = false,
		onOpenPolicySettings
	}: {
		companyId: string;
		initialBalances?: LeaveBalance[];
		initialPeriod?: LeavePeriod;
		onRequestLeave: () => void;
		isAdmin?: boolean;
		onOpenPolicySettings?: () => void;
	} = $props();

	let balances = $state<LeaveBalance[]>([]);
	let period = $state<LeavePeriod | null>(null);
	let loading = $state(false);
	let error = $state('');

	$effect(() => {
		if (initialBalances && initialBalances.length > 0) {
			balances = initialBalances;
		}
	});

	$effect(() => {
		if (initialPeriod) {
			period = initialPeriod;
		}
	});

	onMount(() => {
		if (balances.length === 0) {
			loadBalances();
		}
	});

	export async function loadBalances() {
		loading = true;
		error = '';
		try {
			const res = await getLeaveBalances();
			balances = res.balances;
			period = res.period;
		} catch (err: any) {
			error = err?.message || 'Failed to load leave allowances';
		} finally {
			loading = false;
		}
	}

	function formatPeriodDate(dateStr?: string): string {
		if (!dateStr) return '';
		try {
			return new Date(dateStr).toLocaleDateString([], {
				month: 'short',
				day: 'numeric',
				year: 'numeric'
			});
		} catch {
			return dateStr;
		}
	}
</script>

<div class="p-4 sm:p-5 rounded-xl bg-[var(--surface)] border border-[var(--border)] space-y-4 shadow-sm">
	
	<!-- Header & Active Period Information -->
	<div class="flex items-center justify-between flex-wrap gap-2">
		<div class="flex items-center gap-2">
			<CalendarCheck class="w-4 h-4 text-blue-400" />
			<h3 class="text-xs font-bold tracking-tight text-[var(--text-main)]">Available Time Off</h3>
		</div>

		{#if period}
			<div
				class="group relative flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-[var(--surface-raised)] border border-[var(--border)] text-[10px] font-mono text-[var(--text-sub)] cursor-help"
				title="Active quota period based on company reset policy"
			>
				<Repeat class="w-3 h-3 text-[#f97040]" />
				<span>
					{formatPeriodDate(period.start_date)} &ndash; {formatPeriodDate(period.end_date)}
				</span>
				
				<!-- Hover Tooltip -->
				<div class="pointer-events-none absolute bottom-full right-0 mb-1.5 hidden group-hover:block w-56 p-2 rounded-lg bg-[var(--surface-raised)] border border-[var(--border)] shadow-xl text-[10px] text-[var(--text-main)] z-50">
					<div class="font-semibold text-[#f97040] mb-0.5">Dynamic Cycle Policy</div>
					Leave allowances and rollover limits apply between these dates and reset automatically.
				</div>
			</div>
		{/if}
	</div>

	<!-- Balances Grid -->
	{#if loading && balances.length === 0}
		<div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
			<div class="h-20 rounded-lg bg-[var(--surface-raised)] animate-pulse border border-[var(--border)]"></div>
			<div class="h-20 rounded-lg bg-[var(--surface-raised)] animate-pulse border border-[var(--border)]"></div>
		</div>
	{:else if error}
		<div class="p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-xs text-red-400 flex items-center justify-between">
			<span>{error}</span>
			<button class="underline ml-2" onclick={loadBalances}>Retry</button>
		</div>
	{:else if balances.length === 0}
		<div class="p-4 rounded-lg bg-[var(--surface-raised)] border border-[var(--border)] text-center text-xs text-[var(--text-mute)]">
			No leave categories configured for this organization.
		</div>
	{:else}
		<div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
			{#each balances as b (b.leave_type_id)}
				{@const pctUsed = b.total_allocated_days > 0 ? Math.min(100, Math.round((b.used_days / b.total_allocated_days) * 100)) : 0}
				<div class="p-3 rounded-lg bg-[var(--surface-raised)] border border-[var(--border)] hover:border-[var(--border-strong)] transition-all flex flex-col justify-between space-y-2 group">
					
					<div class="flex items-start justify-between gap-1.5">
						<div class="space-y-0.5">
							<span class="text-[11px] font-semibold text-[var(--text-main)] tracking-tight">
								{b.leave_type_name}
							</span>
							<div class="flex items-center gap-1 text-[9px] font-mono">
								<span class="px-1.5 py-0.2 rounded border {b.is_paid ? 'bg-emerald-500/10 border-emerald-500/25 text-emerald-400' : 'bg-purple-500/10 border-purple-500/25 text-purple-400'}">
									{b.is_paid ? 'Paid' : 'Unpaid'}
								</span>
								{#if b.carried_over_days > 0}
									<span class="px-1.5 py-0.2 rounded bg-blue-500/10 border border-blue-500/25 text-blue-400" title="Days carried over from previous reset cycle">
										+{b.carried_over_days}d rollover
									</span>
								{/if}
							</div>
						</div>

						<div class="text-right">
							{#if b.is_flexible}
								<span class="font-display font-bold text-xs text-purple-400">Flexible</span>
								<div class="text-[9px] font-mono text-[var(--text-mute)]">Manager approval</div>
							{:else}
								<span class="font-display font-bold text-base text-emerald-400">
									{b.remaining_days}
								</span>
								<span class="text-[10px] text-[var(--text-mute)] font-mono">/ {b.total_allocated_days}d</span>
							{/if}
						</div>
					</div>

					<!-- Visual Bar (Only for non-flexible/allocated types) -->
					{#if !b.is_flexible && b.total_allocated_days > 0}
						<div class="space-y-1">
							<div class="w-full h-1.5 rounded-full bg-[var(--bg)] overflow-hidden flex">
								<div
									class="h-full bg-emerald-500 transition-all duration-500"
									style="width: {100 - pctUsed}%"
								></div>
								<div
									class="h-full bg-neutral-500/40 transition-all duration-500"
									style="width: {pctUsed}%"
								></div>
							</div>
							<div class="flex items-center justify-between text-[9px] font-mono text-[var(--text-mute)]">
								<span>{b.used_days} used</span>
								{#if b.pending_days > 0}
									<span class="text-amber-400">{b.pending_days} pending</span>
								{/if}
								<span>{b.remaining_days} left</span>
							</div>
						</div>
					{/if}

				</div>
			{/each}
		</div>
	{/if}

	<!-- Action Trigger -->
	<Button
		variant="secondary"
		class="w-full h-9 text-xs"
		onclick={onRequestLeave}
	>
		<Send class="w-3.5 h-3.5 mr-1.5 text-[#f97040]" />
		<span>Request Time Off</span>
	</Button>

	<!-- Admin Link to Policy Config -->
	{#if isAdmin && onOpenPolicySettings}
		<div class="pt-1 flex items-center justify-between text-[11px] text-[var(--text-mute)]">
			<button
				type="button"
				class="flex items-center gap-1 hover:text-[#f97040] transition-colors cursor-pointer"
				onclick={onOpenPolicySettings}
			>
				<Sliders class="w-3 h-3 text-[#f97040]" />
				<span>Configure Reset Cycle & Rollover Policy &rarr;</span>
			</button>
		</div>
	{/if}

</div>
