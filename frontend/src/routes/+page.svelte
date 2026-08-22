<script lang="ts">
	import { page } from '$app/state';
	import { goto, invalidateAll } from '$app/navigation';
	import { login } from '$lib/api/auth';
	import { ArrowRight, Shield, Zap, Users, Sparkles, AlertCircle } from '@lucide/svelte';
	import sushiLogo from '$lib/assets/sushi_logo_without_bg.png';

	// Reusable Components
	import Logo from '$lib/components/common/Logo.svelte';
	import ThemeToggle from '$lib/components/common/ThemeToggle.svelte';
	import StatusBadge from '$lib/components/common/StatusBadge.svelte';
	import FeatureCard from '$lib/components/common/FeatureCard.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Button from '$lib/components/ui/Button.svelte';

	let email = $state('');
	let password = $state('');
	let rememberPassword = $state(false);
	let isLoading = $state(false);
	let errorMessage = $state('');

	const user = $derived(page.data.user);

	$effect(() => {
		if (typeof window !== 'undefined') {
			const savedEmail = localStorage.getItem('sushi_remember_email');
			const savedPass = localStorage.getItem('sushi_remember_password');
			if (savedEmail && savedPass) {
				email = savedEmail;
				password = savedPass;
				rememberPassword = true;
			}
		}
	});

	async function handleSignIn(e: Event) {
		e.preventDefault();
		errorMessage = '';
		isLoading = true;

		try {
			if (typeof window !== 'undefined') {
				if (rememberPassword) {
					localStorage.setItem('sushi_remember_email', email);
					localStorage.setItem('sushi_remember_password', password);
				} else {
					localStorage.removeItem('sushi_remember_email');
					localStorage.removeItem('sushi_remember_password');
				}
			}
			await login(email, password);
			await invalidateAll();
			await goto('/dashboard');
		} catch (err: unknown) {
			if (err instanceof Error) {
				errorMessage = err.message;
			} else {
				errorMessage = 'An unexpected error occurred. Please try again.';
			}
		} finally {
			isLoading = false;
		}
	}
</script>

<svelte:head>
	<title>🍣 Sushi Clocks — Clock In/Out & Payroll Reports</title>
	<meta
		name="description"
		content="Multi-tenant clock-in/out attendance engine with integrated payroll report generation."
	/>
</svelte:head>

