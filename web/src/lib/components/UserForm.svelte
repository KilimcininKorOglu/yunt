<script lang="ts">
	/**
	 * UserForm Component
	 * A form for creating and editing users with validation.
	 * Handles both create and edit modes based on whether a user is provided.
	 */

	import type { UserProfile, UserCreateInput, UserUpdateInput, UserRole, UserStatus } from '$lib/api';

	interface Props {
		/** User to edit (null for create mode) */
		user?: UserProfile | null;
		/** Whether the form is in a loading/submitting state */
		isLoading?: boolean;
		/** Error message to display */
		error?: string | null;
		/** Callback when form is submitted */
		onSubmit: (data: UserCreateInput | UserUpdateInput, isEdit: boolean) => void | Promise<void>;
		/** Callback when form is cancelled */
		onCancel: () => void;
	}

	const {
		user = null,
		isLoading = false,
		error = null,
		onSubmit,
		onCancel
	}: Props = $props();

	// Determine if we're in edit mode
	const isEdit = $derived(user !== null);
	const formTitle = $derived(isEdit ? 'Edit User' : 'Create User');
	const submitText = $derived(isEdit ? 'Update User' : 'Create User');

	// Form fields - initialized and synced via $effect
	let username = $state('');
	let email = $state('');
	let displayName = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let role = $state<UserRole>('user');
	let status = $state<UserStatus>('active');

	// Sync form fields when user prop changes
	$effect(() => {
		username = user?.username ?? '';
		email = user?.email ?? '';
		displayName = user?.displayName ?? '';
		password = '';
		confirmPassword = '';
		role = user?.role ?? 'user';
		status = user?.status ?? 'active';
	});

	// Field validation errors
	let usernameError = $state<string | null>(null);
	let emailError = $state<string | null>(null);
	let passwordError = $state<string | null>(null);
	let confirmPasswordError = $state<string | null>(null);

	// Available options
	const roles: { value: UserRole; label: string }[] = [
		{ value: 'admin', label: 'Admin' },
		{ value: 'user', label: 'User' },
		{ value: 'viewer', label: 'Viewer' }
	];

	const statuses: { value: UserStatus; label: string }[] = [
		{ value: 'active', label: 'Active' },
		{ value: 'inactive', label: 'Inactive' },
		{ value: 'pending', label: 'Pending' }
	];

	// Validate username
	function validateUsername(): boolean {
		if (!username.trim()) {
			usernameError = 'Username is required';
			return false;
		}
		if (username.trim().length < 3) {
			usernameError = 'Username must be at least 3 characters';
			return false;
		}
		if (!/^[a-zA-Z0-9_-]+$/.test(username.trim())) {
			usernameError = 'Username can only contain letters, numbers, underscores, and hyphens';
			return false;
		}
		usernameError = null;
		return true;
	}

	// Validate email
	function validateEmail(): boolean {
		if (!email.trim()) {
			emailError = 'Email is required';
			return false;
		}
		const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
		if (!emailRegex.test(email.trim())) {
			emailError = 'Please enter a valid email address';
			return false;
		}
		emailError = null;
		return true;
	}

	// Validate password
	function validatePassword(): boolean {
		// Password is optional in edit mode
		if (isEdit && !password) {
			passwordError = null;
			return true;
		}

		if (!isEdit && !password) {
			passwordError = 'Password is required';
			return false;
		}

		if (password && password.length < 8) {
			passwordError = 'Password must be at least 8 characters';
			return false;
		}

		passwordError = null;
		return true;
	}

	// Validate confirm password
	function validateConfirmPassword(): boolean {
		// Only required if password is set
		if (!password) {
			confirmPasswordError = null;
			return true;
		}

		if (password !== confirmPassword) {
			confirmPasswordError = 'Passwords do not match';
			return false;
		}

		confirmPasswordError = null;
		return true;
	}

	// Validate all fields
	function validateForm(): boolean {
		const isUsernameValid = validateUsername();
		const isEmailValid = validateEmail();
		const isPasswordValid = validatePassword();
		const isConfirmPasswordValid = validateConfirmPassword();

		return isUsernameValid && isEmailValid && isPasswordValid && isConfirmPasswordValid;
	}

	// Check if form can be submitted
	const canSubmit = $derived(
		username.trim().length >= 3 &&
		email.trim().length > 0 &&
		(isEdit || password.length >= 8) &&
		(!password || password === confirmPassword) &&
		!isLoading
	);

	// Handle form submission
	async function handleSubmit(event: Event): Promise<void> {
		event.preventDefault();

		if (!validateForm()) {
			return;
		}

		if (isEdit) {
			// Build update payload - only include changed fields
			const updateData: UserUpdateInput = {};

			if (displayName !== (user?.displayName ?? '')) {
				updateData.displayName = displayName.trim() || undefined;
			}
			if (email.trim() !== user?.email) {
				updateData.email = email.trim();
			}
			if (role !== user?.role) {
				updateData.role = role;
			}
			if (status !== user?.status) {
				updateData.status = status;
			}

			await onSubmit(updateData, true);
		} else {
			// Build create payload
			const createData: UserCreateInput = {
				username: username.trim(),
				email: email.trim(),
				password: password,
				displayName: displayName.trim() || undefined,
				role: role
			};

			await onSubmit(createData, false);
		}
	}

	// Handle keyboard events
	function handleKeydown(event: KeyboardEvent): void {
		if (event.key === 'Escape' && !isLoading) {
			onCancel();
		}
	}
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="card p-6">
	<!-- Header -->
	<div class="mb-6 flex items-center justify-between">
		<h2 class="text-xl font-semibold text-secondary-900">{formTitle}</h2>
		<button
			type="button"
			class="text-secondary-400 hover:text-secondary-600"
			onclick={onCancel}
			disabled={isLoading}
			aria-label="Close form"
		>
			<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M6 18L18 6M6 6l12 12"
				/>
			</svg>
		</button>
	</div>

	<!-- Error message -->
	{#if error}
		<div class="mb-6 rounded-lg border border-red-200 bg-red-50 p-4" role="alert">
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
				<p class="text-sm text-red-700">{error}</p>
			</div>
		</div>
	{/if}

	<!-- Form -->
	<form onsubmit={handleSubmit} class="space-y-5">
		<!-- Username field (readonly in edit mode) -->
		<div>
			<label for="username" class="mb-2 block text-sm font-medium text-secondary-700">
				Username <span class="text-red-500">*</span>
			</label>
			<input
				type="text"
				id="username"
				name="username"
				bind:value={username}
				onblur={validateUsername}
				class="input"
				class:border-red-500={usernameError}
				placeholder="Enter username"
				disabled={isEdit || isLoading}
				autocomplete="username"
			/>
			{#if usernameError}
				<p class="mt-1 text-sm text-red-500">{usernameError}</p>
			{/if}
			{#if isEdit}
				<p class="mt-1 text-xs text-secondary-500">Username cannot be changed</p>
			{/if}
		</div>

		<!-- Email field -->
		<div>
			<label for="email" class="mb-2 block text-sm font-medium text-secondary-700">
				Email <span class="text-red-500">*</span>
			</label>
			<input
				type="email"
				id="email"
				name="email"
				bind:value={email}
				onblur={validateEmail}
				class="input"
				class:border-red-500={emailError}
				placeholder="user@example.com"
				disabled={isLoading}
				autocomplete="email"
			/>
			{#if emailError}
				<p class="mt-1 text-sm text-red-500">{emailError}</p>
			{/if}
		</div>

		<!-- Display name field -->
		<div>
			<label for="displayName" class="mb-2 block text-sm font-medium text-secondary-700">
				Display Name
			</label>
			<input
				type="text"
				id="displayName"
				name="displayName"
				bind:value={displayName}
				class="input"
				placeholder="Enter display name (optional)"
				disabled={isLoading}
				autocomplete="name"
			/>
		</div>

		<!-- Password field -->
		<div>
			<label for="password" class="mb-2 block text-sm font-medium text-secondary-700">
				{isEdit ? 'New Password' : 'Password'} {#if !isEdit}<span class="text-red-500">*</span>{/if}
			</label>
			<input
				type="password"
				id="password"
				name="password"
				bind:value={password}
				onblur={validatePassword}
				class="input"
				class:border-red-500={passwordError}
				placeholder={isEdit ? 'Leave blank to keep current password' : 'Enter password'}
				disabled={isLoading}
				autocomplete="new-password"
			/>
			{#if passwordError}
				<p class="mt-1 text-sm text-red-500">{passwordError}</p>
			{:else if !isEdit}
				<p class="mt-1 text-xs text-secondary-500">Minimum 8 characters</p>
			{/if}
		</div>

		<!-- Confirm password field (only show when password is entered) -->
		{#if password}
			<div>
				<label for="confirmPassword" class="mb-2 block text-sm font-medium text-secondary-700">
					Confirm Password <span class="text-red-500">*</span>
				</label>
				<input
					type="password"
					id="confirmPassword"
					name="confirmPassword"
					bind:value={confirmPassword}
					onblur={validateConfirmPassword}
					class="input"
					class:border-red-500={confirmPasswordError}
					placeholder="Confirm password"
					disabled={isLoading}
					autocomplete="new-password"
				/>
				{#if confirmPasswordError}
					<p class="mt-1 text-sm text-red-500">{confirmPasswordError}</p>
				{/if}
			</div>
		{/if}

		<!-- Role and Status row -->
		<div class="grid grid-cols-1 gap-5 sm:grid-cols-2">
			<!-- Role field -->
			<div>
				<label for="role" class="mb-2 block text-sm font-medium text-secondary-700">
					Role
				</label>
				<select
					id="role"
					name="role"
					bind:value={role}
					class="input"
					disabled={isLoading}
				>
					{#each roles as r}
						<option value={r.value}>{r.label}</option>
					{/each}
				</select>
			</div>

			<!-- Status field (only in edit mode) -->
			{#if isEdit}
				<div>
					<label for="status" class="mb-2 block text-sm font-medium text-secondary-700">
						Status
					</label>
					<select
						id="status"
						name="status"
						bind:value={status}
						class="input"
						disabled={isLoading}
					>
						{#each statuses as s}
							<option value={s.value}>{s.label}</option>
						{/each}
					</select>
				</div>
			{/if}
		</div>

		<!-- Form actions -->
		<div class="flex justify-end gap-3 pt-4 border-t border-secondary-200">
			<button
				type="button"
				class="btn-secondary"
				onclick={onCancel}
				disabled={isLoading}
			>
				Cancel
			</button>
			<button
				type="submit"
				class="btn-primary"
				disabled={!canSubmit}
			>
				{#if isLoading}
					<span class="flex items-center gap-2">
						<svg class="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
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
						Saving...
					</span>
				{:else}
					{submitText}
				{/if}
			</button>
		</div>
	</form>
</div>
