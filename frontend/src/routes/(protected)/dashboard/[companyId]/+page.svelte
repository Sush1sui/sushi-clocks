<script lang="ts">
	import { page } from '$app/state';
	import { goto, invalidateAll } from '$app/navigation';
	import { logout } from '$lib/api/auth';
	import type { Company } from '$lib/api/companies';
	import {
		clockIn,
		clockOut,
		getCompanyAttendanceSummary,
		type Timesheet,
		type AttendanceSummary
	} from '$lib/api/timesheets';
	import sushiLogo from '$lib/assets/sushi_logo_without_bg.png';
	import {
		Building2,
		Shield,
		Clock,
		CalendarCheck,
		FileSpreadsheet,
		Users,
		LogOut,
		ArrowLeft,
		UserCheck,
		UserX,
		CheckCircle2,
		Play,
		Square,
		Send,
		SlidersHorizontal,
		ChevronRight,
		AlertCircle
	} from '@lucide/svelte';

	import Logo from '$lib/components/common/Logo.svelte';
	import ThemeToggle from '$lib/components/common/ThemeToggle.svelte';
	import StatusBadge from '$lib/components/common/StatusBadge.svelte';
	import Button from '$lib/components/ui/Button.svelte';

	let isLoggingOut = $state(false);

	const user = $derived(page.data.user);
	const company = $derived(page.data.company as Company);

	const isSuperAdminInspecting = $derived(user?.system_role === 'super_admin');
	const isCompanyAdmin = $derived(user?.system_role === 'admin');
	const isHR = $derived(user?.system_role === 'hr');
	const isEmployee = $derived(user?.system_role === 'employee');

	// Initial data from server load
	const initialStatus = $derived(page.data.timesheetStatus);
	const initialSummary = $derived(page.data.attendanceSummary);

	let activeShift = $state<Timesheet | null>(null);
	let attendanceSummary = $state<AttendanceSummary | null>(null);
	let isPunching = $state(false);
	let punchError = $state<string | null>(null);

	// Sync when server data updates
	$effect(() => {
		if (initialStatus !== undefined) {
			activeShift = initialStatus?.shift || null;
		}
	});

	$effect(() => {
		if (initialSummary !== undefined) {
			attendanceSummary = initialSummary || null;
		}
	});

	const isClockedIn = $derived(activeShift !== null && activeShift.status === 'active');
	const clockTime = $derived.by(() => {
		if (!activeShift?.clock_in_time) return null;
		try {
			return new Date(activeShift.clock_in_time).toLocaleTimeString([], {
				hour: '2-digit',
				minute: '2-digit',
				timeZone: company?.timezone || undefined
			});
		} catch {
			return new Date(activeShift.clock_in_time).toLocaleTimeString([], {
				hour: '2-digit',
				minute: '2-digit'
			});
		}
	});

	const totalStaffCount = $derived(attendanceSummary?.total_staff ?? 0);
	const liveClockedInCount = $derived(attendanceSummary?.clocked_in_count ?? 0);
	const liveClockedOutCount = $derived(attendanceSummary?.clocked_out_count ?? 0);

	async function refreshAttendance() {
		if (!company?.id) return;
		if (isCompanyAdmin || isHR || isSuperAdminInspecting) {
			try {
				const updated = await getCompanyAttendanceSummary(company.id);
				attendanceSummary = updated;
			} catch (err) {
				console.error('Failed to refresh attendance summary:', err);
			}
		}
	}

	async function toggleClock() {
		punchError = null;
		isPunching = true;
		try {
			if (isClockedIn) {
				await clockOut();
				activeShift = null;
			} else {
				const res = await clockIn();
				activeShift = res.timesheet;
			}
			await refreshAttendance();
			await invalidateAll();
		} catch (err: unknown) {
			if (err instanceof Error) {
				punchError = err.message;
			} else {
				punchError = 'Failed to record shift. Please try again.';
			}
		} finally {
			isPunching = false;
		}
	}

	async function handleLogout() {
		isLoggingOut = true;
		try {
			await logout();
		} catch (err) {
			console.error('Logout error:', err);
		} finally {
			await invalidateAll();
			await goto('/');
		}
	}

	const roleDisplayMap: Record<string, { label: string; badgeClass: string }> = {
		super_admin: {
			label: 'System Admin (Inspecting)',
			badgeClass: 'bg-purple-500/15 border-purple-500/30 text-purple-400'
		},
		admin: {
			label: 'Company Admin',
			badgeClass: 'bg-blue-500/15 border-blue-500/30 text-blue-400'
		},
		hr: {
			label: 'HR Manager',
			badgeClass: 'bg-amber-500/15 border-amber-500/30 text-amber-400'
		},
		employee: {
			label: 'Employee',
			badgeClass: 'bg-emerald-500/15 border-emerald-500/30 text-emerald-400'
		}
	};
