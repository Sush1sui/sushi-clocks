<script lang="ts">
	import { onMount } from 'svelte';
	import { History, ChevronLeft, ChevronRight, MessageSquarePlus, CheckCircle2, AlertCircle, X, Loader2 } from '@lucide/svelte';
	import type { Timesheet } from '$lib/api/timesheets';
	import { getShiftHistory, requestAdjustment } from '$lib/api/timesheets';

	let history = $state<Timesheet[]>([]);
	let page = $state(1);
	let total = $state(0);
	let limit = 5;
	let loading = $state(false);
	let error = $state('');

	// Adjustment modal state
	let showModal = $state(false);
	let selectedShift = $state<Timesheet | null>(null);
	let reason = $state('');
	let submitting = $state(false);
	let modalError = $state('');
	let modalSuccess = $state('');

	onMount(() => {
		loadHistory();
	});

	export async function loadHistory() {
		loading = true;
		error = '';
		try {
			const res = await getShiftHistory(page, limit);
			history = res.history;
			total = res.total;
		} catch (err: any) {
			error = err?.message || 'Failed to load shift history';
		} finally {
			loading = false;
		}
	}

	function openAdjustmentModal(shift: Timesheet) {
		selectedShift = shift;
		reason = shift.adjustment_reason || '';
		modalError = '';
		modalSuccess = '';
		showModal = true;
	}

	async function submitAdjustment() {
		if (!selectedShift || !reason.trim()) {
			modalError = 'Please provide a reason for the adjustment.';
			return;
		}

		submitting = true;
		modalError = '';
		try {
			const res = await requestAdjustment(selectedShift.id, reason.trim());
			modalSuccess = 'Adjustment request submitted for manager review.';
			// Update local shift
			const idx = history.findIndex((h) => h.id === selectedShift?.id);
			if (idx !== -1) {
				history[idx] = res.timesheet;
			}
			setTimeout(() => {
				showModal = false;
			}, 1200);
		} catch (err: any) {
			modalError = err?.message || 'Failed to submit adjustment request';
		} finally {
			submitting = false;
		}
	}

	function formatDate(dateStr: string): string {
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

	function formatTime(dateStr?: string | null): string {
		if (!dateStr) return '—';
		try {
			return new Date(dateStr).toLocaleTimeString([], {
				hour: '2-digit',
				minute: '2-digit'
			});
		} catch {
			return dateStr;
		}
	}

	function calculateDuration(inStr: string, outStr?: string | null): string {
		if (!outStr) return 'In Progress';
		const start = new Date(inStr).getTime();
		const end = new Date(outStr).getTime();
		const diff = Math.max(0, Math.floor((end - start) / 1000));
		const hours = Math.floor(diff / 3600);
		const minutes = Math.floor((diff % 3600) / 60);
		return `${hours}h ${minutes}m`;
	}

	function getStatusBadge(status: string) {
		switch (status) {
			case 'active':
				return { label: 'Active', class: 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20' };
			case 'completed':
				return { label: 'Completed', class: 'bg-blue-500/10 text-blue-400 border-blue-500/20' };
			case 'flagged_for_review':
				return { label: 'Under Review', class: 'bg-amber-500/10 text-amber-400 border-amber-500/20' };
			case 'rejected':
				return { label: 'Rejected', class: 'bg-red-500/10 text-red-400 border-red-500/20' };
			default:
				return { label: status, class: 'bg-[var(--surface-hover)] text-[var(--text-sub)] border-[var(--border)]' };
		}
	}

	const totalPages = $derived(Math.max(1, Math.ceil(total / limit)));
</script>

<div class="rounded-2xl bg-[var(--surface)] border border-[var(--border)] p-6 shadow-sm flex flex-col gap-4">
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-2.5">
			<div class="w-8 h-8 rounded-lg bg-[var(--accent)]/10 text-[var(--accent)] flex items-center justify-center">
				<History class="w-4 h-4" />
			</div>
			<div>
				<h3 class="text-sm font-semibold text-[var(--text-main)]">My Shift History</h3>
				<p class="text-xs text-[var(--text-sub)]">Your logged shifts & adjustments</p>
			</div>
		</div>

		<div class="flex items-center gap-2">
			<button
				onclick={() => { page = Math.max(1, page - 1); loadHistory(); }}
				disabled={page <= 1 || loading}
				class="p-1 rounded-lg border border-[var(--border)] text-[var(--text-sub)] hover:text-[var(--text-main)] disabled:opacity-40"
			>
				<ChevronLeft class="w-4 h-4" />
			</button>
			<span class="text-xs font-mono text-[var(--text-sub)]">
				{page} / {totalPages}
			</span>
			<button
				onclick={() => { page = Math.min(totalPages, page + 1); loadHistory(); }}
				disabled={page >= totalPages || loading}
				class="p-1 rounded-lg border border-[var(--border)] text-[var(--text-sub)] hover:text-[var(--text-main)] disabled:opacity-40"
			>
				<ChevronRight class="w-4 h-4" />
			</button>
		</div>
	</div>

	{#if error}
		<div class="p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-xs text-red-400 flex items-center gap-2">
			<AlertCircle class="w-4 h-4 shrink-0" />
			<span>{error}</span>
		</div>
	{/if}

	{#if loading && history.length === 0}
		<div class="space-y-2 py-4">
			{#each [1, 2, 3] as _}
				<div class="h-10 rounded-xl bg-[var(--surface-hover)] animate-pulse"></div>
			{/each}
		</div>
	{:else if history.length === 0}
		<div class="text-center py-8 text-[var(--text-sub)] text-xs">
			No shift records found.
		</div>
	{:else}
		<div class="overflow-x-auto">
			<table class="w-full text-left text-xs border-collapse">
				<thead>
					<tr class="border-b border-[var(--border)] text-[var(--text-sub)] font-mono uppercase tracking-wider">
						<th class="py-2.5 px-3">Date</th>
						<th class="py-2.5 px-3">In</th>
						<th class="py-2.5 px-3">Out</th>
						<th class="py-2.5 px-3">Duration</th>
						<th class="py-2.5 px-3">Status</th>
						<th class="py-2.5 px-3 text-right">Action</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-[var(--border)] font-sans">
					{#each history as item (item.id)}
						{@const badge = getStatusBadge(item.status)}
						<tr class="hover:bg-[var(--surface-hover)] transition-colors">
							<td class="py-3 px-3 font-medium text-[var(--text-main)]">
								{formatDate(item.clock_in_time)}
							</td>
							<td class="py-3 px-3 font-mono text-[var(--text-main)]">
								{formatTime(item.clock_in_time)}
							</td>
							<td class="py-3 px-3 font-mono text-[var(--text-main)]">
								{formatTime(item.clock_out_time)}
							</td>
							<td class="py-3 px-3 font-mono text-[var(--text-sub)]">
								{calculateDuration(item.clock_in_time, item.clock_out_time)}
							</td>
							<td class="py-3 px-3">
								<span class="inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-mono border {badge.class}">
									{badge.label}
								</span>
							</td>
							<td class="py-3 px-3 text-right">
								{#if item.status === 'completed' || item.status === 'rejected'}
									<button
										onclick={() => openAdjustmentModal(item)}
										class="inline-flex items-center gap-1 px-2 py-1 rounded text-[11px] font-medium border border-[var(--border)] text-[var(--text-sub)] hover:text-[var(--text-main)] hover:bg-[var(--surface-hover)] transition"
										title="Request time adjustment"
									>
										<MessageSquarePlus class="w-3 h-3" />
										Adjust
									</button>
								{:else if item.status === 'flagged_for_review'}
									<span class="text-[10px] text-amber-400 font-mono">Pending HR</span>
								{:else}
									<span class="text-[10px] text-[var(--text-sub)]">—</span>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<!-- Modal for Requesting Shift Adjustment -->
{#if showModal && selectedShift}
	<div class="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4">
		<div class="w-full max-w-md bg-[var(--surface)] border border-[var(--border)] rounded-2xl shadow-xl overflow-hidden p-6 flex flex-col gap-4">
			<div class="flex items-center justify-between">
				<h3 class="text-sm font-semibold text-[var(--text-main)]">Request Punch Adjustment</h3>
				<button onclick={() => showModal = false} class="p-1 text-[var(--text-sub)] hover:text-[var(--text-main)]">
					<X class="w-4 h-4" />
				</button>
			</div>

			<div class="p-3 rounded-xl bg-[var(--surface-hover)] border border-[var(--border)] text-xs flex flex-col gap-1">
				<p><span class="text-[var(--text-sub)]">Shift Date:</span> <span class="font-medium text-[var(--text-main)]">{formatDate(selectedShift.clock_in_time)}</span></p>
				<p><span class="text-[var(--text-sub)]">Original Punches:</span> <span class="font-mono text-[var(--text-main)]">{formatTime(selectedShift.clock_in_time)} → {formatTime(selectedShift.clock_out_time)}</span></p>
			</div>

			<div class="flex flex-col gap-1.5">
				<label for="adjustment-reason" class="text-xs font-medium text-[var(--text-main)]">
					Reason / Note for Reviewer <span class="text-red-400">*</span>
				</label>
				<textarea
					id="adjustment-reason"
					bind:value={reason}
					rows="3"
					placeholder="e.g. Forgot to punch out due to offsite meeting, left at 5:30 PM."
					class="w-full text-xs rounded-xl bg-[var(--surface-hover)] border border-[var(--border)] p-3 text-[var(--text-main)] placeholder-[var(--text-sub)] focus:outline-hidden focus:border-[var(--accent)]"
				></textarea>
			</div>

			{#if modalError}
				<div class="p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-xs text-red-400 flex items-center gap-2">
					<AlertCircle class="w-4 h-4 shrink-0" />
					<span>{modalError}</span>
				</div>
			{/if}

			{#if modalSuccess}
				<div class="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-xs text-emerald-400 flex items-center gap-2">
					<CheckCircle2 class="w-4 h-4 shrink-0" />
					<span>{modalSuccess}</span>
				</div>
			{/if}

			<div class="flex items-center justify-end gap-2 pt-2 border-t border-[var(--border)]">
				<button
					onclick={() => showModal = false}
					class="px-3 py-1.5 rounded-lg text-xs font-medium text-[var(--text-sub)] hover:text-[var(--text-main)]"
				>
					Cancel
				</button>
				<button
					onclick={submitAdjustment}
					disabled={submitting}
					class="px-4 py-1.5 rounded-lg text-xs font-semibold bg-[var(--accent)] text-white hover:opacity-90 disabled:opacity-50 flex items-center gap-1.5"
				>
					{#if submitting}
						<Loader2 class="w-3.5 h-3.5 animate-spin" />
					{/if}
					Submit Request
				</button>
			</div>
		</div>
	</div>
{/if}
