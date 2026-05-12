<script lang="ts">
	import type { UserProfile, UserRole, UserStatus, PaginationInfo } from '$lib/api';

	interface Props {
		users: UserProfile[];
		pagination?: PaginationInfo;
		isLoading?: boolean;
		currentUserId?: string;
		onEdit: (user: UserProfile) => void;
		onDelete: (user: UserProfile) => void;
		onPageChange?: (page: number) => void;
	}

	const {
		users,
		pagination,
		isLoading = false,
		currentUserId,
		onEdit,
		onDelete,
		onPageChange
	}: Props = $props();

	function formatRole(role: UserRole): string {
		return role.charAt(0).toUpperCase() + role.slice(1);
	}

	function formatStatus(status: UserStatus): string {
		return status.charAt(0).toUpperCase() + status.slice(1);
	}

	function formatDate(dateStr: string | undefined): string {
		if (!dateStr) return 'Never';
		return new Date(dateStr).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
	}

	function canDelete(user: UserProfile): boolean {
		return user.id !== currentUserId;
	}
</script>

<table class="msg-table">
	<thead>
		<tr>
			<th>User</th>
			<th>Email</th>
			<th>Role</th>
			<th>Status</th>
			<th>Last Login</th>
			<th>Created</th>
			<th style="text-align:right;">Actions</th>
		</tr>
	</thead>
	<tbody>
		{#if isLoading}
			<tr>
				<td colspan="7" class="loading-cell">
					<div class="loading-spinner"></div>
					Loading users...
				</td>
			</tr>
		{:else if users.length === 0}
			<tr>
				<td colspan="7" class="empty-cell">No users found</td>
			</tr>
		{:else}
			{#each users as user (user.id)}
				<tr>
					<td><b>{user.displayName || user.username}</b><br><span style="color:var(--text-muted);font-size:10px;">@{user.username}</span></td>
					<td>{user.email}</td>
					<td>{formatRole(user.role)}</td>
					<td>{formatStatus(user.status)}</td>
					<td>{formatDate(user.lastLoginAt)}</td>
					<td>{formatDate(user.createdAt)}</td>
					<td style="text-align:right;">
						<button type="button" class="hotmail-btn" onclick={() => onEdit(user)}>Edit</button>
						{#if canDelete(user)}
							<button type="button" class="hotmail-btn" style="color:var(--error-red);" onclick={() => onDelete(user)}>Delete</button>
						{/if}
					</td>
				</tr>
			{/each}
		{/if}
	</tbody>
</table>

{#if pagination && pagination.totalPages > 1}
	<div class="pagination-bar">
		<span class="pagination-info">
			Page {pagination.page} of {pagination.totalPages} ({pagination.totalItems} users)
		</span>
		<div class="pagination-buttons">
			<button type="button" class="hotmail-btn" disabled={!pagination.hasPrev} onclick={() => onPageChange?.(pagination.page - 1)}>&laquo; Prev</button>
			<button type="button" class="hotmail-btn" disabled={!pagination.hasNext} onclick={() => onPageChange?.(pagination.page + 1)}>Next &raquo;</button>
		</div>
	</div>
{/if}
