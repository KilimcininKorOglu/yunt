<script lang="ts">
	/**
	 * MessageActions Component
	 * Provides action buttons for message operations: delete, move, star, mark read/unread.
	 */

	import type { Message, Mailbox, ID } from '$lib/api/types';
	import ConfirmDialog from './ConfirmDialog.svelte';

	interface Props {
		/** The message object */
		message: Message;
		/** Available mailboxes for move operation */
		mailboxes: Mailbox[];
		/** Whether any action is in progress */
		isLoading?: boolean;
		/** Callback for back navigation */
		onBack?: () => void;
		/** Callback for delete action */
		onDelete?: () => void | Promise<void>;
		/** Callback for move action */
		onMove?: (targetMailboxId: ID) => void | Promise<void>;
		/** Callback for star/unstar action */
		onToggleStar?: () => void | Promise<void>;
		/** Callback for mark read/unread action */
		onToggleRead?: () => void | Promise<void>;
		/** Callback for mark as spam action */
		onMarkSpam?: () => void | Promise<void>;
		/** Callback for download raw action */
		onDownloadRaw?: () => void | Promise<void>;
	}

	const {
		message,
		mailboxes,
		isLoading = false,
		onBack,
		onDelete,
		onMove,
		onToggleStar,
		onToggleRead,
		onMarkSpam,
		onDownloadRaw
	}: Props = $props();

	// Dialog states
	let showDeleteConfirm = $state(false);
	let showMoveDropdown = $state(false);
	let showMoreDropdown = $state(false);

	// Filter mailboxes to exclude current mailbox
	const moveMailboxes = $derived(mailboxes.filter((mb) => mb.id !== message.mailboxId));

	/**
	 * Handle delete confirmation
	 */
	async function handleDeleteConfirm(): Promise<void> {
		showDeleteConfirm = false;
		await onDelete?.();
	}

	/**
	 * Handle move to mailbox
	 */
	async function handleMove(mailboxId: ID): Promise<void> {
		showMoveDropdown = false;
		await onMove?.(mailboxId);
	}

	/**
	 * Close dropdowns when clicking outside
	 */
	function handleClickOutside(event: MouseEvent): void {
		const target = event.target as HTMLElement;
		if (!target.closest('[data-dropdown]')) {
			showMoveDropdown = false;
			showMoreDropdown = false;
		}
	}
</script>

<svelte:window onclick={handleClickOutside} />

