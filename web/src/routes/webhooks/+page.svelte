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

	function handleLogout(): void {
		authStore.logout();
		goto('/login');
	}
</script>

<div class="min-h-screen bg-secondary-50">
	<header class="border-b border-secondary-200 bg-white">
		<div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
			<div class="flex h-16 items-center justify-between">
				<div class="flex items-center gap-4">
					<a href="/" class="text-2xl font-bold text-primary-600">Yunt</a>
					<span class="text-secondary-300">/</span>
					<h1 class="text-lg font-medium text-secondary-900">Webhooks</h1>
				</div>
				<div class="flex items-center gap-4">
					{#if authStore.user}
						<span class="text-sm text-secondary-500">
							{authStore.user.displayName || authStore.user.username}
						</span>
					{/if}
					<button class="btn-secondary text-sm" onclick={handleLogout}>Logout</button>
				</div>
			</div>
		</div>
	</header>

	<div class="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
		<div class="mb-6 flex items-center justify-between">
			<div>
				<h2 class="text-xl font-semibold text-secondary-900">Webhook Management</h2>
				<p class="mt-1 text-sm text-secondary-500">
					Configure webhooks to receive notifications when events occur.
				</p>
			</div>
			{#if !showForm}
				<button class="btn-primary" onclick={() => openForm()}>New Webhook</button>
			{/if}
		</div>

		{#if showForm}
			<div class="card mb-6 p-6">
				<h3 class="mb-4 text-lg font-medium text-secondary-900">
					{editingWebhook ? 'Edit Webhook' : 'Create Webhook'}
				</h3>
				<WebhookForm
					webhook={editingWebhook}
					onSuccess={handleSuccess}
					onCancel={closeForm}
				/>
			</div>
		{/if}

		<WebhookList onEdit={(webhook: Webhook) => openForm(webhook)} />
	</div>
</div>
