<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { getMessagesApi, getMailboxesApi } from '$lib/api';
	import { authStore } from '$stores/auth.svelte';
	import type { Mailbox, SendMessageInput, AttachmentUploadResult } from '$lib/api/types';

	const messagesApi = getMessagesApi();
	const mailboxesApi = getMailboxesApi();

	let relayEnabled = $state(false);
	let mailboxes = $state<Mailbox[]>([]);
	let fromMailboxId = $state('');

	let toField = $state('');
	let ccField = $state('');
	let bccField = $state('');
	let subject = $state('');
	let textBody = $state('');
	let htmlBody = $state('');
	let useRichText = $state(false);

	let attachments = $state<AttachmentUploadResult[]>([]);
	let draftId = $state<string | null>(null);

	let sending = $state(false);
	let savingDraft = $state(false);
	let error = $state<string | null>(null);
	let success = $state<string | null>(null);
	let loading = $state(false);

	let autoSaveTimer: ReturnType<typeof setInterval> | null = null;
	let autoSaveInProgress = false;
	let lastSavedContent = '';
	let fileInput: HTMLInputElement;

	const replyToId = $derived($page.url.searchParams.get('replyTo'));
	const forwardId = $derived($page.url.searchParams.get('forward'));
	const draftParam = $derived($page.url.searchParams.get('draft'));

	onMount(async () => {
		await loadRelayStatus();
		await loadMailboxes();

		if (draftParam) {
			await loadDraft(draftParam);
		} else if (replyToId) {
			await loadOriginal(replyToId, 'reply');
		} else if (forwardId) {
			await loadOriginal(forwardId, 'forward');
		}

		if (authStore.user?.signature) {
			if (!textBody.includes('---')) {
				textBody += '\n\n---\n' + authStore.user.signature;
			}
		}

		autoSaveTimer = setInterval(autoSave, 30000);
		return () => { if (autoSaveTimer) clearInterval(autoSaveTimer); };
	});

	async function loadRelayStatus(): Promise<void> {
		try {
			const resp = await fetch('/api/v1/stats', {
				headers: { 'Authorization': `Bearer ${authStore.accessToken}` }
			});
			const data = await resp.json();
			relayEnabled = data?.data?.relayEnabled ?? false;
		} catch { relayEnabled = false; }
	}

	async function loadMailboxes(): Promise<void> {
		try {
			const resp = await mailboxesApi.list();
			const items = resp?.items ?? resp ?? [];
			mailboxes = (items as Mailbox[]).filter(m =>
				!m.isCatchAll && !['Sent', 'Drafts', 'Trash', 'Spam', 'Junk E-Mail'].includes(m.name)
			);
			if (mailboxes.length > 0 && !fromMailboxId) {
				fromMailboxId = mailboxes[0].id;
			}
		} catch { /* ignore */ }
	}

	async function loadOriginal(id: string, mode: 'reply' | 'forward'): Promise<void> {
		loading = true;
		try {
			const msg = await messagesApi.get(id);
			if (mode === 'reply') {
				toField = msg.from.address;
				subject = msg.subject.startsWith('Re:') ? msg.subject : `Re: ${msg.subject}`;
				textBody = `\n\n--- Original Message ---\nFrom: ${msg.from.name || msg.from.address}\nDate: ${new Date(msg.receivedAt).toLocaleString()}\n\n${msg.textBody || ''}`;
			} else {
				subject = msg.subject.startsWith('Fwd:') ? msg.subject : `Fwd: ${msg.subject}`;
				textBody = `\n\n--- Forwarded Message ---\nFrom: ${msg.from.name || msg.from.address}\nTo: ${msg.to.map((t: {address: string}) => t.address).join(', ')}\nDate: ${new Date(msg.receivedAt).toLocaleString()}\nSubject: ${msg.subject}\n\n${msg.textBody || ''}`;
			}
		} catch (err) {
			error = 'Failed to load original message.';
		} finally {
			loading = false;
		}
	}

	async function loadDraft(id: string): Promise<void> {
		loading = true;
		try {
			const msg = await messagesApi.get(id);
			draftId = msg.id;
			subject = msg.subject || '';
			textBody = msg.textBody || '';
			htmlBody = msg.htmlBody || '';
			toField = msg.to?.map((t: {address: string}) => t.address).join(', ') || '';
			ccField = msg.cc?.map((t: {address: string}) => t.address).join(', ') || '';
			bccField = msg.bcc?.map((t: {address: string}) => t.address).join(', ') || '';
			const attList = await messagesApi.listAttachments(msg.id);
			attachments = (attList as AttachmentUploadResult[]) || [];
		} catch {
			error = 'Failed to load draft.';
		} finally {
			loading = false;
		}
	}

	function parseAddresses(field: string): string[] {
		return field.split(/[,;]/).map(s => s.trim()).filter(Boolean);
	}

	function currentContent(): string {
		return `${toField}|${ccField}|${bccField}|${subject}|${textBody}|${htmlBody}`;
	}

	async function autoSave(): Promise<void> {
		if (autoSaveInProgress || sending || !subject.trim()) return;
		const content = currentContent();
		if (content === lastSavedContent) return;

		autoSaveInProgress = true;
		try {
			const input = {
				fromMailboxId: fromMailboxId || undefined,
				to: parseAddresses(toField),
				cc: parseAddresses(ccField),
				bcc: parseAddresses(bccField),
				subject,
				textBody,
				htmlBody: useRichText ? htmlBody : undefined
			};
			if (draftId) {
				await messagesApi.updateDraft(draftId, input);
			} else {
				const result = await messagesApi.saveDraft(input);
				draftId = result.id;
			}
			lastSavedContent = content;
		} catch { /* silent auto-save failure */ }
		finally { autoSaveInProgress = false; }
	}

	async function handleSend(): Promise<void> {
		if (sending) return;
		const toAddrs = parseAddresses(toField);
		if (toAddrs.length === 0) { error = 'At least one recipient is required.'; return; }
		if (!subject.trim()) { error = 'Subject is required.'; return; }
		if (!fromMailboxId) { error = 'Please select a From address.'; return; }

		sending = true;
		error = null;
		try {
			const input: SendMessageInput = {
				fromMailboxId,
				to: toAddrs,
				cc: parseAddresses(ccField),
				bcc: parseAddresses(bccField),
				subject,
				textBody,
				htmlBody: useRichText ? htmlBody : undefined
			};
			const result = await messagesApi.send(input);
			if (result.failedRecipients && result.failedRecipients.length > 0) {
				success = `Message sent to ${result.recipients.length} recipient(s). Failed: ${result.failedRecipients.join(', ')}`;
			} else {
				success = 'Message sent successfully!';
			}
			if (draftId) {
				try { await messagesApi.delete(draftId); } catch { /* draft cleanup */ }
			}
			setTimeout(() => goto('/inbox'), 2000);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to send message.';
		} finally {
			sending = false;
		}
	}

	async function handleSaveDraft(): Promise<void> {
		if (savingDraft) return;
		savingDraft = true;
		error = null;
		try {
			const input = {
				fromMailboxId: fromMailboxId || undefined,
				to: parseAddresses(toField),
				cc: parseAddresses(ccField),
				bcc: parseAddresses(bccField),
				subject: subject || '(No Subject)',
				textBody,
				htmlBody: useRichText ? htmlBody : undefined
			};
			if (draftId) {
				await messagesApi.updateDraft(draftId, input);
			} else {
				const result = await messagesApi.saveDraft(input);
				draftId = result.id;
			}
			lastSavedContent = currentContent();
			success = 'Draft saved.';
			setTimeout(() => { success = null; }, 3000);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save draft.';
		} finally {
			savingDraft = false;
		}
	}

	async function handleAttach(): Promise<void> {
		fileInput?.click();
	}

	async function handleFileSelected(e: Event): Promise<void> {
		const input = e.target as HTMLInputElement;
		const files = input.files;
		if (!files || files.length === 0) return;

		if (!draftId) {
			await handleSaveDraft();
			if (!draftId) return;
		}

		for (const file of files) {
			try {
				const result = await messagesApi.uploadDraftAttachment(draftId, file);
				attachments = [...attachments, result];
			} catch (err) {
				error = `Failed to upload ${file.name}: ${err instanceof Error ? err.message : 'Unknown error'}`;
			}
		}
		input.value = '';
	}

	async function handleRemoveAttachment(attId: string): Promise<void> {
		if (!draftId) return;
		try {
			await messagesApi.deleteDraftAttachment(draftId, attId);
			attachments = attachments.filter(a => a.id !== attId);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to remove attachment.';
		}
	}

	function formatFileSize(bytes: number): string {
		if (bytes >= 1048576) return (bytes / 1048576).toFixed(1) + ' MB';
		if (bytes >= 1024) return (bytes / 1024).toFixed(0) + ' KB';
		return bytes + ' B';
	}
