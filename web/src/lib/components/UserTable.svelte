<script lang="ts">
	/**
	 * UserTable Component
	 * Displays a table of users with actions for edit and delete.
	 * Includes role and status badges, search, and pagination.
	 */

	import type { UserProfile, UserRole, UserStatus, PaginationInfo } from '$lib/api';

	interface Props {
		/** List of users to display */
		users: UserProfile[];
		/** Pagination information */
		pagination?: PaginationInfo;
		/** Whether the table is in loading state */
		isLoading?: boolean;
		/** Search query string */
		searchQuery?: string;
		/** Currently selected role filter */
		roleFilter?: UserRole | '';
		/** Currently selected status filter */
		statusFilter?: UserStatus | '';
		/** The current user's ID (to prevent self-delete) */
		currentUserId?: string;
		/** Callback when edit is clicked */
		onEdit: (user: UserProfile) => void;
		/** Callback when delete is clicked */
		onDelete: (user: UserProfile) => void;
		/** Callback when page changes */
		onPageChange?: (page: number) => void;
		/** Callback when search changes */
		onSearchChange?: (query: string) => void;
		/** Callback when role filter changes */
		onRoleFilterChange?: (role: UserRole | '') => void;
		/** Callback when status filter changes */
		onStatusFilterChange?: (status: UserStatus | '') => void;
	}

	const {
		users,
		pagination,
		isLoading = false,
		searchQuery = '',
		roleFilter = '',
		statusFilter = '',
		currentUserId,
		onEdit,
		onDelete,
		onPageChange,
		onSearchChange,
		onRoleFilterChange,
		onStatusFilterChange
	}: Props = $props();

	// Local search state for debouncing - synced via $effect
	let localSearch = $state('');
	let searchTimeout: ReturnType<typeof setTimeout> | null = null;

	// Sync local search with prop when prop changes externally
	$effect(() => {
		localSearch = searchQuery;
	});

	// Handle search input with debounce
	function handleSearchInput(event: Event): void {
		const target = event.target as HTMLInputElement;
		localSearch = target.value;

		if (searchTimeout) {
			clearTimeout(searchTimeout);
		}

		searchTimeout = setTimeout(() => {
			onSearchChange?.(localSearch);
		}, 300);
	}

	// Get role badge styling
	function getRoleBadgeClass(role: UserRole): string {
		switch (role) {
			case 'admin':
				return 'bg-purple-100 text-purple-700';
			case 'user':
				return 'bg-blue-100 text-blue-700';
			case 'viewer':
				return 'bg-gray-100 text-gray-700';
			default:
				return 'bg-gray-100 text-gray-700';
		}
	}

	// Get status badge styling
	function getStatusBadgeClass(status: UserStatus): string {
		switch (status) {
			case 'active':
				return 'bg-green-100 text-green-700';
			case 'inactive':
				return 'bg-red-100 text-red-700';
			case 'pending':
				return 'bg-yellow-100 text-yellow-700';
			default:
				return 'bg-gray-100 text-gray-700';
		}
	}

	// Format role label
	function formatRole(role: UserRole): string {
		return role.charAt(0).toUpperCase() + role.slice(1);
	}

	// Format status label
	function formatStatus(status: UserStatus): string {
		return status.charAt(0).toUpperCase() + status.slice(1);
	}

	// Format date
	function formatDate(dateStr: string | undefined): string {
		if (!dateStr) return 'Never';
		const date = new Date(dateStr);
		return date.toLocaleDateString(undefined, {
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		});
	}

	// Check if user can be deleted (not the current user)
	function canDelete(user: UserProfile): boolean {
		return user.id !== currentUserId;
	}

	// Available role options
	const roleOptions: { value: UserRole | ''; label: string }[] = [
		{ value: '', label: 'All Roles' },
		{ value: 'admin', label: 'Admin' },
		{ value: 'user', label: 'User' },
		{ value: 'viewer', label: 'Viewer' }
	];

	// Available status options
	const statusOptions: { value: UserStatus | ''; label: string }[] = [
		{ value: '', label: 'All Statuses' },
		{ value: 'active', label: 'Active' },
		{ value: 'inactive', label: 'Inactive' },
		{ value: 'pending', label: 'Pending' }
	];
</script>

