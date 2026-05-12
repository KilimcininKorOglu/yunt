<script lang="ts">
	import type { Mailbox, ID } from '$lib/api/types';

	interface Props {
		mailboxes: Mailbox[];
		selectedMailboxId: ID | null;
		totalUnreadCount: number;
		onSelectMailbox: (mailboxId: ID | null) => void;
		collapsed?: boolean;
		onToggleCollapse?: () => void;
	}

	const {
		mailboxes,
		selectedMailboxId,
		totalUnreadCount,
		onSelectMailbox,
		collapsed = false,
		onToggleCollapse
	}: Props = $props();

	const systemFolders = [
		{ id: null, name: 'Inbox', icon: '📥' },
		{ id: 'starred', name: 'Starred', icon: '⭐' },
		{ id: 'spam', name: 'Junk E-Mail', icon: '⚠️' }
	] as const;

	function handleSelectMailbox(id: ID | null | 'starred' | 'spam'): void {
		if (id === 'starred' || id === 'spam') {
			onSelectMailbox(id as ID | null);
		} else {
			onSelectMailbox(id);
		}
	}

	const totalMessages = $derived(
		mailboxes.reduce((sum, m) => sum + (m.messageCount || 0), 0)
	);

	const storagePercent = $derived(Math.min(Math.round((totalMessages / 1000) * 100), 100));
</script>

<aside class="sidebar" class:collapsed>
	<div class="sidebar-header">
		{#if !collapsed}
			<span class="sidebar-title">Folders</span>
		{/if}
		{#if onToggleCollapse}
			<button type="button" class="collapse-btn" onclick={onToggleCollapse} title={collapsed ? 'Expand' : 'Collapse'}>
				{collapsed ? '»' : '«'}
			</button>
		{/if}
	</div>

	{#if !collapsed}
		<div class="folder-list">
			{#each systemFolders as folder (folder.id)}
				{@const isSelected = folder.id === null ? selectedMailboxId === null : selectedMailboxId === folder.id}
				<button
					type="button"
					class="folder-item"
					class:active={isSelected}
					onclick={() => handleSelectMailbox(folder.id)}
				>
					<span class="folder-icon">{folder.icon}</span>
					<span class="folder-name">{folder.name}</span>
					{#if folder.id === null && totalUnreadCount > 0}
						<span class="folder-count">({totalUnreadCount})</span>
					{/if}
				</button>
			{/each}

			{#if mailboxes.length > 0}
				<div class="folder-separator"></div>
				{#each mailboxes as mailbox (mailbox.id)}
					{@const isSelected = selectedMailboxId === mailbox.id}
					<button
						type="button"
						class="folder-item"
						class:active={isSelected}
						onclick={() => handleSelectMailbox(mailbox.id)}
					>
						<span class="folder-icon">📁</span>
						<span class="folder-name">{mailbox.name}</span>
						{#if mailbox.unreadCount > 0}
							<span class="folder-count">({mailbox.unreadCount})</span>
						{/if}
					</button>
				{/each}
			{/if}
		</div>

		<div class="sidebar-footer">
			<a href="/settings" class="manage-link">Manage Folders</a>
			<div class="storage-bar">
				<div class="storage-label">
					<span>Mail</span>
					<span>{storagePercent}% used</span>
				</div>
				<div class="storage-track">
					<div class="storage-fill" style="width:{storagePercent}%"></div>
				</div>
				<div class="storage-info">{totalMessages} messages</div>
			</div>
		</div>
	{/if}
</aside>
