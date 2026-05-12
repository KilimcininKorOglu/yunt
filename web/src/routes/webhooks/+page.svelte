<script lang="ts">
	import { goto } from '$app/navigation';
	import { WebhookForm, WebhookList } from '$components';
	import type { Webhook } from '$lib/api';
	import { authStore } from '$stores/auth';

	let showForm = $state(false);
	let editingWebhook = $state<Webhook | undefined>(undefined);

	function openForm(webhook?: Webhook): void {
		editingWebhook = webhook;
		showForm = true;
	}

	function closeForm(): void {
		editingWebhook = undefined;
		showForm = false;
	}

	function handleSuccess(): void {
		closeForm();
	}
</script>

<svelte:head>
	<title>Webhooks - Yunt Mail</title>
</svelte:head>

<div class="options-area">
	<h2>Webhook Management</h2>
	<p style="font-size:11px;color:var(--text-muted);margin-bottom:10px;">
		Configure webhooks to receive notifications when events occur.
	</p>

	{#if !showForm}
		<button type="button" class="hotmail-btn" style="margin-bottom:10px;" onclick={() => openForm()}>
			New Webhook
		</button>
	{/if}

	{#if showForm}
		<div class="info-box" style="margin-bottom:12px;">
			<div class="info-box-header">{editingWebhook ? 'Edit Webhook' : 'Create Webhook'}</div>
			<div class="info-box-body">
				<WebhookForm webhook={editingWebhook} onSuccess={handleSuccess} onCancel={closeForm} />
			</div>
		</div>
	{/if}

	<WebhookList onEdit={(webhook) => openForm(webhook)} />
</div>
