<script lang="ts">
	import { onMount } from 'svelte';
	import { ClipboardCheck, Check, X, Clock, AlertCircle, Loader2, Edit3 } from '@lucide/svelte';
	import type { TimesheetWithUser } from '$lib/api/timesheets';
	import { getPendingAdjustments, resolveAdjustment, directOverrideTimesheet } from '$lib/api/timesheets';

	let {
		companyId,
		onResolved
	}: {
		companyId: string;
		onResolved?: () => void;
	} = $props();

	let adjustments = $state<TimesheetWithUser[]>([]);
	let loading = $state(false);
	let error = $state('');
	let actionInProgress = $state<string | null>(null);

	// Direct Edit modal state
	let editTarget = $state<TimesheetWithUser | null>(null);
	let editInTime = $state('');
	let editOutTime = $state('');
	let editReason = $state('');
	let editError = $state('');
	let editSubmitting = $state(false);

	onMount(() => {
		loadAdjustments();
	});

	export async function loadAdjustments() {
		loading = true;
		error = '';
		try {
			adjustments = await getPendingAdjustments(companyId);
		} catch (err: any) {
			error = err?.message || 'Failed to load adjustment queue';
		} finally {
			loading = false;
		}
	}

	async function handleResolve(id: string, action: 'approve' | 'reject') {
		actionInProgress = id;
		try {
			await resolveAdjustment(id, action);
			adjustments = adjustments.filter((a) => a.id !== id);
			if (onResolved) onResolved();
		} catch (err: any) {
			error = err?.message || `Failed to ${action} adjustment`;
		} finally {
			actionInProgress = null;
		}
	}

	function openDirectEdit(shift: TimesheetWithUser) {
		editTarget = shift;
		editInTime = shift.clock_in_time ? new Date(shift.clock_in_time).toISOString().slice(0, 16) : '';
		editOutTime = shift.clock_out_time ? new Date(shift.clock_out_time).toISOString().slice(0, 16) : '';
		editReason = shift.adjustment_reason || 'Manual manager adjustment';
		editError = '';
	}

	async function submitDirectEdit() {
		if (!editTarget || !editInTime || !editReason.trim()) {
			editError = 'Clock In time and Reason are required.';
			return;
		}

		editSubmitting = true;
		editError = '';
		try {
			await directOverrideTimesheet(editTarget.id, editInTime, editOutTime || null, editReason.trim());
			adjustments = adjustments.filter((a) => a.id !== editTarget?.id);
			editTarget = null;
			if (onResolved) onResolved();
		} catch (err: any) {
			editError = err?.message || 'Failed to override timesheet';
		} finally {
			editSubmitting = false;
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

	function formatDate(dateStr: string): string {
		try {
			return new Date(dateStr).toLocaleDateString([], {
				month: 'short',
				day: 'numeric'
			});
		} catch {
			return dateStr;
		}
	}
</script>

<div class="rounded-2xl bg-[var(--surface)] border border-[var(--border)] p-6 shadow-sm flex flex-col gap-4">
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-2.5">
			<div class="w-8 h-8 rounded-lg bg-amber-500/10 text-amber-500 flex items-center justify-center">
				<ClipboardCheck class="w-4 h-4" />
			</div>
			<div>
				<h3 class="text-sm font-semibold text-[var(--text-main)]">Attendance Adjustments</h3>
				<p class="text-xs text-[var(--text-sub)]">Employee correction requests awaiting HR review</p>
			</div>
		</div>

		{#if adjustments.length > 0}
			<span class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-xs font-mono font-medium bg-amber-500/10 text-amber-500 border border-amber-500/20">
				{adjustments.length} Pending
			</span>
		{/if}
	</div>

	{#if error}
		<div class="p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-xs text-red-400 flex items-center gap-2">
			<AlertCircle class="w-4 h-4 shrink-0" />
			<span>{error}</span>
		</div>
	{/if}

	{#if loading && adjustments.length === 0}
		<div class="space-y-2 py-4">
			{#each [1, 2] as _}
				<div class="h-14 rounded-xl bg-[var(--surface-hover)] animate-pulse"></div>
			{/each}
		</div>
	{:else if adjustments.length === 0}
		<div class="text-center py-6 text-[var(--text-sub)] text-xs flex flex-col items-center gap-1.5">
			<Clock class="w-6 h-6 opacity-30 text-[var(--text-sub)]" />
			<p>No pending adjustment requests.</p>
		</div>
	{:else}
		<div class="space-y-3">
			{#each adjustments as adj (adj.id)}
				<div class="p-4 rounded-xl bg-[var(--surface-hover)] border border-[var(--border)] flex flex-col sm:flex-row sm:items-center justify-between gap-3">
					<div class="flex flex-col gap-1">
						<div class="flex items-center gap-2">
							<span class="font-medium text-xs text-[var(--text-main)]">
								{adj.user_first_name} {adj.user_last_name}
							</span>
							<span class="text-[10px] font-mono text-[var(--text-sub)]">
								• {formatDate(adj.clock_in_time)} ({formatTime(adj.clock_in_time)} → {formatTime(adj.clock_out_time)})
							</span>
						</div>
						{#if adj.adjustment_reason}
							<p class="text-xs text-[var(--text-sub)] italic bg-[var(--surface)] p-2 rounded-lg border border-[var(--border)]">
								"{adj.adjustment_reason}"
							</p>
						{/if}
					</div>

					<div class="flex items-center gap-2 shrink-0 self-end sm:self-center">
						<button
							onclick={() => openDirectEdit(adj)}
							class="p-1.5 rounded-lg border border-[var(--border)] text-[var(--text-sub)] hover:text-[var(--text-main)] hover:bg-[var(--surface)] transition text-xs flex items-center gap-1"
							title="Edit punches before approving"
						>
							<Edit3 class="w-3.5 h-3.5" />
							<span>Edit</span>
						</button>
						<button
							onclick={() => handleResolve(adj.id, 'reject')}
							disabled={actionInProgress === adj.id}
							class="px-2.5 py-1.5 rounded-lg text-xs font-medium border border-red-500/30 text-red-400 hover:bg-red-500/10 transition disabled:opacity-50 flex items-center gap-1"
						>
							<X class="w-3.5 h-3.5" />
							Reject
						</button>
						<button
							onclick={() => handleResolve(adj.id, 'approve')}
							disabled={actionInProgress === adj.id}
							class="px-3 py-1.5 rounded-lg text-xs font-semibold bg-emerald-500 text-white hover:bg-emerald-600 transition disabled:opacity-50 flex items-center gap-1"
						>
							{#if actionInProgress === adj.id}
								<Loader2 class="w-3.5 h-3.5 animate-spin" />
							{:else}
								<Check class="w-3.5 h-3.5" />
							{/if}
							Approve
						</button>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

<!-- Modal for Direct Override / Correction -->
{#if editTarget}
	<div class="fixed inset-0 z-50 bg-black/60 backdrop-blur-xs flex items-center justify-center p-4">
		<div class="w-full max-w-md bg-[var(--surface)] border border-[var(--border)] rounded-2xl shadow-xl p-6 flex flex-col gap-4">
			<div class="flex items-center justify-between">
				<h3 class="text-sm font-semibold text-[var(--text-main)]">Edit Shift Punches</h3>
				<button onclick={() => editTarget = null} class="p-1 text-[var(--text-sub)] hover:text-[var(--text-main)]">
					<X class="w-4 h-4" />
				</button>
			</div>

			<div class="flex flex-col gap-3 text-xs">
				<div class="flex flex-col gap-1">
					<label for="edit-clock-in" class="font-medium text-[var(--text-main)]">Clock In Time</label>
					<input
						id="edit-clock-in"
						type="datetime-local"
						bind:value={editInTime}
						class="w-full rounded-xl bg-[var(--surface-hover)] border border-[var(--border)] p-2.5 text-[var(--text-main)] font-mono focus:outline-hidden focus:border-[var(--accent)]"
					/>
				</div>

				<div class="flex flex-col gap-1">
					<label for="edit-clock-out" class="font-medium text-[var(--text-main)]">Clock Out Time</label>
					<input
						id="edit-clock-out"
						type="datetime-local"
						bind:value={editOutTime}
						class="w-full rounded-xl bg-[var(--surface-hover)] border border-[var(--border)] p-2.5 text-[var(--text-main)] font-mono focus:outline-hidden focus:border-[var(--accent)]"
					/>
				</div>

				<div class="flex flex-col gap-1">
					<label for="edit-reason-input" class="font-medium text-[var(--text-main)]">Manager Note / Override Reason</label>
					<textarea
						id="edit-reason-input"
						bind:value={editReason}
						rows="2"
						placeholder="Reason for adjustment"
						class="w-full rounded-xl bg-[var(--surface-hover)] border border-[var(--border)] p-2.5 text-[var(--text-main)] focus:outline-hidden focus:border-[var(--accent)]"
					></textarea>
				</div>
			</div>

			{#if editError}
				<div class="p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-xs text-red-400 flex items-center gap-2">
					<AlertCircle class="w-4 h-4 shrink-0" />
					<span>{editError}</span>
				</div>
			{/if}

			<div class="flex items-center justify-end gap-2 pt-2 border-t border-[var(--border)]">
				<button
					onclick={() => editTarget = null}
					class="px-3 py-1.5 rounded-lg text-xs font-medium text-[var(--text-sub)] hover:text-[var(--text-main)]"
				>
					Cancel
				</button>
				<button
					onclick={submitDirectEdit}
					disabled={editSubmitting}
					class="px-4 py-1.5 rounded-lg text-xs font-semibold bg-emerald-500 text-white hover:bg-emerald-600 disabled:opacity-50 flex items-center gap-1.5"
				>
					{#if editSubmitting}
						<Loader2 class="w-3.5 h-3.5 animate-spin" />
					{/if}
					Save & Approve
				</button>
			</div>
		</div>
	</div>
{/if}
