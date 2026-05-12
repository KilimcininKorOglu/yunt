<script lang="ts">
	import type { UserProfile, UserCreateInput, UserUpdateInput, UserRole, UserStatus } from '$lib/api';

	interface Props {
		user?: UserProfile | null;
		isLoading?: boolean;
		error?: string | null;
		onSubmit: (data: UserCreateInput | UserUpdateInput, isEdit: boolean) => void | Promise<void>;
		onCancel: () => void;
	}

	const { user = null, isLoading = false, error = null, onSubmit, onCancel }: Props = $props();

	const isEdit = $derived(user !== null);
	const submitText = $derived(isEdit ? 'Update User' : 'Create User');

	let username = $state('');
	let email = $state('');
	let displayName = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let role = $state<UserRole>('user');
	let status = $state<UserStatus>('active');

	$effect(() => {
		username = user?.username ?? '';
		email = user?.email ?? '';
		displayName = user?.displayName ?? '';
		password = '';
		confirmPassword = '';
		role = user?.role ?? 'user';
		status = user?.status ?? 'active';
	});

	let usernameError = $state<string | null>(null);
	let emailError = $state<string | null>(null);
	let passwordError = $state<string | null>(null);
	let confirmPasswordError = $state<string | null>(null);

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

	function validateUsername(): boolean {
		if (!username.trim()) { usernameError = 'Username is required'; return false; }
		if (username.trim().length < 3) { usernameError = 'Min 3 characters'; return false; }
		if (!/^[a-zA-Z0-9_-]+$/.test(username.trim())) { usernameError = 'Letters, numbers, _ and - only'; return false; }
		usernameError = null; return true;
	}

	function validateEmail(): boolean {
		if (!email.trim()) { emailError = 'Email is required'; return false; }
		if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())) { emailError = 'Invalid email'; return false; }
		emailError = null; return true;
	}

	function validatePassword(): boolean {
		if (isEdit && !password) { passwordError = null; return true; }
		if (!isEdit && !password) { passwordError = 'Password is required'; return false; }
		if (password && password.length < 8) { passwordError = 'Min 8 characters'; return false; }
		passwordError = null; return true;
	}

	function validateConfirmPassword(): boolean {
		if (!password) { confirmPasswordError = null; return true; }
		if (password !== confirmPassword) { confirmPasswordError = 'Passwords do not match'; return false; }
		confirmPasswordError = null; return true;
	}

	function validateForm(): boolean {
		return validateUsername() && validateEmail() && validatePassword() && validateConfirmPassword();
	}

	const canSubmit = $derived(
		username.trim().length >= 3 && email.trim().length > 0 &&
		(isEdit || password.length >= 8) && (!password || password === confirmPassword) && !isLoading
	);

	async function handleSubmit(event: Event): Promise<void> {
		event.preventDefault();
		if (!validateForm()) return;

		if (isEdit) {
			const updateData: UserUpdateInput = {};
			if (displayName !== (user?.displayName ?? '')) updateData.displayName = displayName.trim() || undefined;
			if (email.trim() !== user?.email) updateData.email = email.trim();
			if (role !== user?.role) updateData.role = role;
			if (status !== user?.status) updateData.status = status;
			await onSubmit(updateData, true);
		} else {
			await onSubmit({
				username: username.trim(), email: email.trim(), password,
				displayName: displayName.trim() || undefined, role
			} as UserCreateInput, false);
		}
	}
</script>

{#if error}
	<div class="alert alert-error" style="margin-bottom:8px;">{error}</div>
{/if}

<form onsubmit={handleSubmit}>
	<table class="server-info-table"><tbody>
		<tr>
			<td class="lbl">Username *</td>
			<td>
				<input type="text" class="hotmail-input" bind:value={username} onblur={validateUsername} disabled={isEdit || isLoading} style="width:200px;" />
				{#if usernameError}<br><span style="color:var(--error-red);font-size:10px;">{usernameError}</span>{/if}
				{#if isEdit}<br><span style="font-size:10px;color:var(--text-muted);">Cannot be changed</span>{/if}
			</td>
		</tr>
		<tr>
			<td class="lbl">Email *</td>
			<td>
				<input type="email" class="hotmail-input" bind:value={email} onblur={validateEmail} disabled={isLoading} style="width:250px;" />
				{#if emailError}<br><span style="color:var(--error-red);font-size:10px;">{emailError}</span>{/if}
			</td>
		</tr>
		<tr>
			<td class="lbl">Display Name</td>
			<td><input type="text" class="hotmail-input" bind:value={displayName} disabled={isLoading} style="width:200px;" /></td>
		</tr>
		<tr>
			<td class="lbl">{isEdit ? 'New Password' : 'Password *'}</td>
			<td>
				<input type="password" class="hotmail-input" bind:value={password} onblur={validatePassword} disabled={isLoading} style="width:200px;" placeholder={isEdit ? 'Leave blank to keep' : ''} />
				{#if passwordError}<br><span style="color:var(--error-red);font-size:10px;">{passwordError}</span>{/if}
			</td>
		</tr>
		{#if password}
			<tr>
				<td class="lbl">Confirm Password *</td>
				<td>
					<input type="password" class="hotmail-input" bind:value={confirmPassword} onblur={validateConfirmPassword} disabled={isLoading} style="width:200px;" />
					{#if confirmPasswordError}<br><span style="color:var(--error-red);font-size:10px;">{confirmPasswordError}</span>{/if}
				</td>
			</tr>
		{/if}
		<tr>
			<td class="lbl">Role</td>
			<td>
				<select class="hotmail-select" bind:value={role} disabled={isLoading}>
					{#each roles as r}<option value={r.value}>{r.label}</option>{/each}
				</select>
			</td>
		</tr>
		{#if isEdit}
			<tr>
				<td class="lbl">Status</td>
				<td>
					<select class="hotmail-select" bind:value={status} disabled={isLoading}>
						{#each statuses as s}<option value={s.value}>{s.label}</option>{/each}
					</select>
				</td>
			</tr>
		{/if}
	</tbody></table>

	<div style="margin-top:10px;display:flex;gap:6px;">
		<button type="submit" class="hotmail-btn toolbar-btn-primary" disabled={!canSubmit}>
			{isLoading ? 'Saving...' : submitText}
		</button>
		<button type="button" class="hotmail-btn" onclick={onCancel} disabled={isLoading}>Cancel</button>
	</div>
</form>
