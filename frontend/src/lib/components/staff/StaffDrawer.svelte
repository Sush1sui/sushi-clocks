<script lang="ts">
	import { X, UserPlus, Users, Mail, Phone, Shield, ChevronDown, AlertCircle, CheckCircle, Loader2 } from '@lucide/svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import { getCompanyUsers, createCompanyUser } from '$lib/api/users';
	import type { UserResponse, CreateStaffPayload } from '$lib/api/users';
	import { APIError } from '$lib/api/client';

	let {
		open = $bindable(false),
		companyId,
		role
	}: {
		open: boolean;
		companyId: string;
		role: string;
	} = $props();

	type DrawerTab = 'list' | 'add';

	let activeTab = $state<DrawerTab>('list');
	let users = $state<UserResponse[]>([]);
	let listLoading = $state(false);
	let listError = $state('');

	// Add Staff form state
	let firstName = $state('');
	let lastName = $state('');
	let email = $state('');
	let password = $state('');
	let mobileNumber = $state('');
	let staffRole = $state<'hr' | 'employee'>('employee');
	let formLoading = $state(false);
	let formError = $state('');
	let formSuccess = $state('');

	const isAdmin = $derived(role === 'admin');

	// Load users whenever drawer opens
	$effect(() => {
		if (open) {
			loadUsers();
		}
	});

	// Escape key closes the drawer
	function handleKeyDown(e: KeyboardEvent) {
		if (e.key === 'Escape') open = false;
	}

	async function loadUsers() {
		listLoading = true;
		listError = '';
		try {
			users = await getCompanyUsers(companyId);
		} catch (err) {
			listError = err instanceof APIError ? err.message : 'Failed to load staff list.';
		} finally {
			listLoading = false;
		}
	}

	function resetForm() {
		firstName = '';
		lastName = '';
		email = '';
		password = '';
		mobileNumber = '';
		staffRole = 'employee';
		formError = '';
		formSuccess = '';
	}

	async function handleAddStaff(e: Event) {
		e.preventDefault();
		formError = '';
		formSuccess = '';
		formLoading = true;

		const payload: CreateStaffPayload = {
			first_name: firstName,
			last_name: lastName,
			email,
			password,
			system_role: staffRole,
			...(mobileNumber.trim() ? { mobile_number: mobileNumber.trim() } : {})
		};

		try {
			const newUser = await createCompanyUser(companyId, payload);
			formSuccess = `${newUser.first_name} ${newUser.last_name} added successfully.`;
			// Refresh list and return to it after short delay
			await loadUsers();
			resetForm();
			setTimeout(() => {
				formSuccess = '';
				activeTab = 'list';
			}, 1800);
		} catch (err) {
			formError = err instanceof APIError ? err.message : 'Failed to create staff account.';
		} finally {
			formLoading = false;
		}
	}

	function getRoleBadge(r: string): { label: string; cls: string } {
		switch (r) {
			case 'admin':
				return { label: 'Admin', cls: 'bg-[#f97040]/15 text-[#f97040] border-[#f97040]/25' };
			case 'hr':
				return { label: 'HR', cls: 'bg-purple-500/15 text-purple-400 border-purple-500/25' };
			default:
				return { label: 'Employee', cls: 'bg-blue-500/15 text-blue-400 border-blue-500/25' };
		}
	}

	function formatDate(iso: string) {
		return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
	}
</script>

<svelte:window onkeydown={handleKeyDown} />

