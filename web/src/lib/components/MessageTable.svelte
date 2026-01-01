<script module lang="ts">
	/**
	 * Calculate page number for pagination display
	 */
	function getPageNumber(pagination: PaginationInfo, idx: number): number {
		const { page, totalPages } = pagination;
		const maxVisible = 7;

		if (totalPages <= maxVisible) {
			return idx + 1;
		}

		// Always show first and last, with ellipsis
		if (idx === 0) return 1;
		if (idx === maxVisible - 1) return totalPages;

		// Show pages around current
		const start = Math.max(2, page - 2);
		const end = Math.min(totalPages - 1, page + 2);
		const range = [];

		for (let i = start; i <= end; i++) {
			range.push(i);
		}

		// Pad to fill gaps
		while (range.length < maxVisible - 2) {
			if (range[0] > 2) {
				range.unshift(range[0] - 1);
			} else if (range[range.length - 1] < totalPages - 1) {
				range.push(range[range.length - 1] + 1);
			} else {
				break;
			}
		}

		// Add ellipsis indicators (-1)
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
	/**
	 * MessageTable Component
	 * Displays a sortable, selectable table of messages.
	 */

	import type { Message, ID, SortOrder, PaginationInfo } from '$lib/api/types';
	import type { MessageSortField } from '$stores/messages';

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

	// Helper to format date
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

	// Helper to format sender
	function formatSender(from: { name?: string; address: string }): string {
		return from.name || from.address.split('@')[0];
	}

	// Helper to format size
	function formatSize(bytes: number): string {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}

	// Sort icon component
	function getSortIcon(field: MessageSortField): string {
		if (sortField !== field) return '';
		return sortOrder === 'asc' ? '↑' : '↓';
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

<div class="flex flex-col">
	<!-- Table Container -->
	<div class="overflow-x-auto">
		<table class="min-w-full divide-y divide-secondary-200">
			<thead class="bg-secondary-50">
				<tr>
					<!-- Checkbox Column -->
					<th scope="col" class="w-10 px-3 py-3">
						<input
							type="checkbox"
							checked={allSelected && messages.length > 0}
							onchange={onToggleSelectAll}
							class="h-4 w-4 rounded border-secondary-300 text-primary-600 focus:ring-primary-500"
							aria-label="Select all messages"
						/>
					</th>

					<!-- Star Column -->
					<th scope="col" class="w-10 px-2 py-3">
						<span class="sr-only">Star</span>
					</th>

					<!-- From Column -->
					<th scope="col" class="px-3 py-3 text-left">
						<button
							type="button"
							onclick={() => onSort('from')}
							class="flex items-center gap-1 text-xs font-semibold uppercase tracking-wider text-secondary-500 hover:text-secondary-700"
						>
							From
							<span class="text-primary-600">{getSortIcon('from')}</span>
						</button>
					</th>

					<!-- Subject Column -->
					<th scope="col" class="min-w-[200px] px-3 py-3 text-left">
						<button
							type="button"
							onclick={() => onSort('subject')}
							class="flex items-center gap-1 text-xs font-semibold uppercase tracking-wider text-secondary-500 hover:text-secondary-700"
						>
							Subject
							<span class="text-primary-600">{getSortIcon('subject')}</span>
						</button>
					</th>

					<!-- Attachments Column -->
					<th scope="col" class="w-10 px-2 py-3">
						<span class="sr-only">Attachments</span>
					</th>

					<!-- Size Column -->
					<th scope="col" class="w-20 px-3 py-3 text-right">
						<button
							type="button"
							onclick={() => onSort('size')}
							class="flex w-full items-center justify-end gap-1 text-xs font-semibold uppercase tracking-wider text-secondary-500 hover:text-secondary-700"
						>
							Size
							<span class="text-primary-600">{getSortIcon('size')}</span>
						</button>
					</th>

					<!-- Date Column -->
					<th scope="col" class="w-28 px-3 py-3 text-right">
						<button
							type="button"
							onclick={() => onSort('receivedAt')}
							class="flex w-full items-center justify-end gap-1 text-xs font-semibold uppercase tracking-wider text-secondary-500 hover:text-secondary-700"
						>
							Date
							<span class="text-primary-600">{getSortIcon('receivedAt')}</span>
						</button>
					</th>
				</tr>
			</thead>
			<tbody class="divide-y divide-secondary-100 bg-white">
				{#if isLoading}
					<!-- Loading State -->
					{#each Array(5) as _, i (i)}
						<tr class="animate-pulse">
							<td class="px-3 py-4">
								<div class="h-4 w-4 rounded bg-secondary-200"></div>
							</td>
							<td class="px-2 py-4">
								<div class="h-4 w-4 rounded bg-secondary-200"></div>
							</td>
							<td class="px-3 py-4">
								<div class="h-4 w-24 rounded bg-secondary-200"></div>
							</td>
							<td class="px-3 py-4">
								<div class="h-4 w-48 rounded bg-secondary-200"></div>
							</td>
							<td class="px-2 py-4">
								<div class="h-4 w-4 rounded bg-secondary-200"></div>
							</td>
							<td class="px-3 py-4">
								<div class="ml-auto h-4 w-12 rounded bg-secondary-200"></div>
							</td>
							<td class="px-3 py-4">
								<div class="ml-auto h-4 w-16 rounded bg-secondary-200"></div>
							</td>
						</tr>
					{/each}
				{:else if messages.length === 0}
					<!-- Empty State -->
					<tr>
						<td colspan="7" class="px-6 py-12 text-center">
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
										d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
									/>
								</svg>
								<p class="text-sm text-secondary-500">No messages found</p>
							</div>
						</td>
					</tr>
				{:else}
					<!-- Message Rows -->
					{#each messages as message (message.id)}
						<tr
							class="cursor-pointer transition-colors hover:bg-secondary-50 {message.status ===
							'unread'
								? 'font-semibold'
								: ''} {selectedIds.has(message.id) ? 'bg-primary-50' : ''}"
							onclick={() => handleRowClick(message)}
							onkeydown={(e) => handleRowKeydown(e, message)}
							role="button"
							tabindex="0"
						>
							<!-- Checkbox -->
							<td class="px-3 py-3" onclick={(e) => e.stopPropagation()}>
								<input
									type="checkbox"
									checked={selectedIds.has(message.id)}
									onchange={() => onToggleSelect(message.id)}
									class="h-4 w-4 rounded border-secondary-300 text-primary-600 focus:ring-primary-500"
									aria-label="Select message"
								/>
							</td>

							<!-- Star -->
							<td class="px-2 py-3" onclick={(e) => e.stopPropagation()}>
								<button
									type="button"
									onclick={() => onToggleStar(message.id)}
									class="text-secondary-300 hover:text-yellow-400 {message.isStarred
										? 'text-yellow-400'
										: ''}"
									aria-label={message.isStarred ? 'Remove star' : 'Add star'}
								>
									<svg
										class="h-5 w-5"
										fill={message.isStarred ? 'currentColor' : 'none'}
										viewBox="0 0 24 24"
										stroke="currentColor"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
										/>
									</svg>
								</button>
							</td>

							<!-- From -->
							<td class="max-w-[150px] truncate px-3 py-3 text-sm text-secondary-900">
								{formatSender(message.from)}
							</td>

							<!-- Subject -->
							<td class="px-3 py-3">
								<div class="flex items-center gap-2">
									{#if message.status === 'unread'}
										<span
											class="h-2 w-2 flex-shrink-0 rounded-full bg-primary-500"
											aria-label="Unread"
										></span>
									{/if}
									<span
										class="truncate text-sm {message.status === 'unread'
											? 'text-secondary-900'
											: 'text-secondary-700'}"
									>
										{message.subject || '(No Subject)'}
									</span>
								</div>
							</td>

							<!-- Attachments -->
							<td class="px-2 py-3">
								{#if message.attachmentCount > 0}
									<span
										class="text-secondary-400"
										title="{message.attachmentCount} attachment(s)"
									>
										<svg
											class="h-4 w-4"
											fill="none"
											viewBox="0 0 24 24"
											stroke="currentColor"
										>
											<path
												stroke-linecap="round"
												stroke-linejoin="round"
												stroke-width="2"
												d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13"
											/>
										</svg>
									</span>
								{/if}
							</td>

							<!-- Size -->
							<td class="px-3 py-3 text-right text-sm text-secondary-500">
								{formatSize(message.size)}
							</td>

							<!-- Date -->
							<td class="px-3 py-3 text-right text-sm text-secondary-500">
								{formatDate(message.receivedAt)}
							</td>
						</tr>
					{/each}
				{/if}
			</tbody>
		</table>
	</div>

	<!-- Pagination -->
	{#if pagination && pagination.totalPages > 1}
		<div
			class="flex items-center justify-between border-t border-secondary-200 bg-white px-4 py-3"
		>
			<div class="flex flex-1 items-center justify-between sm:hidden">
				<button
					type="button"
					disabled={!pagination.hasPrev}
					onclick={() => onPageChange(pagination.page - 1)}
					class="btn-secondary relative"
				>
					Previous
				</button>
				<span class="text-sm text-secondary-700">
					Page {pagination.page} of {pagination.totalPages}
				</span>
				<button
					type="button"
					disabled={!pagination.hasNext}
					onclick={() => onPageChange(pagination.page + 1)}
					class="btn-secondary relative"
				>
					Next
				</button>
			</div>
			<div class="hidden sm:flex sm:flex-1 sm:items-center sm:justify-between">
				<div>
					<p class="text-sm text-secondary-700">
						Showing
						<span class="font-medium"
							>{(pagination.page - 1) * pagination.pageSize + 1}</span
						>
						to
						<span class="font-medium">
							{Math.min(pagination.page * pagination.pageSize, pagination.totalItems)}
						</span>
						of
						<span class="font-medium">{pagination.totalItems}</span>
						results
					</p>
				</div>
				<nav
					class="isolate inline-flex -space-x-px rounded-md shadow-sm"
					aria-label="Pagination"
				>
					<!-- Previous Button -->
					<button
						type="button"
						disabled={!pagination.hasPrev}
						onclick={() => onPageChange(pagination.page - 1)}
						class="relative inline-flex items-center rounded-l-md border border-secondary-300 bg-white px-2 py-2 text-sm font-medium text-secondary-500 hover:bg-secondary-50 disabled:cursor-not-allowed disabled:opacity-50"
					>
						<span class="sr-only">Previous</span>
						<svg
							class="h-5 w-5"
							viewBox="0 0 20 20"
							fill="currentColor"
							aria-hidden="true"
						>
							<path
								fill-rule="evenodd"
								d="M12.79 5.23a.75.75 0 01-.02 1.06L8.832 10l3.938 3.71a.75.75 0 11-1.04 1.08l-4.5-4.25a.75.75 0 010-1.08l4.5-4.25a.75.75 0 011.06.02z"
								clip-rule="evenodd"
							/>
						</svg>
					</button>

					<!-- Page Numbers -->
					{#each Array(Math.min(pagination.totalPages, 7)) as _, idx (idx)}
						{@const pageNum = getPageNumber(pagination, idx)}
						{#if pageNum === -1}
							<span
								class="relative inline-flex items-center border border-secondary-300 bg-white px-4 py-2 text-sm font-medium text-secondary-700"
							>
								...
							</span>
						{:else}
							<button
								type="button"
								onclick={() => onPageChange(pageNum)}
								class="relative inline-flex items-center border px-4 py-2 text-sm font-medium {pagination.page ===
								pageNum
									? 'z-10 border-primary-500 bg-primary-50 text-primary-600'
									: 'border-secondary-300 bg-white text-secondary-500 hover:bg-secondary-50'}"
							>
								{pageNum}
							</button>
						{/if}
					{/each}

					<!-- Next Button -->
					<button
						type="button"
						disabled={!pagination.hasNext}
						onclick={() => onPageChange(pagination.page + 1)}
						class="relative inline-flex items-center rounded-r-md border border-secondary-300 bg-white px-2 py-2 text-sm font-medium text-secondary-500 hover:bg-secondary-50 disabled:cursor-not-allowed disabled:opacity-50"
					>
						<span class="sr-only">Next</span>
						<svg
							class="h-5 w-5"
							viewBox="0 0 20 20"
							fill="currentColor"
							aria-hidden="true"
						>
							<path
								fill-rule="evenodd"
								d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z"
								clip-rule="evenodd"
							/>
						</svg>
					</button>
				</nav>
			</div>
		</div>
	{/if}
</div>
