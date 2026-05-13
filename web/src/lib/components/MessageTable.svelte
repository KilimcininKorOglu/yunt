<script module lang="ts">
	function getPageNumber(pagination: PaginationInfo, idx: number): number {
		const { page, totalPages } = pagination;
		const maxVisible = 7;

		if (totalPages <= maxVisible) {
			return idx + 1;
		}

		if (idx === 0) return 1;
		if (idx === maxVisible - 1) return totalPages;

		const start = Math.max(2, page - 2);
		const end = Math.min(totalPages - 1, page + 2);
		const range = [];

		for (let i = start; i <= end; i++) {
			range.push(i);
		}

		while (range.length < maxVisible - 2) {
			if (range[0] > 2) {
				range.unshift(range[0] - 1);
			} else if (range[range.length - 1] < totalPages - 1) {
				range.push(range[range.length - 1] + 1);
			} else {
				break;
			}
		}

		const result: number[] = [1];
		if (range[0] > 2) {
			result.push(-1);
		}
		result.push(...range);
		if (range[range.length - 1] < totalPages - 1) {
			result.push(-1);
		}
		result.push(totalPages);

		return result[idx] ?? -1;
	}
</script>

<script lang="ts">
	import type { Message, ID, SortOrder, PaginationInfo } from '$lib/api/types';
	import type { MessageSortField } from '$stores/messages.svelte';

	interface Props {
		messages: Message[];
		selectedIds: Set<ID>;
		allSelected: boolean;
		sortField: MessageSortField;
		sortOrder: SortOrder;
		pagination: PaginationInfo | null;
		isLoading: boolean;
		onToggleSelect: (id: ID) => void;
		onToggleSelectAll: () => void;
		onToggleStar: (id: ID) => void;
		onSort: (field: MessageSortField) => void;
		onPageChange: (page: number) => void;
		onMessageClick?: (message: Message) => void;
	}

	const {
		messages,
		selectedIds,
		allSelected,
		sortField,
		sortOrder,
		pagination,
		isLoading,
		onToggleSelect,
		onToggleSelectAll,
		onToggleStar,
		onSort,
		onPageChange,
		onMessageClick
	}: Props = $props();

	function formatDate(dateStr: string): string {
		const date = new Date(dateStr);
		const now = new Date();
		const diffDays = Math.floor((now.getTime() - date.getTime()) / (1000 * 60 * 60 * 24));

		if (diffDays === 0) {
			return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
		} else if (diffDays < 7) {
			return date.toLocaleDateString([], { weekday: 'short' });
		} else if (diffDays < 365) {
			return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
		} else {
			return date.toLocaleDateString([], { year: 'numeric', month: 'short', day: 'numeric' });
		}
	}

	function formatSender(from: { name?: string; address: string }): string {
		return from.name || from.address.split('@')[0];
	}

	function formatSize(bytes: number): string {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}

	function getSortIndicator(field: MessageSortField): string {
		if (sortField !== field) return '';
		return sortOrder === 'asc' ? ' ▲' : ' ▼';
	}

	function handleRowClick(message: Message): void {
		onMessageClick?.(message);
	}

	function handleRowKeydown(event: KeyboardEvent, message: Message): void {
		if (event.key === 'Enter' || event.key === ' ') {
			event.preventDefault();
			handleRowClick(message);
		}
	}
</script>

<div class="msg-table-wrapper">
	<table class="msg-table">
		<thead>
			<tr>
				<th class="col-check">
					<input
						type="checkbox"
						checked={allSelected && messages.length > 0}
						onchange={onToggleSelectAll}
						title="Select all"
					/>
				</th>
				<th class="col-star"></th>
				<th class="col-from">
					<button type="button" class="sort-btn" onclick={() => onSort('from')}>
						From{getSortIndicator('from')}
					</button>
				</th>
				<th class="col-subject">
					<button type="button" class="sort-btn" onclick={() => onSort('subject')}>
						Subject{getSortIndicator('subject')}
					</button>
				</th>
				<th class="col-attach"></th>
				<th class="col-size">
					<button type="button" class="sort-btn" onclick={() => onSort('size')}>
						Size{getSortIndicator('size')}
					</button>
				</th>
				<th class="col-date">
					<button type="button" class="sort-btn" onclick={() => onSort('receivedAt')}>
						Date{getSortIndicator('receivedAt')}
					</button>
				</th>
			</tr>
		</thead>
		<tbody>
			{#if isLoading}
				<tr>
					<td colspan="7" class="loading-cell">
						<div class="loading-spinner"></div>
						Loading messages...
					</td>
				</tr>
			{:else if messages.length === 0}
				<tr>
					<td colspan="7" class="empty-cell">
						📭 No messages found
					</td>
				</tr>
			{:else}
				{#each messages as message (message.id)}
					<tr
						class:unread={message.status === 'unread'}
						class:selected={selectedIds.has(message.id)}
						onclick={() => handleRowClick(message)}
						onkeydown={(e) => handleRowKeydown(e, message)}
						role="button"
						tabindex="0"
					>
						<td class="col-check" onclick={(e) => e.stopPropagation()}>
							<input
								type="checkbox"
								checked={selectedIds.has(message.id)}
								onchange={() => onToggleSelect(message.id)}
							/>
						</td>
						<td class="col-star" onclick={(e) => e.stopPropagation()}>
							<button type="button" class="star-btn" class:starred={message.isStarred} onclick={() => onToggleStar(message.id)}>
								{message.isStarred ? '★' : '☆'}
							</button>
						</td>
						<td class="col-from">{formatSender(message.from)}</td>
						<td class="col-subject">
							{message.subject || '(No Subject)'}
						</td>
						<td class="col-attach">
							{#if message.attachmentCount > 0}
								<span title="{message.attachmentCount} attachment(s)">📎</span>
							{/if}
						</td>
						<td class="col-size">{formatSize(message.size)}</td>
						<td class="col-date">{formatDate(message.receivedAt)}</td>
					</tr>
				{/each}
			{/if}
		</tbody>
	</table>

	{#if pagination && pagination.totalPages > 1}
		<div class="pagination-bar">
			<span class="pagination-info">
				Page {pagination.page} of {pagination.totalPages}
				({pagination.totalItems} messages)
			</span>
			<div class="pagination-buttons">
				<button
					type="button"
					class="hotmail-btn"
					disabled={!pagination.hasPrev}
					onclick={() => onPageChange(pagination.page - 1)}
				>
					&laquo; Prev
				</button>
				{#each Array(Math.min(pagination.totalPages, 7)) as _, idx (idx)}
					{@const pageNum = getPageNumber(pagination, idx)}
					{#if pageNum === -1}
						<span class="page-ellipsis">...</span>
					{:else}
						<button
							type="button"
							class="hotmail-btn"
							class:active={pagination.page === pageNum}
							onclick={() => onPageChange(pageNum)}
						>
							{pageNum}
						</button>
					{/if}
				{/each}
				<button
					type="button"
					class="hotmail-btn"
					disabled={!pagination.hasNext}
					onclick={() => onPageChange(pagination.page + 1)}
				>
					Next &raquo;
				</button>
			</div>
		</div>
	{/if}
</div>
