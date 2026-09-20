<script lang="ts">
	import { X, Calendar, AlertCircle, CheckCircle2, Clock, Send, ShieldAlert, Sparkles } from '@lucide/svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import { submitLeaveRequest, type LeaveBalance } from '$lib/api/leave';

	let {
		open = $bindable(false),
		balances = [],
		onSubmitted
	}: {
		open: boolean;
		balances: LeaveBalance[];
		onSubmitted?: () => void;
	} = $props();

	let selectedTypeId = $state('');
	let startDate = $state(new Date().toISOString().slice(0, 10));
	let endDate = $state(new Date().toISOString().slice(0, 10));
	let reason = $state('');
	let submitting = $state(false);
	let error = $state('');
	let success = $state('');

	// Pre-select first balance if none selected
	$effect(() => {
		if (open && !selectedTypeId && balances.length > 0) {
			selectedTypeId = balances[0].leave_type_id;
		}
	});

	const selectedBalance = $derived(
		balances.find((b) => b.leave_type_id === selectedTypeId) || null
	);

	// Calendar days calculation: (end_date - start_date) + 1
	const durationDays = $derived.by(() => {
		if (!startDate || !endDate) return 0;
		const start = new Date(startDate);
		const end = new Date(endDate);
		if (end < start) return 0;
		const diffTime = end.getTime() - start.getTime();
		return Math.floor(diffTime / (1000 * 60 * 60 * 24)) + 1;
	});

	const isExceedingQuota = $derived.by(() => {
		if (!selectedBalance) return false;
		if (selectedBalance.is_flexible) return false;
		if (selectedBalance.is_paid && durationDays > selectedBalance.remaining_days) return true;
		if (!selectedBalance.is_paid && selectedBalance.total_allocated_days > 0 && durationDays > selectedBalance.remaining_days) return true;
		return false;
	});

	function handleKeyDown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			open = false;
		}
	}

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		error = '';
		success = '';

		if (!selectedTypeId) {
			error = 'Please select a leave category.';
			return;
		}

		if (!startDate || !endDate) {
			error = 'Please specify both start and end dates.';
			return;
		}

		if (new Date(endDate) < new Date(startDate)) {
			error = 'End date cannot be earlier than start date.';
			return;
		}

		if (isExceedingQuota) {
			error = `Requested duration (${durationDays} days) exceeds available quota (${selectedBalance?.remaining_days} days).`;
			return;
		}

		submitting = true;
		try {
			await submitLeaveRequest({
				leave_type_id: selectedTypeId,
				start_date: startDate,
				end_date: endDate,
				reason: reason.trim()
			});
			success = 'Leave request submitted successfully!';
			setTimeout(() => {
				open = false;
				success = '';
				reason = '';
				if (onSubmitted) onSubmitted();
			}, 900);
		} catch (err: any) {
			error = err?.message || 'Failed to submit leave request. Please verify inputs.';
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:window onkeydown={handleKeyDown} />

{#if open}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6 bg-black/60 backdrop-blur-sm animate-in fade-in duration-200"
		role="dialog"
		aria-modal="true"
	>
		<!-- Modal Box -->
		<div class="relative w-full max-w-lg rounded-2xl bg-[var(--surface)] border border-[var(--border)] shadow-2xl overflow-hidden animate-in zoom-in-95 duration-200">
			
			<!-- Header -->
			<div class="p-5 border-b border-[var(--border)] flex items-center justify-between bg-[var(--surface-raised)]">
				<div class="flex items-center gap-2.5">
					<div class="w-8 h-8 rounded-lg bg-[#f97040]/10 text-[#f97040] flex items-center justify-center">
						<Calendar class="w-4 h-4" />
					</div>
					<div>
						<h2 class="text-sm font-bold tracking-tight text-[var(--text-main)]">Request Time Off</h2>
						<p class="text-[11px] text-[var(--text-mute)] font-mono">Submit request for management approval</p>
					</div>
				</div>

				<button
					type="button"
					onclick={() => (open = false)}
					class="w-7 h-7 rounded-md hover:bg-[var(--surface)] flex items-center justify-center text-[var(--text-mute)] hover:text-[var(--text-main)] transition-colors cursor-pointer"
					aria-label="Close modal"
				>
					<X class="w-4 h-4" />
				</button>
			</div>

			<!-- Body Form -->
			<form onsubmit={handleSubmit} class="p-5 space-y-4">
				
				{#if error}
					<div class="p-3 rounded-lg bg-red-500/10 border border-red-500/30 flex items-start gap-2 text-xs text-red-400 animate-in fade-in duration-150">
						<AlertCircle class="w-4 h-4 shrink-0 mt-0.5" />
						<span>{error}</span>
					</div>
				{/if}

				{#if success}
					<div class="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/30 flex items-center gap-2 text-xs text-emerald-400 animate-in fade-in duration-150">
						<CheckCircle2 class="w-4 h-4 shrink-0" />
						<span>{success}</span>
					</div>
				{/if}

				<!-- Leave Category Selector -->
				<div class="space-y-1.5">
					<label for="leave-type-select" class="block text-xs font-semibold text-[var(--text-main)]">
						Leave Category
					</label>
					<select
						id="leave-type-select"
						bind:value={selectedTypeId}
						class="w-full h-10 px-3 rounded-lg bg-[var(--bg)] border border-[var(--border)] text-xs text-[var(--text-main)] focus:outline-none focus:border-[#f97040] transition-colors cursor-pointer"
						required
					>
						{#each balances as b (b.leave_type_id)}
							<option value={b.leave_type_id}>
								{b.leave_type_name} ({b.is_paid ? 'Paid' : 'Unpaid'}) &mdash; {b.is_flexible ? 'Flexible' : `${b.remaining_days} days left`}
							</option>
						{/each}
					</select>

					{#if selectedBalance}
						<div class="flex items-center justify-between text-[11px] font-mono pt-1 text-[var(--text-sub)]">
							<span>
								Allowance: 
								{#if selectedBalance.is_flexible}
									<strong class="text-purple-400">Flexible (Uncapped)</strong>
								{:else}
									<strong class="text-emerald-400">{selectedBalance.remaining_days} days left</strong> of {selectedBalance.total_allocated_days}d
								{/if}
							</span>
							{#if selectedBalance.carried_over_days > 0}
								<span class="text-blue-400 font-medium">Includes {selectedBalance.carried_over_days}d rollover</span>
							{/if}
						</div>
					{/if}
				</div>

				<!-- Date Range Grid -->
				<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
					<div class="space-y-1.5">
						<label for="start-date-input" class="block text-xs font-semibold text-[var(--text-main)]">
							Start Date
						</label>
						<input
							id="start-date-input"
							type="date"
							bind:value={startDate}
							class="w-full h-10 px-3 rounded-lg bg-[var(--bg)] border border-[var(--border)] text-xs text-[var(--text-main)] focus:outline-none focus:border-[#f97040] transition-colors"
							required
						/>
					</div>

					<div class="space-y-1.5">
						<label for="end-date-input" class="block text-xs font-semibold text-[var(--text-main)]">
							End Date
						</label>
						<input
							id="end-date-input"
							type="date"
							bind:value={endDate}
							min={startDate}
							class="w-full h-10 px-3 rounded-lg bg-[var(--bg)] border border-[var(--border)] text-xs text-[var(--text-main)] focus:outline-none focus:border-[#f97040] transition-colors"
							required
						/>
					</div>
				</div>

				<!-- Calendar Duration Calculation Banner -->
				<div class="p-3 rounded-lg bg-[var(--surface-raised)] border border-[var(--border)] flex items-center justify-between">
					<div class="flex items-center gap-2">
						<Clock class="w-4 h-4 text-[#f97040]" />
						<div class="text-xs">
							<span class="text-[var(--text-mute)]">Duration: </span>
							<strong class="font-display text-sm font-bold {isExceedingQuota ? 'text-red-400' : 'text-[var(--text-main)]'}">
								{durationDays} {durationDays === 1 ? 'Calendar Day' : 'Calendar Days'}
							</strong>
						</div>
					</div>

					<span class="text-[10px] font-mono text-[var(--text-mute)]">
						Consecutive Days
					</span>
				</div>

				{#if isExceedingQuota}
					<div class="p-2.5 rounded-lg bg-amber-500/10 border border-amber-500/30 text-xs text-amber-400 flex items-start gap-2">
						<ShieldAlert class="w-4 h-4 shrink-0 mt-0.5" />
						<span>
							Requested duration ({durationDays} days) exceeds available quota ({selectedBalance?.remaining_days} days). Please select fewer dates or request an unpaid leave.
						</span>
					</div>
				{/if}

				<!-- Reason Field -->
				<div class="space-y-1.5">
					<label for="reason-input" class="block text-xs font-semibold text-[var(--text-main)]">
						Reason / Notes <span class="font-normal text-[var(--text-mute)]">(Optional)</span>
					</label>
					<textarea
						id="reason-input"
						bind:value={reason}
						rows="3"
						placeholder="Brief reason for time off (e.g. medical appointment, annual family trip)..."
						class="w-full p-3 rounded-lg bg-[var(--bg)] border border-[var(--border)] text-xs text-[var(--text-main)] placeholder:text-[var(--text-mute)] focus:outline-none focus:border-[#f97040] transition-colors resize-none"
					></textarea>
				</div>

				<!-- Footer Buttons -->
				<div class="pt-2 flex items-center justify-end gap-2.5 border-t border-[var(--border)]">
					<Button
						type="button"
						variant="ghost"
						class="h-9 px-3 text-xs"
						onclick={() => (open = false)}
						disabled={submitting}
					>
						Cancel
					</Button>

					<Button
						type="submit"
						variant="primary"
						loading={submitting}
						disabled={submitting || isExceedingQuota || durationDays <= 0}
						class="h-9 px-4 text-xs font-semibold"
					>
						<Send class="w-3.5 h-3.5 mr-1.5" />
						<span>Submit Request</span>
					</Button>
				</div>

			</form>

		</div>
	</div>
{/if}
