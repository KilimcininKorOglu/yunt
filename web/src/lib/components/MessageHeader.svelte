<script lang="ts">
	import type { Message, EmailAddress } from '$lib/api/types';

	interface Props {
		message: Message;
	}

	const { message }: Props = $props();

	function formatEmailAddress(addr: EmailAddress): string {
		if (addr.name) {
			return `${addr.name} <${addr.address}>`;
		}
		return addr.address;
	}

	function formatEmailAddresses(addrs: EmailAddress[]): string {
		return addrs.map(formatEmailAddress).join(', ');
	}

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
</script>

<div class="read-header">
	<div class="read-subject">
		{#if message.isStarred}★{/if}
		{message.subject || '(No subject)'}
	</div>
	<table class="read-meta"><tbody>
		<tr>
			<td class="lbl">From:</td>
			<td class="val">{formatEmailAddress(message.from)}</td>
		</tr>
		<tr>
			<td class="lbl">To:</td>
			<td class="val">{formatEmailAddresses(message.to)}</td>
		</tr>
		{#if message.cc && message.cc.length > 0}
			<tr>
				<td class="lbl">Cc:</td>
				<td class="val">{formatEmailAddresses(message.cc)}</td>
			</tr>
		{/if}
		{#if message.bcc && message.bcc.length > 0}
			<tr>
				<td class="lbl">Bcc:</td>
				<td class="val">{formatEmailAddresses(message.bcc)}</td>
			</tr>
		{/if}
		{#if message.replyTo}
			<tr>
				<td class="lbl">Reply-To:</td>
				<td class="val">{formatEmailAddress(message.replyTo)}</td>
			</tr>
		{/if}
		<tr>
			<td class="lbl">Date:</td>
			<td class="val">{formatDate(message.receivedAt)}</td>
		</tr>
	</tbody></table>
</div>
