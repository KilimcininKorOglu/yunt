<script lang="ts">
	/**
	 * Inbox Page
	 * Main mail inbox view with sidebar, filters, search, and message table.
	 */

	import { goto } from '$app/navigation';
	import { authStore } from '$stores/auth';
	import { messagesStore } from '$stores/messages';
	import Sidebar from '$components/Sidebar.svelte';
	import SearchBar from '$components/SearchBar.svelte';
	import FilterControls from '$components/FilterControls.svelte';
	import BulkActions from '$components/BulkActions.svelte';
	import MessageTable from '$components/MessageTable.svelte';
	import type { Message, ID } from '$lib/api/types';

	// Sidebar state
	let sidebarCollapsed = $state(false);

	// Initialize data on mount
	$effect(() => {
		if (authStore.isAuthenticated && !authStore.isLoading) {
			messagesStore.loadMailboxes();
			messagesStore.loadMessages();
		}
	});

	// Handlers
	function handleSearch(query: string): void {
		messagesStore.search(query);
	}

	function handleClearSearch(): void {
		messagesStore.clearSearch();
	}

	function handleFilterChange(filters: { status?: 'read' | 'unread'; isStarred?: boolean }): void {
		messagesStore.setFilters(filters);
	}

	function handleClearFilters(): void {
		messagesStore.clearFilters();
	}

	function handleSelectMailbox(mailboxId: ID | null): void {
		// Handle special filters
		if (mailboxId === 'starred') {
			messagesStore.selectMailbox(null);
			messagesStore.setFilters({ isStarred: true });
		} else if (mailboxId === 'spam') {
			// Spam filter - would need API support
			messagesStore.selectMailbox(null);
		} else {
			messagesStore.selectMailbox(mailboxId);
		}
	}

	function handleToggleSidebar(): void {
		sidebarCollapsed = !sidebarCollapsed;
	}

	async function handleLogout(): Promise<void> {
		await authStore.logout();
		goto('/login');
	}

	function handleMessageClick(message: Message): void {
		// Mark as read and navigate to message view
		if (message.status === 'unread') {
			messagesStore.markAsRead(message.id);
		}
		// Navigate to message detail (placeholder for now)
		// goto(`/inbox/${message.id}`);
	}

	function handleToggleSelect(id: ID): void {
		messagesStore.toggleSelect(id);
	}

	function handleToggleSelectAll(): void {
		messagesStore.toggleSelectAll();
	}

	function handleToggleStar(id: ID): void {
		messagesStore.toggleStar(id);
	}

	function handleSort(field: Parameters<typeof messagesStore.setSort>[0]): void {
		messagesStore.setSort(field);
	}

	function handlePageChange(page: number): void {
		messagesStore.goToPage(page);
	}

	// Bulk action handlers
	function handleMarkAsRead(): void {
		messagesStore.markSelectedAsRead();
	}

	function handleMarkAsUnread(): void {
		messagesStore.markSelectedAsUnread();
	}

	function handleDelete(): void {
		messagesStore.deleteSelected();
	}

	function handleStar(): void {
		messagesStore.starSelected();
	}

	function handleUnstar(): void {
		messagesStore.unstarSelected();
	}

	function handleMove(targetMailboxId: ID): void {
		messagesStore.moveSelected(targetMailboxId);
	}

	function handleRefresh(): void {
		messagesStore.refresh();
	}
</script>

<svelte:head>
	<title>Inbox - Yunt</title>
</svelte:head>

<div class="flex h-screen overflow-hidden bg-secondary-50">
	<!-- Sidebar -->
	<Sidebar
		mailboxes={messagesStore.mailboxes}
		selectedMailboxId={messagesStore.selectedMailboxId}
		totalUnreadCount={messagesStore.totalUnreadCount}
		onSelectMailbox={handleSelectMailbox}
		onLogout={handleLogout}
		collapsed={sidebarCollapsed}
		onToggleCollapse={handleToggleSidebar}
	/>

	<!-- Main Content -->
	<main class="flex flex-1 flex-col overflow-hidden">
		<!-- Top Bar -->
		<header class="flex items-center gap-4 border-b border-secondary-200 bg-white px-4 py-3">
			<!-- Mobile menu button -->
			<button
				type="button"
				class="rounded-lg p-2 text-secondary-500 hover:bg-secondary-100 lg:hidden"
				onclick={handleToggleSidebar}
				aria-label="Toggle sidebar"
			>
				<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M4 6h16M4 12h16M4 18h16"
					/>
				</svg>
			</button>

			<!-- Search Bar -->
			<SearchBar
				value={messagesStore.filters.searchQuery ?? ''}
				onSearch={handleSearch}
				onClear={handleClearSearch}
				disabled={messagesStore.isLoading}
			/>

			<!-- Refresh Button -->
			<button
				type="button"
				onclick={handleRefresh}
				disabled={messagesStore.isLoading}
				class="rounded-lg p-2 text-secondary-500 hover:bg-secondary-100 disabled:opacity-50"
				title="Refresh"
				aria-label="Refresh messages"
			>
				<svg
					class="h-5 w-5 {messagesStore.isLoading ? 'animate-spin' : ''}"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
					/>
				</svg>
			</button>
		</header>

		<!-- Filter Bar -->
		<div class="flex items-center justify-between border-b border-secondary-200 bg-white px-4 py-2">
			{#if messagesStore.hasSelection}
				<!-- Bulk Actions -->
				<BulkActions
					selectedCount={messagesStore.selectedIds.size}
					mailboxes={messagesStore.mailboxes}
					onMarkAsRead={handleMarkAsRead}
					onMarkAsUnread={handleMarkAsUnread}
					onDelete={handleDelete}
					onStar={handleStar}
					onUnstar={handleUnstar}
					onMove={handleMove}
					disabled={messagesStore.isLoading}
				/>
			{:else}
				<!-- Filter Controls -->
				<FilterControls
					filters={messagesStore.filters}
					onFilterChange={handleFilterChange}
					onClearFilters={handleClearFilters}
					disabled={messagesStore.isLoading}
				/>
			{/if}

			<!-- Message Count -->
			<div class="text-sm text-secondary-500">
				{#if messagesStore.pagination}
					{messagesStore.pagination.totalItems} message{messagesStore.pagination.totalItems === 1
						? ''
						: 's'}
				{/if}
			</div>
		</div>

		<!-- Error Alert -->
		{#if messagesStore.error}
			<div class="border-b border-red-200 bg-red-50 px-4 py-3" role="alert">
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-2">
						<svg class="h-5 w-5 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
						<p class="text-sm text-red-700">{messagesStore.error}</p>
					</div>
					<button
						type="button"
						onclick={() => messagesStore.clearError()}
						class="text-red-500 hover:text-red-700"
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

		<!-- Message Table -->
		<div class="flex-1 overflow-auto">
			<MessageTable
				messages={messagesStore.messages}
				selectedIds={messagesStore.selectedIds}
				allSelected={messagesStore.allSelected}
				sortField={messagesStore.sortField}
				sortOrder={messagesStore.sortOrder}
				pagination={messagesStore.pagination}
				isLoading={messagesStore.isLoading}
				onToggleSelect={handleToggleSelect}
				onToggleSelectAll={handleToggleSelectAll}
				onToggleStar={handleToggleStar}
				onSort={handleSort}
				onPageChange={handlePageChange}
				onMessageClick={handleMessageClick}
			/>
		</div>
	</main>
</div>
