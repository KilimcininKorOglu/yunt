<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { authStore } from '$stores/auth';
	import { getMessagesApi, getMailboxesApi, getAttachmentsApi } from '$lib/api';
	import type { Message, Mailbox, AttachmentSummary, ID } from '$lib/api/types';
	import MessageHeader from '$lib/components/MessageHeader.svelte';
	import MessageBody from '$lib/components/MessageBody.svelte';
	import AttachmentList from '$lib/components/AttachmentList.svelte';
	import MessageActions from '$lib/components/MessageActions.svelte';

	const messagesApi = getMessagesApi();
	const mailboxesApi = getMailboxesApi();
	const attachmentsApi = getAttachmentsApi();

	let message = $state<Message | null>(null);
	let attachments = $state<AttachmentSummary[]>([]);
	let mailboxes = $state<Mailbox[]>([]);
	let inlineAttachments = $state<Map<string, string>>(new Map());
	let isLoading = $state(true);
	let isActionLoading = $state(false);
	let error = $state<string | null>(null);

	const messageId = $derived($page.params.id);

	$effect(() => {
		if (messageId && authStore.isAuthenticated && !authStore.isLoading) {
			loadMessage(messageId);
			loadMailboxes();
		}
	});

	async function loadMessage(id: ID): Promise<void> {
		isLoading = true;
		error = null;

		try {
			const [messageData, attachmentsData] = await Promise.all([
				messagesApi.get(id),
				messagesApi.listAttachments(id)
			]);

			message = messageData;
			attachments = attachmentsData;

			if (message.status === 'unread') {
				try {
					await messagesApi.markAsRead(id);
					message = { ...message, status: 'read' };
				} catch (err) {
					console.warn('Failed to mark message as read:', err);
				}
			}

			await loadInlineAttachments();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load message';
			message = null;
			attachments = [];
		} finally {
			isLoading = false;
		}
	}

	async function loadMailboxes(): Promise<void> {
		try {
			const response = await mailboxesApi.list({ pageSize: 100 });
			mailboxes = response.items;
		} catch (err) {
			console.warn('Failed to load mailboxes:', err);
		}
	}

	async function loadInlineAttachments(): Promise<void> {
		const inlineAtts = attachments.filter((a) => a.isInline);
		if (inlineAtts.length === 0 || !message) return;

		const newMap = new Map<string, string>();

		for (const att of inlineAtts) {
			try {
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

	function handleBack(): void {
		if (window.history.length > 1) {
			window.history.back();
		} else {
			goto('/inbox');
		}
	}

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
	<title>{message?.subject || 'Message'} - Yunt Mail</title>
</svelte:head>

{#if isLoading}
	<div style="text-align:center;padding:40px;">
		<div class="loading-spinner" style="width:24px;height:24px;margin:0 auto 10px;"></div>
		<p>Loading message...</p>
	</div>
{:else if error && !message}
	<div style="padding:20px;">
		<div class="alert alert-error">
			{error}
		</div>
		<button type="button" class="hotmail-btn" onclick={() => loadMessage(messageId)}>Try Again</button>
		<button type="button" class="hotmail-btn" onclick={handleBack}>Go Back</button>
	</div>
{:else if message}
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

	{#if error}
		<div class="alert alert-error" style="margin:8px 10px;">
			{error}
			<button type="button" class="alert-close" onclick={() => (error = null)}>X</button>
		</div>
	{/if}

	<MessageHeader {message} />

	{#if attachments.length > 0}
		<AttachmentList messageId={message.id} {attachments} />
	{/if}

	<MessageBody {message} {inlineAttachments} />
{/if}