</script>

<svelte:head>
	<title>Compose - Yunt Mail</title>
</svelte:head>

<input type="file" multiple bind:this={fileInput} onchange={handleFileSelected} style="display:none;" />

<div class="toolbar">
	<div class="toolbar-left">
		<button type="button" class="hotmail-btn toolbar-btn-primary"
			onclick={handleSend}
			disabled={sending || !relayEnabled}>
			{sending ? 'Sending...' : 'Send'}
		</button>
		<span class="toolbar-sep">|</span>
		<button type="button" class="hotmail-btn" onclick={handleSaveDraft} disabled={savingDraft}>
			{savingDraft ? 'Saving...' : 'Save Draft'}
		</button>
		<button type="button" class="hotmail-btn" onclick={handleAttach}>Attach</button>
		<span class="toolbar-sep">|</span>
		<a href="/inbox" class="hotmail-btn">Cancel</a>
	</div>
</div>

{#if !relayEnabled}
	<div class="alert alert-info" style="margin:8px 10px;">
		Mail sending requires relay configuration. Configure SMTP relay in server settings to enable sending.
	</div>
{/if}

{#if error}
	<div class="alert alert-error" style="margin:8px 10px;">
		{error}
		<button type="button" class="alert-close" onclick={() => (error = null)}>X</button>
	</div>
{/if}

{#if success}
	<div class="alert alert-success" style="margin:8px 10px;">
		{success}
	</div>
{/if}

<div class="compose-area">
	{#if loading}
		<div style="text-align:center;padding:20px;">
			<div class="loading-spinner"></div>
			Loading...
		</div>
	{:else}
		<div class="field-row">
			<span class="field-label">From:</span>
			<div class="field-input">
				<select class="hotmail-select" bind:value={fromMailboxId}>
					{#each mailboxes as mb (mb.id)}
						<option value={mb.id}>{mb.address}</option>
					{/each}
				</select>
			</div>
		</div>
		<div class="field-row">
			<span class="field-label">To:</span>
			<div class="field-input"><input type="text" bind:value={toField} placeholder="recipient@example.com" /></div>
		</div>
		<div class="field-row">
			<span class="field-label">Cc:</span>
			<div class="field-input"><input type="text" bind:value={ccField} /></div>
		</div>
		<div class="field-row">
			<span class="field-label">Bcc:</span>
			<div class="field-input"><input type="text" bind:value={bccField} /></div>
		</div>
		<div class="field-row">
			<span class="field-label">Subject:</span>
			<div class="field-input"><input type="text" bind:value={subject} /></div>
		</div>

		{#if attachments.length > 0}
			<div class="field-row" style="align-items:flex-start;">
				<span class="field-label">Attach:</span>
				<div class="field-input" style="display:flex;flex-wrap:wrap;gap:4px;">
					{#each attachments as att (att.id)}
						<span class="attachment-badge">
							📎 {att.filename} ({formatFileSize(att.size)})
							<button type="button" class="attachment-remove" onclick={() => handleRemoveAttachment(att.id)} title="Remove">X</button>
						</span>
					{/each}
				</div>
			</div>
		{/if}

		<div class="editor">
			<div class="editor-toolbar">
				<label style="font-size:10px;margin-right:8px;">
					<input type="checkbox" bind:checked={useRichText} /> Rich Text
				</label>
				{#if useRichText}
					<button type="button" onclick={() => document.execCommand('bold')}><b>B</b></button>
					<button type="button" onclick={() => document.execCommand('italic')}><i>I</i></button>
					<button type="button" onclick={() => document.execCommand('underline')}><u>U</u></button>
				{/if}
			</div>
			{#if useRichText}
				<div
					class="compose-textarea"
					contenteditable="true"
					style="min-height:250px;overflow-y:auto;padding:8px;background:#fff;border:1px solid var(--toolbar-border);"
					bind:innerHTML={htmlBody}
					role="textbox"
					tabindex="0"
				></div>
			{:else}
				<textarea class="compose-textarea" bind:value={textBody} placeholder="Type your message here..."></textarea>
			{/if}
		</div>

		{#if draftId}
			<div style="font-size:10px;color:var(--text-muted);padding:4px 10px;">
				Draft saved {draftId ? '(auto-save active)' : ''}
			</div>
		{/if}
	{/if}
</div>

<style>
	.attachment-badge {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		padding: 2px 8px;
		background: var(--toolbar-bg);
		border: 1px solid var(--toolbar-border);
		border-radius: 3px;
		font-size: 11px;
	}
	.attachment-remove {
		background: none;
		border: none;
		color: var(--error-red, #c00);
		cursor: pointer;
		font-size: 10px;
		font-weight: bold;
		padding: 0 2px;
	}
</style>
