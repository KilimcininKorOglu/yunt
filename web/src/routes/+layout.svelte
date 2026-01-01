<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { authStore } from '$stores/auth';
	import { isPublicRoute, isGuestOnlyRoute } from '$lib/guards/auth';
	import { pollingService } from '$lib/services/polling';
	import Toast from '$components/Toast.svelte';

	interface Props {
		children?: import('svelte').Snippet;
	}

	const { children }: Props = $props();

	// Track initialization
	let initialized = $state(false);

	// Initialize auth on mount
	$effect(() => {
		if (!initialized) {
			authStore.initialize().then(() => {
				initialized = true;
			});
		}
	});

	// Handle route protection after initialization
	$effect(() => {
		if (!initialized || authStore.isLoading) return;

		const currentPath = $page.url.pathname;

		// If user is authenticated and on a guest-only route, redirect to home
		if (authStore.isAuthenticated && isGuestOnlyRoute(currentPath)) {
			const redirect = $page.url.searchParams.get('redirect');
			goto(redirect || '/');
			return;
		}

		// If user is not authenticated and on a protected route, redirect to login
		if (!authStore.isAuthenticated && !isPublicRoute(currentPath)) {
			const redirectUrl =
				currentPath !== '/' ? `?redirect=${encodeURIComponent(currentPath)}` : '';
			goto(`/login${redirectUrl}`);
		}
	});

	// Start/stop polling based on authentication state
	$effect(() => {
		if (initialized && authStore.isAuthenticated) {
			pollingService.start();
		} else {
			pollingService.stop();
		}

		// Cleanup on unmount
		return () => {
			pollingService.stop();
		};
	});
</script>

<!-- Toast notifications (always rendered) -->
<Toast />

{#if !initialized || authStore.isLoading}
	<!-- Loading state while checking authentication -->
	<div class="flex min-h-screen items-center justify-center bg-secondary-50">
		<div class="text-center">
			<div
				class="mb-4 inline-block h-12 w-12 animate-spin rounded-full border-4 border-primary-200 border-t-primary-600"
			></div>
			<p class="text-secondary-500">Loading...</p>
		</div>
	</div>
{:else}
	<div class="min-h-screen bg-secondary-50">
		{@render children?.()}
	</div>
{/if}
