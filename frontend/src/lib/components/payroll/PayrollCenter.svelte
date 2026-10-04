<script lang="ts">
	import {
		FileSpreadsheet,
		Download,
		Calculator,
		DollarSign,
		Users,
		AlertCircle,
		CheckCircle2,
		TrendingUp
	} from '@lucide/svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import {
		calculatePayroll,
		exportPayrollCSV,
		type PayrollPreview
	} from '$lib/api/payroll';

	let {
		companyId
	}: {
		companyId: string;
	} = $props();

	// Default to current month-to-date
	const now = new Date();
	const firstDayStr = new Date(now.getFullYear(), now.getMonth(), 1).toISOString().slice(0, 10);
	const todayStr = now.toISOString().slice(0, 10);

	let startDate = $state(firstDayStr);
	let endDate = $state(todayStr);

	let loading = $state(false);
	let exporting = $state(false);
	let error = $state('');
	let successMsg = $state('');
	let preview = $state<PayrollPreview | null>(null);

	// Presets
	function setPreset(preset: 'thisMonth' | 'lastMonth' | 'firstHalf' | 'secondHalf') {
		const cur = new Date();
		if (preset === 'thisMonth') {
			startDate = new Date(cur.getFullYear(), cur.getMonth(), 1).toISOString().slice(0, 10);
			endDate = new Date(cur.getFullYear(), cur.getMonth() + 1, 0).toISOString().slice(0, 10);
		} else if (preset === 'lastMonth') {
			startDate = new Date(cur.getFullYear(), cur.getMonth() - 1, 1).toISOString().slice(0, 10);
			endDate = new Date(cur.getFullYear(), cur.getMonth(), 0).toISOString().slice(0, 10);
		} else if (preset === 'firstHalf') {
			startDate = new Date(cur.getFullYear(), cur.getMonth(), 1).toISOString().slice(0, 10);
			endDate = new Date(cur.getFullYear(), cur.getMonth(), 15).toISOString().slice(0, 10);
		} else if (preset === 'secondHalf') {
			startDate = new Date(cur.getFullYear(), cur.getMonth(), 16).toISOString().slice(0, 10);
			endDate = new Date(cur.getFullYear(), cur.getMonth() + 1, 0).toISOString().slice(0, 10);
		}
	}

	export async function handleCalculate() {
		if (!companyId) return;
		if (!startDate || !endDate) {
			error = 'Please select both start and end dates.';
			return;
		}
		if (new Date(endDate) < new Date(startDate)) {
			error = 'End date cannot be earlier than start date.';
			return;
		}

		loading = true;
		error = '';
		successMsg = '';

		try {
			preview = await calculatePayroll(companyId, startDate, endDate);
		} catch (err: any) {
			error = err?.message || 'Failed to calculate payroll preview.';
			preview = null;
		} finally {
			loading = false;
		}
	}

	async function handleExport() {
		if (!companyId || !preview) return;
		exporting = true;
		error = '';
		successMsg = '';

		try {
			await exportPayrollCSV(companyId, startDate, endDate);
			successMsg = 'CSV export initiated successfully.';
			setTimeout(() => {
				successMsg = '';
			}, 4000);
		} catch (err: any) {
			error = err?.message || 'Failed to export payroll CSV.';
		} finally {
			exporting = false;
		}
	}

	function formatCurrency(amount: number, code = 'USD'): string {
		return new Intl.NumberFormat('en-US', {
			style: 'currency',
			currency: code || 'USD',
			minimumFractionDigits: 2,
			maximumFractionDigits: 2
		}).format(amount);
	}
</script>

