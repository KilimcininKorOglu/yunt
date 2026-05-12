<script lang="ts">
	import { page } from '$app/stores';
	import { getMessagesApi } from '$lib/api';
	import type { Message } from '$lib/api/types';

	const messagesApi = getMessagesApi();

	let to = $state('');
	let cc = $state('');
	let bcc = $state('');
	let subject = $state('');
	let body = $state('');
	let showInfo = $state(false);
	let loading = $state(false);

	const replyToId = $derived($page.url.searchParams.get('replyTo'));
	const forwardId = $derived($page.url.searchParams.get('forward'));

	$effect(() => {
		if (replyToId) {
			loadOriginal(replyToId, 'reply');
		} else if (forwardId) {
			loadOriginal(forwardId, 'forward');
		}
	});

	async function loadOriginal(id: string, mode: 'reply' | 'forward'): Promise<void> {
		loading = true;
		try {
			const msg = await messagesApi.get(id);
			if (mode === 'reply') {
				to = msg.from.address;
				subject = msg.subject.startsWith('Re:') ? msg.subject : `Re: ${msg.subject}`;
				body = `\n\n--- Original Message ---\nFrom: ${msg.from.name || msg.from.address}\nDate: ${new Date(msg.receivedAt).toLocaleString()}\n\n${msg.textBody || ''}`;
			} else {
				subject = msg.subject.startsWith('Fwd:') ? msg.subject : `Fwd: ${msg.subject}`;
				body = `\n\n--- Forwarded Message ---\nFrom: ${msg.from.name || msg.from.address}\nTo: ${msg.to.map(t => t.address).join(', ')}\nDate: ${new Date(msg.receivedAt).toLocaleString()}\nSubject: ${msg.subject}\n\n${msg.textBody || ''}`;
			}
		} catch (err) {
			console.error('Failed to load original message:', err);
		} finally {
			loading = false;
		}
	}

	function handleSend(): void {
		showInfo = true;
	}
</script>

<svelte:head>
	<title>Compose - Yunt Mail</title>
</svelte:head>

<div class="toolbar">
	<div class="toolbar-left">
		<button type="button" class="hotmail-btn toolbar-btn-primary" onclick={handleSend}>Send</button>
		<span class="toolbar-sep">|</span>
		<button type="button" class="hotmail-btn" disabled>Save Draft</button>
		<button type="button" class="hotmail-btn" disabled>Attach</button>
		<span class="toolbar-sep">|</span>
		<a href="/inbox" class="hotmail-btn">Cancel</a>
	</div>
</div>

{#if showInfo}
	<div class="alert alert-info" style="margin:8px 10px;">
		Mail sending is not supported — Yunt is a receiving mail server.
		<button type="button" class="alert-close" onclick={() => (showInfo = false)}>X</button>
	</div>
{/if}

<div class="compose-area">
	{#if loading}
		<div style="text-align:center;padding:20px;">
			<div class="loading-spinner"></div>
			Loading original message...
		</div>
	{:else}
		<div class="field-row">
			<span class="field-label">To:</span>
			<div class="field-input"><input type="text" bind:value={to} /></div>
		</div>
		<div class="field-row">
			<span class="field-label">Cc:</span>
			<div class="field-input"><input type="text" bind:value={cc} /></div>
		</div>
		<div class="field-row">
			<span class="field-label">Bcc:</span>
			<div class="field-input"><input type="text" bind:value={bcc} /></div>
		</div>
		<div class="field-row">
			<span class="field-label">Subject:</span>
			<div class="field-input"><input type="text" bind:value={subject} /></div>
		</div>

		<div class="editor">
			<div class="editor-toolbar">
				<select disabled><option>Verdana</option></select>
				<select disabled><option>10pt</option></select>
				<button type="button" disabled><b>B</b></button>
				<button type="button" disabled><i>I</i></button>
				<button type="button" disabled><u>U</u></button>
			</div>
			<textarea class="compose-textarea" bind:value={body} placeholder="Type your message here..."></textarea>
		</div>
	{/if}
</div>
