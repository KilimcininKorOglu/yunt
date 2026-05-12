<script lang="ts">
	import type { AttachmentSummary, ID } from '$lib/api/types';
	import { getMessagesApi } from '$lib/api';

	interface Props {
		messageId: ID;
		attachments: AttachmentSummary[];
	}

	const { messageId, attachments }: Props = $props();

	const messagesApi = getMessagesApi();

	let downloading = $state<Set<ID>>(new Set());

	const regularAttachments = $derived(attachments.filter((a) => !a.isInline));

	async function downloadAttachment(attachment: AttachmentSummary): Promise<void> {
		if (downloading.has(attachment.id)) return;

		downloading = new Set(downloading).add(attachment.id);

		try {
			await messagesApi.saveAttachment(messageId, attachment.id, attachment.filename);
		} catch (err) {
			console.error('Failed to download attachment:', err);
		} finally {
			const newSet = new Set(downloading);
			newSet.delete(attachment.id);
			downloading = newSet;
		}
	}

	async function downloadAll(): Promise<void> {
		for (const attachment of regularAttachments) {
			await downloadAttachment(attachment);
		}
	}
</script>

{#if regularAttachments.length > 0}
	<div class="read-attachments">
		<div class="att-title">
			📎 Attachments ({regularAttachments.length})
			{#if regularAttachments.length > 1}
				<button type="button" class="hotmail-btn" style="margin-left:10px;font-size:10px;" onclick={downloadAll}>
					Download All
				</button>
			{/if}
		</div>
		{#each regularAttachments as attachment (attachment.id)}
			{@const isDownloading = downloading.has(attachment.id)}
			<button
				type="button"
				class="att-item"
				onclick={() => downloadAttachment(attachment)}
				disabled={isDownloading}
			>
				📄 {attachment.filename} ({attachment.sizeFormatted})
				{#if isDownloading}
					<span class="loading-spinner" style="width:10px;height:10px;margin-left:4px;"></span>
				{/if}
			</button>
		{/each}
	</div>
{/if}
