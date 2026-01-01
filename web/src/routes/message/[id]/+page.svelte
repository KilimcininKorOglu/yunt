<script lang="ts">
	/**
	 * Message Detail Page
	 * Displays full message content with header, body, attachments, and actions.
	 * Automatically marks message as read on load.
	 */

	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { authStore } from '$stores/auth';
	import { getMessagesApi, getMailboxesApi, getAttachmentsApi } from '$lib/api';
	import type { Message, Mailbox, AttachmentSummary, ID } from '$lib/api/types';
	import MessageHeader from '$lib/components/MessageHeader.svelte';
	import MessageBody from '$lib/components/MessageBody.svelte';
	import AttachmentList from '$lib/components/AttachmentList.svelte';
	import MessageActions from '$lib/components/MessageActions.svelte';

	// API instances
	const messagesApi = getMessagesApi();
	const mailboxesApi = getMailboxesApi();
	const attachmentsApi = getAttachmentsApi();

	// State
	let message = $state<Message | null>(null);
	let attachments = $state<AttachmentSummary[]>([]);
	let mailboxes = $state<Mailbox[]>([]);
	let inlineAttachments = $state<Map<string, string>>(new Map());
	let isLoading = $state(true);
	let isActionLoading = $state(false);
	let error = $state<string | null>(null);

	// Get message ID from URL
	const messageId = $derived($page.params.id);

	// Load message data when ID changes
	$effect(() => {
		if (messageId && authStore.isAuthenticated && !authStore.isLoading) {
			loadMessage(messageId);
			loadMailboxes();
		}
	});

	/**
	 * Load message details
	 */
	async function loadMessage(id: ID): Promise<void> {
		isLoading = true;
		error = null;

		try {
			// Load message and attachments in parallel
			const [messageData, attachmentsData] = await Promise.all([
				messagesApi.get(id),
				messagesApi.listAttachments(id)
			]);

			message = messageData;
			attachments = attachmentsData;

			// Mark as read if unread
			if (message.status === 'unread') {
				try {
					await messagesApi.markAsRead(id);
					message = { ...message, status: 'read' };
				} catch (err) {
					console.warn('Failed to mark message as read:', err);
				}
			}

			// Load inline attachments for HTML rendering
			await loadInlineAttachments();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load message';
			message = null;
			attachments = [];
		} finally {
			isLoading = false;
		}
	}

	/**
	 * Load mailboxes for move operation
	 */
	async function loadMailboxes(): Promise<void> {
		try {
			const response = await mailboxesApi.list({ pageSize: 100 });
			mailboxes = response.items;
		} catch (err) {
			console.warn('Failed to load mailboxes:', err);
		}
	}

	/**
	 * Load inline attachments (images) for HTML body rendering
	 */
	async function loadInlineAttachments(): Promise<void> {
		const inlineAtts = attachments.filter((a) => a.isInline);
		if (inlineAtts.length === 0 || !message) return;

		const newMap = new Map<string, string>();

		for (const att of inlineAtts) {
			try {
				// Get the attachment details to get the contentId
				const attDetails = await messagesApi.getAttachment(message.id, att.id);
				if (attDetails.contentId) {
					const dataUrl = await attachmentsApi.getDataUrl(att.id);
					newMap.set(attDetails.contentId, dataUrl);
				}
			} catch (err) {
				console.warn(`Failed to load inline attachment ${att.id}:`, err);
			}
		}

		inlineAttachments = newMap;
	}

	/**
	 * Navigate back to inbox
	 */
	function handleBack(): void {
		// Check if we came from a specific page
		if (window.history.length > 1) {
			window.history.back();
		} else {
			goto('/inbox');
		}
	}

	/**
	 * Delete message and navigate back
	 */
	async function handleDelete(): Promise<void> {
		if (!message) return;

		isActionLoading = true;
		try {
			await messagesApi.delete(message.id);
			goto('/inbox');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete message';
		} finally {
			isActionLoading = false;
		}
	}

	/**
	 * Move message to another mailbox
	 */
	async function handleMove(targetMailboxId: ID): Promise<void> {
		if (!message) return;

		isActionLoading = true;
		try {
			await messagesApi.move(message.id, targetMailboxId);
			message = { ...message, mailboxId: targetMailboxId };
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to move message';
		} finally {
			isActionLoading = false;
		}
	}

	/**
	 * Toggle star status
	 */
	async function handleToggleStar(): Promise<void> {
		if (!message) return;

		isActionLoading = true;
		try {
			if (message.isStarred) {
				await messagesApi.unstar(message.id);
			} else {
				await messagesApi.star(message.id);
			}
			message = { ...message, isStarred: !message.isStarred };
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to update star';
		} finally {
			isActionLoading = false;
		}
	}

	/**
	 * Toggle read/unread status
	 */
	async function handleToggleRead(): Promise<void> {
		if (!message) return;

		isActionLoading = true;
		try {
			if (message.status === 'unread') {
				await messagesApi.markAsRead(message.id);
				message = { ...message, status: 'read' };
			} else {
				await messagesApi.markAsUnread(message.id);
				message = { ...message, status: 'unread' };
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to update read status';
		} finally {
			isActionLoading = false;
		}
	}

	/**
	 * Toggle spam status
	 */
	async function handleMarkSpam(): Promise<void> {
		if (!message) return;

		isActionLoading = true;
		try {
			if (message.isSpam) {
				await messagesApi.markAsNotSpam(message.id);
			} else {
				await messagesApi.markAsSpam(message.id);
			}
			message = { ...message, isSpam: !message.isSpam };
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to update spam status';
		} finally {
			isActionLoading = false;
		}
	}

	/**
	 * Download raw message
	 */
	async function handleDownloadRaw(): Promise<void> {
		if (!message) return;

		isActionLoading = true;
		try {
			const filename = `${message.subject || 'message'}.eml`.replace(/[/\\?%*:|"<>]/g, '-');
			await messagesApi.downloadRaw(message.id, filename);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to download message';
		} finally {
			isActionLoading = false;
		}
	}
</script>

<svelte:head>
	<title>{message?.subject || 'Message'} - Yunt</title>
</svelte:head>

<div class="flex h-screen flex-col overflow-hidden bg-secondary-50">
	{#if isLoading}
		<!-- Loading state -->
		<div class="flex flex-1 items-center justify-center">
			<div class="text-center">
				<div
					class="mb-4 inline-block h-10 w-10 animate-spin rounded-full border-4 border-primary-200 border-t-primary-600"
				></div>
				<p class="text-secondary-500">Loading message...</p>
			</div>
		</div>
	{:else if error && !message}
		<!-- Error state -->
		<div class="flex flex-1 flex-col items-center justify-center p-8">
			<div
				class="mx-auto max-w-md rounded-xl border border-red-200 bg-white p-8 text-center shadow-sm"
			>
				<div
					class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-red-100"
				>
					<svg
						class="h-7 w-7 text-red-600"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
						stroke-width="2"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
						/>
					</svg>
				</div>
				<h2 class="mb-2 text-lg font-semibold text-secondary-900">
					Unable to Load Message
				</h2>
				<p class="mb-6 text-sm text-secondary-600">{error}</p>
				<div class="flex justify-center gap-3">
					<button
						type="button"
						onclick={() => loadMessage(messageId)}
						class="btn-primary"
					>
						Try Again
					</button>
					<button type="button" onclick={handleBack} class="btn-secondary">
						Go Back
					</button>
				</div>
			</div>
		</div>
	{:else if message}
		<!-- Message content -->
		<div class="flex flex-1 flex-col overflow-hidden">
			<!-- Actions bar -->
			<MessageActions
				{message}
				{mailboxes}
				isLoading={isActionLoading}
				onBack={handleBack}
				onDelete={handleDelete}
				onMove={handleMove}
				onToggleStar={handleToggleStar}
				onToggleRead={handleToggleRead}
				onMarkSpam={handleMarkSpam}
				onDownloadRaw={handleDownloadRaw}
			/>

			<!-- Error alert (for action errors) -->
			{#if error}
				<div class="border-b border-red-200 bg-red-50 px-6 py-3" role="alert">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-2">
							<svg
								class="h-5 w-5 text-red-500"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
								/>
							</svg>
							<p class="text-sm text-red-700">{error}</p>
						</div>
						<button
							type="button"
							onclick={() => (error = null)}
							class="text-red-500 hover:text-red-700"
							aria-label="Dismiss error"
						>
							<svg
								class="h-5 w-5"
								fill="none"
								viewBox="0 0 24 24"
								stroke="currentColor"
							>
								<path
									stroke-linecap="round"
									stroke-linejoin="round"
									stroke-width="2"
									d="M6 18L18 6M6 6l12 12"
								/>
							</svg>
						</button>
					</div>
				</div>
			{/if}

			<!-- Scrollable content area -->
			<div class="flex-1 overflow-auto">
				<div class="mx-auto max-w-5xl">
					<!-- Message card -->
					<div
						class="m-4 overflow-hidden rounded-xl border border-secondary-200 bg-white shadow-sm"
					>
						<!-- Header -->
						<MessageHeader {message} />

						<!-- Attachments -->
						{#if attachments.length > 0}
							<AttachmentList messageId={message.id} {attachments} />
						{/if}

						<!-- Body -->
						<MessageBody {message} {inlineAttachments} />
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>
