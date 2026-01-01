<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { authStore } from '$stores/auth';
	import { requireGuest, handleGuardResult } from '$lib/guards/auth';

	// Form state
	let username = $state('');
	let password = $state('');
	let isSubmitting = $state(false);
	let formError = $state<string | null>(null);

	// Input validation state
	let usernameError = $state<string | null>(null);
	let passwordError = $state<string | null>(null);

	// Check if form is valid
	const isFormValid = $derived(
		username.trim().length > 0 && password.length > 0 && !usernameError && !passwordError
	);

	// Get redirect URL from query params
	function getRedirectUrl(): string {
		const redirect = $page.url.searchParams.get('redirect');
		return redirect || '/';
	}

	// Validate username
	function validateUsername(): void {
		if (!username.trim()) {
			usernameError = 'Username is required';
		} else if (username.trim().length < 3) {
			usernameError = 'Username must be at least 3 characters';
		} else {
			usernameError = null;
		}
	}

	// Validate password
	function validatePassword(): void {
		if (!password) {
			passwordError = 'Password is required';
		} else if (password.length < 4) {
			passwordError = 'Password must be at least 4 characters';
		} else {
			passwordError = null;
		}
	}

	// Handle form submission
	async function handleSubmit(event: Event): Promise<void> {
		event.preventDefault();

		// Validate all fields
		validateUsername();
		validatePassword();

		if (!isFormValid) {
			return;
		}

		isSubmitting = true;
		formError = null;

		try {
			await authStore.login({ username: username.trim(), password });
			// Redirect to dashboard or intended page
			await goto(getRedirectUrl());
		} catch (err) {
			if (err instanceof Error) {
				// Handle specific error messages
				if (err.message.includes('credentials') || err.message.includes('unauthorized')) {
					formError = 'Invalid username or password';
				} else if (err.message.includes('network') || err.message.includes('Network')) {
					formError = 'Network error. Please check your connection and try again.';
				} else {
					formError = err.message;
				}
			} else {
				formError = 'An unexpected error occurred. Please try again.';
			}
		} finally {
			isSubmitting = false;
		}
	}

	// Redirect if already authenticated
	$effect(() => {
		if (!authStore.isLoading) {
			const result = requireGuest({ redirectTo: getRedirectUrl() });
			handleGuardResult(result);
		}
	});
</script>

<svelte:head>
	<title>Login - Yunt</title>
</svelte:head>

<main class="flex min-h-screen items-center justify-center p-4">
	<div class="card w-full max-w-md p-8">
		<!-- Logo and Title -->
		<div class="mb-8 text-center">
			<h1 class="mb-2 text-3xl font-bold text-primary-600">Yunt</h1>
			<p class="text-secondary-500">Sign in to your account</p>
		</div>

		<!-- Loading State -->
		{#if authStore.isLoading}
			<div class="flex items-center justify-center py-8">
				<div
					class="h-8 w-8 animate-spin rounded-full border-4 border-primary-200 border-t-primary-600"
				></div>
			</div>
		{:else}
			<!-- Login Form -->
			<form onsubmit={handleSubmit} class="space-y-6">
				<!-- Error Alert -->
				{#if formError}
					<div class="rounded-lg border border-red-200 bg-red-50 p-4" role="alert">
						<div class="flex items-start gap-3">
							<svg
								class="mt-0.5 h-5 w-5 flex-shrink-0 text-red-500"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
								/>
							</svg>
							<p class="text-sm text-red-700">{formError}</p>
						</div>
					</div>
				{/if}

				<!-- Username Field -->
				<div>
					<label for="username" class="mb-2 block text-sm font-medium text-secondary-700">
						Username
					</label>
					<input
						type="text"
						id="username"
						name="username"
						bind:value={username}
						onblur={validateUsername}
						class="input"
						class:border-red-500={usernameError}
						placeholder="Enter your username"
						autocomplete="username"
						disabled={isSubmitting}
					/>
					{#if usernameError}
						<p class="mt-1 text-sm text-red-500">{usernameError}</p>
					{/if}
				</div>

				<!-- Password Field -->
				<div>
					<label for="password" class="mb-2 block text-sm font-medium text-secondary-700">
						Password
					</label>
					<input
						type="password"
						id="password"
						name="password"
						bind:value={password}
						onblur={validatePassword}
						class="input"
						class:border-red-500={passwordError}
						placeholder="Enter your password"
						autocomplete="current-password"
						disabled={isSubmitting}
					/>
					{#if passwordError}
						<p class="mt-1 text-sm text-red-500">{passwordError}</p>
					{/if}
				</div>

				<!-- Submit Button -->
				<button
					type="submit"
					class="btn-primary w-full"
					disabled={isSubmitting || !isFormValid}
				>
					{#if isSubmitting}
						<span class="flex items-center justify-center gap-2">
							<svg class="h-5 w-5 animate-spin" fill="none" viewBox="0 0 24 24">
								<circle
									class="opacity-25"
									cx="12"
									cy="12"
									r="10"
									stroke="currentColor"
									stroke-width="4"
								></circle>
								<path
									class="opacity-75"
									fill="currentColor"
									d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
								></path>
							</svg>
							Signing in...
						</span>
					{:else}
						Sign in
					{/if}
				</button>
			</form>
		{/if}

		<!-- Footer -->
		<div class="mt-8 text-center text-sm text-secondary-500">
			<p>Development Mail Server</p>
		</div>
	</div>
</main>
