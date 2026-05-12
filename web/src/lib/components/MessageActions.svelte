<script lang="ts">
	import type { Message, Mailbox, ID } from '$lib/api/types';
	import ConfirmDialog from './ConfirmDialog.svelte';

	interface Props {
		message: Message;
		mailboxes: Mailbox[];
		isLoading?: boolean;
		onBack?: () => void;
		onDelete?: () => void | Promise<void>;
		onMove?: (targetMailboxId: ID) => void | Promise<void>;
		onToggleStar?: () => void | Promise<void>;
		onToggleRead?: () => void | Promise<void>;
		onMarkSpam?: () => void | Promise<void>;
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

	let showDeleteConfirm = $state(false);
	let showMoveDropdown = $state(false);

	const moveMailboxes = $derived(mailboxes.filter((mb) => mb.id !== message.mailboxId));

	async function handleDeleteConfirm(): Promise<void> {
		showDeleteConfirm = false;
		await onDelete?.();
	}

	async function handleMove(mailboxId: ID): Promise<void> {
		showMoveDropdown = false;
		await onMove?.(mailboxId);
	}
</script>

<div class="toolbar">
	<div class="toolbar-left">
		{#if onBack}
			<button type="button" class="hotmail-btn" onclick={onBack} disabled={isLoading}>
				&laquo; Back to Inbox
			</button>
		{/if}
		<span class="toolbar-sep">|</span>
		<a href="/compose?replyTo={message.id}" class="hotmail-btn">Reply</a>
		<a href="/compose?forward={message.id}" class="hotmail-btn">Forward</a>
		<span class="toolbar-sep">|</span>
		<button type="button" class="hotmail-btn" onclick={() => (showDeleteConfirm = true)} disabled={isLoading}>
			Delete
		</button>
		<button type="button" class="hotmail-btn" onclick={onMarkSpam} disabled={isLoading}>
			{message.isSpam ? 'Not Junk' : 'Junk'}
		</button>
		<div class="move-dropdown">
			<button type="button" class="hotmail-btn" onclick={() => (showMoveDropdown = !showMoveDropdown)} disabled={isLoading || moveMailboxes.length === 0}>
				Put in Folder ▾
			</button>
			{#if showMoveDropdown}
				<button type="button" class="dropdown-backdrop" onclick={() => (showMoveDropdown = false)}></button>
				<div class="dropdown-menu">
					{#each moveMailboxes as mailbox (mailbox.id)}
						<button type="button" class="dropdown-item" onclick={() => handleMove(mailbox.id)}>
							📁 {mailbox.name}
						</button>
					{/each}
				</div>
			{/if}
		</div>
	</div>
	<div class="toolbar-right">
		<button type="button" class="hotmail-btn" onclick={onToggleStar} disabled={isLoading}>
			{message.isStarred ? '★ Starred' : '☆ Star'}
		</button>
		<button type="button" class="hotmail-btn" onclick={onToggleRead} disabled={isLoading}>
			{message.status === 'unread' ? 'Mark Read' : 'Mark Unread'}
		</button>
		<button type="button" class="hotmail-btn" onclick={onDownloadRaw} disabled={isLoading}>
			Download .eml
		</button>
	</div>
</div>

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