<div class="flex items-center justify-between border-b border-secondary-200 bg-white px-4 py-2">
	<!-- Left side: Back button -->
	<div class="flex items-center gap-2">
		{#if onBack}
			<button
				type="button"
				onclick={onBack}
				disabled={isLoading}
				class="flex items-center gap-1.5 rounded-lg px-3 py-2 text-sm font-medium text-secondary-600 transition-colors hover:bg-secondary-100 hover:text-secondary-900 disabled:opacity-50"
			>
				<svg
					class="h-4 w-4"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M10 19l-7-7m0 0l7-7m-7 7h18"
					/>
				</svg>
				Back
			</button>
		{/if}
	</div>

	<!-- Right side: Action buttons -->
	<div class="flex items-center gap-1">
		<!-- Mark as Read/Unread -->
		<button
			type="button"
			onclick={onToggleRead}
			disabled={isLoading}
			class="flex items-center gap-1.5 rounded-lg px-3 py-2 text-sm font-medium text-secondary-600 transition-colors hover:bg-secondary-100 hover:text-secondary-900 disabled:opacity-50"
			title={message.status === 'unread' ? 'Mark as read' : 'Mark as unread'}
		>
			{#if message.status === 'unread'}
				<svg
					class="h-4 w-4"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M3 19v-8.93a2 2 0 01.89-1.664l7-4.666a2 2 0 012.22 0l7 4.666A2 2 0 0121 10.07V19M3 19a2 2 0 002 2h14a2 2 0 002-2M3 19l6.75-4.5M21 19l-6.75-4.5M3 10l6.75 4.5M21 10l-6.75 4.5m0 0l-1.14.76a2 2 0 01-2.22 0l-1.14-.76"
					/>
				</svg>
				<span class="hidden sm:inline">Mark read</span>
			{:else}
				<svg
					class="h-4 w-4"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
					/>
				</svg>
				<span class="hidden sm:inline">Mark unread</span>
			{/if}
		</button>

		<!-- Star/Unstar -->
		<button
			type="button"
			onclick={onToggleStar}
			disabled={isLoading}
			class="flex items-center gap-1.5 rounded-lg px-3 py-2 text-sm font-medium transition-colors disabled:opacity-50 {message.isStarred
				? 'text-amber-500 hover:bg-amber-50'
				: 'text-secondary-600 hover:bg-secondary-100 hover:text-secondary-900'}"
			title={message.isStarred ? 'Remove star' : 'Add star'}
		>
			{#if message.isStarred}
				<svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
					<path
						fill-rule="evenodd"
						d="M10.788 3.21c.448-1.077 1.976-1.077 2.424 0l2.082 5.007 5.404.433c1.164.093 1.636 1.545.749 2.305l-4.117 3.527 1.257 5.273c.271 1.136-.964 2.033-1.96 1.425L12 18.354 7.373 21.18c-.996.608-2.231-.29-1.96-1.425l1.257-5.273-4.117-3.527c-.887-.76-.415-2.212.749-2.305l5.404-.433 2.082-5.006z"
						clip-rule="evenodd"
					/>
				</svg>
			{:else}
				<svg
					class="h-4 w-4"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
					/>
				</svg>
			{/if}
			<span class="hidden sm:inline">{message.isStarred ? 'Starred' : 'Star'}</span>
		</button>

		<!-- Move dropdown -->
		<div class="relative" data-dropdown>
			<button
				type="button"
				onclick={() => {
					showMoveDropdown = !showMoveDropdown;
					showMoreDropdown = false;
				}}
				disabled={isLoading || moveMailboxes.length === 0}
				class="flex items-center gap-1.5 rounded-lg px-3 py-2 text-sm font-medium text-secondary-600 transition-colors hover:bg-secondary-100 hover:text-secondary-900 disabled:opacity-50"
				title="Move to mailbox"
			>
				<svg
					class="h-4 w-4"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"
					/>
				</svg>
				<span class="hidden sm:inline">Move</span>
				<svg
					class="h-3 w-3"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
				</svg>
			</button>

			{#if showMoveDropdown}
				<div
					class="absolute right-0 top-full z-10 mt-1 w-48 rounded-lg border border-secondary-200 bg-white py-1 shadow-lg"
				>
					{#each moveMailboxes as mailbox (mailbox.id)}
						<button
							type="button"
							onclick={() => handleMove(mailbox.id)}
							class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-secondary-700 transition-colors hover:bg-secondary-50"
						>
							<svg
								class="h-4 w-4 text-secondary-400"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
								stroke-width="2"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
								/>
							</svg>
							<span class="truncate">{mailbox.name}</span>
						</button>
					{/each}
					{#if moveMailboxes.length === 0}
						<p class="px-4 py-2 text-sm text-secondary-500">No other mailboxes</p>
					{/if}
				</div>
			{/if}
		</div>

		<!-- More actions dropdown -->
		<div class="relative" data-dropdown>
			<button
				type="button"
				onclick={() => {
					showMoreDropdown = !showMoreDropdown;
					showMoveDropdown = false;
				}}
				disabled={isLoading}
				class="flex items-center gap-1.5 rounded-lg px-3 py-2 text-sm font-medium text-secondary-600 transition-colors hover:bg-secondary-100 hover:text-secondary-900 disabled:opacity-50"
				title="More actions"
			>
				<svg
					class="h-4 w-4"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z"
					/>
				</svg>
			</button>

			{#if showMoreDropdown}
				<div
					class="absolute right-0 top-full z-10 mt-1 w-48 rounded-lg border border-secondary-200 bg-white py-1 shadow-lg"
				>
					{#if onMarkSpam}
						<button
							type="button"
							onclick={() => {
								showMoreDropdown = false;
								onMarkSpam?.();
							}}
							class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-secondary-700 transition-colors hover:bg-secondary-50"
						>
							<svg
								class="h-4 w-4 text-secondary-400"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
								stroke-width="2"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
								/>
							</svg>
							{message.isSpam ? 'Not spam' : 'Mark as spam'}
						</button>
					{/if}
					{#if onDownloadRaw}
						<button
							type="button"
							onclick={() => {
								showMoreDropdown = false;
								onDownloadRaw?.();
							}}
							class="flex w-full items-center gap-2 px-4 py-2 text-left text-sm text-secondary-700 transition-colors hover:bg-secondary-50"
						>
							<svg
								class="h-4 w-4 text-secondary-400"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
								stroke-width="2"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
								/>
							</svg>
							Download EML
						</button>
					{/if}
				</div>
			{/if}
		</div>

		<!-- Delete button -->
		<button
			type="button"
			onclick={() => (showDeleteConfirm = true)}
			disabled={isLoading}
			class="flex items-center gap-1.5 rounded-lg px-3 py-2 text-sm font-medium text-red-600 transition-colors hover:bg-red-50 disabled:opacity-50"
			title="Delete message"
		>
			<svg
				class="h-4 w-4"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
				stroke-width="2"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
				/>
			</svg>
			<span class="hidden sm:inline">Delete</span>
		</button>
	</div>
</div>

<!-- Delete confirmation dialog -->
<ConfirmDialog
	open={showDeleteConfirm}
	title="Delete Message"
	message="Are you sure you want to delete this message? This action cannot be undone."
	confirmText="Delete"
	variant="danger"
	{isLoading}
	onConfirm={handleDeleteConfirm}
	onCancel={() => (showDeleteConfirm = false)}
/>
