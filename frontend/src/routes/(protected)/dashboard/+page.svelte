<script lang="ts">
	import { page } from '$app/state';
	import { goto, invalidateAll } from '$app/navigation';
	import { logout } from '$lib/api/auth';
	import { createCompany, type CompanyWithStats } from '$lib/api/companies';
	import sushiLogo from '$lib/assets/sushi_logo_without_bg.png';
	import {
		Building2,
		Plus,
		Users,
		Globe,
		Coins,
		ExternalLink,
		Search,
		LogOut,
		X,
		AlertCircle,
		CheckCircle2,
		UploadCloud
	} from '@lucide/svelte';

	import Logo from '$lib/components/common/Logo.svelte';
	import ThemeToggle from '$lib/components/common/ThemeToggle.svelte';
	import StatusBadge from '$lib/components/common/StatusBadge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';

	let isLoggingOut = $state(false);
	let searchQuery = $state('');
	let showCreateModal = $state(false);
	let isCreating = $state(false);
	let createError = $state('');
	let createSuccess = $state('');

	// Create Company Form state
	let formName = $state('');
	let formCurrency = $state('USD');
	let formTimezone = $state('Asia/Manila');
	let formAdminFirstName = $state('');
	let formAdminLastName = $state('');
	let formAdminEmail = $state('');
	let formAdminPassword = $state('');

	const user = $derived(page.data.user);
	const companies = $derived((page.data.companies || []) as CompanyWithStats[]);

	const filteredCompanies = $derived(
		companies.filter(
			(c) =>
				c.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
				c.currency_code.toLowerCase().includes(searchQuery.toLowerCase()) ||
				(c.admin_email && c.admin_email.toLowerCase().includes(searchQuery.toLowerCase()))
		)
	);

	const totalEmployeesAcrossPlatform = $derived(
		companies.reduce((acc, c) => acc + (c.total_users || 0), 0)
	);

	const uniqueCurrencies = $derived(
		new Set(companies.map((c) => c.currency_code)).size
	);

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

	async function handleCreateCompany(e: Event) {
		e.preventDefault();
		createError = '';
		createSuccess = '';
		isCreating = true;

		try {
			await createCompany({
				name: formName,
				currency_code: formCurrency,
				timezone: formTimezone,
				admin_first_name: formAdminFirstName,
				admin_last_name: formAdminLastName,
				admin_email: formAdminEmail,
				admin_password: formAdminPassword
			});

			createSuccess = `Organization "${formName}" provisioned!`;
			formName = '';
			formAdminFirstName = '';
			formAdminLastName = '';
			formAdminEmail = '';
			formAdminPassword = '';

			await invalidateAll();

			setTimeout(() => {
				showCreateModal = false;
				createSuccess = '';
			}, 900);
		} catch (err: unknown) {
			if (err instanceof Error) {
				createError = err.message;
			} else {
				createError = 'Failed to provision company.';
			}
		} finally {
			isCreating = false;
		}
	}
</script>