<div class="min-h-screen w-full flex flex-col md:flex-row bg-[var(--bg)] text-[var(--text-main)] selection:bg-[#f97040] selection:text-[#0c0d14] transition-colors duration-150">
	
	<!-- LEFT PANEL — Editorial Brand & Feature Showcase -->
	<section
		aria-label="Brand Overview"
		class="relative flex-1 md:w-[54%] lg:w-[58%] flex flex-col justify-between p-8 sm:p-12 lg:p-16 border-b md:border-b-0 md:border-r border-[var(--border)] overflow-hidden"
	>
		<!-- 2 Large Faded Logo Background Accents -->
		<div
			class="pointer-events-none absolute top-8 -right-20 w-[420px] h-[420px] watermark-logo rotate-[15deg] select-none"
			aria-hidden="true"
		>
			<img src={sushiLogo} alt="" class="w-full h-full object-contain" />
		</div>
		<div
			class="pointer-events-none absolute -bottom-20 -left-12 w-[460px] h-[460px] watermark-logo rotate-[-12deg] select-none"
			aria-hidden="true"
		>
			<img src={sushiLogo} alt="" class="w-full h-full object-contain" />
		</div>

		<!-- Brand Header -->
		<header class="relative z-10 flex items-center justify-between">
			<Logo size="md" />

			<div class="flex items-center gap-2.5">
				<StatusBadge label="WebSocket Active" dotColor="emerald" />
				<ThemeToggle />
			</div>
		</header>

		<!-- Main Left Content -->
		<div class="relative z-10 my-10 md:my-auto max-w-xl space-y-7">
			<!-- Mascot & Eyebrow Badge -->
			<div class="flex items-center gap-4">
				<div class="relative w-18 h-18 sm:w-20 sm:h-20 rounded-xl bg-[var(--surface)] border border-[#f97040]/25 p-2 flex items-center justify-center shrink-0 shadow-sm">
					<img
						src={sushiLogo}
						alt="Sushi Timekeeper Mascot"
						class="w-full h-full object-contain animate-float select-none"
					/>
				</div>
				<div class="space-y-1">
					<div class="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-md bg-[#f97040]/10 border border-[#f97040]/25 text-[#f97040] text-xs font-mono font-medium">
						<Sparkles class="w-3.5 h-3.5" />
						<span>Time & Attendance</span>
					</div>
					<div class="text-xs text-[var(--text-sub)]">Accurate attendance and payroll management</div>
				</div>
			</div>

			<!-- Flat Editorial Headline -->
			<div class="space-y-3">
				<h1 class="font-display text-4xl sm:text-5xl lg:text-6xl font-extrabold tracking-tight leading-[1.08]">
					<span class="text-[var(--text-main)] block">Clock in. Clock out.</span>
					<span class="text-[#f97040] block">Payroll done right.</span>
				</h1>
				<p class="text-[var(--text-sub)] text-sm sm:text-base leading-relaxed max-w-lg">
					Real-time attendance tracking for your entire workforce. Record accurate shifts, deduct approved leave, and export payroll CSV reports in seconds.
				</p>
			</div>

			<!-- Clean Reusable Feature Cards (Sharp rounded-lg) -->
			<div class="grid grid-cols-1 sm:grid-cols-3 gap-3 pt-2">
				<FeatureCard
					icon={Zap}
					title="Live Attendance"
					description="Real-time employee activity"
				/>
				<FeatureCard
					icon={Shield}
					title="Payroll Reports"
					description="1-click wage CSV export"
				/>
				<FeatureCard
					icon={Users}
					title="Leave Tracking"
					description="Automatic balance tracking"
				/>
			</div>
		</div>

		<!-- Left Footer -->
		<footer class="relative z-10 flex items-center justify-between text-xs text-[var(--text-mute)] pt-4 font-mono">
			<span>SECURE & ENCRYPTED WORKSPACE</span>
			<span>GO + SVELTEKIT</span>
		</footer>
	</section>

	<!-- RIGHT PANEL — Sign In Card Area -->
	<section
		aria-label="Sign In Section"
		class="relative flex-1 md:w-[46%] lg:w-[42%] bg-[var(--bg)] flex flex-col justify-center items-center p-6 sm:p-10 lg:p-12 overflow-hidden transition-colors duration-150"
	>
		<div class="w-full max-w-md space-y-6">
			
			<!-- Mobile Brand Header & Theme Toggle (Visible only on small screens) -->
			<div class="md:hidden flex items-center justify-between mb-2">
				<Logo size="sm" />
				<ThemeToggle />
			</div>

			<!-- Already Signed In Prompt -->
			{#if user}
				<div class="p-6 rounded-xl bg-[var(--surface)] border border-[#f97040]/30 shadow-lg space-y-4 text-center">
					<div class="w-11 h-11 mx-auto rounded-lg bg-[#f97040]/15 text-[#f97040] flex items-center justify-center">
						<Sparkles class="w-5 h-5" />
					</div>
					<div>
						<h2 class="font-display text-lg font-bold text-[var(--text-main)]">Already Signed In</h2>
						<p class="text-xs text-[var(--text-sub)] mt-1 font-mono">
							Logged in as {user.email} ({user.system_role})
						</p>
					</div>
					<Button
						variant="primary"
						class="w-full h-10 text-sm"
						onclick={() => goto('/dashboard')}
					>
						<span>Go to Dashboard</span>
						<ArrowRight class="w-4 h-4 ml-1" />
					</Button>
				</div>
			{:else}
				<!-- Sign In Header -->
				<div class="space-y-1">
					<h2 class="font-display text-2xl sm:text-3xl font-bold text-[var(--text-main)] tracking-tight">Sign In</h2>
					<p class="text-xs sm:text-sm text-[var(--text-sub)]">
						Enter your organization credentials to open your dashboard.
					</p>
				</div>

				<!-- Card Container (Sharp rounded-xl, flat surface) -->
				<div class="p-6 sm:p-7 rounded-xl bg-[var(--surface)] border border-[var(--border)] shadow-md space-y-5">
					
					{#if errorMessage}
						<div class="p-3 rounded-lg bg-red-500/10 border border-red-500/30 flex items-start gap-2.5 text-xs text-red-400">
							<AlertCircle class="w-4 h-4 shrink-0 mt-0.5" />
							<span>{errorMessage}</span>
						</div>
					{/if}

					<form onsubmit={handleSignIn} class="space-y-4">
						
						<!-- Reusable Work Email Input -->
						<Input
							id="email"
							type="email"
							label="Work Email"
							bind:value={email}
							placeholder="admin@sushi.clocks"
							required
						/>

						<!-- Reusable Password Input -->
						<div class="space-y-1.5">
							<div class="flex items-center justify-between">
								<span class="block text-xs font-mono uppercase tracking-wider text-[var(--text-sub)]">
									Password
								</span>
								<a href="#forgot" class="text-xs text-[#f97040] hover:underline">
									Forgot password?
								</a>
							</div>
							<Input
								id="password"
								type="password"
								bind:value={password}
								placeholder="••••••••••••"
								required
							/>
						</div>

						<!-- Remember Password & Security Meta -->
						<div class="flex items-center justify-between text-xs text-[var(--text-sub)] pt-1">
							<label class="flex items-center gap-2 cursor-pointer select-none">
								<input
									type="checkbox"
									bind:checked={rememberPassword}
									class="rounded bg-[var(--surface-raised)] border-[var(--border)] text-[#f97040] focus:ring-0 focus:ring-offset-0 cursor-pointer"
								/>
								<span>Remember password</span>
							</label>
							<span class="text-[var(--text-mute)] text-[11px] font-mono">256-bit encrypted</span>
						</div>

						<!-- Reusable Flat Submit Button -->
						<Button
							type="submit"
							variant="primary"
							loading={isLoading}
							class="w-full h-10 text-sm font-semibold"
						>
							<span>Sign in to Dashboard</span>
							<ArrowRight class="w-4 h-4 ml-1" />
						</Button>
					</form>

					<!-- Security Note -->
					<div class="pt-2 border-t border-[var(--border)] text-center text-xs text-[var(--text-mute)]">
						Access is provisioned by your company administrator.
					</div>
				</div>

				<!-- Role Hierarchy Box (Sharp rounded-lg) -->
				<div class="p-3 rounded-lg bg-[var(--surface)] border border-[var(--border)] text-xs text-[var(--text-mute)] text-center space-y-1">
					<div class="text-[var(--text-sub)] font-medium text-[11px]">Role Hierarchy Supported</div>
					<div class="font-mono text-[11px]">Company Admin • HR Manager • Employee</div>
				</div>
			{/if}
		</div>
	</section>
</div>
