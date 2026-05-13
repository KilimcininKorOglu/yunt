<script lang="ts">
	import { goto } from '$app/navigation';
	import { authStore } from '$stores/auth.svelte';
	import { messagesStore } from '$stores/messages.svelte';
	import Sidebar from '$components/Sidebar.svelte';
	import SearchBar from '$components/SearchBar.svelte';
	import FilterControls from '$components/FilterControls.svelte';
	import BulkActions from '$components/BulkActions.svelte';
	import MessageTable from '$components/MessageTable.svelte';
	import type { Message, ID } from '$lib/api/types';

	let sidebarCollapsed = $state(false);

	$effect(() => {
		if (authStore.isAuthenticated && !authStore.isLoading) {
			messagesStore.loadMailboxes();
			messagesStore.loadMessages();
		}
	});

	function handleSearch(query: string): void {
		messagesStore.search(query);
	}

	function handleClearSearch(): void {
		messagesStore.clearSearch();
	}

	function handleFilterChange(filters: {
		status?: 'read' | 'unread';
		isStarred?: boolean;
	}): void {
		messagesStore.setFilters(filters);
	}

	function handleClearFilters(): void {
		messagesStore.clearFilters();
	}

	function handleSelectMailbox(mailboxId: ID | null): void {
		if (mailboxId === 'starred') {
			messagesStore.selectMailbox(null);
			messagesStore.setFilters({ isStarred: true });
		} else if (mailboxId === 'spam') {
			messagesStore.selectMailbox(null);
		} else {
			messagesStore.selectMailbox(mailboxId);
		}
	}

	function handleToggleSidebar(): void {
		sidebarCollapsed = !sidebarCollapsed;
	}

	function handleMessageClick(message: Message): void {
		if (message.status === 'unread') {
			messagesStore.markAsRead(message.id);
		}
		goto(`/message/${message.id}`);
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
	<title>Inbox - Yunt Mail</title>
</svelte:head>

<div class="inbox-layout">
	<Sidebar
		mailboxes={messagesStore.mailboxes}
		selectedMailboxId={messagesStore.selectedMailboxId}
		totalUnreadCount={messagesStore.totalUnreadCount}
		onSelectMailbox={handleSelectMailbox}
		collapsed={sidebarCollapsed}
		onToggleCollapse={handleToggleSidebar}
	/>

	<div class="inbox-content">
		<!-- Toolbar -->
		<div class="toolbar">
			<div class="toolbar-left">
				<a href="/compose" class="hotmail-btn toolbar-btn-primary">New</a>
				<span class="toolbar-sep">|</span>
				<button type="button" class="hotmail-btn" onclick={handleDelete} disabled={!messagesStore.hasSelection}>Delete</button>
				<button type="button" class="hotmail-btn" onclick={handleRefresh} disabled={messagesStore.isLoading}>
					{messagesStore.isLoading ? 'Loading...' : 'Refresh'}
				</button>
			</div>
			<div class="toolbar-right">
				<SearchBar
					value={messagesStore.filters.searchQuery ?? ''}
					onSearch={handleSearch}
					onClear={handleClearSearch}
					disabled={messagesStore.isLoading}
					placeholder="Find"
				/>
			</div>
		</div>

		<!-- Filter/Bulk Actions Bar -->
		<div class="filter-bar">
			{#if messagesStore.hasSelection}
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
				<FilterControls
					filters={messagesStore.filters}
					onFilterChange={handleFilterChange}
					onClearFilters={handleClearFilters}
					disabled={messagesStore.isLoading}
				/>
				<span class="msg-count">
					{#if messagesStore.pagination}
						{messagesStore.pagination.totalItems} Message(s), {messagesStore.totalUnreadCount} Unread
					{/if}
				</span>
			{/if}
		</div>

		<!-- Error -->
		{#if messagesStore.error}
			<div class="alert alert-error">
				{messagesStore.error}
				<button type="button" class="alert-close" onclick={() => messagesStore.clearError()}>X</button>
			</div>
		{/if}

		<!-- Message Table -->
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
</div>