<div id="payroll-center-section" class="rounded-xl border border-[var(--border)] bg-[var(--surface)] overflow-hidden shadow-sm">
	<!-- Header -->
	<div class="p-4 sm:p-5 border-b border-[var(--border)] flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-[var(--surface-raised)]/40">
		<div class="flex items-center gap-3">
			<div class="w-9 h-9 rounded-lg bg-[#f97040]/10 text-[#f97040] flex items-center justify-center">
				<FileSpreadsheet class="w-5 h-5" />
			</div>
			<div>
				<h2 class="text-sm sm:text-base font-semibold text-[var(--text-main)] flex items-center gap-2">
					Payroll Calculation & Export
					<span class="text-[10px] uppercase font-bold tracking-wider px-1.5 py-0.5 rounded bg-[#f97040]/10 text-[#f97040]">
						Phase 5
					</span>
				</h2>
				<p class="text-xs text-[var(--text-sub)]">
					Evaluate completed shifts, leaves, overrides, and modifiers with exact proration.
				</p>
			</div>
		</div>

		<!-- Date Presets -->
		<div class="flex flex-wrap items-center gap-1.5 text-xs">
			<button
				type="button"
				onclick={() => setPreset('firstHalf')}
				class="px-2.5 py-1 rounded-md bg-[var(--surface)] hover:bg-[var(--surface-hover)] border border-[var(--border)] text-[var(--text-sub)] transition-colors cursor-pointer"
			>
				1st–15th
			</button>
			<button
				type="button"
				onclick={() => setPreset('secondHalf')}
				class="px-2.5 py-1 rounded-md bg-[var(--surface)] hover:bg-[var(--surface-hover)] border border-[var(--border)] text-[var(--text-sub)] transition-colors cursor-pointer"
			>
				16th–End
			</button>
			<button
				type="button"
				onclick={() => setPreset('thisMonth')}
				class="px-2.5 py-1 rounded-md bg-[var(--surface)] hover:bg-[var(--surface-hover)] border border-[var(--border)] text-[var(--text-sub)] transition-colors cursor-pointer"
			>
				This Month
			</button>
			<button
				type="button"
				onclick={() => setPreset('lastMonth')}
				class="px-2.5 py-1 rounded-md bg-[var(--surface)] hover:bg-[var(--surface-hover)] border border-[var(--border)] text-[var(--text-sub)] transition-colors cursor-pointer"
			>
				Last Month
			</button>
		</div>
	</div>

	<!-- Controls Bar -->
	<div class="p-4 sm:p-5 border-b border-[var(--border)] bg-[var(--surface)]">
		<div class="flex flex-col sm:flex-row items-stretch sm:items-end justify-between gap-4">
			<div class="grid grid-cols-1 sm:grid-cols-2 gap-3 flex-1 max-w-lg">
				<div>
					<label for="payroll-start" class="block text-xs font-medium text-[var(--text-sub)] mb-1">
						Pay Period Start
					</label>
					<div class="relative">
						<input
							id="payroll-start"
							type="date"
							bind:value={startDate}
							class="w-full px-3 py-2 text-xs rounded-lg border border-[var(--border)] bg-[var(--surface-raised)] text-[var(--text-main)] focus:outline-none focus:border-[#f97040]"
						/>
					</div>
				</div>

				<div>
					<label for="payroll-end" class="block text-xs font-medium text-[var(--text-sub)] mb-1">
						Pay Period End
					</label>
					<div class="relative">
						<input
							id="payroll-end"
							type="date"
							bind:value={endDate}
							class="w-full px-3 py-2 text-xs rounded-lg border border-[var(--border)] bg-[var(--surface-raised)] text-[var(--text-main)] focus:outline-none focus:border-[#f97040]"
						/>
					</div>
				</div>
			</div>

			<div class="flex items-center gap-2">
				<Button
					variant="primary"
					loading={loading}
					onclick={handleCalculate}
					class="px-4 py-2 text-xs font-semibold cursor-pointer"
				>
					<Calculator class="w-3.5 h-3.5" />
					Calculate Preview
				</Button>

				<Button
					variant="secondary"
					disabled={!preview || exporting}
					loading={exporting}
					onclick={handleExport}
					class="px-4 py-2 text-xs font-semibold cursor-pointer"
				>
					<Download class="w-3.5 h-3.5" />
					Export CSV
				</Button>
			</div>
		</div>

		<!-- Feedback notices -->
		{#if error}
			<div class="mt-4 p-3 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-xs flex items-center gap-2">
				<AlertCircle class="w-4 h-4 shrink-0" />
				<span>{error}</span>
			</div>
		{/if}

		{#if successMsg}
			<div class="mt-4 p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-xs flex items-center gap-2">
				<CheckCircle2 class="w-4 h-4 shrink-0" />
				<span>{successMsg}</span>
			</div>
		{/if}
	</div>

	<!-- Results Preview -->
	{#if preview}
		<!-- Metric Summary Cards -->
		<div class="p-4 sm:p-5 border-b border-[var(--border)] bg-[var(--surface-raised)]/20">
			<div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
				<div class="p-3.5 rounded-lg border border-[var(--border)] bg-[var(--surface)]">
					<div class="text-[11px] font-medium text-[var(--text-sub)] flex items-center justify-between">
						<span>Total Gross Wages</span>
						<TrendingUp class="w-3.5 h-3.5 text-emerald-400" />
					</div>
					<div class="text-lg sm:text-xl font-bold text-[var(--text-main)] mt-1">
						{formatCurrency(preview.total_gross, preview.currency_code)}
					</div>
					<div class="text-[10px] text-[var(--text-mute)] mt-0.5">
						Before deductions & additions
					</div>
				</div>

				<div class="p-3.5 rounded-lg border border-[var(--border)] bg-[var(--surface)]">
					<div class="text-[11px] font-medium text-[var(--text-sub)] flex items-center justify-between">
						<span>Total Net Payout</span>
						<DollarSign class="w-3.5 h-3.5 text-[#f97040]" />
					</div>
					<div class="text-lg sm:text-xl font-bold text-[#f97040] mt-1">
						{formatCurrency(preview.total_net, preview.currency_code)}
					</div>
					<div class="text-[10px] text-[var(--text-mute)] mt-0.5">
						Final payable balance
					</div>
				</div>

				<div class="p-3.5 rounded-lg border border-[var(--border)] bg-[var(--surface)]">
					<div class="text-[11px] font-medium text-[var(--text-sub)] flex items-center justify-between">
						<span>Staff Headcount</span>
						<Users class="w-3.5 h-3.5 text-blue-400" />
					</div>
					<div class="text-lg sm:text-xl font-bold text-[var(--text-main)] mt-1">
						{preview.employees.length}
					</div>
					<div class="text-[10px] text-[var(--text-mute)] mt-0.5">
						Active in period
					</div>
				</div>
			</div>
		</div>

		<!-- Employee Table -->
		<div class="overflow-x-auto">
			<table class="w-full text-left text-xs">
				<thead class="bg-[var(--surface-raised)] border-b border-[var(--border)] text-[var(--text-sub)] font-medium">
					<tr>
						<th class="px-4 py-3">Employee</th>
						<th class="px-4 py-3">Role</th>
						<th class="px-4 py-3">Wage Type & Rate</th>
						<th class="px-4 py-3">Worked Time</th>
						<th class="px-4 py-3">Leave (P/U)</th>
						<th class="px-4 py-3">Modifiers</th>
						<th class="px-4 py-3 text-right">Gross Pay</th>
						<th class="px-4 py-3 text-right">Net Pay</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-[var(--border)]">
					{#each preview.employees as emp (emp.user_id)}
						<tr class="hover:bg-[var(--surface-raised)]/50 transition-colors">
							<!-- Name & Email -->
							<td class="px-4 py-3">
								<div class="font-medium text-[var(--text-main)]">
									{emp.first_name} {emp.last_name}
								</div>
								<div class="text-[10px] text-[var(--text-mute)]">{emp.email}</div>
							</td>

							<!-- Role -->
							<td class="px-4 py-3 text-[var(--text-sub)]">
								<span class="inline-block px-2 py-0.5 rounded bg-[var(--surface-raised)] border border-[var(--border)] text-[11px]">
									{emp.role_name}
								</span>
							</td>

							<!-- Wage & Rate -->
							<td class="px-4 py-3">
								<div class="font-medium text-[var(--text-main)]">
									{formatCurrency(emp.base_rate, emp.currency_code)}
									<span class="text-[10px] text-[var(--text-sub)] font-normal">/{emp.wage_type}</span>
								</div>
								<div class="mt-0.5">
									{#if emp.wage_source === 'override'}
										<span class="px-1.5 py-0.2 rounded bg-amber-500/10 text-amber-400 text-[9px] font-semibold">
											override
										</span>
									{:else if emp.wage_source === 'role'}
										<span class="px-1.5 py-0.2 rounded bg-blue-500/10 text-blue-400 text-[9px] font-semibold">
											role default
										</span>
									{:else}
										<span class="px-1.5 py-0.2 rounded bg-neutral-500/10 text-neutral-400 text-[9px]">
											none
										</span>
									{/if}
								</div>
							</td>

							<!-- Worked Time -->
							<td class="px-4 py-3 text-[var(--text-sub)]">
								<div>
									<span class="font-medium text-[var(--text-main)]">
										{(emp.worked_minutes / 60).toFixed(1)}
									</span> hrs
								</div>
								<div class="text-[10px] text-[var(--text-mute)]">
									{emp.worked_days} {emp.worked_days === 1 ? 'day' : 'days'}
								</div>
							</td>

							<!-- Leave Days -->
							<td class="px-4 py-3">
								<div class="flex items-center gap-1.5 text-[11px]">
									<span class="text-emerald-400" title="Paid Leave">{emp.paid_leave_days}d paid</span>
									<span class="text-[var(--text-mute)]">/</span>
									<span class="text-rose-400" title="Unpaid Leave">{emp.unpaid_leave_days}d unpaid</span>
								</div>
							</td>

							<!-- Modifiers -->
							<td class="px-4 py-3">
								{#if emp.modifiers.length > 0}
									<div class="flex flex-wrap gap-1 max-w-xs">
										{#each emp.modifiers as m (m.id)}
											{#if m.modifier_type === 'addition'}
												<span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] bg-emerald-500/10 text-emerald-400 border border-emerald-500/20" title="{m.name} ({m.calculation_method})">
													+{formatCurrency(m.amount, emp.currency_code)}
												</span>
											{:else}
												<span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] bg-rose-500/10 text-rose-400 border border-rose-500/20" title="{m.name} ({m.calculation_method})">
													-{formatCurrency(m.amount, emp.currency_code)}
												</span>
											{/if}
										{/each}
									</div>
								{:else}
									<span class="text-[var(--text-mute)] text-[11px]">—</span>
								{/if}
							</td>

							<!-- Gross Pay -->
							<td class="px-4 py-3 text-right font-medium text-[var(--text-main)]">
								{formatCurrency(emp.gross_pay, emp.currency_code)}
							</td>

							<!-- Net Pay -->
							<td class="px-4 py-3 text-right font-bold text-[#f97040]">
								{formatCurrency(emp.net_pay, emp.currency_code)}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{:else if !loading}
		<!-- Empty / Prompt State -->
		<div class="p-10 text-center text-xs text-[var(--text-sub)]">
			<FileSpreadsheet class="w-8 h-8 mx-auto text-[var(--text-mute)] mb-2 stroke-[1.5]" />
			<p class="font-medium text-[var(--text-main)]">No Payroll Preview Generated</p>
			<p class="text-[var(--text-mute)] mt-1">
				Select a date range above and click <strong>Calculate Preview</strong> to evaluate shifts, leaves, and modifiers.
			</p>
		</div>
	{/if}
</div>
