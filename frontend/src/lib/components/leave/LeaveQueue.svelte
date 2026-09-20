<script lang="ts">
	import { onMount } from 'svelte';
	import { CalendarCheck, Check, X, Clock, AlertCircle, Loader2, MessageSquare, User, FileText, ChevronRight } from '@lucide/svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import { getCompanyLeaveRequests, resolveLeaveRequest, type LeaveRequest } from '$lib/api/leave';

	let {
		companyId,
		onResolved,
		open = $bindable(false)
	}: {
		companyId: string;
		onResolved?: () => void;
		open?: boolean;
	} = $props();

	type FilterStatus = 'pending' | 'approved' | 'rejected' | 'all';

	let activeFilter = $state<FilterStatus>('pending');
	let requests = $state<LeaveRequest[]>([]);
	let loading = $state(false);
	let error = $state('');
	let actionInProgress = $state<string | null>(null);

	// Rejection modal state
	let rejectTarget = $state<LeaveRequest | null>(null);
	let rejectNotes = $state('');
	let rejectSubmitting = $state(false);
	let rejectError = $state('');

	onMount(() => {
		if (companyId) {
			loadRequests();
		}
	});

	// Reload when companyId or activeFilter changes
	$effect(() => {
		if (companyId && activeFilter) {
			loadRequests();
		}
	});

	// If rendered as drawer, reload on open
	$effect(() => {
		if (open && companyId) {
			loadRequests();
		}
	});

	export async function loadRequests() {
		loading = true;
		error = '';
		try {
			const res = await getCompanyLeaveRequests(companyId, activeFilter);
			requests = res.requests;
		} catch (err: any) {
			error = err?.message || 'Failed to load leave review queue';
		} finally {
			loading = false;
		}
	}

	async function handleApprove(id: string) {
		actionInProgress = id;
		try {
			await resolveLeaveRequest(id, 'approve');
			requests = requests.filter((r) => r.id !== id);
			if (onResolved) onResolved();
		} catch (err: any) {
			error = err?.message || 'Failed to approve leave request';
		} finally {
			actionInProgress = null;
		}
	}

	function openRejectModal(req: LeaveRequest) {
		rejectTarget = req;
		rejectNotes = '';
		rejectError = '';
	}

	async function submitRejection() {
		if (!rejectTarget) return;
		rejectSubmitting = true;
		rejectError = '';
		try {
			await resolveLeaveRequest(rejectTarget.id, 'reject', rejectNotes.trim());
			requests = requests.filter((r) => r.id !== rejectTarget?.id);
			rejectTarget = null;
			if (onResolved) onResolved();
		} catch (err: any) {
			rejectError = err?.message || 'Failed to reject leave request';
		} finally {
			rejectSubmitting = false;
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

	const pendingCount = $derived(
		requests.filter((r) => r.status === 'pending').length
	);
</script>

<!-- Embedded Section in Dashboard / Management Hub -->
<div class="p-5 rounded-xl bg-[var(--surface)] border border-[var(--border)] space-y-4 shadow-sm">
	
	<!-- Header & Filter Tabs -->
	<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
		<div class="flex items-center gap-2">
			<CalendarCheck class="w-4 h-4 text-amber-400" />
			<h2 class="text-sm font-bold tracking-tight text-[var(--text-main)]">Leave Requests & Review Queue</h2>
		</div>

		<!-- Status Filter Pills -->
		<div class="flex items-center p-1 rounded-lg bg-[var(--surface-raised)] border border-[var(--border)] text-xs font-mono">
			<button
				type="button"
				class="px-2.5 py-1 rounded-md transition-colors cursor-pointer {activeFilter === 'pending' ? 'bg-[#f97040] text-white font-bold' : 'text-[var(--text-mute)] hover:text-[var(--text-main)]'}"
				onclick={() => (activeFilter = 'pending')}
			>
				Pending
			</button>
			<button
				type="button"
				class="px-2.5 py-1 rounded-md transition-colors cursor-pointer {activeFilter === 'approved' ? 'bg-[#f97040] text-white font-bold' : 'text-[var(--text-mute)] hover:text-[var(--text-main)]'}"
				onclick={() => (activeFilter = 'approved')}
			>
				Approved
			</button>
			<button
				type="button"
				class="px-2.5 py-1 rounded-md transition-colors cursor-pointer {activeFilter === 'rejected' ? 'bg-[#f97040] text-white font-bold' : 'text-[var(--text-mute)] hover:text-[var(--text-main)]'}"
				onclick={() => (activeFilter = 'rejected')}
			>
				Rejected
			</button>
			<button
				type="button"
				class="px-2.5 py-1 rounded-md transition-colors cursor-pointer {activeFilter === 'all' ? 'bg-[#f97040] text-white font-bold' : 'text-[var(--text-mute)] hover:text-[var(--text-main)]'}"
				onclick={() => (activeFilter = 'all')}
			>
				All
			</button>
		</div>
	</div>

	{#if error}
		<div class="p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-xs text-red-400 flex items-center justify-between">
			<span>{error}</span>
			<button class="underline ml-2" onclick={loadRequests}>Retry</button>
		</div>
	{/if}

	<!-- Table Content -->
	{#if loading && requests.length === 0}
		<div class="py-8 flex flex-col items-center justify-center gap-2 text-xs text-[var(--text-mute)]">
			<Loader2 class="w-5 h-5 animate-spin text-[#f97040]" />
			<span>Loading leave queue...</span>
		</div>
	{:else if requests.length === 0}
		<div class="p-8 rounded-lg bg-[var(--surface-raised)] border border-[var(--border)] text-center text-xs text-[var(--text-mute)]">
			No {activeFilter === 'all' ? '' : activeFilter} leave requests found.
		</div>
	{:else}
		<div class="overflow-x-auto rounded-lg border border-[var(--border)]">
			<table class="w-full text-left text-xs">
				<thead class="bg-[var(--surface-raised)] border-b border-[var(--border)] text-[var(--text-mute)] font-mono text-[11px]">
					<tr>
						<th class="py-2.5 px-3">Employee</th>
						<th class="py-2.5 px-3">Leave Category</th>
						<th class="py-2.5 px-3">Dates & Duration</th>
						<th class="py-2.5 px-3">Reason</th>
						<th class="py-2.5 px-3">Status</th>
						<th class="py-2.5 px-3 text-right">Actions</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-[var(--border)] bg-[var(--surface)]">
					{#each requests as req (req.id)}
						<tr class="hover:bg-[var(--surface-raised)]/60 transition-colors">
							
							<!-- Employee Name -->
							<td class="py-2.5 px-3">
								<div class="font-semibold text-[var(--text-main)]">
									{req.user_first_name} {req.user_last_name}
								</div>
								<div class="text-[10px] text-[var(--text-mute)] font-mono">
									{req.role_name || req.user_email}
								</div>
							</td>

							<!-- Category -->
							<td class="py-2.5 px-3">
								<div class="font-medium text-[var(--text-main)]">{req.leave_type_name}</div>
								<span class="inline-block px-1.5 py-0.2 rounded text-[9px] font-mono border {req.is_paid ? 'bg-emerald-500/10 border-emerald-500/25 text-emerald-400' : 'bg-purple-500/10 border-purple-500/25 text-purple-400'}">
									{req.is_paid ? 'Paid' : 'Unpaid'}
								</span>
							</td>

							<!-- Dates & Calendar Duration -->
							<td class="py-2.5 px-3 font-mono">
								<div class="text-[var(--text-main)]">
									{formatDate(req.start_date)} &ndash; {formatDate(req.end_date)}
								</div>
								<div class="text-[10px] text-[#f97040] font-semibold">
									{req.calendar_days} {req.calendar_days === 1 ? 'calendar day' : 'calendar days'}
								</div>
							</td>

							<!-- Reason -->
							<td class="py-2.5 px-3 max-w-xs">
								{#if req.reason}
									<p class="text-[var(--text-sub)] line-clamp-2" title={req.reason}>
										{req.reason}
									</p>
								{:else}
									<span class="text-[var(--text-mute)] font-mono">&mdash;</span>
								{/if}
								{#if req.review_notes}
									<p class="text-[10px] text-amber-400 font-mono mt-0.5" title={req.review_notes}>
										Note: {req.review_notes}
									</p>
								{/if}
							</td>

							<!-- Status Badge -->
							<td class="py-2.5 px-3">
								{#if req.status === 'pending'}
									<span class="px-2 py-0.5 rounded text-[10px] font-mono font-semibold bg-amber-500/10 border border-amber-500/25 text-amber-400">
										Pending
									</span>
								{:else if req.status === 'approved'}
									<span class="px-2 py-0.5 rounded text-[10px] font-mono font-semibold bg-emerald-500/10 border border-emerald-500/25 text-emerald-400">
										Approved
									</span>
								{:else if req.status === 'rejected'}
									<span class="px-2 py-0.5 rounded text-[10px] font-mono font-semibold bg-red-500/10 border border-red-500/25 text-red-400">
										Rejected
									</span>
								{/if}
							</td>

							<!-- Actions -->
							<td class="py-2.5 px-3 text-right">
								{#if req.status === 'pending'}
									<div class="flex items-center justify-end gap-1.5">
										<button
											type="button"
											onclick={() => handleApprove(req.id)}
											disabled={actionInProgress === req.id}
											class="px-2.5 py-1 rounded bg-emerald-500/15 hover:bg-emerald-500/25 text-emerald-400 border border-emerald-500/30 text-[11px] font-semibold transition-colors flex items-center gap-1 cursor-pointer disabled:opacity-50"
										>
											{#if actionInProgress === req.id}
												<Loader2 class="w-3 h-3 animate-spin" />
											{:else}
												<Check class="w-3 h-3" />
											{/if}
											<span>Approve</span>
										</button>

										<button
											type="button"
											onclick={() => openRejectModal(req)}
											disabled={actionInProgress === req.id}
											class="px-2.5 py-1 rounded bg-red-500/15 hover:bg-red-500/25 text-red-400 border border-red-500/30 text-[11px] font-semibold transition-colors flex items-center gap-1 cursor-pointer disabled:opacity-50"
										>
											<X class="w-3 h-3" />
											<span>Reject</span>
										</button>
									</div>
								{:else}
									<span class="text-[10px] font-mono text-[var(--text-mute)]">Resolved</span>
								{/if}
							</td>

						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

</div>

<!-- Rejection Prompt Modal -->
{#if rejectTarget}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-in fade-in duration-150"
		role="dialog"
		aria-modal="true"
	>
		<div class="relative w-full max-w-md rounded-2xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-4 shadow-2xl animate-in zoom-in-95 duration-150">
			
			<div class="flex items-center justify-between">
				<h3 class="text-sm font-bold text-[var(--text-main)]">Reject Leave Request</h3>
				<button
					type="button"
					onclick={() => (rejectTarget = null)}
					class="text-[var(--text-mute)] hover:text-[var(--text-main)]"
				>
					<X class="w-4 h-4" />
				</button>
			</div>

			<p class="text-xs text-[var(--text-sub)]">
				You are rejecting the <strong>{rejectTarget.leave_type_name}</strong> request from <strong>{rejectTarget.user_first_name} {rejectTarget.user_last_name}</strong> ({rejectTarget.calendar_days} calendar days).
			</p>

			{#if rejectError}
				<div class="p-2.5 rounded-lg bg-red-500/10 border border-red-500/30 text-xs text-red-400">
					{rejectError}
				</div>
			{/if}

			<div class="space-y-1.5">
				<label for="reject-notes" class="block text-xs font-semibold text-[var(--text-main)]">
					Reason for Rejection <span class="font-normal text-[var(--text-mute)]">(Shared with employee)</span>
				</label>
				<textarea
					id="reject-notes"
					bind:value={rejectNotes}
					rows="3"
					placeholder="e.g. Inadequate team coverage, please reschedule..."
					class="w-full p-2.5 rounded-lg bg-[var(--bg)] border border-[var(--border)] text-xs text-[var(--text-main)] focus:outline-none focus:border-red-400 transition-colors resize-none"
				></textarea>
			</div>

			<div class="flex items-center justify-end gap-2 pt-2 border-t border-[var(--border)]">
				<Button
					variant="ghost"
					class="h-8 text-xs"
					onclick={() => (rejectTarget = null)}
					disabled={rejectSubmitting}
				>
					Cancel
				</Button>
				<Button
					variant="primary"
					class="h-8 text-xs bg-red-600 hover:bg-red-500 text-white"
					loading={rejectSubmitting}
					disabled={rejectSubmitting}
					onclick={submitRejection}
				>
					Confirm Rejection
				</Button>
			</div>

		</div>
	</div>
{/if}
