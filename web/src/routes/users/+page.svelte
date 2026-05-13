<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { authStore } from '$stores/auth.svelte';
	import { requireAdmin, handleGuardResult } from '$lib/guards/auth';
	import { getUsersApi } from '$lib/api';
	import type {
		UserProfile,
		UserCreateInput,
		UserUpdateInput,
		UserRole,
		UserStatus,
		PaginationInfo
	} from '$lib/api';
	import UserTable from '$lib/components/UserTable.svelte';
	import UserForm from '$lib/components/UserForm.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';

	const usersApi = getUsersApi();

	let users = $state<UserProfile[]>([]);
	let pagination = $state<PaginationInfo | undefined>(undefined);
	let isLoading = $state(true);
	let error = $state<string | null>(null);

	let showForm = $state(false);
	let editingUser = $state<UserProfile | null>(null);
	let formError = $state<string | null>(null);
	let isSubmitting = $state(false);

	let showDeleteConfirm = $state(false);
	let userToDelete = $state<UserProfile | null>(null);
	let isDeleting = $state(false);

	let currentPage = $state(1);
	let searchQuery = $state('');
	let roleFilter = $state<UserRole | ''>('');
	let statusFilter = $state<UserStatus | ''>('');

	const pageSize = 10;

	$effect(() => {
		if (!authStore.isLoading) {
			const result = requireAdmin({ redirectTo: '/' });
			handleGuardResult(result);
		}
	});

	async function loadUsers(): Promise<void> {
		isLoading = true;
		error = null;

		try {
			const response = await usersApi.list({
				page: currentPage,
				pageSize: pageSize,
				search: searchQuery || undefined,
				role: roleFilter || undefined,
				status: statusFilter || undefined
			});

			users = response.items;
			pagination = response.pagination;
		} catch (err) {
			if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Failed to load users. Please try again.';
			}
			users = [];
			pagination = undefined;
		} finally {
			isLoading = false;
		}
	}

	$effect(() => {
		const page = currentPage;
		const search = searchQuery;
		const role = roleFilter;
		const status = statusFilter;
		void page; void search; void role; void status;

		if (!authStore.isLoading && authStore.isAuthenticated && authStore.user?.role === 'admin') {
			loadUsers();
		}
	});

	onMount(() => {
		if (authStore.user?.role === 'admin') {
			loadUsers();
		}
	});

	function handlePageChange(page: number): void {
		currentPage = page;
	}

	function handleSearchChange(query: string): void {
		searchQuery = query;
		currentPage = 1;
	}

	function handleRoleFilterChange(role: UserRole | ''): void {
		roleFilter = role;
		currentPage = 1;
	}

	function handleStatusFilterChange(status: UserStatus | ''): void {
		statusFilter = status;
		currentPage = 1;
	}

	function handleCreateUser(): void {
		editingUser = null;
		formError = null;
		showForm = true;
	}

	function handleEditUser(user: UserProfile): void {
		editingUser = user;
		formError = null;
		showForm = true;
	}

	function handleCloseForm(): void {
		showForm = false;
		editingUser = null;
		formError = null;
	}

	async function handleFormSubmit(data: UserCreateInput | UserUpdateInput, isEdit: boolean): Promise<void> {
		isSubmitting = true;
		formError = null;

		try {
			if (isEdit && editingUser) {
				await usersApi.update(editingUser.id, data as UserUpdateInput);
				const createData = data as UserCreateInput;
				if (createData.password) {
					await usersApi.updatePassword(editingUser.id, { newPassword: createData.password });
				}
			} else {
				await usersApi.create(data as UserCreateInput);
			}
			handleCloseForm();
			await loadUsers();
		} catch (err) {
			if (err instanceof Error) {
				if (err.message.includes('already exists') || err.message.includes('duplicate')) {
					formError = 'A user with this username or email already exists.';
				} else if (err.message.includes('forbidden') || err.message.includes('Forbidden')) {
					formError = 'You do not have permission to perform this action.';
				} else {
					formError = err.message;
				}
			} else {
				formError = isEdit ? 'Failed to update user.' : 'Failed to create user.';
			}
		} finally {
			isSubmitting = false;
		}
	}

	function handleDeleteUser(user: UserProfile): void {
		userToDelete = user;
		showDeleteConfirm = true;
	}

	function handleCancelDelete(): void {
		showDeleteConfirm = false;
		userToDelete = null;
	}

	async function handleConfirmDelete(): Promise<void> {
		if (!userToDelete) return;
		isDeleting = true;

		try {
			await usersApi.delete(userToDelete.id);
			handleCancelDelete();
			await loadUsers();
		} catch (err) {
			handleCancelDelete();
			if (err instanceof Error) {
				error = err.message;
			} else {
				error = 'Failed to delete user.';
			}
		} finally {
			isDeleting = false;
		}
	}

	const isAdmin = $derived(authStore.user?.role === 'admin');
</script>

<svelte:head>
	<title>Users - Yunt Mail</title>
</svelte:head>

<div class="options-area">
	<div class="today-header">
		<h2>User Management</h2>
		<button type="button" class="hotmail-btn toolbar-btn-primary" onclick={handleCreateUser} disabled={!isAdmin}>
			New User
		</button>
	</div>

	{#if error}
		<div class="alert alert-error">
			{error}
			<button type="button" class="alert-close" onclick={() => (error = null)}>X</button>
		</div>
	{/if}

	{#if showForm}
		<div class="info-box" style="margin-bottom:12px;">
			<div class="info-box-header">{editingUser ? 'Edit User' : 'Create User'}</div>
			<div class="info-box-body">
				<UserForm
					user={editingUser}
					isLoading={isSubmitting}
					error={formError}
					onSubmit={handleFormSubmit}
					onCancel={handleCloseForm}
				/>
			</div>
		</div>
	{/if}

	<!-- Filters -->
	<div class="filter-bar" style="margin-bottom:0;border-bottom:none;">
		<div class="filter-controls">
			<span class="filter-label">Search:</span>
			<input
				type="text"
				class="hotmail-input"
				style="width:150px;"
				placeholder="Username or email..."
				value={searchQuery}
				oninput={(e) => handleSearchChange((e.target as HTMLInputElement).value)}
			/>
			<span class="filter-label">Role:</span>
			<select class="hotmail-select" onchange={(e) => handleRoleFilterChange((e.target as HTMLSelectElement).value as UserRole | '')}>
				<option value="">All</option>
				<option value="admin">Admin</option>
				<option value="user">User</option>
			</select>
			<span class="filter-label">Status:</span>
			<select class="hotmail-select" onchange={(e) => handleStatusFilterChange((e.target as HTMLSelectElement).value as UserStatus | '')}>
				<option value="">All</option>
				<option value="active">Active</option>
				<option value="inactive">Inactive</option>
			</select>
		</div>
	</div>

	<UserTable
		{users}
		{pagination}
		{isLoading}
		onEdit={handleEditUser}
		onDelete={handleDeleteUser}
		onPageChange={handlePageChange}
	/>
</div>

<ConfirmDialog
	open={showDeleteConfirm}
	title="Delete User"
	message="Are you sure you want to delete {userToDelete?.username}? This action cannot be undone."
	confirmText="Delete"
	variant="danger"
	isLoading={isDeleting}
	onConfirm={handleConfirmDelete}
	onCancel={handleCancelDelete}
/>