</script>

<svelte:head>
	<title>{company?.name || 'Company'} Dashboard — 🍣 Sushi Clocks</title>
</svelte:head>

<div class="relative min-h-screen bg-[var(--bg)] text-[var(--text-main)] overflow-x-hidden">
	
	<!-- 2 Large Faded Background Logos -->
	<div
		class="pointer-events-none absolute top-14 -right-24 w-[480px] h-[480px] watermark-logo rotate-[12deg] select-none"
		aria-hidden="true"
	>
		<img src={sushiLogo} alt="" class="w-full h-full object-contain" />
	</div>
	<div
		class="pointer-events-none absolute bottom-12 -left-20 w-[420px] h-[420px] watermark-logo rotate-[-15deg] select-none"
		aria-hidden="true"
	>
		<img src={sushiLogo} alt="" class="w-full h-full object-contain" />
	</div>

	<!-- Top Navigation Bar -->
	<header class="border-b border-[var(--border)] bg-[var(--bg)]/95 backdrop-blur-sm sticky top-0 z-30">
		<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-14 flex items-center justify-between">
			<div class="flex items-center gap-3">
				{#if isSuperAdminInspecting}
					<Button
						variant="ghost"
						class="h-7.5 px-2 text-xs text-[var(--text-mute)] hover:text-[var(--text-main)] border border-[var(--border)]"
						onclick={() => goto('/dashboard')}
					>
						<ArrowLeft class="w-3.5 h-3.5 mr-1" />
						<span>Platform Hub</span>
					</Button>
					<div class="h-4 w-px bg-[var(--border)]"></div>
				{/if}

				<Logo size="sm" />
				<div class="hidden sm:block h-4 w-px bg-[var(--border)]"></div>
				<div class="flex items-center gap-2">
					<span class="font-display font-bold text-sm tracking-tight">{company?.name}</span>
					<span class="px-1.5 py-0.2 rounded text-[10px] font-mono font-semibold bg-[#f97040]/10 border border-[#f97040]/30 text-[#f97040]">
						{company?.currency_code}
					</span>
				</div>
			</div>

			<div class="flex items-center gap-2.5">
				<StatusBadge label="Live Sync" dotColor="emerald" />
				<ThemeToggle />
				<Button
					variant="ghost"
					loading={isLoggingOut}
					onclick={handleLogout}
					class="text-xs h-8 px-2.5 border border-[var(--border)] hover:border-red-500/30 hover:text-red-400"
				>
					<LogOut class="w-3.5 h-3.5 mr-1" />
					<span>Sign Out</span>
				</Button>
			</div>
		</div>
	</header>

	<!-- Main Workspace -->
	<main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-5">
		
		<!-- Inspection Mode Banner for Super Admin -->
		{#if isSuperAdminInspecting}
			<div class="p-3 rounded-lg bg-purple-500/10 border border-purple-500/30 flex items-center justify-between gap-3 text-xs">
				<div class="flex items-center gap-2 text-purple-400">
					<Shield class="w-4 h-4 shrink-0" />
					<div>
						<span class="font-bold">Inspection Mode:</span> Viewing <strong>{company.name}</strong> as Platform Administrator.
					</div>
				</div>
				<Button
					variant="secondary"
					class="h-7 text-xs px-2.5 text-purple-300 border-purple-500/30 hover:bg-purple-500/10"
					onclick={() => goto('/dashboard')}
				>
					Return to Platform Hub
				</Button>
			</div>
		{/if}

		<!-- Unified Workspace Header & Live Attendance Bar -->
		<section class="p-4 sm:p-5 rounded-xl bg-[var(--surface)] border border-[var(--border)] shadow-sm">
			<div class="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
				
				<!-- Left: Organization Identity -->
				<div class="space-y-1">
					<div class="flex flex-wrap items-center gap-2">
						{#if user}
							<span class="px-2 py-0.5 rounded text-[10px] font-mono font-medium border {roleDisplayMap[user.system_role]?.badgeClass}">
								{roleDisplayMap[user.system_role]?.label || user.system_role}
							</span>
						{/if}
						<span class="text-[10px] font-mono text-[var(--text-mute)]">
							{user?.email}
						</span>
					</div>

					<h1 class="font-display text-xl sm:text-2xl font-bold tracking-tight text-[var(--text-main)]">
						{company.name}
					</h1>

					<div class="flex flex-wrap items-center gap-3 text-xs text-[var(--text-sub)] pt-0.5 font-mono">
						<span>Timezone: {company.timezone}</span>
						<span>•</span>
						<span>Currency: {company.currency_code}</span>
					</div>
				</div>

				<!-- Right: Live Attendance Summary Chips (Visible for Admin / HR) -->
				{#if isCompanyAdmin || isHR || isSuperAdminInspecting}
					<div class="flex flex-wrap items-center gap-2 sm:gap-3 bg-[var(--surface-raised)] p-2.5 sm:p-3 rounded-lg border border-[var(--border)]">
						
						<!-- Clocked In Chip -->
						<div class="flex items-center gap-2 px-3 py-1.5 rounded-md bg-emerald-500/10 border border-emerald-500/25 text-emerald-400">
							<span class="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></span>
							<div class="text-xs font-semibold">
								<span class="font-bold">{liveClockedInCount}</span>
								<span class="font-normal opacity-90 text-[11px] ml-1">Clocked In</span>
							</div>
						</div>

						<!-- Clocked Out Chip -->
						<div class="flex items-center gap-2 px-3 py-1.5 rounded-md bg-[var(--surface)] border border-[var(--border)] text-[var(--text-sub)]">
							<span class="w-2 h-2 rounded-full bg-neutral-400 dark:bg-neutral-500"></span>
							<div class="text-xs">
								<span class="font-bold text-[var(--text-main)]">{liveClockedOutCount}</span>
								<span class="text-[11px] ml-1">Clocked Out</span>
							</div>
						</div>

						<!-- Total Team Chip -->
						<div class="hidden sm:flex items-center gap-1.5 px-3 py-1.5 text-xs text-[var(--text-mute)] font-mono">
							<span>{totalStaffCount} Staff</span>
						</div>
					</div>
				{/if}
			</div>
		</section>

		<!-- Streamlined 2-Column Dashboard Grid -->
		<div class="grid grid-cols-1 lg:grid-cols-12 gap-5">
			
			<!-- Left Column: Primary Self-Service Action Center (Time Tracker & Leave) -->
			<div class="lg:col-span-5 space-y-4">
				
				<!-- Time Card -->
				<div class="p-5 rounded-xl bg-[var(--surface)] border border-[var(--border)] space-y-4 shadow-sm">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-2">
							<Clock class="w-4 h-4 text-[#f97040]" />
							<h2 class="text-sm font-bold tracking-tight">Time Card</h2>
						</div>
						<span class="px-2 py-0.5 rounded text-[10px] font-mono font-medium {isClockedIn ? 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30' : 'bg-neutral-500/15 text-neutral-400 border border-neutral-500/30'}">
							{isClockedIn ? 'CLOCKED IN' : 'CLOCKED OUT'}
						</span>
					</div>

					{#if punchError}
						<div class="p-2.5 rounded-lg bg-red-500/10 border border-red-500/30 flex items-start gap-2 text-xs text-red-400">
							<AlertCircle class="w-3.5 h-3.5 shrink-0 mt-0.5" />
							<span>{punchError}</span>
						</div>
					{/if}

					<div class="p-4 rounded-lg bg-[var(--surface-raised)] border border-[var(--border)] text-center space-y-1">
						<div class="text-[10px] text-[var(--text-mute)] font-mono uppercase tracking-wider">
							{isClockedIn ? 'Active Shift Started' : 'Current Shift Status'}
						</div>
						<div class="font-display font-bold text-xl tracking-tight {isClockedIn ? 'text-emerald-400' : 'text-[#f97040]'}">
							{isClockedIn ? `Clocked in at ${clockTime}` : 'Ready to clock in'}
						</div>
						<div class="text-[11px] text-[var(--text-mute)]">
							{isClockedIn ? 'Work hours are currently being recorded' : 'Press button below to start shift recording'}
						</div>
					</div>

					<!-- Single-Click Action Button -->
					<Button
						variant={isClockedIn ? 'outline' : 'primary'}
						loading={isPunching}
						disabled={isPunching}
						class="w-full h-11 text-sm font-semibold {isClockedIn ? 'border-red-500/40 text-red-400 hover:bg-red-500/10' : ''}"
						onclick={toggleClock}
					>
						{#if isClockedIn}
							<Square class="w-4 h-4 mr-1.5" />
							<span>Clock Out</span>
						{:else}
							<Play class="w-4 h-4 mr-1.5 fill-current" />
							<span>Clock In</span>
						{/if}
					</Button>
				</div>

				<!-- Leave Balances & Fast Request -->
				<div class="p-4 rounded-xl bg-[var(--surface)] border border-[var(--border)] space-y-3 shadow-sm">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-2">
							<CalendarCheck class="w-4 h-4 text-blue-400" />
							<h3 class="text-xs font-bold tracking-tight">Available Time Off</h3>
						</div>
						<span class="text-[10px] font-mono text-[var(--text-mute)]">2026 Allowance</span>
					</div>

					<div class="grid grid-cols-2 gap-2 text-xs">
						<div class="p-2.5 rounded-lg bg-[var(--surface-raised)] border border-[var(--border)]">
							<span class="text-[10px] text-[var(--text-mute)] font-mono">Vacation</span>
							<div class="font-display text-sm font-bold text-emerald-400 mt-0.5">12 Days</div>
						</div>
						<div class="p-2.5 rounded-lg bg-[var(--surface-raised)] border border-[var(--border)]">
							<span class="text-[10px] text-[var(--text-mute)] font-mono">Sick Leave</span>
							<div class="font-display text-sm font-bold text-blue-400 mt-0.5">8 Days</div>
						</div>
					</div>

					<Button
						variant="secondary"
						class="w-full h-9 text-xs"
					>
						<Send class="w-3.5 h-3.5 mr-1.5 text-[#f97040]" />
						<span>Request Time Off</span>
					</Button>
				</div>

			</div>

			<!-- Right Column: Operational Quick Access (For Admin & HR) -->
			<div class="lg:col-span-7 space-y-4">
				
				{#if isCompanyAdmin || isHR || isSuperAdminInspecting}
					
					<!-- Operational Management Hub -->
					<div class="p-5 rounded-xl bg-[var(--surface)] border border-[var(--border)] space-y-4 shadow-sm">
						<div class="flex items-center justify-between">
							<div class="flex items-center gap-2">
								<Building2 class="w-4 h-4 text-[#f97040]" />
								<h2 class="text-sm font-bold tracking-tight">Management & Operations</h2>
							</div>
							<span class="text-[10px] font-mono text-[var(--text-mute)]">Quick Access</span>
						</div>

						<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
							
							<!-- Users & Staff -->
							<button
								type="button"
								class="p-3.5 rounded-lg bg-[var(--surface-raised)] border border-[var(--border)] hover:border-[#f97040]/50 transition-colors text-left group cursor-pointer"
							>
								<div class="flex items-start justify-between">
									<div class="w-7 h-7 rounded bg-blue-500/10 text-blue-400 flex items-center justify-center mb-2">
										<Users class="w-3.5 h-3.5" />
									</div>
									<ChevronRight class="w-3.5 h-3.5 text-[var(--text-mute)] group-hover:text-[var(--text-main)] transition-colors" />
								</div>
								<div class="text-xs font-semibold text-[var(--text-main)]">Users & Staff</div>
								<div class="text-[11px] text-[var(--text-sub)] mt-0.5">Manage employee accounts and roles.</div>
							</button>

							<!-- Live Attendance -->
							<button
								type="button"
								class="p-3.5 rounded-lg bg-[var(--surface-raised)] border border-[var(--border)] hover:border-[#f97040]/50 transition-colors text-left group cursor-pointer"
							>
								<div class="flex items-start justify-between">
									<div class="w-7 h-7 rounded bg-emerald-500/10 text-emerald-400 flex items-center justify-center mb-2">
										<UserCheck class="w-3.5 h-3.5" />
									</div>
									<ChevronRight class="w-3.5 h-3.5 text-[var(--text-mute)] group-hover:text-[var(--text-main)] transition-colors" />
								</div>
								<div class="text-xs font-semibold text-[var(--text-main)]">Live Attendance</div>
								<div class="text-[11px] text-[var(--text-sub)] mt-0.5">View active shifts and timestamps.</div>
							</button>

							<!-- Leave Requests -->
							<button
								type="button"
								class="p-3.5 rounded-lg bg-[var(--surface-raised)] border border-[var(--border)] hover:border-[#f97040]/50 transition-colors text-left group cursor-pointer"
							>
								<div class="flex items-start justify-between">
									<div class="w-7 h-7 rounded bg-amber-500/10 text-amber-400 flex items-center justify-center mb-2">
										<CalendarCheck class="w-3.5 h-3.5" />
									</div>
									<ChevronRight class="w-3.5 h-3.5 text-[var(--text-mute)] group-hover:text-[var(--text-main)] transition-colors" />
								</div>
								<div class="text-xs font-semibold text-[var(--text-main)]">Leave Requests</div>
								<div class="text-[11px] text-[var(--text-sub)] mt-0.5">Approve or reject time-off requests.</div>
							</button>

							<!-- Payroll Reports -->
							<button
								type="button"
								class="p-3.5 rounded-lg bg-[var(--surface-raised)] border border-[var(--border)] hover:border-[#f97040]/50 transition-colors text-left group cursor-pointer"
							>
								<div class="flex items-start justify-between">
									<div class="w-7 h-7 rounded bg-[#f97040]/10 text-[#f97040] flex items-center justify-center mb-2">
										<FileSpreadsheet class="w-3.5 h-3.5" />
									</div>
									<ChevronRight class="w-3.5 h-3.5 text-[var(--text-mute)] group-hover:text-[var(--text-main)] transition-colors" />
								</div>
								<div class="text-xs font-semibold text-[var(--text-main)]">Payroll Reports</div>
								<div class="text-[11px] text-[var(--text-sub)] mt-0.5">Calculate wages and export CSV.</div>
							</button>

						</div>

						<!-- Admin Settings Row -->
						{#if isCompanyAdmin || isSuperAdminInspecting}
							<div class="pt-3 border-t border-[var(--border)] flex items-center justify-between text-xs text-[var(--text-sub)]">
								<div class="flex items-center gap-1.5">
									<SlidersHorizontal class="w-3.5 h-3.5 text-[var(--text-mute)]" />
									<span>Company Policies & Pay Rates</span>
								</div>
								<Button variant="ghost" class="h-7 text-xs px-2 text-[#f97040] hover:underline">
									Manage Policies →
								</Button>
							</div>
						{/if}

					</div>

				{:else}
					
					<!-- Employee Info / Tips Card (When logged in as Employee) -->
					<div class="p-5 rounded-xl bg-[var(--surface)] border border-[var(--border)] space-y-3 shadow-sm">
						<h3 class="text-xs font-bold tracking-tight text-[var(--text-main)]">Quick Tips</h3>
						<ul class="text-xs text-[var(--text-sub)] space-y-2 list-disc list-inside">
							<li>Remember to clock in at the start of your shift and clock out before leaving.</li>
							<li>Leave requests should be submitted at least 3 business days in advance.</li>
							<li>Your attendance and hours will automatically be reflected in monthly payroll reports.</li>
						</ul>
					</div>

				{/if}

			</div>

		</div>

	</main>
</div>
