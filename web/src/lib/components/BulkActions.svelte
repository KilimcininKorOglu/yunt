<script lang="ts">
	/**
	 * BulkActions Component
	 * Provides bulk action buttons for selected messages.
	 */

	import type { ID, Mailbox } from '$lib/api/types';

	interface Props {
		selectedCount: number;
		mailboxes?: Mailbox[];
		onMarkAsRead: () => void;
		onMarkAsUnread: () => void;
		onDelete: () => void;
		onStar: () => void;
		onUnstar: () => void;
		onMove?: (targetMailboxId: ID) => void;
		disabled?: boolean;
	}

	const {
		selectedCount,
		mailboxes = [],
		onMarkAsRead,
		onMarkAsUnread,
		onDelete,
		onStar,
		onUnstar,
		onMove,
		disabled = false
	}: Props = $props();

	let showMoveMenu = $state(false);
	let showDeleteConfirm = $state(false);

	function handleMove(mailboxId: ID): void {
		onMove?.(mailboxId);
		showMoveMenu = false;
	}

	function handleDelete(): void {
		if (showDeleteConfirm) {
			onDelete();
			showDeleteConfirm = false;
		} else {
			showDeleteConfirm = true;
			// Auto-hide confirmation after 3 seconds
			setTimeout(() => {
				showDeleteConfirm = false;
			}, 3000);
		}
	}

	function closeMoveMenu(): void {
		showMoveMenu = false;
	}
</script>

<div class="flex items-center gap-2">
	<span class="text-sm font-medium text-secondary-700">
		{selectedCount} selected
	</span>

	<div class="h-4 w-px bg-secondary-300"></div>

	<!-- Mark as Read -->
	<button
		type="button"
		{disabled}
		onclick={onMarkAsRead}
		class="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-sm text-secondary-700 hover:bg-secondary-100"
		title="Mark as read"
	>
		<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				stroke-width="2"
				d="M3 19v-8.93a2 2 0 01.89-1.664l7-4.666a2 2 0 012.22 0l7 4.666A2 2 0 0121 10.07V19M3 19a2 2 0 002 2h14a2 2 0 002-2M3 19l6.75-4.5M21 19l-6.75-4.5M3 10l6.75 4.5M21 10l-6.75 4.5m0 0l-1.14.76a2 2 0 01-2.22 0l-1.14-.76"
			/>
		</svg>
		<span class="hidden sm:inline">Read</span>
	</button>

	<!-- Mark as Unread -->
	<button
		type="button"
		{disabled}
		onclick={onMarkAsUnread}
		class="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-sm text-secondary-700 hover:bg-secondary-100"
		title="Mark as unread"
	>
		<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				stroke-width="2"
				d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
			/>
		</svg>
		<span class="hidden sm:inline">Unread</span>
	</button>

	<!-- Star -->
	<button
		type="button"
		{disabled}
		onclick={onStar}
		class="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-sm text-secondary-700 hover:bg-secondary-100"
		title="Star messages"
	>
		<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				stroke-width="2"
				d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
			/>
		</svg>
		<span class="hidden sm:inline">Star</span>
	</button>

	<!-- Unstar -->
	<button
		type="button"
		{disabled}
		onclick={onUnstar}
		class="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-sm text-secondary-700 hover:bg-secondary-100"
		title="Remove star"
	>
		<svg class="h-4 w-4" fill="currentColor" viewBox="0 0 24 24" stroke="currentColor">
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				stroke-width="2"
				d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
			/>
		</svg>
		<span class="hidden sm:inline">Unstar</span>
	</button>

	<!-- Move to Mailbox -->
	{#if onMove && mailboxes.length > 0}
		<div class="relative">
			<button
				type="button"
				{disabled}
				onclick={() => (showMoveMenu = !showMoveMenu)}
				class="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-sm text-secondary-700 hover:bg-secondary-100"
				title="Move to mailbox"
			>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"
					/>
				</svg>
				<span class="hidden sm:inline">Move</span>
				<svg class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M19 9l-7 7-7-7"
					/>
				</svg>
			</button>

			{#if showMoveMenu}
				<!-- Backdrop -->
				<button
					type="button"
					class="fixed inset-0 z-10"
					onclick={closeMoveMenu}
					aria-label="Close menu"
				></button>

				<!-- Dropdown Menu -->
				<div
					class="absolute left-0 z-20 mt-1 w-48 rounded-lg border border-secondary-200 bg-white py-1 shadow-lg"
				>
					{#each mailboxes as mailbox (mailbox.id)}
						<button
							type="button"
							onclick={() => handleMove(mailbox.id)}
							class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm text-secondary-700 hover:bg-secondary-50"
						>
							<svg class="h-4 w-4 text-secondary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
								/>
							</svg>
							{mailbox.name}
						</button>
					{/each}
				</div>
			{/if}
		</div>
	{/if}

	<div class="h-4 w-px bg-secondary-300"></div>

	<!-- Delete -->
	<button
		type="button"
		{disabled}
		onclick={handleDelete}
		class="inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-sm transition-colors {showDeleteConfirm
			? 'bg-red-600 text-white'
			: 'text-red-600 hover:bg-red-50'}"
		title={showDeleteConfirm ? 'Click again to confirm' : 'Delete messages'}
	>
		<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				stroke-width="2"
				d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
			/>
		</svg>
		<span class="hidden sm:inline">
			{showDeleteConfirm ? 'Confirm?' : 'Delete'}
		</span>
	</button>
</div>
