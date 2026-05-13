<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { authStore } from '$stores/auth.svelte';
	import { requireGuest, handleGuardResult } from '$lib/guards/auth';

	let username = $state('');
	let password = $state('');
	let isSubmitting = $state(false);
	let formError = $state<string | null>(null);

	const isFormValid = $derived(username.trim().length > 0 && password.length > 0);

	function getRedirectUrl(): string {
		return $page.url.searchParams.get('redirect') || '/';
	}

	async function handleSubmit(event: Event): Promise<void> {
		event.preventDefault();
		if (!isFormValid) return;

		isSubmitting = true;
		formError = null;

		try {
			await authStore.login({ username: username.trim(), password });
			await goto(getRedirectUrl());
		} catch (err) {
			if (err instanceof Error) {
				if (err.message.includes('credentials') || err.message.includes('unauthorized')) {
					formError = 'Invalid username or password';
				} else if (err.message.includes('network') || err.message.includes('Network')) {
					formError = 'Network error. Please check your connection.';
				} else {
					formError = err.message;
				}
			} else {
				formError = 'An unexpected error occurred.';
			}
		} finally {
			isSubmitting = false;
		}
	}

	$effect(() => {
		if (!authStore.isLoading) {
			const result = requireGuest({ redirectTo: getRedirectUrl() });
			handleGuardResult(result);
		}
	});
</script>

<svelte:head>
	<title>Sign In - Yunt Mail</title>
</svelte:head>

<div class="login-page">
	<div class="login-left">
		<h2>New to Yunt Mail?</h2>
		<h3 style="color:var(--msn-dark);font-size:13px;margin-bottom:8px;">A development mail server for developers</h3>
		<ul>
			<li><b>Capture all outgoing emails</b><br>No more accidental emails to real users. Yunt captures everything your app sends.</li>
			<li><b>SMTP + IMAP + Web UI</b><br>Full protocol support. Connect with Thunderbird, use the web interface, or hit the REST API.</li>
			<li><b>Multi-database support</b><br>SQLite, PostgreSQL, MySQL, or MongoDB — pick your backend.</li>
			<li><b>Real-time updates</b><br>See new emails instantly with Server-Sent Events.</li>
		</ul>
	</div>
	<div class="login-right">
		<h3>Sign in to Yunt</h3>

		{#if authStore.isLoading}
			<div style="text-align:center;padding:20px;">
				<div class="loading-spinner"></div>
			</div>
		{:else}
			{#if formError}
				<div class="alert alert-error">{formError}</div>
			{/if}

			<form onsubmit={handleSubmit}>
				<label for="username">Username:</label>
				<input type="text" id="username" bind:value={username} class="hotmail-input" autocomplete="username" disabled={isSubmitting}>

				<label for="password">Password:</label>
				<input type="password" id="password" bind:value={password} class="hotmail-input" autocomplete="current-password" disabled={isSubmitting}>

				<br>
				<button type="submit" class="signin-btn" disabled={isSubmitting || !isFormValid}>
					{isSubmitting ? 'Signing in...' : 'Sign In'}
				</button>
			</form>
		{/if}
	</div>
</div>
