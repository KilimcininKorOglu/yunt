<script lang="ts">
	/**
	 * MessageHeader Component
	 * Displays the message header information including sender, recipients, subject, and date.
	 */

	import type { Message, EmailAddress } from '$lib/api/types';

	interface Props {
		/** The message object containing header data */
		message: Message;
	}

	const { message }: Props = $props();

	/**
	 * Format an email address for display
	 */
	function formatEmailAddress(addr: EmailAddress): string {
		if (addr.name) {
			return `${addr.name} <${addr.address}>`;
		}
		return addr.address;
	}

	/**
	 * Format multiple email addresses for display
	 */
	function formatEmailAddresses(addrs: EmailAddress[]): string {
		return addrs.map(formatEmailAddress).join(', ');
	}

	/**
	 * Format date for display
	 */
	function formatDate(timestamp: string): string {
		const date = new Date(timestamp);
		return date.toLocaleString(undefined, {
			weekday: 'short',
			year: 'numeric',
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	/**
	 * Format relative time
	 */
	function formatRelativeTime(timestamp: string): string {
		const date = new Date(timestamp);
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffMins = Math.floor(diffMs / 60000);
		const diffHours = Math.floor(diffMs / 3600000);
		const diffDays = Math.floor(diffMs / 86400000);

		if (diffMins < 1) return 'Just now';
		if (diffMins < 60) return `${diffMins} minute${diffMins === 1 ? '' : 's'} ago`;
		if (diffHours < 24) return `${diffHours} hour${diffHours === 1 ? '' : 's'} ago`;
		if (diffDays < 7) return `${diffDays} day${diffDays === 1 ? '' : 's'} ago`;
		return formatDate(timestamp);
	}

	/**
	 * Get initials from email address
	 */
	function getInitials(addr: EmailAddress): string {
		if (addr.name) {
			const parts = addr.name.trim().split(/\s+/);
			if (parts.length >= 2) {
				return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
			}
			return addr.name.charAt(0).toUpperCase();
		}
		return addr.address.charAt(0).toUpperCase();
	}

	// Expandable details state
	let showDetails = $state(false);
</script>

<div class="border-b border-secondary-200 bg-white px-6 py-4">
	<!-- Main header row -->
	<div class="flex items-start gap-4">
		<!-- Avatar -->
		<div
			class="flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-primary-100 text-primary-700"
		>
			<span class="text-lg font-semibold">{getInitials(message.from)}</span>
		</div>

		<!-- Main info -->
		<div class="min-w-0 flex-1">
			<div class="flex items-start justify-between gap-4">
				<div class="min-w-0 flex-1">
					<!-- Sender name -->
					<div class="flex items-center gap-2">
						<h2 class="truncate text-base font-semibold text-secondary-900">
							{message.from.name || message.from.address}
						</h2>
						{#if message.isStarred}
							<svg
								class="h-5 w-5 flex-shrink-0 text-amber-400"
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
					<!-- Sender email if different from name -->
					{#if message.from.name}
						<p class="truncate text-sm text-secondary-500">
							&lt;{message.from.address}&gt;
						</p>
					{/if}
				</div>

				<!-- Date and time -->
				<div class="flex-shrink-0 text-right">
					<p
						class="text-sm font-medium text-secondary-700"
						title={formatDate(message.receivedAt)}
					>
						{formatRelativeTime(message.receivedAt)}
					</p>
					<p class="text-xs text-secondary-400">{formatDate(message.receivedAt)}</p>
				</div>
			</div>

			<!-- Subject -->
			<h1 class="mt-2 text-xl font-semibold text-secondary-900">
				{message.subject || '(No subject)'}
			</h1>

			<!-- Recipients summary -->
			<div class="mt-2 flex items-center gap-2 text-sm text-secondary-600">
				<span class="font-medium">To:</span>
				<span class="truncate">{formatEmailAddresses(message.to)}</span>
				{#if message.cc && message.cc.length > 0}
					<span class="text-secondary-400">+{message.cc.length} CC</span>
				{/if}
				<button
					type="button"
					onclick={() => (showDetails = !showDetails)}
					class="ml-2 flex items-center gap-1 text-primary-600 hover:text-primary-700"
				>
					<span class="text-xs font-medium">{showDetails ? 'Hide' : 'Show'} details</span>
					<svg
						class="h-4 w-4 transition-transform {showDetails ? 'rotate-180' : ''}"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M19 9l-7 7-7-7"
						/>
					</svg>
				</button>
			</div>
		</div>
	</div>

	<!-- Expandable details -->
	{#if showDetails}
		<div class="mt-4 rounded-lg border border-secondary-200 bg-secondary-50 p-4">
			<dl class="space-y-2 text-sm">
				<div class="flex">
					<dt class="w-24 flex-shrink-0 font-medium text-secondary-600">From:</dt>
					<dd class="text-secondary-900">{formatEmailAddress(message.from)}</dd>
				</div>
				<div class="flex">
					<dt class="w-24 flex-shrink-0 font-medium text-secondary-600">To:</dt>
					<dd class="break-all text-secondary-900">{formatEmailAddresses(message.to)}</dd>
				</div>
				{#if message.cc && message.cc.length > 0}
					<div class="flex">
						<dt class="w-24 flex-shrink-0 font-medium text-secondary-600">Cc:</dt>
						<dd class="break-all text-secondary-900">
							{formatEmailAddresses(message.cc)}
						</dd>
					</div>
				{/if}
				{#if message.bcc && message.bcc.length > 0}
					<div class="flex">
						<dt class="w-24 flex-shrink-0 font-medium text-secondary-600">Bcc:</dt>
						<dd class="break-all text-secondary-900">
							{formatEmailAddresses(message.bcc)}
						</dd>
					</div>
				{/if}
				{#if message.replyTo}
					<div class="flex">
						<dt class="w-24 flex-shrink-0 font-medium text-secondary-600">Reply-To:</dt>
						<dd class="text-secondary-900">{formatEmailAddress(message.replyTo)}</dd>
					</div>
				{/if}
				<div class="flex">
					<dt class="w-24 flex-shrink-0 font-medium text-secondary-600">Date:</dt>
					<dd class="text-secondary-900">{formatDate(message.receivedAt)}</dd>
				</div>
				{#if message.messageId}
					<div class="flex">
						<dt class="w-24 flex-shrink-0 font-medium text-secondary-600">
							Message-ID:
						</dt>
						<dd class="break-all font-mono text-xs text-secondary-700">
							{message.messageId}
						</dd>
					</div>
				{/if}
			</dl>
		</div>
	{/if}
</div>
