<script lang="ts">
	/**
	 * MessageList Component
	 * Displays a list of recent messages with sender, subject, preview, and timestamp.
	 */

	import type { MessageSummary } from '$lib/api';

	interface Props {
		/** Array of message summaries to display */
		messages: MessageSummary[];
		/** Whether data is loading */
		loading?: boolean;
		/** Number of skeleton items to show while loading */
		skeletonCount?: number;
		/** Callback when a message is clicked */
		onMessageClick?: (message: MessageSummary) => void;
		/** Maximum number of messages to show */
		maxItems?: number;
		/** Whether to show the empty state */
		showEmptyState?: boolean;
	}

	const {
		messages = [],
		loading = false,
		skeletonCount = 5,
		onMessageClick,
		maxItems = 10,
		showEmptyState = true
	}: Props = $props();

	const displayMessages = $derived(messages.slice(0, maxItems));

	function formatRelativeTime(timestamp: string): string {
		const date = new Date(timestamp);
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffMins = Math.floor(diffMs / 60000);
		const diffHours = Math.floor(diffMs / 3600000);
		const diffDays = Math.floor(diffMs / 86400000);

		if (diffMins < 1) return 'Just now';
		if (diffMins < 60) return `${diffMins}m ago`;
		if (diffHours < 24) return `${diffHours}h ago`;
		if (diffDays < 7) return `${diffDays}d ago`;
		return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
	}

	function handleClick(message: MessageSummary): void {
		if (onMessageClick) {
			onMessageClick(message);
		}
	}

	function handleKeyDown(event: KeyboardEvent, message: MessageSummary): void {
		if (event.key === 'Enter' || event.key === ' ') {
			event.preventDefault();
			handleClick(message);
		}
	}
</script>

<div class="card">
	<div class="border-b border-secondary-200 px-5 py-4">
		<h3 class="text-base font-semibold text-secondary-900">Recent Messages</h3>
	</div>
	<div class="divide-y divide-secondary-100">
		{#if loading}
			{#each Array(skeletonCount) as _, i (i)}
				<div class="flex items-start gap-3 px-5 py-4">
					<div
						class="h-10 w-10 flex-shrink-0 animate-pulse rounded-full bg-secondary-200"
					></div>
					<div class="min-w-0 flex-1">
						<div class="flex items-center justify-between gap-2">
							<div class="h-4 w-32 animate-pulse rounded bg-secondary-200"></div>
							<div class="h-3 w-12 animate-pulse rounded bg-secondary-200"></div>
						</div>
						<div class="mt-1.5 h-4 w-48 animate-pulse rounded bg-secondary-200"></div>
						<div class="mt-1.5 h-3 w-full animate-pulse rounded bg-secondary-200"></div>
					</div>
				</div>
			{/each}
		{:else if displayMessages.length === 0 && showEmptyState}
			<div class="px-5 py-12 text-center">
				<svg
					class="mx-auto h-12 w-12 text-secondary-300"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="1.5"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M21.75 9v.906a2.25 2.25 0 01-1.183 1.981l-6.478 3.488M2.25 9v.906a2.25 2.25 0 001.183 1.981l6.478 3.488m8.839 2.51l-4.66-2.51m0 0l-1.023-.55a2.25 2.25 0 00-2.134 0l-1.022.55m0 0l-4.661 2.51m16.5 1.615a2.25 2.25 0 01-2.25 2.25h-15a2.25 2.25 0 01-2.25-2.25V8.844a2.25 2.25 0 011.183-1.98l7.5-4.04a2.25 2.25 0 012.134 0l7.5 4.04a2.25 2.25 0 011.183 1.98V19.5z"
					/>
				</svg>
				<p class="mt-3 text-sm font-medium text-secondary-500">No messages yet</p>
				<p class="mt-1 text-sm text-secondary-400">
					Messages will appear here when they arrive.
				</p>
			</div>
		{:else}
			{#each displayMessages as message (message.id)}
				<div
					class="flex cursor-pointer items-start gap-3 px-5 py-4 transition-colors hover:bg-secondary-50"
					role="button"
					tabindex="0"
					onclick={() => handleClick(message)}
					onkeydown={(e) => handleKeyDown(e, message)}
				>
					<div
						class={`flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full ${message.status === 'unread' ? 'bg-primary-100 text-primary-700' : 'bg-secondary-100 text-secondary-600'}`}
					>
						<span class="text-sm font-medium">
							{(message.from.name || message.from.address).charAt(0).toUpperCase()}
						</span>
					</div>
					<div class="min-w-0 flex-1">
						<div class="flex items-center justify-between gap-2">
							<p
								class={`truncate text-sm ${message.status === 'unread' ? 'font-semibold text-secondary-900' : 'font-medium text-secondary-700'}`}
							>
								{message.from.name || message.from.address}
							</p>
							<span class="flex-shrink-0 text-xs text-secondary-400">
								{formatRelativeTime(message.receivedAt)}
							</span>
						</div>
						<p
							class={`mt-0.5 truncate text-sm ${message.status === 'unread' ? 'font-medium text-secondary-800' : 'text-secondary-600'}`}
						>
							{message.subject || '(No subject)'}
						</p>
						<p class="mt-0.5 truncate text-xs text-secondary-400">
							{message.preview}
						</p>
					</div>
					<div class="flex flex-shrink-0 items-center gap-2">
						{#if message.status === 'unread'}
							<span class="h-2 w-2 rounded-full bg-primary-500"></span>
						{/if}
						{#if message.hasAttachments}
							<svg
								class="h-4 w-4 text-secondary-400"
								viewBox="0 0 24 24"
								fill="none"
								stroke="currentColor"
								stroke-width="2"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d="M18.375 12.739l-7.693 7.693a4.5 4.5 0 01-6.364-6.364l10.94-10.94A3 3 0 1119.5 7.372L8.552 18.32m.009-.01l-.01.01m5.699-9.941l-7.81 7.81a1.5 1.5 0 002.112 2.13"
								/>
							</svg>
						{/if}
						{#if message.isStarred}
							<svg
								class="h-4 w-4 text-amber-400"
								viewBox="0 0 24 24"
								fill="currentColor"
							>
								<path
									fill-rule="evenodd"
									d="M10.788 3.21c.448-1.077 1.976-1.077 2.424 0l2.082 5.007 5.404.433c1.164.093 1.636 1.545.749 2.305l-4.117 3.527 1.257 5.273c.271 1.136-.964 2.033-1.96 1.425L12 18.354 7.373 21.18c-.996.608-2.231-.29-1.96-1.425l1.257-5.273-4.117-3.527c-.887-.76-.415-2.212.749-2.305l5.404-.433 2.082-5.006z"
									clip-rule="evenodd"
								/>
							</svg>
						{/if}
					</div>
				</div>
			{/each}
		{/if}
	</div>
	{#if !loading && displayMessages.length > 0}
		<div class="border-t border-secondary-200 px-5 py-3">
			<a
				href="/messages"
				class="text-sm font-medium text-primary-600 transition-colors hover:text-primary-700"
			>
				View all messages &rarr;
			</a>
		</div>
	{/if}
</div>
