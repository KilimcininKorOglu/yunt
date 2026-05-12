<script lang="ts">
	import type { MessageSummary } from '$lib/api';

	interface Props {
		messages: MessageSummary[];
		loading?: boolean;
		onMessageClick?: (message: MessageSummary) => void;
		maxItems?: number;
	}

	const {
		messages = [],
		loading = false,
		onMessageClick,
		maxItems = 5
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
		onMessageClick?.(message);
	}
</script>

<div class="info-box">
	<div class="info-box-header">Recent Messages</div>
	<div class="info-box-body" style="padding:0;">
		{#if loading}
			{#each Array(3) as _, i (i)}
				<div style="padding:6px 8px;border-bottom:1px solid var(--border-light);">
					<div class="loading-skeleton" style="width:60%;margin-bottom:4px;"></div>
					<div class="loading-skeleton" style="width:80%;"></div>
				</div>
			{/each}
		{:else if displayMessages.length === 0}
			<p style="padding:10px;color:var(--text-muted);text-align:center;">No recent messages</p>
		{:else}
			<table class="msg-table"><tbody>
				{#each displayMessages as message (message.id)}
					<tr
						class:unread={message.status === 'unread'}
						onclick={() => handleClick(message)}
						role="button"
						tabindex="0"
						onkeydown={(e) => { if (e.key === 'Enter') handleClick(message); }}
					>
						<td class="col-from" style="max-width:100px;">
							{message.from.name || message.from.address.split('@')[0]}
						</td>
						<td class="col-subject">
							{message.subject || '(No Subject)'}
						</td>
						<td class="col-date">{formatRelativeTime(message.receivedAt)}</td>
					</tr>
				{/each}
			</tbody></table>
		{/if}
	</div>
</div>
