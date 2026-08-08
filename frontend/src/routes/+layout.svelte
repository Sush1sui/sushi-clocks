<script lang="ts">
	import '../app.css';
	import { ModeWatcher } from 'mode-watcher';
	import sushiLogo from '$lib/assets/sushi_logo_without_bg.png';

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={sushiLogo} />
	<link rel="preconnect" href="https://fonts.googleapis.com" />
	<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin="anonymous" />
	<link
		href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Sora:wght@400;600;700;800&family=JetBrains+Mono:wght@400;500&display=swap"
		rel="stylesheet"
	/>
</svelte:head>

<ModeWatcher defaultMode="dark" />
{@render children()}
