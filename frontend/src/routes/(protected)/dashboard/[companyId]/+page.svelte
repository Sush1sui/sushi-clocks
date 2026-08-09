<script lang="ts">
	import { page } from '$app/state';
	import { goto, invalidateAll } from '$app/navigation';
	import { logout } from '$lib/api/auth';
	import type { Company } from '$lib/api/companies';
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
		DollarSign,
		CheckCircle2,
		Play,
		Square,
		Send
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

	// Employee Clock Demo State
	let isClockedIn = $state(false);
	let clockTime = $state<string | null>(null);

	function toggleClock() {
		if (isClockedIn) {
			isClockedIn = false;
			clockTime = null;
		} else {
			isClockedIn = true;
			clockTime = new Date().toLocaleTimeString();
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
			label: 'Platform Super Admin (Inspecting)',
			badgeClass: 'bg-purple-500/15 border-purple-500/30 text-purple-400'
		},
		admin: {
			label: 'Company Administrator',
			badgeClass: 'bg-blue-500/15 border-blue-500/30 text-blue-400'
		},
		hr: {
			label: 'HR Attendance Officer',
			badgeClass: 'bg-amber-500/15 border-amber-500/30 text-amber-400'
		},
		employee: {
			label: 'Staff Member',
			badgeClass: 'bg-emerald-500/15 border-emerald-500/30 text-emerald-400'
		}
	};
</script>

<svelte:head>
	<title>{company?.name || 'Tenant'} Dashboard — 🍣 Sushi Clocks</title>
</svelte:head>

