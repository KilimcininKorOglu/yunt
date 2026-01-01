<script lang="ts">
	/**
	 * User Management Page
	 * Admin-only page for managing users: list, create, edit, and delete.
	 */

	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { authStore } from '$stores/auth';
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

	// API instance
	const usersApi = getUsersApi();

	// State
	let users = $state<UserProfile[]>([]);
	let pagination = $state<PaginationInfo | undefined>(undefined);
	let isLoading = $state(true);
	let error = $state<string | null>(null);

	// Form state
	let showForm = $state(false);
	let editingUser = $state<UserProfile | null>(null);
	let formError = $state<string | null>(null);
	let isSubmitting = $state(false);

	// Delete confirmation state
	let showDeleteConfirm = $state(false);
	let userToDelete = $state<UserProfile | null>(null);
	let isDeleting = $state(false);

	// Filter state
	let currentPage = $state(1);
	let searchQuery = $state('');
	let roleFilter = $state<UserRole | ''>('');
	let statusFilter = $state<UserStatus | ''>('');

	// Page size
	const pageSize = 10;

	// Admin guard - redirect non-admins
	$effect(() => {
		if (!authStore.isLoading) {
			const result = requireAdmin({ redirectTo: '/' });
			handleGuardResult(result);
		}
	});

	// Load users
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

	// Reload when filters change
	$effect(() => {
		// Track dependencies by reading them
		const page = currentPage;
		const search = searchQuery;
		const role = roleFilter;
		const status = statusFilter;

		// Prevent unused variable warnings
		void page;
		void search;
		void role;
		void status;

		// Only load if authenticated as admin
		if (!authStore.isLoading && authStore.isAuthenticated && authStore.user?.role === 'admin') {
			loadUsers();
		}
	});

	// Initial load
	onMount(() => {
		// Guard will redirect if not admin, so just load
		if (authStore.user?.role === 'admin') {
			loadUsers();
		}
	});

	// Handle page change
	function handlePageChange(page: number): void {
		currentPage = page;
	}

	// Handle search change
	function handleSearchChange(query: string): void {
		searchQuery = query;
		currentPage = 1; // Reset to first page on search
	}

	// Handle role filter change
	function handleRoleFilterChange(role: UserRole | ''): void {
		roleFilter = role;
		currentPage = 1;
	}

	// Handle status filter change
	function handleStatusFilterChange(status: UserStatus | ''): void {
		statusFilter = status;
		currentPage = 1;
	}

	// Open create form
	function handleCreateUser(): void {
		editingUser = null;
		formError = null;
		showForm = true;
	}

	// Open edit form
	function handleEditUser(user: UserProfile): void {
		editingUser = user;
		formError = null;
		showForm = true;
	}

	// Close form
	function handleCloseForm(): void {
		showForm = false;
		editingUser = null;
		formError = null;
	}

	// Handle form submission
	async function handleFormSubmit(
		data: UserCreateInput | UserUpdateInput,
		isEdit: boolean
	): Promise<void> {
		isSubmitting = true;
		formError = null;

		try {
			if (isEdit && editingUser) {
				await usersApi.update(editingUser.id, data as UserUpdateInput);

				// If password was provided, update it separately
				const createData = data as UserCreateInput;
				if (createData.password) {
					await usersApi.updatePassword(editingUser.id, {
						newPassword: createData.password
					});
				}
			} else {
				await usersApi.create(data as UserCreateInput);
			}

			// Close form and refresh list
			handleCloseForm();
			await loadUsers();
		} catch (err) {
			if (err instanceof Error) {
				// Handle specific error messages
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

	// Open delete confirmation
	function handleDeleteUser(user: UserProfile): void {
		userToDelete = user;
		showDeleteConfirm = true;
	}

	// Close delete confirmation
	function handleCancelDelete(): void {
		showDeleteConfirm = false;
		userToDelete = null;
	}

	// Confirm delete
	async function handleConfirmDelete(): Promise<void> {
		if (!userToDelete) return;

		isDeleting = true;

		try {
			await usersApi.delete(userToDelete.id);
			handleCancelDelete();
			await loadUsers();
		} catch (err) {
			// Just close the dialog on error, show error via toast or inline
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

	// Check if current user is admin (for rendering)
	const isAdmin = $derived(authStore.user?.role === 'admin');
</script>

<svelte:head>
	<title>User Management - Yunt</title>
</svelte:head>

<main class="min-h-screen p-6">
	<div class="mx-auto max-w-7xl">
		<!-- Header -->
		<div class="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
			<div>
				<h1 class="text-2xl font-bold text-secondary-900">User Management</h1>
				<p class="mt-1 text-sm text-secondary-500">Manage users, roles, and permissions</p>
			</div>

			<div class="flex gap-3">
				<!-- Back to dashboard -->
				<button type="button" class="btn-secondary" onclick={() => goto('/')}>
					<svg class="mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M10 19l-7-7m0 0l7-7m-7 7h18"
						/>
					</svg>
					Back
				</button>

				<!-- Create user button -->
				<button
					type="button"
					class="btn-primary"
					onclick={handleCreateUser}
					disabled={!isAdmin}
				>
					<svg class="mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M12 4v16m8-8H4"
						/>
					</svg>
					Create User
				</button>
			</div>
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
					<div class="flex-1">
						<p class="text-sm text-red-700">{error}</p>
					</div>
					<button
						type="button"
						class="text-red-500 hover:text-red-700"
						onclick={() => (error = null)}
						aria-label="Dismiss error"
					>
						<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M6 18L18 6M6 6l12 12"
							/>
						</svg>
					</button>
				</div>
			</div>
		{/if}

		<!-- Main content -->
		{#if showForm}
			<!-- User form -->
			<UserForm
				user={editingUser}
				isLoading={isSubmitting}
				error={formError}
				onSubmit={handleFormSubmit}
				onCancel={handleCloseForm}
			/>
		{:else}
			<!-- User table -->
			<UserTable
				{users}
				{pagination}
				{isLoading}
				{searchQuery}
				{roleFilter}
				{statusFilter}
				currentUserId={authStore.user?.id}
				onEdit={handleEditUser}
				onDelete={handleDeleteUser}
				onPageChange={handlePageChange}
				onSearchChange={handleSearchChange}
				onRoleFilterChange={handleRoleFilterChange}
				onStatusFilterChange={handleStatusFilterChange}
			/>
		{/if}
	</div>
</main>

<!-- Delete confirmation dialog -->
<ConfirmDialog
	open={showDeleteConfirm}
	title="Delete User"
	message={`Are you sure you want to delete "${userToDelete?.displayName || userToDelete?.username}"? This action cannot be undone.`}
	confirmText="Delete"
	cancelText="Cancel"
	variant="danger"
	isLoading={isDeleting}
	onConfirm={handleConfirmDelete}
	onCancel={handleCancelDelete}
/>
