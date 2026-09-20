<script lang="ts">
	import { onMount } from 'svelte';
	import { X, Sliders, Repeat, AlertCircle, CheckCircle2, ShieldCheck, Info } from '@lucide/svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import { getCompanyLeavePolicy, updateCompanyLeavePolicy, type CompanyLeavePolicy } from '$lib/api/leave';

	let {
		open = $bindable(false),
		companyId,
		onUpdated
	}: {
		open: boolean;
		companyId: string;
		onUpdated?: () => void;
	} = $props();

	let resetMonth = $state(1);
	let resetDay = $state(1);
	let allowCarryover = $state(false);
	let maxCarryoverDays = $state(0);

	let loading = $state(false);
	let saving = $state(false);
	let error = $state('');
	let success = $state('');

	const months = [
		{ value: 1, label: 'January' },
		{ value: 2, label: 'February' },
		{ value: 3, label: 'March' },
		{ value: 4, label: 'April' },
		{ value: 5, label: 'May' },
		{ value: 6, label: 'June' },
		{ value: 7, label: 'July' },
		{ value: 8, label: 'August' },
		{ value: 9, label: 'September' },
		{ value: 10, label: 'October' },
		{ value: 11, label: 'November' },
		{ value: 12, label: 'December' }
	];

	$effect(() => {
		if (open && companyId) {
			loadPolicy();
		}
	});

	async function loadPolicy() {
		loading = true;
		error = '';
		try {
			const policy = await getCompanyLeavePolicy(companyId);
			resetMonth = policy.leave_reset_month || 1;
			resetDay = policy.leave_reset_day || 1;
			allowCarryover = !!policy.allow_leave_carryover;
			maxCarryoverDays = policy.max_carryover_days || 0;
		} catch (err: any) {
			error = err?.message || 'Failed to load leave policy settings';
		} finally {
			loading = false;
		}
	}

	async function handleSave(e: SubmitEvent) {
		e.preventDefault();
		saving = true;
		error = '';
		success = '';
		try {
			await updateCompanyLeavePolicy(companyId, {
				leave_reset_month: Number(resetMonth),
				leave_reset_day: Number(resetDay),
				allow_leave_carryover: allowCarryover,
				max_carryover_days: Number(maxCarryoverDays)
			});
			success = 'Leave reset & rollover policy saved successfully!';
			setTimeout(() => {
				open = false;
				success = '';
				if (onUpdated) onUpdated();
			}, 900);
		} catch (err: any) {
			error = err?.message || 'Failed to update leave policy settings';
		} finally {
			saving = false;
		}
	}

	const selectedMonthName = $derived(
		months.find((m) => m.value === Number(resetMonth))?.label || 'January'
	);
</script>

