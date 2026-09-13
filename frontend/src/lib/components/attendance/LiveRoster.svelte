<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Clock, UserCheck, RefreshCw, AlertCircle } from '@lucide/svelte';
	import type { TimesheetWithUser } from '$lib/api/timesheets';
	import { getLiveRoster } from '$lib/api/timesheets';

	let {
		companyId,
		roster = $bindable<TimesheetWithUser[]>([])
	}: {
		companyId: string;
		roster?: TimesheetWithUser[];
	} = $props();

	let loading = $state(false);
	let error = $state('');
	let now = $state(Date.now());
	let timer: any = null;

	onMount(() => {
		loadRoster();
		timer = setInterval(() => {
			now = Date.now();
		}, 1000);
	});

	onDestroy(() => {
		if (timer) clearInterval(timer);
	});

	export async function loadRoster() {
		loading = true;
		error = '';
		try {
			roster = await getLiveRoster(companyId);
		} catch (err: any) {
			error = err?.message || 'Failed to fetch live roster';
		} finally {
			loading = false;
		}
	}

	function formatDuration(startTimeStr: string, currentTimestamp: number): string {
		const start = new Date(startTimeStr).getTime();
		const diff = Math.max(0, Math.floor((currentTimestamp - start) / 1000));
		const hours = Math.floor(diff / 3600);
		const minutes = Math.floor((diff % 3600) / 60);
		const seconds = diff % 60;

		const pad = (n: number) => n.toString().padStart(2, '0');
		if (hours > 0) {
			return `${hours}h ${pad(minutes)}m ${pad(seconds)}s`;
		}
		return `${minutes}m ${pad(seconds)}s`;
	}

	function formatClockIn(timeStr: string): string {
		try {
			const d = new Date(timeStr);
			return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
		} catch {
			return timeStr;
		}
	}
</script>

<div class="rounded-2xl bg-[var(--surface)] border border-[var(--border)] p-6 shadow-sm flex flex-col gap-4">
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-2.5">
			<div class="w-8 h-8 rounded-lg bg-emerald-500/10 text-emerald-500 flex items-center justify-center">
				<UserCheck class="w-4 h-4" />
			</div>
			<div>
				<h3 class="text-sm font-semibold text-[var(--text-main)]">Live Attendance Roster</h3>
				<p class="text-xs text-[var(--text-sub)]">Real-time presence streamed via SSE</p>
			</div>
		</div>

		<div class="flex items-center gap-2">
			<span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-mono font-medium bg-emerald-500/10 text-emerald-500 border border-emerald-500/20">
				<span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
				{roster.length} Present
			</span>
			<button
				onclick={loadRoster}
				disabled={loading}
				class="p-1.5 rounded-lg border border-[var(--border)] text-[var(--text-sub)] hover:text-[var(--text-main)] hover:bg-[var(--surface-hover)] transition disabled:opacity-50"
				title="Refresh roster"
			>
				<RefreshCw class="w-3.5 h-3.5 {loading ? 'animate-spin' : ''}" />
			</button>
		</div>
	</div>

	{#if error}
		<div class="p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-xs text-red-400 flex items-center gap-2">
			<AlertCircle class="w-4 h-4 shrink-0" />
			<span>{error}</span>
		</div>
	{/if}

	{#if loading && roster.length === 0}
		<div class="space-y-2 py-4">
			{#each [1, 2, 3] as _}
				<div class="h-12 rounded-xl bg-[var(--surface-hover)] animate-pulse"></div>
			{/each}
		</div>
	{:else if roster.length === 0}
		<div class="text-center py-8 text-[var(--text-sub)] flex flex-col items-center gap-2">
			<Clock class="w-8 h-8 opacity-40 text-[var(--text-sub)]" />
			<p class="text-xs">No employees currently clocked in.</p>
		</div>
	{:else}
		<div class="overflow-x-auto">
			<table class="w-full text-left text-xs border-collapse">
				<thead>
					<tr class="border-b border-[var(--border)] text-[var(--text-sub)] font-mono uppercase tracking-wider">
						<th class="py-2.5 px-3">Employee</th>
						<th class="py-2.5 px-3">Role</th>
						<th class="py-2.5 px-3">Clocked In</th>
						<th class="py-2.5 px-3 text-right">Elapsed Time</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-[var(--border)] font-sans">
					{#each roster as staff (staff.id)}
						<tr class="hover:bg-[var(--surface-hover)] transition-colors">
							<td class="py-3 px-3">
								<div class="flex items-center gap-2.5">
									<div class="w-7 h-7 rounded-full bg-[var(--accent)]/15 text-[var(--accent)] flex items-center justify-center font-bold text-xs">
										{staff.user_first_name ? staff.user_first_name[0].toUpperCase() : 'U'}
									</div>
									<div>
										<p class="font-medium text-[var(--text-main)]">
											{staff.user_first_name} {staff.user_last_name}
										</p>
										<p class="text-[10px] text-[var(--text-sub)] font-mono">{staff.user_email}</p>
									</div>
								</div>
							</td>
							<td class="py-3 px-3">
								<span class="inline-block px-2 py-0.5 rounded text-[10px] font-mono capitalize bg-[var(--surface-hover)] border border-[var(--border)] text-[var(--text-sub)]">
									{staff.role_name}
								</span>
							</td>
							<td class="py-3 px-3 font-mono text-[var(--text-main)]">
								{formatClockIn(staff.clock_in_time)}
							</td>
							<td class="py-3 px-3 text-right font-mono font-bold text-emerald-400">
								{formatDuration(staff.clock_in_time, now)}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>
