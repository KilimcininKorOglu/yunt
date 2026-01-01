<script lang="ts">
	/**
	 * Sidebar Component
	 * Provides mailbox navigation with unread counts.
	 */

	import type { Mailbox, ID } from '$lib/api/types';
	import { authStore } from '$stores/auth';

	interface Props {
		mailboxes: Mailbox[];
		selectedMailboxId: ID | null;
		totalUnreadCount: number;
		onSelectMailbox: (mailboxId: ID | null) => void;
		onLogout?: () => void;
		collapsed?: boolean;
		onToggleCollapse?: () => void;
	}

	const {
		mailboxes,
		selectedMailboxId,
		totalUnreadCount,
		onSelectMailbox,
		onLogout,
		collapsed = false,
		onToggleCollapse
	}: Props = $props();

	// System folders for visual distinction
	const systemFolders = [
		{ id: null, name: 'All Mail', icon: 'inbox' },
		{ id: 'starred', name: 'Starred', icon: 'star' },
		{ id: 'spam', name: 'Spam', icon: 'spam' }
	] as const;

	function getIconPath(icon: string): string {
		switch (icon) {
			case 'inbox':
				return 'M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z';
			case 'star':
				return 'M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z';
			case 'spam':
				return 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z';
			default:
				return 'M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z';
		}
	}

	function handleSelectMailbox(id: ID | null | 'starred' | 'spam'): void {
		if (id === 'starred' || id === 'spam') {
			// These are special filters, not mailbox IDs
			// Will be handled by parent component
			onSelectMailbox(id as ID | null);
		} else {
			onSelectMailbox(id);
		}
	}
</script>

<aside
	class="flex h-full flex-col border-r border-secondary-200 bg-white transition-all duration-200 {collapsed
		? 'w-16'
		: 'w-64'}"
>
	<!-- Header -->
	<div class="flex items-center justify-between border-b border-secondary-200 p-4">
		{#if !collapsed}
			<h1 class="text-xl font-bold text-primary-600">Yunt</h1>
		{/if}
		{#if onToggleCollapse}
			<button
				type="button"
				onclick={onToggleCollapse}
				class="rounded-lg p-1.5 text-secondary-500 hover:bg-secondary-100 hover:text-secondary-700"
				aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
			>
				<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					{#if collapsed}
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M13 5l7 7-7 7M5 5l7 7-7 7"
						/>
					{:else}
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M11 19l-7-7 7-7m8 14l-7-7 7-7"
						/>
					{/if}
				</svg>
			</button>
		{/if}
	</div>

	<!-- Navigation -->
	<nav class="flex-1 overflow-y-auto p-3">
		<!-- System Folders -->
		<div class="mb-4 space-y-1">
			{#each systemFolders as folder (folder.id)}
				{@const isSelected =
					folder.id === null
						? selectedMailboxId === null
						: selectedMailboxId === folder.id}
				<button
					type="button"
					onclick={() => handleSelectMailbox(folder.id)}
					class="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left text-sm font-medium transition-colors {isSelected
						? 'bg-primary-50 text-primary-700'
						: 'text-secondary-700 hover:bg-secondary-50'}"
					title={collapsed ? folder.name : undefined}
				>
					<svg
						class="h-5 w-5 flex-shrink-0"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d={getIconPath(folder.icon)}
						/>
					</svg>
					{#if !collapsed}
						<span class="flex-1 truncate">{folder.name}</span>
						{#if folder.id === null && totalUnreadCount > 0}
							<span
								class="ml-auto rounded-full bg-primary-100 px-2 py-0.5 text-xs font-medium text-primary-700"
							>
								{totalUnreadCount > 99 ? '99+' : totalUnreadCount}
							</span>
						{/if}
					{/if}
				</button>
			{/each}
		</div>

		<!-- Mailboxes Section -->
		{#if mailboxes.length > 0}
			<div class="mb-2 border-t border-secondary-200 pt-4">
				{#if !collapsed}
					<h2
						class="mb-2 px-3 text-xs font-semibold uppercase tracking-wider text-secondary-500"
					>
						Mailboxes
					</h2>
				{/if}
				<div class="space-y-1">
					{#each mailboxes as mailbox (mailbox.id)}
						{@const isSelected = selectedMailboxId === mailbox.id}
						<button
							type="button"
							onclick={() => handleSelectMailbox(mailbox.id)}
							class="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left text-sm font-medium transition-colors {isSelected
								? 'bg-primary-50 text-primary-700'
								: 'text-secondary-700 hover:bg-secondary-50'}"
							title={collapsed
								? `${mailbox.name} (${mailbox.unreadCount} unread)`
								: undefined}
						>
							<svg
								class="h-5 w-5 flex-shrink-0"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
								/>
							</svg>
							{#if !collapsed}
								<span class="flex-1 truncate">{mailbox.name}</span>
								{#if mailbox.unreadCount > 0}
									<span
										class="ml-auto rounded-full bg-secondary-100 px-2 py-0.5 text-xs font-medium text-secondary-700"
									>
										{mailbox.unreadCount > 99 ? '99+' : mailbox.unreadCount}
									</span>
								{/if}
							{/if}
						</button>
					{/each}
				</div>
			</div>
		{/if}
	</nav>

	<!-- User Section -->
	<div class="border-t border-secondary-200 p-3">
		<div
			class="flex items-center gap-3 rounded-lg px-3 py-2 {collapsed
				? 'justify-center'
				: 'justify-between'}"
		>
			{#if !collapsed}
				<div class="min-w-0 flex-1">
					<p class="truncate text-sm font-medium text-secondary-900">
						{authStore.user?.displayName || authStore.user?.username || 'User'}
					</p>
					<p class="truncate text-xs text-secondary-500">
						{authStore.user?.email || ''}
					</p>
				</div>
			{/if}
			<button
				type="button"
				onclick={onLogout}
				class="rounded-lg p-1.5 text-secondary-500 hover:bg-secondary-100 hover:text-secondary-700"
				title="Logout"
				aria-label="Logout"
			>
				<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"
					/>
				</svg>
			</button>
		</div>
	</div>
</aside>
