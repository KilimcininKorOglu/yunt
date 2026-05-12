<script lang="ts">
	import { getWebhooksApi } from '$lib/api';
	import type { Webhook, WebhookCreateInput, WebhookUpdateInput, WebhookEvent } from '$lib/api';

	interface Props {
		webhook?: Webhook;
		onSuccess?: (webhook: Webhook) => void;
		onCancel?: () => void;
	}

	const { webhook, onSuccess, onCancel }: Props = $props();

	const webhooksApi = getWebhooksApi();

	let name = $state('');
	let url = $state('');
	let secret = $state('');
	let events = $state<WebhookEvent[]>([]);
	let maxRetries = $state(3);
	let timeoutSeconds = $state(30);
	let headers = $state<{ key: string; value: string }[]>([]);

	$effect(() => {
		if (webhook) {
			name = webhook.name; url = webhook.url;
			events = [...webhook.events]; maxRetries = webhook.maxRetries; timeoutSeconds = webhook.timeoutSeconds;
			headers = webhook.headers ? Object.entries(webhook.headers).map(([key, value]) => ({ key, value })) : [];
		} else {
			name = ''; url = ''; secret = ''; events = []; maxRetries = 3; timeoutSeconds = 30; headers = [];
		}
	});

	let isSubmitting = $state(false);
	let formError = $state<string | null>(null);
	let nameError = $state<string | null>(null);
	let urlError = $state<string | null>(null);

	const isEditing = $derived(!!webhook);

	const availableEvents: { value: WebhookEvent; label: string; description: string }[] = [
		{ value: 'message.received', label: 'Message Received', description: 'When a new email is received' },
		{ value: 'message.deleted', label: 'Message Deleted', description: 'When an email is deleted' },
		{ value: 'mailbox.created', label: 'Mailbox Created', description: 'When a new mailbox is created' },
		{ value: 'mailbox.deleted', label: 'Mailbox Deleted', description: 'When a mailbox is deleted' }
	];

	const isFormValid = $derived(name.trim().length > 0 && url.trim().length > 0 && events.length > 0 && !nameError && !urlError);

	function validateName(): void {
		if (!name.trim()) nameError = 'Name is required';
		else if (name.trim().length < 2) nameError = 'Min 2 characters';
		else nameError = null;
	}

	function validateUrl(): void {
		if (!url.trim()) { urlError = 'URL is required'; return; }
		try {
			const p = new URL(url.trim());
			urlError = !['http:', 'https:'].includes(p.protocol) ? 'Must use HTTP or HTTPS' : null;
		} catch { urlError = 'Invalid URL'; }
	}

	function toggleEvent(event: WebhookEvent): void {
		if (events.includes(event)) events = events.filter((e) => e !== event);
		else events = [...events, event];
	}

	function addHeader(): void { headers = [...headers, { key: '', value: '' }]; }
	function removeHeader(index: number): void { headers = headers.filter((_, i) => i !== index); }
	function updateHeader(index: number, field: 'key' | 'value', value: string): void {
		headers = headers.map((h, i) => (i === index ? { ...h, [field]: value } : h));
	}

	async function handleSubmit(event: Event): Promise<void> {
		event.preventDefault();
		validateName(); validateUrl();
		if (!isFormValid) return;

		isSubmitting = true; formError = null;
		try {
			const headersObj: Record<string, string> = {};
			for (const h of headers) { if (h.key.trim() && h.value.trim()) headersObj[h.key.trim()] = h.value.trim(); }

			let result: Webhook;
			if (isEditing && webhook) {
				const input: WebhookUpdateInput = { name: name.trim(), url: url.trim(), events, maxRetries, timeoutSeconds,
					headers: Object.keys(headersObj).length > 0 ? headersObj : undefined };
				if (secret.trim()) input.secret = secret.trim();
				result = await webhooksApi.update(webhook.id, input);
			} else {
				const input: WebhookCreateInput = { name: name.trim(), url: url.trim(), events, maxRetries, timeoutSeconds,
					headers: Object.keys(headersObj).length > 0 ? headersObj : undefined };
				if (secret.trim()) input.secret = secret.trim();
				result = await webhooksApi.create(input);
			}
			onSuccess?.(result);
		} catch (err) {
			formError = err instanceof Error ? err.message : 'Failed to save webhook';
		} finally { isSubmitting = false; }
	}