{#if open}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 sm:p-6 bg-black/60 backdrop-blur-sm animate-in fade-in duration-200"
		role="dialog"
		aria-modal="true"
	>
		<div class="relative w-full max-w-lg rounded-2xl bg-[var(--surface)] border border-[var(--border)] shadow-2xl overflow-hidden animate-in zoom-in-95 duration-200">
			
			<!-- Modal Header -->
			<div class="p-5 border-b border-[var(--border)] flex items-center justify-between bg-[var(--surface-raised)]">
				<div class="flex items-center gap-2.5">
					<div class="w-8 h-8 rounded-lg bg-[#f97040]/10 text-[#f97040] flex items-center justify-center">
						<Sliders class="w-4 h-4" />
					</div>
					<div>
						<h2 class="text-sm font-bold tracking-tight text-[var(--text-main)]">Leave Reset & Rollover Policy</h2>
						<p class="text-[11px] text-[var(--text-mute)] font-mono">Company Administrator settings</p>
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

			<!-- Form Content -->
			<form onsubmit={handleSave} class="p-5 space-y-4">
				
				{#if error}
					<div class="p-3 rounded-lg bg-red-500/10 border border-red-500/30 flex items-start gap-2 text-xs text-red-400">
						<AlertCircle class="w-4 h-4 shrink-0 mt-0.5" />
						<span>{error}</span>
					</div>
				{/if}

				{#if success}
					<div class="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/30 flex items-center gap-2 text-xs text-emerald-400">
						<CheckCircle2 class="w-4 h-4 shrink-0" />
						<span>{success}</span>
					</div>
				{/if}

				<!-- Section: Reset Cycle -->
				<div class="space-y-2">
					<span class="text-xs font-bold text-[var(--text-main)] flex items-center gap-1.5">
						<Repeat class="w-3.5 h-3.5 text-[#f97040]" />
						<span>Annual Quota Reset Date</span>
					</span>
					<p class="text-[11px] text-[var(--text-mute)]">
						Date when all employee balances renew and previous unused balances are evaluated for rollover.
					</p>

					<div class="grid grid-cols-2 gap-3 pt-1">
						<div class="space-y-1">
							<label for="reset-month-select" class="block text-[11px] font-semibold text-[var(--text-sub)]">
								Month
							</label>
							<select
								id="reset-month-select"
								bind:value={resetMonth}
								class="w-full h-9 px-3 rounded-lg bg-[var(--bg)] border border-[var(--border)] text-xs text-[var(--text-main)] focus:outline-none focus:border-[#f97040] transition-colors cursor-pointer"
							>
								{#each months as m}
									<option value={m.value}>{m.label}</option>
								{/each}
							</select>
						</div>

						<div class="space-y-1">
							<label for="reset-day-input" class="block text-[11px] font-semibold text-[var(--text-sub)]">
								Day
							</label>
							<input
								id="reset-day-input"
								type="number"
								min="1"
								max="31"
								bind:value={resetDay}
								class="w-full h-9 px-3 rounded-lg bg-[var(--bg)] border border-[var(--border)] text-xs text-[var(--text-main)] focus:outline-none focus:border-[#f97040] transition-colors"
								required
							/>
						</div>
					</div>
				</div>

				<!-- Section: Rollover / Stackable Leaves -->
				<div class="p-4 rounded-xl bg-[var(--surface-raised)] border border-[var(--border)] space-y-3">
					<div class="flex items-center justify-between">
						<div>
							<div class="text-xs font-bold text-[var(--text-main)]">Allow Unused Leave Rollover</div>
							<p class="text-[11px] text-[var(--text-mute)] mt-0.5">
								Stack remaining unused days into the next cycle.
							</p>
						</div>

						<!-- Toggle switch -->
						<button
							type="button"
							role="switch"
							aria-label="Toggle allow unused leave rollover"
							aria-checked={allowCarryover}
							onclick={() => (allowCarryover = !allowCarryover)}
							class="w-11 h-6 rounded-full transition-colors relative cursor-pointer {allowCarryover ? 'bg-[#f97040]' : 'bg-[var(--border)]'}"
						>
							<span
								class="block w-4 h-4 rounded-full bg-white transition-transform duration-200 absolute top-1 {allowCarryover ? 'left-6' : 'left-1'}"
							></span>
						</button>
					</div>

					{#if allowCarryover}
						<div class="pt-2 border-t border-[var(--border)] space-y-1.5 animate-in fade-in duration-150">
							<label for="max-carryover-input" class="block text-[11px] font-semibold text-[var(--text-sub)]">
								Maximum Carryover Days <span class="font-normal text-[var(--text-mute)]">(0 = No Cap / Full Rollover)</span>
							</label>
							<input
								id="max-carryover-input"
								type="number"
								min="0"
								max="365"
								bind:value={maxCarryoverDays}
								class="w-full h-9 px-3 rounded-lg bg-[var(--bg)] border border-[var(--border)] text-xs text-[var(--text-main)] focus:outline-none focus:border-[#f97040] transition-colors"
							/>
						</div>
					{/if}
				</div>

				<!-- Summary Preview Banner -->
				<div class="p-3 rounded-lg bg-[var(--bg)] border border-[var(--border)] flex items-start gap-2 text-xs text-[var(--text-sub)]">
					<Info class="w-4 h-4 text-[#f97040] shrink-0 mt-0.5" />
					<p class="leading-relaxed">
						Allowances will renew annually on <strong>{selectedMonthName} {resetDay}</strong>. 
						{#if allowCarryover}
							Unused days <strong>will roll over</strong> into the next cycle
							{#if maxCarryoverDays > 0}
								(capped at {maxCarryoverDays} days).
							{:else}
								with no cap.
							{/if}
						{:else}
							Unused leaves <strong>will not roll over</strong> and will expire upon reset.
						{/if}
					</p>
				</div>

				<!-- Footer Actions -->
				<div class="flex items-center justify-end gap-2.5 pt-2 border-t border-[var(--border)]">
					<Button
						type="button"
						variant="ghost"
						class="h-9 px-3 text-xs"
						onclick={() => (open = false)}
						disabled={saving}
					>
						Cancel
					</Button>

					<Button
						type="submit"
						variant="primary"
						loading={saving}
						disabled={saving}
						class="h-9 px-4 text-xs font-semibold"
					>
						<ShieldCheck class="w-3.5 h-3.5 mr-1.5" />
						<span>Save Policy Settings</span>
					</Button>
				</div>

			</form>

		</div>
	</div>
{/if}