<div class="space-y-4">
	<!-- Filters and search -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<!-- Search -->
		<div class="relative flex-1 max-w-sm">
			<div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
				<svg
					class="h-5 w-5 text-secondary-400"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
					/>
				</svg>
			</div>
			<input
				type="text"
				placeholder="Search users..."
				value={localSearch}
				oninput={handleSearchInput}
				class="input pl-10"
				disabled={isLoading}
			/>
		</div>

		<!-- Filters -->
		<div class="flex gap-3">
			<select
				class="input w-auto"
				value={roleFilter}
				onchange={(e) => onRoleFilterChange?.((e.target as HTMLSelectElement).value as UserRole | '')}
				disabled={isLoading}
			>
				{#each roleOptions as opt}
					<option value={opt.value}>{opt.label}</option>
				{/each}
			</select>

			<select
				class="input w-auto"
				value={statusFilter}
				onchange={(e) => onStatusFilterChange?.((e.target as HTMLSelectElement).value as UserStatus | '')}
				disabled={isLoading}
			>
				{#each statusOptions as opt}
					<option value={opt.value}>{opt.label}</option>
				{/each}
			</select>
		</div>
	</div>

	<!-- Table -->
	<div class="card overflow-hidden">
		<div class="overflow-x-auto">
			<table class="w-full text-left text-sm">
				<thead class="border-b border-secondary-200 bg-secondary-50 text-xs uppercase text-secondary-600">
					<tr>
						<th class="px-4 py-3 font-medium">User</th>
						<th class="px-4 py-3 font-medium">Role</th>
						<th class="px-4 py-3 font-medium">Status</th>
						<th class="px-4 py-3 font-medium">Last Login</th>
						<th class="px-4 py-3 font-medium">Created</th>
						<th class="px-4 py-3 font-medium text-right">Actions</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-secondary-100">
					{#if isLoading}
						<!-- Loading skeleton -->
						{#each Array(5) as _}
							<tr class="animate-pulse">
								<td class="px-4 py-3">
									<div class="flex items-center gap-3">
										<div class="h-10 w-10 rounded-full bg-secondary-200"></div>
										<div class="space-y-1">
											<div class="h-4 w-24 rounded bg-secondary-200"></div>
											<div class="h-3 w-32 rounded bg-secondary-100"></div>
										</div>
									</div>
								</td>
								<td class="px-4 py-3">
									<div class="h-5 w-16 rounded bg-secondary-200"></div>
								</td>
								<td class="px-4 py-3">
									<div class="h-5 w-16 rounded bg-secondary-200"></div>
								</td>
								<td class="px-4 py-3">
									<div class="h-4 w-20 rounded bg-secondary-200"></div>
								</td>
								<td class="px-4 py-3">
									<div class="h-4 w-20 rounded bg-secondary-200"></div>
								</td>
								<td class="px-4 py-3">
									<div class="flex justify-end gap-2">
										<div class="h-8 w-8 rounded bg-secondary-200"></div>
										<div class="h-8 w-8 rounded bg-secondary-200"></div>
									</div>
								</td>
							</tr>
						{/each}
					{:else if users.length === 0}
						<!-- Empty state -->
						<tr>
							<td colspan="6" class="px-4 py-12 text-center">
								<div class="flex flex-col items-center gap-2">
									<svg
										class="h-12 w-12 text-secondary-300"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="1.5"
											d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"
										/>
									</svg>
									<p class="text-secondary-500">No users found</p>
									{#if searchQuery || roleFilter || statusFilter}
										<p class="text-sm text-secondary-400">Try adjusting your filters</p>
									{/if}
								</div>
							</td>
						</tr>
					{:else}
						{#each users as user (user.id)}
							<tr class="hover:bg-secondary-50 transition-colors">
								<!-- User info -->
								<td class="px-4 py-3">
									<div class="flex items-center gap-3">
										<!-- Avatar -->
										<div class="flex h-10 w-10 items-center justify-center rounded-full bg-primary-100 text-primary-600 font-medium">
											{user.username.charAt(0).toUpperCase()}
										</div>
										<!-- Details -->
										<div>
											<p class="font-medium text-secondary-900">
												{user.displayName || user.username}
											</p>
											<p class="text-xs text-secondary-500">
												@{user.username} &middot; {user.email}
											</p>
										</div>
									</div>
								</td>

								<!-- Role -->
								<td class="px-4 py-3">
									<span class={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${getRoleBadgeClass(user.role)}`}>
										{formatRole(user.role)}
									</span>
								</td>

								<!-- Status -->
								<td class="px-4 py-3">
									<span class={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${getStatusBadgeClass(user.status)}`}>
										{formatStatus(user.status)}
									</span>
								</td>

								<!-- Last Login -->
								<td class="px-4 py-3 text-secondary-600">
									{formatDate(user.lastLoginAt)}
								</td>

								<!-- Created -->
								<td class="px-4 py-3 text-secondary-600">
									{formatDate(user.createdAt)}
								</td>

								<!-- Actions -->
								<td class="px-4 py-3">
									<div class="flex justify-end gap-1">
										<button
											type="button"
											class="rounded p-2 text-secondary-400 hover:bg-secondary-100 hover:text-secondary-600 transition-colors"
											onclick={() => onEdit(user)}
											title="Edit user"
										>
											<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
												<path
													stroke-linecap="round"
													stroke-linejoin="round"
													stroke-width="2"
													d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
												/>
											</svg>
										</button>
										{#if canDelete(user)}
											<button
												type="button"
												class="rounded p-2 text-secondary-400 hover:bg-red-50 hover:text-red-600 transition-colors"
												onclick={() => onDelete(user)}
												title="Delete user"
											>
												<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
													<path
														stroke-linecap="round"
														stroke-linejoin="round"
														stroke-width="2"
														d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
													/>
												</svg>
											</button>
										{:else}
											<span
												class="rounded p-2 text-secondary-300 cursor-not-allowed"
												title="Cannot delete your own account"
											>
												<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
													<path
														stroke-linecap="round"
														stroke-linejoin="round"
														stroke-width="2"
														d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
													/>
												</svg>
											</span>
										{/if}
									</div>
								</td>
							</tr>
						{/each}
					{/if}
				</tbody>
			</table>
		</div>
	</div>

	<!-- Pagination -->
	{#if pagination && pagination.totalPages > 1}
		<div class="flex items-center justify-between px-1">
			<p class="text-sm text-secondary-600">
				Showing {(pagination.page - 1) * pagination.pageSize + 1} to {Math.min(pagination.page * pagination.pageSize, pagination.totalItems)} of {pagination.totalItems} users
			</p>

			<nav class="flex items-center gap-1" aria-label="Pagination">
				<button
					type="button"
					class="rounded p-2 text-secondary-400 hover:bg-secondary-100 hover:text-secondary-600 disabled:opacity-50 disabled:cursor-not-allowed"
					onclick={() => onPageChange?.(pagination.page - 1)}
					disabled={!pagination.hasPrev || isLoading}
					aria-label="Previous page"
				>
					<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
					</svg>
				</button>

				<!-- Page numbers -->
				{#each Array(pagination.totalPages) as _, i}
					{@const pageNum = i + 1}
					{#if pagination.totalPages <= 7 || pageNum === 1 || pageNum === pagination.totalPages || Math.abs(pageNum - pagination.page) <= 1}
						<button
							type="button"
							class="min-w-[2rem] rounded px-2 py-1 text-sm font-medium transition-colors"
							class:bg-primary-600={pageNum === pagination.page}
							class:text-white={pageNum === pagination.page}
							class:text-secondary-600={pageNum !== pagination.page}
							class:hover:bg-secondary-100={pageNum !== pagination.page}
							onclick={() => onPageChange?.(pageNum)}
							disabled={isLoading}
						>
							{pageNum}
						</button>
					{:else if pageNum === 2 && pagination.page > 4}
						<span class="px-1 text-secondary-400">...</span>
					{:else if pageNum === pagination.totalPages - 1 && pagination.page < pagination.totalPages - 3}
						<span class="px-1 text-secondary-400">...</span>
					{/if}
				{/each}

				<button
					type="button"
					class="rounded p-2 text-secondary-400 hover:bg-secondary-100 hover:text-secondary-600 disabled:opacity-50 disabled:cursor-not-allowed"
					onclick={() => onPageChange?.(pagination.page + 1)}
					disabled={!pagination.hasNext || isLoading}
					aria-label="Next page"
				>
					<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
					</svg>
				</button>
			</nav>
		</div>
	{/if}
</div>
