<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { authStore } from '$stores/auth';
	import { isPublicRoute, isGuestOnlyRoute } from '$lib/guards/auth';
	import { sseService } from '$lib/services/sse';
	import { pollingService } from '$lib/services/polling';
	import Toast from '$components/Toast.svelte';

	interface Props {
		children?: import('svelte').Snippet;
	}

	const { children }: Props = $props();

	let initialized = $state(false);

	$effect(() => {
		if (!initialized) {
			authStore.initialize().then(() => {
				initialized = true;
			});
		}
	});

	$effect(() => {
		if (!initialized || authStore.isLoading) return;

		const currentPath = $page.url.pathname;

		if (authStore.isAuthenticated && isGuestOnlyRoute(currentPath)) {
			const redirect = $page.url.searchParams.get('redirect');
			goto(redirect || '/');
			return;
		}

		if (!authStore.isAuthenticated && !isPublicRoute(currentPath)) {
			const redirectUrl =
				currentPath !== '/' ? `?redirect=${encodeURIComponent(currentPath)}` : '';
			goto(`/login${redirectUrl}`);
		}
	});

	$effect(() => {
		if (initialized && authStore.isAuthenticated) {
			const token = authStore.getAccessToken();
			if (token) {
				sseService.start(token);
			} else {
				pollingService.start();
			}
		} else {
			sseService.stop();
			pollingService.stop();
		}

		return () => {
			sseService.stop();
			pollingService.stop();
		};
	});

	function getActiveTab(pathname: string): string {
		if (pathname === '/') return 'today';
		if (pathname.startsWith('/inbox') || pathname.startsWith('/message') || pathname.startsWith('/compose')) return 'mail';
		if (pathname.startsWith('/calendar')) return 'calendar';
		if (pathname.startsWith('/contacts')) return 'contacts';
		if (pathname.startsWith('/settings')) return 'options';
		return 'mail';
	}

	function handleSignOut() {
		authStore.logout();
		goto('/login');
	}
</script>

<Toast />

{#if !initialized || authStore.isLoading}
	<div style="display:flex;align-items:center;justify-content:center;min-height:100vh;background:var(--page-bg);">
		<div style="text-align:center;color:#fff;">
			<div class="loading-spinner" style="width:32px;height:32px;margin:0 auto 12px;"></div>
			<p>Loading...</p>
		</div>
	</div>
{:else if !authStore.isAuthenticated || $page.url.pathname === '/login'}
	{@render children?.()}
{:else}
	<!-- MSN Top Bar -->
	<div class="topbar">
		<div>
			<a href="/">Yunt Home</a>
			<span class="sep">|</span>
			<a href="/inbox">Mail</a>
			<span class="sep">|</span>
			<a href="/settings">Settings</a>
			<span class="sep">|</span>
			<a href="/users">Users</a>
			<span class="sep">|</span>
			<a href="/webhooks">Webhooks</a>
		</div>
		<div class="topbar-right">
			<button class="signout-btn" onclick={handleSignOut}>Sign Out</button>
		</div>
	</div>

	<!-- MSN Hotmail Header -->
	<div class="msn-header">
		<div class="msn-brand">
			<div class="msn-logo">
				<span class="logo-icon">&#x1F434;</span>
				Yunt
			</div>
			<span class="hotmail-text">Mail</span>
		</div>
		<div style="display:flex;align-items:flex-end;gap:20px;">
			<div class="nav-tabs">
				<a class="tab" class:active={getActiveTab($page.url.pathname) === 'today'} href="/">Today</a>
				<a class="tab" class:active={getActiveTab($page.url.pathname) === 'mail'} href="/inbox">Mail</a>
				<a class="tab" class:active={getActiveTab($page.url.pathname) === 'calendar'} href="/calendar">Calendar</a>
				<a class="tab" class:active={getActiveTab($page.url.pathname) === 'contacts'} href="/contacts">Contacts</a>
			</div>
			<div class="header-right">
				<a href="/settings">Options</a>
				<a href="https://github.com/KilimcininKorOglu/yunt" target="_blank">Help</a>
			</div>
		</div>
	</div>

	<!-- Sub Header -->
	<div class="sub-header">
		<span class="email">{authStore.user?.email || authStore.user?.username}</span>
		<span class="messenger">Yunt Mail Server</span>
	</div>

	<!-- Page Content -->
	{@render children?.()}

	<!-- Footer -->
	<div class="footer-bar">
		<a href="/">Yunt Home</a> &nbsp;|&nbsp;
		<a href="/inbox">Mail</a> &nbsp;|&nbsp;
		<a href="/settings">Settings</a> &nbsp;|&nbsp;
		<a href="/users">Users</a> &nbsp;|&nbsp;
		<a href="/webhooks">Webhooks</a>
	</div>
	<div class="footer-bottom">
		<span>Yunt Mail Server &nbsp; <a href="https://github.com/KilimcininKorOglu/yunt" target="_blank">GitHub</a></span>
		<span><a href="/settings">Options</a></span>
	</div>
{/if}