<svelte:head>
	<title>Platform Super Admin — 🍣 Sushi Clocks</title>
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
				<Logo size="sm" />
				<div class="hidden sm:block h-4 w-px bg-[var(--border)]"></div>
				<div class="hidden sm:flex items-center gap-2">
					<span class="px-2 py-0.5 rounded text-[10px] font-mono font-medium bg-purple-500/15 border border-purple-500/30 text-purple-400">
						SUPER ADMIN
					</span>
					<span class="text-xs text-[var(--text-mute)] font-mono">Platform Control Hub</span>
				</div>
			</div>

			<div class="flex items-center gap-2.5">
				<StatusBadge label="Aiven DB Live" dotColor="emerald" />
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
		
		<!-- Workspace Header Panel (Flattened, no floating island) -->
		<section class="p-5 rounded-lg bg-[var(--surface)] border border-[var(--border)]">
			<div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
				<div class="space-y-1">
					<h1 class="font-display text-xl sm:text-2xl font-bold tracking-tight text-[var(--text-main)]">
						Multi-Tenant Organizations Hub
					</h1>
					<p class="text-xs sm:text-sm text-[var(--text-sub)] leading-relaxed max-w-2xl">
						Manage platform organizations, provision initial tenant administrators, and oversee tenant isolation across PostgreSQL & MongoDB telemetry.
					</p>
				</div>

				<div class="shrink-0">
					<Button
						variant="primary"
						class="h-9 px-4 text-xs font-semibold"
						onclick={() => (showCreateModal = true)}
					>
						<Plus class="w-3.5 h-3.5 mr-1" />
						<span>Provision Organization</span>
					</Button>
				</div>
			</div>
		</section>

		<!-- Dense Metrics Grid -->
		<section class="grid grid-cols-2 lg:grid-cols-4 gap-3">
			
			<div class="p-3.5 sm:p-4 rounded-lg bg-[var(--surface)] border border-[var(--border)] space-y-1.5">
				<div class="flex items-center justify-between">
					<span class="text-[0.65rem] font-mono tracking-widest uppercase text-[var(--text-mute)]">TOTAL TENANTS</span>
					<div class="w-6 h-6 rounded bg-[#f97040]/10 text-[#f97040] flex items-center justify-center">
						<Building2 class="w-3 h-3" />
					</div>
				</div>
				<div class="font-display text-xl font-bold tracking-tight">{companies.length}</div>
				<div class="text-[11px] text-[var(--text-sub)]">Provisioned Organizations</div>
			</div>

			<div class="p-3.5 sm:p-4 rounded-lg bg-[var(--surface)] border border-[var(--border)] space-y-1.5">
				<div class="flex items-center justify-between">
					<span class="text-[0.65rem] font-mono tracking-widest uppercase text-[var(--text-mute)]">TOTAL USERS</span>
					<div class="w-6 h-6 rounded bg-blue-500/10 text-blue-400 flex items-center justify-center">
						<Users class="w-3 h-3" />
					</div>
				</div>
				<div class="font-display text-xl font-bold tracking-tight">{totalEmployeesAcrossPlatform}</div>
				<div class="text-[11px] text-[var(--text-sub)]">Users across all tenants</div>
			</div>

			<div class="p-3.5 sm:p-4 rounded-lg bg-[var(--surface)] border border-[var(--border)] space-y-1.5">
				<div class="flex items-center justify-between">
					<span class="text-[0.65rem] font-mono tracking-widest uppercase text-[var(--text-mute)]">CURRENCIES</span>
					<div class="w-6 h-6 rounded bg-emerald-500/10 text-emerald-400 flex items-center justify-center">
						<Coins class="w-3 h-3" />
					</div>
				</div>
				<div class="font-display text-xl font-bold tracking-tight">{uniqueCurrencies || 1} Active</div>
				<div class="text-[11px] text-[var(--text-sub)]">Multi-currency engine</div>
			</div>

			<div class="p-3.5 sm:p-4 rounded-lg bg-[var(--surface)] border border-[var(--border)] space-y-1.5">
				<div class="flex items-center justify-between">
					<span class="text-[0.65rem] font-mono tracking-widest uppercase text-[var(--text-mute)]">ARCHIVAL ENGINE</span>
					<div class="w-6 h-6 rounded bg-purple-500/10 text-purple-400 flex items-center justify-center">
						<UploadCloud class="w-3 h-3" />
					</div>
				</div>
				<div class="font-display text-sm font-bold text-emerald-400">Timezone-Aware</div>
				<div class="text-[11px] text-[var(--text-sub)]">Monthly Cloudflare R2 archival</div>
			</div>

		</section>

		<!-- Organizations Section (Dense Data Layout) -->
		<section class="space-y-3">
			<div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2.5">
				<div class="flex items-center gap-2">
					<Building2 class="w-4 h-4 text-[#f97040]" />
					<h2 class="text-sm font-bold tracking-tight">Active Organizations</h2>
					<span class="px-1.5 py-0.2 rounded text-[10px] font-mono bg-[var(--surface-raised)] border border-[var(--border)] text-[var(--text-mute)]">
						{filteredCompanies.length}
					</span>
				</div>

				<!-- Search Input -->
				<div class="relative w-full sm:w-64">
					<Search class="w-3.5 h-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-[var(--text-mute)]" />
					<input
						type="text"
						placeholder="Search organizations..."
						bind:value={searchQuery}
						class="w-full h-8.5 pl-8 pr-3 rounded-lg bg-[var(--surface)] border border-[var(--border)] text-xs text-[var(--text-main)] placeholder-[var(--text-mute)] focus:outline-none focus:border-[#f97040] transition-colors"
					/>
				</div>
			</div>

			{#if filteredCompanies.length === 0}
				<div class="p-8 text-center rounded-lg bg-[var(--surface)] border border-[var(--border)] space-y-3">
					<div class="w-9 h-9 mx-auto rounded bg-[var(--surface-raised)] border border-[var(--border)] text-[var(--text-mute)] flex items-center justify-center">
						<Building2 class="w-4 h-4" />
					</div>
					<div class="space-y-0.5">
						<h3 class="text-sm font-semibold">No organizations found</h3>
						<p class="text-xs text-[var(--text-sub)] max-w-sm mx-auto">
							{searchQuery ? 'No organizations matched your search filter.' : 'Provision your first tenant organization with an initial administrator account to get started.'}
						</p>
					</div>
					{#if !searchQuery}
						<Button
							variant="primary"
							class="h-8.5 text-xs px-3.5"
							onclick={() => (showCreateModal = true)}
						>
							<Plus class="w-3 h-3 mr-1" />
							<span>Provision First Organization</span>
						</Button>
					{/if}
				</div>
			{:else}
				<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
					{#each filteredCompanies as company (company.id)}
						<div class="p-4 rounded-lg bg-[var(--surface)] border border-[var(--border)] hover:border-[#f97040]/30 transition-all duration-100 flex flex-col justify-between space-y-3 group">
							<div class="space-y-2">
								<div class="flex items-start justify-between gap-2">
									<div class="space-y-0.5 min-w-0">
										<h3 class="font-display font-bold text-sm text-[var(--text-main)] group-hover:text-[#f97040] transition-colors truncate">
											{company.name}
										</h3>
										<span class="text-[10px] font-mono text-[var(--text-mute)] block truncate">
											{company.id}
										</span>
									</div>

									<span class="px-1.5 py-0.5 rounded text-[10px] font-mono font-semibold bg-[#f97040]/10 border border-[#f97040]/30 text-[#f97040] shrink-0">
										{company.currency_code}
									</span>
								</div>

								<div class="space-y-0.5 text-xs text-[var(--text-sub)] pt-1.5 border-t border-[var(--border)]">
									<div class="flex items-center justify-between text-[11px]">
										<span class="text-[var(--text-mute)]">Timezone:</span>
										<span class="font-mono">{company.timezone}</span>
									</div>
									<div class="flex items-center justify-between text-[11px]">
										<span class="text-[var(--text-mute)]">Staff:</span>
										<span class="font-mono font-medium text-emerald-400">{company.total_users} user{company.total_users === 1 ? '' : 's'}</span>
									</div>
									{#if company.admin_email}
										<div class="flex items-center justify-between text-[11px]">
											<span class="text-[var(--text-mute)]">Admin:</span>
											<span class="font-mono truncate max-w-[140px]">{company.admin_email}</span>
										</div>
									{/if}
								</div>
							</div>

							<div class="pt-1.5 border-t border-[var(--border)]">
								<Button
									variant="secondary"
									class="w-full h-7.5 text-xs justify-between group-hover:bg-[#f97040]/10 group-hover:border-[#f97040]/30 transition-colors"
									onclick={() => goto(`/dashboard/${company.id}`)}
								>
									<span>Inspect Tenant</span>
									<ExternalLink class="w-3 h-3 text-[#f97040]" />
								</Button>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</section>
	</main>

	<!-- PROVISION ORGANIZATION MODAL (Sharp flat panel) -->
	{#if showCreateModal}
		<div
			class="fixed inset-0 z-50 bg-black/70 backdrop-blur-xs flex items-center justify-center p-4 overflow-y-auto"
		>
			<div class="relative w-full max-w-md rounded-lg bg-[var(--surface)] border border-[var(--border)] shadow-xl p-5 space-y-3.5 my-6">
				
				<!-- Modal Header -->
				<div class="flex items-center justify-between border-b border-[var(--border)] pb-2.5">
					<div class="space-y-0.5">
						<h3 class="font-display text-base font-bold tracking-tight text-[var(--text-main)]">
							Provision New Organization
						</h3>
						<p class="text-[11px] text-[var(--text-sub)]">
							Configure company settings and initial administrator.
						</p>
					</div>
					<button
						type="button"
						class="p-1 rounded text-[var(--text-mute)] hover:text-[var(--text-main)] hover:bg-[var(--surface-raised)] transition-colors"
						onclick={() => (showCreateModal = false)}
					>
						<X class="w-4 h-4" />
					</button>
				</div>

				{#if createError}
					<div class="p-2.5 rounded bg-red-500/10 border border-red-500/30 flex items-start gap-2 text-xs text-red-400">
						<AlertCircle class="w-3.5 h-3.5 shrink-0 mt-0.5" />
						<span>{createError}</span>
					</div>
				{/if}

				{#if createSuccess}
					<div class="p-2.5 rounded bg-emerald-500/10 border border-emerald-500/30 flex items-start gap-2 text-xs text-emerald-400">
						<CheckCircle2 class="w-3.5 h-3.5 shrink-0 mt-0.5" />
						<span>{createSuccess}</span>
					</div>
				{/if}

				<!-- Modal Form -->
				<form onsubmit={handleCreateCompany} class="space-y-3">
					
					<!-- Organization Details -->
					<div class="space-y-2">
						<span class="text-[10px] font-mono font-semibold uppercase tracking-widest text-[#f97040]">
							1. Organization Profile
						</span>

						<Input
							id="compName"
							label="Organization Name"
							bind:value={formName}
							placeholder="e.g. Acme Corporation"
							required
						/>

						<div class="grid grid-cols-2 gap-2">
							<div class="space-y-1">
								<label for="currencySelect" class="block text-xs font-mono uppercase tracking-wider text-[var(--text-sub)]">
									Currency
								</label>
								<select
									id="currencySelect"
									bind:value={formCurrency}
									class="w-full h-10 px-3 rounded-lg bg-[var(--surface-raised)] border border-[var(--border)] text-[var(--text-main)] text-xs focus:outline-none focus:border-[#f97040] transition-colors"
								>
									<option value="USD">USD ($)</option>
									<option value="PHP">PHP (₱)</option>
									<option value="EUR">EUR (€)</option>
									<option value="GBP">GBP (£)</option>
									<option value="JPY">JPY (¥)</option>
									<option value="CAD">CAD ($)</option>
									<option value="AUD">AUD ($)</option>
								</select>
							</div>

							<div class="space-y-1">
								<label for="timezoneSelect" class="block text-xs font-mono uppercase tracking-wider text-[var(--text-sub)]">
									Timezone
								</label>
								<select
									id="timezoneSelect"
									bind:value={formTimezone}
									class="w-full h-10 px-3 rounded-lg bg-[var(--surface-raised)] border border-[var(--border)] text-[var(--text-main)] text-xs focus:outline-none focus:border-[#f97040] transition-colors"
								>
									<option value="Asia/Manila">Asia/Manila (UTC+8)</option>
									<option value="UTC">UTC (UTC+0)</option>
									<option value="America/New_York">America/New_York (EST)</option>
									<option value="America/Los_Angeles">America/Los_Angeles (PST)</option>
									<option value="Europe/London">Europe/London (GMT)</option>
									<option value="Asia/Tokyo">Asia/Tokyo (UTC+9)</option>
								</select>
							</div>
						</div>
					</div>

					<!-- Initial Company Admin -->
					<div class="space-y-2 pt-1.5 border-t border-[var(--border)]">
						<span class="text-[10px] font-mono font-semibold uppercase tracking-widest text-[#f97040]">
							2. Initial Company Admin
						</span>

						<div class="grid grid-cols-2 gap-2">
							<Input
								id="adminFirst"
								label="First Name"
								bind:value={formAdminFirstName}
								placeholder="Jane"
								required
							/>
							<Input
								id="adminLast"
								label="Last Name"
								bind:value={formAdminLastName}
								placeholder="Doe"
								required
							/>
						</div>

						<Input
							id="adminEmail"
							type="email"
							label="Admin Email"
							bind:value={formAdminEmail}
							placeholder="admin@acmecorp.com"
							required
						/>

						<Input
							id="adminPass"
							type="password"
							label="Admin Password"
							bind:value={formAdminPassword}
							placeholder="Minimum 6 characters"
							required
						/>
					</div>

					<!-- Modal Actions -->
					<div class="flex items-center justify-end gap-2 pt-2.5 border-t border-[var(--border)]">
						<Button
							type="button"
							variant="ghost"
							disabled={isCreating}
							onclick={() => (showCreateModal = false)}
							class="h-8 text-xs"
						>
							Cancel
						</Button>
						<Button
							type="submit"
							variant="primary"
							loading={isCreating}
							class="h-8 px-3.5 text-xs font-semibold"
						>
							<span>Provision Organization</span>
						</Button>
					</div>
				</form>
			</div>
		</div>
	{/if}
</div>
