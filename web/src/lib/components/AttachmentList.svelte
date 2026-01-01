<script lang="ts">
	/**
	 * AttachmentList Component
	 * Displays a list of message attachments with download and preview capabilities.
	 */

	import type { AttachmentSummary, ID } from '$lib/api/types';
	import { getMessagesApi } from '$lib/api';

	interface Props {
		/** Message ID */
		messageId: ID;
		/** List of attachment summaries */
		attachments: AttachmentSummary[];
		/** Callback for inline attachment loading (reserved for future use) */
		onInlineAttachmentLoaded?: (contentId: string, dataUrl: string) => void;
	}

	const {
		messageId,
		attachments,
		onInlineAttachmentLoaded: _onInlineAttachmentLoaded
	}: Props = $props();

	const messagesApi = getMessagesApi();

	// Track download progress
	let downloading = $state<Set<ID>>(new Set());

	// Separate inline and regular attachments
	const regularAttachments = $derived(attachments.filter((a) => !a.isInline));
	const inlineAttachments = $derived(attachments.filter((a) => a.isInline));

	/**
	 * Get file icon based on content type
	 */
	function getFileIcon(contentType: string): { path: string; color: string } {
		if (contentType.startsWith('image/')) {
			return {
				path: 'M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z',
				color: 'text-green-500'
			};
		}
		if (contentType.startsWith('video/')) {
			return {
				path: 'M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z',
				color: 'text-purple-500'
			};
		}
		if (contentType.startsWith('audio/')) {
			return {
				path: 'M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3',
				color: 'text-blue-500'
			};
		}
		if (contentType === 'application/pdf') {
			return {
				path: 'M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z',
				color: 'text-red-500'
			};
		}
		if (
			contentType.includes('spreadsheet') ||
			contentType.includes('excel') ||
			contentType === 'text/csv'
		) {
			return {
				path: 'M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z',
				color: 'text-green-600'
			};
		}
		if (contentType.includes('word') || contentType.includes('document')) {
			return {
				path: 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z',
				color: 'text-blue-600'
			};
		}
		if (
			contentType.includes('zip') ||
			contentType.includes('archive') ||
			contentType.includes('compressed')
		) {
			return {
				path: 'M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4',
				color: 'text-amber-500'
			};
		}
		if (contentType.startsWith('text/')) {
			return {
				path: 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z',
				color: 'text-secondary-500'
			};
		}
		// Default file icon
		return {
			path: 'M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z',
			color: 'text-secondary-400'
		};
	}

	/**
	 * Check if attachment can be previewed
	 */
	function canPreview(contentType: string): boolean {
		return contentType.startsWith('image/') || contentType === 'application/pdf';
	}

	/**
	 * Download an attachment
	 */
	async function downloadAttachment(attachment: AttachmentSummary): Promise<void> {
		if (downloading.has(attachment.id)) return;

		downloading = new Set(downloading).add(attachment.id);

		try {
			await messagesApi.saveAttachment(messageId, attachment.id, attachment.filename);
		} catch (err) {
			console.error('Failed to download attachment:', err);
			// Could show a toast notification here
		} finally {
			const newSet = new Set(downloading);
			newSet.delete(attachment.id);
			downloading = newSet;
		}
	}

	/**
	 * Preview an attachment (opens in new tab)
	 */
	async function previewAttachment(attachment: AttachmentSummary): Promise<void> {
		if (downloading.has(attachment.id)) return;

		downloading = new Set(downloading).add(attachment.id);

		try {
			const blob = await messagesApi.downloadAttachment(messageId, attachment.id);
			const url = URL.createObjectURL(blob);
			window.open(url, '_blank');
			// Note: We don't revoke immediately as the new tab needs time to load
			setTimeout(() => URL.revokeObjectURL(url), 60000);
		} catch (err) {
			console.error('Failed to preview attachment:', err);
		} finally {
			const newSet = new Set(downloading);
			newSet.delete(attachment.id);
			downloading = newSet;
		}
	}

	/**
	 * Download all attachments
	 */
	async function downloadAll(): Promise<void> {
		for (const attachment of regularAttachments) {
			await downloadAttachment(attachment);
		}
	}
</script>

{#if attachments.length > 0}
	<div class="border-t border-secondary-200 bg-secondary-50">
		<!-- Header -->
		<div class="flex items-center justify-between px-6 py-3">
			<div class="flex items-center gap-2">
				<svg
					class="h-5 w-5 text-secondary-500"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13"
					/>
				</svg>
				<span class="text-sm font-medium text-secondary-700">
					{regularAttachments.length} attachment{regularAttachments.length === 1
						? ''
						: 's'}
				</span>
				{#if inlineAttachments.length > 0}
					<span class="text-xs text-secondary-500">
						(+{inlineAttachments.length} inline)
					</span>
				{/if}
			</div>

			{#if regularAttachments.length > 1}
				<button
					type="button"
					onclick={downloadAll}
					class="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-medium text-secondary-600 transition-colors hover:bg-secondary-200 hover:text-secondary-900"
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
							d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
						/>
					</svg>
					Download all
				</button>
			{/if}
		</div>

		<!-- Attachment list -->
		<div class="px-6 pb-4">
			<div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
				{#each regularAttachments as attachment (attachment.id)}
					{@const icon = getFileIcon(attachment.contentType)}
					{@const isDownloading = downloading.has(attachment.id)}
					<div
						class="group flex items-center gap-3 rounded-lg border border-secondary-200 bg-white p-3 transition-colors hover:border-secondary-300 hover:bg-secondary-50"
					>
						<!-- File icon -->
						<div
							class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-lg bg-secondary-100 {icon.color}"
						>
							<svg
								class="h-5 w-5"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
								stroke-width="2"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									d={icon.path}
								/>
							</svg>
						</div>

						<!-- File info -->
						<div class="min-w-0 flex-1">
							<p
								class="truncate text-sm font-medium text-secondary-900"
								title={attachment.filename}
							>
								{attachment.filename}
							</p>
							<p class="text-xs text-secondary-500">
								{attachment.sizeFormatted}
							</p>
						</div>

						<!-- Actions -->
						<div
							class="flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100"
						>
							{#if canPreview(attachment.contentType)}
								<button
									type="button"
									onclick={() => previewAttachment(attachment)}
									disabled={isDownloading}
									class="rounded-lg p-1.5 text-secondary-500 transition-colors hover:bg-secondary-100 hover:text-secondary-700 disabled:opacity-50"
									title="Preview"
									aria-label="Preview attachment"
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
											d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
										/>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
										/>
									</svg>
								</button>
							{/if}
							<button
								type="button"
								onclick={() => downloadAttachment(attachment)}
								disabled={isDownloading}
								class="rounded-lg p-1.5 text-secondary-500 transition-colors hover:bg-secondary-100 hover:text-secondary-700 disabled:opacity-50"
								title="Download"
								aria-label="Download attachment"
							>
								{#if isDownloading}
									<svg
										class="h-4 w-4 animate-spin"
										fill="none"
										viewBox="0 0 24 24"
									>
										<circle
											class="opacity-25"
											cx="12"
											cy="12"
											r="10"
											stroke="currentColor"
											stroke-width="4"
										></circle>
										<path
											class="opacity-75"
											fill="currentColor"
											d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
										></path>
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
											d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
										/>
									</svg>
								{/if}
							</button>
						</div>
					</div>
				{/each}
			</div>
		</div>
	</div>
{/if}
