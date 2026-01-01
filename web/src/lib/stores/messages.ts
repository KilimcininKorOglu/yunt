/**
 * Messages Store
 * Manages message list state with Svelte 5 runes for reactivity.
 * Handles filtering, sorting, pagination, and bulk operations.
 */

import { getMessagesApi, getMailboxesApi } from '$lib/api';
import type {
	ID,
	Message,
	Mailbox,
	MessageStatus,
	MessageListFilter,
	SortOrder,
	PaginationInfo
} from '$lib/api/types';

// ============================================================================
// Types
// ============================================================================

export type MessageSortField = 'receivedAt' | 'sentAt' | 'subject' | 'from' | 'size' | 'status';

export interface MessagesFilter {
	mailboxId?: ID;
	status?: MessageStatus;
	isStarred?: boolean;
	searchQuery?: string;
}

export interface MessagesState {
	/** List of messages for current view */
	messages: Message[];
	/** All mailboxes for sidebar */
	mailboxes: Mailbox[];
	/** Currently selected mailbox ID */
	selectedMailboxId: ID | null;
	/** Selected message IDs for bulk operations */
	selectedIds: Set<ID>;
	/** Pagination information */
	pagination: PaginationInfo | null;
	/** Current sort field */
	sortField: MessageSortField;
	/** Current sort order */
	sortOrder: SortOrder;
	/** Active filters */
	filters: MessagesFilter;
	/** Whether data is loading */
	isLoading: boolean;
	/** Error message from last operation */
	error: string | null;
}

// ============================================================================
// Messages Store Implementation
// ============================================================================

/**
 * Creates the messages store with Svelte 5 runes.
 */