</script>

{#if formError}
	<div class="alert alert-error" style="margin-bottom:8px;">{formError}</div>
{/if}

<form onsubmit={handleSubmit}>
	<table class="server-info-table"><tbody>
		<tr>
			<td class="lbl">Name *</td>
			<td>
				<input type="text" class="hotmail-input" bind:value={name} onblur={validateName} disabled={isSubmitting} style="width:250px;" />
				{#if nameError}<br><span style="color:var(--error-red);font-size:10px;">{nameError}</span>{/if}
			</td>
		</tr>
		<tr>
			<td class="lbl">URL *</td>
			<td>
				<input type="url" class="hotmail-input" bind:value={url} onblur={validateUrl} disabled={isSubmitting} style="width:350px;font-family:monospace;font-size:10px;" />
				{#if urlError}<br><span style="color:var(--error-red);font-size:10px;">{urlError}</span>{/if}
			</td>
		</tr>
		<tr>
			<td class="lbl">Secret</td>
			<td>
				<input type="password" class="hotmail-input" bind:value={secret} disabled={isSubmitting} style="width:250px;" placeholder={isEditing ? 'Leave blank to keep' : 'Optional'} />
			</td>
		</tr>
		<tr>
			<td class="lbl">Events *</td>
			<td>
				{#each availableEvents as ev (ev.value)}
					<label style="display:block;padding:2px 0;">
						<input type="checkbox" checked={events.includes(ev.value)} onchange={() => toggleEvent(ev.value)} disabled={isSubmitting} />
						<b>{ev.label}</b> <span style="color:var(--text-muted);font-size:10px;">— {ev.description}</span>
					</label>
				{/each}
			</td>
		</tr>
		<tr>
			<td class="lbl">Max Retries</td>
			<td><input type="number" class="hotmail-input" bind:value={maxRetries} min="0" max="10" disabled={isSubmitting} style="width:60px;" /></td>
		</tr>
		<tr>
			<td class="lbl">Timeout (s)</td>
			<td><input type="number" class="hotmail-input" bind:value={timeoutSeconds} min="5" max="60" disabled={isSubmitting} style="width:60px;" /></td>
		</tr>
		<tr>
			<td class="lbl">Headers</td>
			<td>
				{#each headers as header, index (index)}
					<div style="display:flex;gap:4px;margin-bottom:4px;">
						<input type="text" class="hotmail-input" value={header.key} oninput={(e) => updateHeader(index, 'key', (e.target as HTMLInputElement).value)} placeholder="Name" disabled={isSubmitting} style="width:120px;" />
						<input type="text" class="hotmail-input" value={header.value} oninput={(e) => updateHeader(index, 'value', (e.target as HTMLInputElement).value)} placeholder="Value" disabled={isSubmitting} style="width:180px;" />
						<button type="button" class="hotmail-btn" onclick={() => removeHeader(index)} disabled={isSubmitting} style="color:var(--error-red);">X</button>
					</div>
				{/each}
				<button type="button" class="hotmail-btn" onclick={addHeader} disabled={isSubmitting}>+ Add Header</button>
			</td>
		</tr>
	</tbody></table>

	<div style="margin-top:10px;display:flex;gap:6px;">
		<button type="submit" class="hotmail-btn toolbar-btn-primary" disabled={isSubmitting || !isFormValid}>
			{isSubmitting ? 'Saving...' : (isEditing ? 'Update Webhook' : 'Create Webhook')}
		</button>
		<button type="button" class="hotmail-btn" onclick={onCancel} disabled={isSubmitting}>Cancel</button>
	</div>
</form>
