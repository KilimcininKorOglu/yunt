<script lang="ts">
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
			setTimeout(() => {
				showDeleteConfirm = false;
			}, 3000);
		}
	}
</script>

<div class="bulk-actions">
	<span class="bulk-count">{selectedCount} selected</span>
	<span class="toolbar-sep">|</span>
	<button type="button" {disabled} onclick={onMarkAsRead} class="hotmail-btn">Mark Read</button>
	<button type="button" {disabled} onclick={onMarkAsUnread} class="hotmail-btn">Mark Unread</button>
	<button type="button" {disabled} onclick={onStar} class="hotmail-btn">Star</button>
	<button type="button" {disabled} onclick={onUnstar} class="hotmail-btn">Unstar</button>

	{#if onMove && mailboxes.length > 0}
		<div class="move-dropdown">
			<button type="button" {disabled} onclick={() => (showMoveMenu = !showMoveMenu)} class="hotmail-btn">
				Put in Folder ▾
			</button>
			{#if showMoveMenu}
				<button type="button" class="dropdown-backdrop" onclick={() => (showMoveMenu = false)}></button>
				<div class="dropdown-menu">
					{#each mailboxes as mailbox (mailbox.id)}
						<button type="button" class="dropdown-item" onclick={() => handleMove(mailbox.id)}>
							📁 {mailbox.name}
						</button>
					{/each}
				</div>
			{/if}
		</div>
	{/if}

	<span class="toolbar-sep">|</span>
	<button
		type="button"
		{disabled}
		onclick={handleDelete}
		class="hotmail-btn delete-btn"
		class:confirm={showDeleteConfirm}
	>
		{showDeleteConfirm ? 'Confirm Delete?' : 'Delete'}
	</button>
</div>