<!-- Backdrop -->
{#if open}
	<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 z-40 bg-black/50 backdrop-blur-[2px] transition-opacity"
		onclick={() => (open = false)}
		aria-hidden="true"
	></div>
{/if}

<!-- Drawer Panel -->
<div
	id="staff-drawer"
	role="dialog"
	aria-modal="true"
	aria-label="Staff Management"
	class="fixed top-0 right-0 z-50 h-full w-full max-w-[480px] flex flex-col bg-[var(--bg)] border-l border-[var(--border)] shadow-2xl transition-transform duration-300 ease-out"
	class:translate-x-0={open}
	class:translate-x-full={!open}
>
	<!-- Header -->
	<div class="flex items-center justify-between px-5 py-4 border-b border-[var(--border)] shrink-0">
		<div class="flex items-center gap-2.5">
			<div class="w-8 h-8 rounded-lg bg-blue-500/10 text-blue-400 flex items-center justify-center">
				<Users class="w-4 h-4" />
			</div>
			<div>
				<h2 class="text-sm font-bold text-[var(--text-main)] tracking-tight">Users & Staff</h2>
				<p class="text-[11px] text-[var(--text-mute)] font-mono">Manage employee accounts</p>
			</div>
		</div>
		<button
			type="button"
			onclick={() => (open = false)}
			class="w-8 h-8 rounded-lg flex items-center justify-center text-[var(--text-mute)] hover:text-[var(--text-main)] hover:bg-[var(--surface)] transition-colors"
			aria-label="Close staff drawer"
		>
			<X class="w-4 h-4" />
		</button>
	</div>

	<!-- Tabs (Admin only sees Add Staff tab) -->
	<div class="flex border-b border-[var(--border)] shrink-0">
		<button
			type="button"
			onclick={() => (activeTab = 'list')}
			class="flex-1 px-4 py-2.5 text-xs font-semibold font-mono transition-colors border-b-2 {activeTab === 'list' ? 'text-[#f97040] border-[#f97040]' : 'text-[var(--text-sub)] border-transparent hover:text-[var(--text-main)]'}"
		>
			Staff List
		</button>
		{#if isAdmin}
			<button
				type="button"
				onclick={() => (activeTab = 'add')}
				class="flex-1 px-4 py-2.5 text-xs font-semibold font-mono transition-colors border-b-2 {activeTab === 'add' ? 'text-[#f97040] border-[#f97040]' : 'text-[var(--text-sub)] border-transparent hover:text-[var(--text-main)]'}"
			>
				Add Staff
			</button>
		{/if}
	</div>

	<!-- Tab Content -->
	<div class="flex-1 overflow-y-auto">

		<!-- ── Staff List Tab ── -->
		{#if activeTab === 'list'}
			<div class="p-4 space-y-2">
				{#if listLoading}
					<!-- Skeleton rows -->
					{#each Array(4) as _}
						<div class="p-3 rounded-lg bg-[var(--surface)] border border-[var(--border)] animate-pulse flex items-center gap-3">
							<div class="w-8 h-8 rounded-full bg-[var(--surface-raised)]"></div>
							<div class="flex-1 space-y-1.5">
								<div class="h-2.5 rounded bg-[var(--surface-raised)] w-32"></div>
								<div class="h-2 rounded bg-[var(--surface-raised)] w-44"></div>
							</div>
						</div>
					{/each}
				{:else if listError}
					<div class="p-3 rounded-lg bg-red-500/10 border border-red-500/30 flex items-start gap-2.5 text-xs text-red-400">
						<AlertCircle class="w-4 h-4 shrink-0 mt-0.5" />
						<span>{listError}</span>
					</div>
				{:else if users.length === 0}
					<div class="py-12 text-center space-y-3">
						<div class="w-12 h-12 mx-auto rounded-xl bg-[var(--surface)] flex items-center justify-center">
							<Users class="w-5 h-5 text-[var(--text-mute)]" />
						</div>
						<div>
							<p class="text-sm font-medium text-[var(--text-main)]">No staff yet</p>
							<p class="text-xs text-[var(--text-sub)] mt-1">
								{isAdmin ? 'Use the Add Staff tab to onboard your first employee.' : 'No staff members have been added yet.'}
							</p>
						</div>
					</div>
				{:else}
					<!-- Count chip -->
					<div class="flex items-center justify-between mb-3">
						<span class="text-[11px] font-mono text-[var(--text-mute)]">{users.length} member{users.length !== 1 ? 's' : ''}</span>
						<button
							type="button"
							onclick={loadUsers}
							class="text-[11px] font-mono text-[var(--text-mute)] hover:text-[#f97040] transition-colors"
						>
							Refresh
						</button>
					</div>

					{#each users as user (user.id)}
						{@const badge = getRoleBadge(user.system_role)}
						<div class="p-3 rounded-lg bg-[var(--surface)] border border-[var(--border)] flex items-start justify-between gap-3 hover:border-[var(--border)] transition-colors">
							<!-- Avatar + info -->
							<div class="flex items-start gap-3 min-w-0">
								<div class="w-8 h-8 rounded-full bg-[#f97040]/15 text-[#f97040] flex items-center justify-center shrink-0 text-xs font-bold uppercase">
									{user.first_name[0]}{user.last_name[0]}
								</div>
								<div class="min-w-0">
									<p class="text-xs font-semibold text-[var(--text-main)] truncate">
										{user.first_name} {user.last_name}
									</p>
									<div class="flex items-center gap-1 mt-0.5">
										<Mail class="w-3 h-3 text-[var(--text-mute)] shrink-0" />
										<span class="text-[11px] text-[var(--text-sub)] truncate">{user.email}</span>
									</div>
									{#if user.mobile_number}
										<div class="flex items-center gap-1 mt-0.5">
											<Phone class="w-3 h-3 text-[var(--text-mute)] shrink-0" />
											<span class="text-[11px] text-[var(--text-sub)]">{user.mobile_number}</span>
										</div>
									{/if}
									<p class="text-[10px] text-[var(--text-mute)] mt-1 font-mono">Joined {formatDate(user.created_at)}</p>
								</div>
							</div>
							<!-- Role badge -->
							<span class="shrink-0 inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-mono font-medium border {badge.cls}">
								{badge.label}
							</span>
						</div>
					{/each}
				{/if}
			</div>

		<!-- ── Add Staff Tab (Admin only) ── -->
		{:else if activeTab === 'add' && isAdmin}
			<form onsubmit={handleAddStaff} class="p-5 space-y-4">

				{#if formSuccess}
					<div class="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/30 flex items-start gap-2.5 text-xs text-emerald-400">
						<CheckCircle class="w-4 h-4 shrink-0 mt-0.5" />
						<span>{formSuccess}</span>
					</div>
				{/if}

				{#if formError}
					<div class="p-3 rounded-lg bg-red-500/10 border border-red-500/30 flex items-start gap-2.5 text-xs text-red-400">
						<AlertCircle class="w-4 h-4 shrink-0 mt-0.5" />
						<span>{formError}</span>
					</div>
				{/if}

				<div class="grid grid-cols-2 gap-3">
					<Input
						id="staff-first-name"
						label="First Name"
						bind:value={firstName}
						placeholder="Jane"
						required
					/>
					<Input
						id="staff-last-name"
						label="Last Name"
						bind:value={lastName}
						placeholder="Doe"
						required
					/>
				</div>

				<Input
					id="staff-email"
					type="email"
					label="Work Email"
					bind:value={email}
					placeholder="jane@company.com"
					required
				/>

				<Input
					id="staff-password"
					type="password"
					label="Temporary Password"
					bind:value={password}
					placeholder="Min. 6 characters"
					required
				/>

				<Input
					id="staff-mobile"
					type="tel"
					label="Mobile Number (Optional)"
					bind:value={mobileNumber}
					placeholder="+63 912 345 6789"
				/>

				<!-- Role Selector -->
				<div class="space-y-1.5">
					<label for="staff-role" class="block text-xs font-mono uppercase tracking-wider text-[var(--text-sub)]">
						Role
					</label>
					<div class="relative">
						<select
							id="staff-role"
							bind:value={staffRole}
							class="w-full h-10 px-3.5 pr-8 rounded-lg bg-[var(--surface-raised)] border border-[var(--border)] text-[var(--text-main)] text-xs sm:text-sm focus:outline-none focus:border-[#f97040] focus:ring-1 focus:ring-[#f97040]/30 transition-colors appearance-none cursor-pointer"
						>
							<option value="employee">Employee</option>
							<option value="hr">HR Manager</option>
						</select>
						<ChevronDown class="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-[var(--text-mute)]" />
					</div>
					<p class="text-[11px] text-[var(--text-mute)] font-mono">
						<Shield class="inline w-3 h-3 mr-0.5 -mt-0.5" />Admin role cannot be assigned here.
					</p>
				</div>

				<div class="pt-2 flex gap-2.5">
					<button
						type="button"
						onclick={resetForm}
						class="flex-1 h-10 rounded-lg border border-[var(--border)] text-xs font-semibold text-[var(--text-sub)] hover:text-[var(--text-main)] hover:bg-[var(--surface)] transition-colors"
					>
						Clear
					</button>
					<Button
						type="submit"
						variant="primary"
						loading={formLoading}
						class="flex-1 h-10 text-xs font-semibold"
					>
						<UserPlus class="w-3.5 h-3.5 mr-1.5" />
						Add Staff Member
					</Button>
				</div>

				<div class="text-[11px] text-[var(--text-mute)] font-mono text-center pt-1">
					Account will receive credentials from the company admin.
				</div>
			</form>
		{/if}
	</div>
</div>