<div class="relative min-h-screen bg-[var(--bg)] text-[var(--text-main)] overflow-x-hidden">
	
	<!-- 2 Large Faded Logo Background Accents -->
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

	<!-- Top Navigation Bar (Full Width Sticky Anchored) -->
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
				<StatusBadge label="Tenant Isolated" dotColor="emerald" />
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
	<main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-6">
		
		<!-- Inspection Mode Banner for Super Admin -->
		{#if isSuperAdminInspecting}
			<div class="p-3 rounded-lg bg-purple-500/10 border border-purple-500/30 flex items-center justify-between gap-3 text-xs">
				<div class="flex items-center gap-2 text-purple-400">
					<Shield class="w-4 h-4 shrink-0" />
					<div>
						<span class="font-bold">Tenant Inspection Active:</span> Viewing <strong>{company.name}</strong> as Platform Super Admin.
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

		<!-- Organization Workspace Header -->
		<section class="p-5 rounded-lg bg-[var(--surface)] border border-[var(--border)] space-y-3">
			<div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
				<div class="space-y-1">
					<div class="flex flex-wrap items-center gap-2">
						{#if user}
							<span class="px-2 py-0.5 rounded text-[10px] font-mono font-medium border {roleDisplayMap[user.system_role]?.badgeClass}">
								{roleDisplayMap[user.system_role]?.label || user.system_role}
							</span>
						{/if}
						<span class="text-[10px] font-mono text-[var(--text-mute)]">
							Tenant UUID: {company.id}
						</span>
					</div>

					<h1 class="font-display text-xl sm:text-2xl font-bold tracking-tight text-[var(--text-main)]">
						{company.name} Workspace
					</h1>

					<div class="flex flex-wrap items-center gap-4 text-xs text-[var(--text-sub)] pt-0.5">
						<div class="flex items-center gap-1.5 font-mono">
							<Clock class="w-3.5 h-3.5 text-[var(--text-mute)]" />
							<span>Timezone: {company.timezone}</span>
						</div>
						<div class="flex items-center gap-1.5 font-mono">
							<DollarSign class="w-3.5 h-3.5 text-[var(--text-mute)]" />
							<span>Currency: {company.currency_code}</span>
						</div>
					</div>
				</div>

				<div class="p-3 rounded-lg bg-[var(--surface-raised)] border border-[var(--border)] flex items-center gap-2.5 text-xs">
					<div class="w-7 h-7 rounded bg-emerald-500/10 border border-emerald-500/30 flex items-center justify-center text-emerald-400">
						<CheckCircle2 class="w-3.5 h-3.5" />
					</div>
					<div>
						<div class="text-[10px] text-[var(--text-mute)] font-mono uppercase tracking-wider">RBAC SCOPE</div>
						<div class="font-bold text-xs text-[var(--text-main)]">Tenant-Isolated Boundary</div>
					</div>
				</div>
			</div>
		</section>

		<!-- ================================================================= -->
		<!-- 1. COMPANY ADMIN VIEW -->
		<!-- ================================================================= -->
		{#if isCompanyAdmin || isSuperAdminInspecting}
			<section class="space-y-3">
				<div class="flex items-center gap-2">
					<Shield class="w-4 h-4 text-[#f97040]" />
					<h2 class="text-sm font-bold tracking-tight">Company Administration Modules</h2>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
					<div class="p-3.5 sm:p-4 rounded-lg bg-[var(--surface)] border border-[var(--border)] space-y-2">
						<div class="w-7 h-7 rounded bg-blue-500/10 text-blue-400 flex items-center justify-center">
							<Users class="w-3.5 h-3.5" />
						</div>
						<div>
							<div class="text-xs font-semibold">User Roster</div>
							<div class="text-[11px] text-[var(--text-sub)] mt-0.5">Provision HR & Employee accounts.</div>
						</div>
					</div>

					<div class="p-3.5 sm:p-4 rounded-lg bg-[var(--surface)] border border-[var(--border)] space-y-2">
						<div class="w-7 h-7 rounded bg-emerald-500/10 text-emerald-400 flex items-center justify-center">
							<DollarSign class="w-3.5 h-3.5" />
						</div>
						<div>
							<div class="text-xs font-semibold">Wage & Roles Engine</div>
							<div class="text-[11px] text-[var(--text-sub)] mt-0.5">Role wages with cascading overrides.</div>
						</div>
					</div>

					<div class="p-3.5 sm:p-4 rounded-lg bg-[var(--surface)] border border-[var(--border)] space-y-2">
						<div class="w-7 h-7 rounded bg-amber-500/10 text-amber-400 flex items-center justify-center">
							<CalendarCheck class="w-3.5 h-3.5" />
						</div>
						<div>
							<div class="text-xs font-semibold">Custom Leave Rules</div>
							<div class="text-[11px] text-[var(--text-sub)] mt-0.5">Paid/unpaid categories & role max limits.</div>
						</div>
					</div>

					<div class="p-3.5 sm:p-4 rounded-lg bg-[var(--surface)] border border-[var(--border)] space-y-2">
						<div class="w-7 h-7 rounded bg-purple-500/10 text-purple-400 flex items-center justify-center">
							<FileSpreadsheet class="w-3.5 h-3.5" />
						</div>
						<div>
							<div class="text-xs font-semibold">Payroll Modifiers</div>
							<div class="text-[11px] text-[var(--text-sub)] mt-0.5">Late penalty rates, tax & bonus rules.</div>
						</div>
					</div>
				</div>
			</section>
		{/if}

		<!-- ================================================================= -->
		<!-- 2. HR MANAGER VIEW -->
		<!-- ================================================================= -->
		{#if isHR || isCompanyAdmin || isSuperAdminInspecting}
			<section class="space-y-3">
				<div class="flex items-center gap-2">
					<UserCheck class="w-4 h-4 text-amber-400" />
					<h2 class="text-sm font-bold tracking-tight">HR Attendance & Payroll Operations</h2>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-3 gap-3">
					<div class="p-3.5 sm:p-4 rounded-lg bg-[var(--surface)] border border-[var(--border)] space-y-2">
						<div class="w-7 h-7 rounded bg-emerald-500/10 text-emerald-400 flex items-center justify-center">
							<Clock class="w-3.5 h-3.5" />
						</div>
						<div>
							<div class="text-xs font-semibold">Live Presence Dashboard</div>
							<div class="text-[11px] text-[var(--text-sub)] mt-0.5">Zero-polling real-time presence channel.</div>
						</div>
					</div>

					<div class="p-3.5 sm:p-4 rounded-lg bg-[var(--surface)] border border-[var(--border)] space-y-2">
						<div class="w-7 h-7 rounded bg-amber-500/10 text-amber-400 flex items-center justify-center">
							<CalendarCheck class="w-3.5 h-3.5" />
						</div>
						<div>
							<div class="text-xs font-semibold">Leave Approvals Queue</div>
							<div class="text-[11px] text-[var(--text-sub)] mt-0.5">Approve and reject pending leave requests.</div>
						</div>
					</div>

					<div class="p-3.5 sm:p-4 rounded-lg bg-[var(--surface)] border border-[var(--border)] space-y-2">
						<div class="w-7 h-7 rounded bg-[#f97040]/10 text-[#f97040] flex items-center justify-center">
							<FileSpreadsheet class="w-3.5 h-3.5" />
						</div>
						<div>
							<div class="text-xs font-semibold">Generate Payroll Report</div>
							<div class="text-[11px] text-[var(--text-sub)] mt-0.5">Calculate net wages and stream CSV download.</div>
						</div>
					</div>
				</div>
			</section>
		{/if}

		<!-- ================================================================= -->
		<!-- 3. EMPLOYEE TIMECLOCK & LEAVE VIEW -->
		<!-- ================================================================= -->
		{#if isEmployee || isCompanyAdmin || isHR || isSuperAdminInspecting}
			<section class="space-y-3">
				<div class="flex items-center gap-2">
					<Clock class="w-4 h-4 text-emerald-400" />
					<h2 class="text-sm font-bold tracking-tight">Employee Self-Service Timeclock</h2>
				</div>

				<div class="grid grid-cols-1 md:grid-cols-2 gap-3">
					
					<!-- Punch In / Out Card -->
					<div class="p-4 rounded-lg bg-[var(--surface)] border border-[var(--border)] space-y-3">
						<div class="flex items-center justify-between">
							<div class="space-y-0.5">
								<h3 class="text-xs font-semibold">Shift Punch Authority</h3>
								<p class="text-[10px] text-[var(--text-mute)]">Server-authoritative timestamp logging</p>
							</div>
							<span class="px-2 py-0.5 rounded text-[10px] font-mono font-medium {isClockedIn ? 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30' : 'bg-neutral-500/15 text-neutral-400 border border-neutral-500/30'}">
								{isClockedIn ? 'ACTIVE SHIFT' : 'CLOCKED OUT'}
							</span>
						</div>

						<div class="p-3 rounded-lg bg-[var(--surface-raised)] border border-[var(--border)] space-y-0.5 text-center">
							<div class="text-[10px] text-[var(--text-mute)] font-mono">SERVER AUTHORITY TIME</div>
							<div class="font-display font-bold text-lg tracking-tight text-[#f97040]">
								{clockTime || 'Ready for punch'}
							</div>
						</div>

						<Button
							variant={isClockedIn ? 'outline' : 'primary'}
							class="w-full h-9 text-xs font-semibold {isClockedIn ? 'border-red-500/40 text-red-400 hover:bg-red-500/10' : ''}"
							onclick={toggleClock}
						>
							{#if isClockedIn}
								<Square class="w-3.5 h-3.5 mr-1" />
								<span>Clock Out Shift</span>
							{:else}
								<Play class="w-3.5 h-3.5 mr-1" />
								<span>Clock In (Capture IP & Telemetry)</span>
							{/if}
						</Button>
					</div>

					<!-- Leave Balance & Request Card -->
					<div class="p-4 rounded-lg bg-[var(--surface)] border border-[var(--border)] space-y-3">
						<div class="flex items-center justify-between">
							<div class="space-y-0.5">
								<h3 class="text-xs font-semibold">Leave Balances</h3>
								<p class="text-[10px] text-[var(--text-mute)]">Cascading role allowances</p>
							</div>
						</div>

						<div class="grid grid-cols-2 gap-2 text-xs">
							<div class="p-2.5 rounded-lg bg-[var(--surface-raised)] border border-[var(--border)] space-y-0.5">
								<span class="text-[10px] text-[var(--text-mute)] font-mono">Vacation</span>
								<div class="font-display text-sm font-bold text-emerald-400">12 Days</div>
							</div>
							<div class="p-2.5 rounded-lg bg-[var(--surface-raised)] border border-[var(--border)] space-y-0.5">
								<span class="text-[10px] text-[var(--text-mute)] font-mono">Sick</span>
								<div class="font-display text-sm font-bold text-blue-400">8 Days</div>
							</div>
						</div>

						<Button
							variant="secondary"
							class="w-full h-9 text-xs"
						>
							<Send class="w-3.5 h-3.5 mr-1.5 text-[#f97040]" />
							<span>Submit Leave Request</span>
						</Button>
					</div>

				</div>
			</section>
		{/if}

	</main>
</div>