function createMessagesStore() {
	// State using Svelte 5 runes
	let messages = $state<Message[]>([]);
	let mailboxes = $state<Mailbox[]>([]);
	let selectedMailboxId = $state<ID | null>(null);
	let selectedIds = $state<Set<ID>>(new Set());
	let pagination = $state<PaginationInfo | null>(null);
	let sortField = $state<MessageSortField>('receivedAt');
	let sortOrder = $state<SortOrder>('desc');
	let filters = $state<MessagesFilter>({});
	let isLoading = $state(false);
	let error = $state<string | null>(null);

	// Derived state
	const hasSelection = $derived(selectedIds.size > 0);
	const allSelected = $derived(messages.length > 0 && selectedIds.size === messages.length);
	const unreadCount = $derived(messages.filter((m) => m.status === 'unread').length);
	const totalUnreadCount = $derived(mailboxes.reduce((sum, mb) => sum + mb.unreadCount, 0));

	// API instances
	const messagesApi = getMessagesApi();
	const mailboxesApi = getMailboxesApi();

	/**
	 * Load mailboxes for sidebar
	 */
	async function loadMailboxes(): Promise<void> {
		try {
			const response = await mailboxesApi.list({ pageSize: 100 });
			mailboxes = response.items;
		} catch (err) {
			console.error('Failed to load mailboxes:', err);
		}
	}

	/**
	 * Load messages with current filters and pagination
	 */
	async function loadMessages(page = 1): Promise<void> {
		isLoading = true;
		error = null;

		try {
			const filter: MessageListFilter = {
				page,
				pageSize: 25,
				sort: sortField,
				order: sortOrder,
				mailboxId: selectedMailboxId ?? filters.mailboxId,
				status: filters.status,
				isStarred: filters.isStarred
			};

			let response;
			if (filters.searchQuery && filters.searchQuery.trim()) {
				response = await messagesApi.search(filters.searchQuery.trim(), filter);
			} else {
				response = await messagesApi.list(filter);
			}

			messages = response.items;
			pagination = response.pagination;
			// Clear selection when loading new messages
			selectedIds = new Set();
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Failed to load messages';
			error = message;
			messages = [];
			pagination = null;
		} finally {
			isLoading = false;
		}
	}

	/**
	 * Refresh current view
	 */
	async function refresh(): Promise<void> {
		await Promise.all([loadMailboxes(), loadMessages(pagination?.page ?? 1)]);
	}

	/**
	 * Select a mailbox
	 */
	async function selectMailbox(mailboxId: ID | null): Promise<void> {
		selectedMailboxId = mailboxId;
		await loadMessages(1);
	}

	/**
	 * Update filters and reload
	 */
	async function setFilters(newFilters: Partial<MessagesFilter>): Promise<void> {
		filters = { ...filters, ...newFilters };
		await loadMessages(1);
	}

	/**
	 * Clear all filters
	 */
	async function clearFilters(): Promise<void> {
		filters = {};
		await loadMessages(1);
	}

	/**
	 * Set search query
	 */
	async function search(query: string): Promise<void> {
		filters = { ...filters, searchQuery: query };
		await loadMessages(1);
	}

	/**
	 * Clear search
	 */
	async function clearSearch(): Promise<void> {
		filters = { ...filters, searchQuery: undefined };
		await loadMessages(1);
	}

	/**
	 * Update sort and reload
	 */
	async function setSort(field: MessageSortField, order?: SortOrder): Promise<void> {
		if (sortField === field && !order) {
			// Toggle order if same field
			sortOrder = sortOrder === 'asc' ? 'desc' : 'asc';
		} else {
			sortField = field;
			sortOrder = order ?? 'desc';
		}
		await loadMessages(1);
	}

	/**
	 * Navigate to page
	 */
	async function goToPage(page: number): Promise<void> {
		if (!pagination) return;
		if (page < 1 || page > pagination.totalPages) return;
		await loadMessages(page);
	}

	/**
	 * Toggle message selection
	 */
	function toggleSelect(id: ID): void {
		const newSet = new Set(selectedIds);
		if (newSet.has(id)) {
			newSet.delete(id);
		} else {
			newSet.add(id);
		}
		selectedIds = newSet;
	}

	/**
	 * Select all messages on current page
	 */
	function selectAll(): void {
		selectedIds = new Set(messages.map((m) => m.id));
	}

	/**
	 * Deselect all messages
	 */
	function deselectAll(): void {
		selectedIds = new Set();
	}

	/**
	 * Toggle select all
	 */
	function toggleSelectAll(): void {
		if (allSelected) {
			deselectAll();
		} else {
			selectAll();
		}
	}

	// ========================================================================
	// Bulk Operations
	// ========================================================================

	/**
	 * Mark selected messages as read
	 */
	async function markSelectedAsRead(): Promise<void> {
		if (selectedIds.size === 0) return;

		isLoading = true;
		try {
			await messagesApi.bulkMarkAsRead(Array.from(selectedIds));
			await refresh();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to mark as read';
		} finally {
			isLoading = false;
		}
	}

	/**
	 * Mark selected messages as unread
	 */
	async function markSelectedAsUnread(): Promise<void> {
		if (selectedIds.size === 0) return;

		isLoading = true;
		try {
			await messagesApi.bulkMarkAsUnread(Array.from(selectedIds));
			await refresh();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to mark as unread';
		} finally {
			isLoading = false;
		}
	}

	/**
	 * Delete selected messages
	 */
	async function deleteSelected(): Promise<void> {
		if (selectedIds.size === 0) return;

		isLoading = true;
		try {
			await messagesApi.bulkDelete(Array.from(selectedIds));
			await refresh();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete messages';
		} finally {
			isLoading = false;
		}
	}

	/**
	 * Star selected messages
	 */
	async function starSelected(): Promise<void> {
		if (selectedIds.size === 0) return;

		isLoading = true;
		try {
			await messagesApi.bulkStar(Array.from(selectedIds));
			await refresh();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to star messages';
		} finally {
			isLoading = false;
		}
	}

	/**
	 * Unstar selected messages
	 */
	async function unstarSelected(): Promise<void> {
		if (selectedIds.size === 0) return;

		isLoading = true;
		try {
			await messagesApi.bulkUnstar(Array.from(selectedIds));
			await refresh();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to unstar messages';
		} finally {
			isLoading = false;
		}
	}

	/**
	 * Move selected messages to another mailbox
	 */
	async function moveSelected(targetMailboxId: ID): Promise<void> {
		if (selectedIds.size === 0) return;

		isLoading = true;
		try {
			await messagesApi.bulkMove(Array.from(selectedIds), targetMailboxId);
			await refresh();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to move messages';
		} finally {
			isLoading = false;
		}
	}

	// ========================================================================
	// Single Message Operations
	// ========================================================================

	/**
	 * Toggle star on a single message
	 */
	async function toggleStar(id: ID): Promise<void> {
		const message = messages.find((m) => m.id === id);
		if (!message) return;

		try {
			if (message.isStarred) {
				await messagesApi.unstar(id);
			} else {
				await messagesApi.star(id);
			}
			// Update local state
			messages = messages.map((m) => (m.id === id ? { ...m, isStarred: !m.isStarred } : m));
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to update star';
		}
	}

	/**
	 * Mark a single message as read
	 */
	async function markAsRead(id: ID): Promise<void> {
		try {
			await messagesApi.markAsRead(id);
			messages = messages.map((m) => (m.id === id ? { ...m, status: 'read' as const } : m));
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to mark as read';
		}
	}

	/**
	 * Delete a single message
	 */
	async function deleteMessage(id: ID): Promise<void> {
		try {
			await messagesApi.delete(id);
			await refresh();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete message';
		}
	}

	/**
	 * Clear error state
	 */
	function clearError(): void {
		error = null;
	}

	return {
		// State getters
		get messages() {
			return messages;
		},
		get mailboxes() {
			return mailboxes;
		},
		get selectedMailboxId() {
			return selectedMailboxId;
		},
		get selectedIds() {
			return selectedIds;
		},
		get pagination() {
			return pagination;
		},
		get sortField() {
			return sortField;
		},
		get sortOrder() {
			return sortOrder;
		},
		get filters() {
			return filters;
		},
		get isLoading() {
			return isLoading;
		},
		get error() {
			return error;
		},
		get hasSelection() {
			return hasSelection;
		},
		get allSelected() {
			return allSelected;
		},
		get unreadCount() {
			return unreadCount;
		},
		get totalUnreadCount() {
			return totalUnreadCount;
		},

		// Actions
		loadMailboxes,
		loadMessages,
		refresh,
		selectMailbox,
		setFilters,
		clearFilters,
		search,
		clearSearch,
		setSort,
		goToPage,
		toggleSelect,
		selectAll,
		deselectAll,
		toggleSelectAll,
		markSelectedAsRead,
		markSelectedAsUnread,
		deleteSelected,
		starSelected,
		unstarSelected,
		moveSelected,
		toggleStar,
		markAsRead,
		deleteMessage,
		clearError
	};
}

// ============================================================================
// Singleton Instance
// ============================================================================

export const messagesStore = createMessagesStore();

export type MessagesStore = ReturnType<typeof createMessagesStore>;
