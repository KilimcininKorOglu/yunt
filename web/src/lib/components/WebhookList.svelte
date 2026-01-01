<script lang="ts">
	import { getWebhooksApi } from '$lib/api';
	import type { Webhook, WebhookStatus } from '$lib/api';

	interface Props {
		/** Callback when edit is requested */
		onEdit?: (webhook: Webhook) => void;
		/** Callback when webhook list changes */
		onRefresh?: () => void;
	}

	const { onEdit, onRefresh }: Props = $props();

	// API instance
	const webhooksApi = getWebhooksApi();

	// State
	let webhooks = $state<Webhook[]>([]);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let testingId = $state<string | null>(null);
	let deletingId = $state<string | null>(null);
	let togglingId = $state<string | null>(null);
	let testResult = $state<{ webhookId: string; success: boolean; message: string } | null>(null);

	// Load webhooks on mount
	$effect(() => {
		loadWebhooks();
	});

	async function loadWebhooks(): Promise<void> {
		isLoading = true;
		error = null;

		try {
			const response = await webhooksApi.list();
			webhooks = response.items;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load webhooks';
		} finally {
			isLoading = false;
		}
	}

	async function handleTest(webhook: Webhook): Promise<void> {
		testingId = webhook.id;
		testResult = null;

		try {
			const delivery = await webhooksApi.test(webhook.id);
			testResult = {
				webhookId: webhook.id,
				success: delivery.success,
				message: delivery.success
					? `Test successful (${delivery.statusCode})`
					: `Test failed: ${delivery.error || 'Unknown error'}`
			};
			// Refresh to get updated stats
			await loadWebhooks();
		} catch (err) {
			testResult = {
				webhookId: webhook.id,
				success: false,
				message: err instanceof Error ? err.message : 'Test failed'
			};
		} finally {
			testingId = null;
		}
	}

	async function handleDelete(webhook: Webhook): Promise<void> {
		if (!confirm(`Are you sure you want to delete the webhook "${webhook.name}"?`)) {
			return;
		}

		deletingId = webhook.id;

		try {
			await webhooksApi.delete(webhook.id);
			await loadWebhooks();
			onRefresh?.();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete webhook';
		} finally {
			deletingId = null;
		}
	}

	async function handleToggleStatus(webhook: Webhook): Promise<void> {
		togglingId = webhook.id;

		try {
			if (webhook.status === 'active') {
				await webhooksApi.deactivate(webhook.id);
			} else {
				await webhooksApi.activate(webhook.id);
			}
			await loadWebhooks();
			onRefresh?.();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to update webhook status';
		} finally {
			togglingId = null;
		}
	}

	function getStatusColor(status: WebhookStatus): string {
		switch (status) {
			case 'active':
				return 'bg-green-100 text-green-700';
			case 'inactive':
				return 'bg-secondary-100 text-secondary-600';
			case 'failed':
				return 'bg-red-100 text-red-700';
			default:
				return 'bg-secondary-100 text-secondary-600';
		}
	}

	function formatDate(timestamp: string | undefined): string {
		if (!timestamp) return 'Never';
		return new Date(timestamp).toLocaleString();
	}

	function clearTestResult(): void {
		testResult = null;
	}
</script>

<div class="space-y-4">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<h3 class="text-lg font-semibold text-secondary-900">Webhooks</h3>
		<button class="btn-secondary text-sm" onclick={loadWebhooks} disabled={isLoading}>
			{#if isLoading}
				<svg class="mr-2 h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
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
			{/if}
			Refresh
		</button>
	</div>

	<!-- Error Alert -->
	{#if error}
		<div class="rounded-lg border border-red-200 bg-red-50 p-4" role="alert">
			<div class="flex items-start gap-3">
				<svg
					class="mt-0.5 h-5 w-5 flex-shrink-0 text-red-500"
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
				<button
					class="ml-auto text-red-500 hover:text-red-700"
					onclick={() => (error = null)}
					aria-label="Dismiss error"
				>
					<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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

	<!-- Test Result Alert -->
	{#if testResult}
		<div
			class="rounded-lg border p-4 {testResult.success
				? 'border-green-200 bg-green-50'
				: 'border-red-200 bg-red-50'}"
			role="alert"
		>
			<div class="flex items-start gap-3">
				{#if testResult.success}
					<svg
						class="mt-0.5 h-5 w-5 flex-shrink-0 text-green-500"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M5 13l4 4L19 7"
						/>
					</svg>
				{:else}
					<svg
						class="mt-0.5 h-5 w-5 flex-shrink-0 text-red-500"
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
				{/if}
				<p class="text-sm {testResult.success ? 'text-green-700' : 'text-red-700'}">
					{testResult.message}
				</p>
				<button
					class="ml-auto {testResult.success
						? 'text-green-500 hover:text-green-700'
						: 'text-red-500 hover:text-red-700'}"
					onclick={clearTestResult}
					aria-label="Dismiss test result"
				>
					<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
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

	<!-- Loading State -->
	{#if isLoading && webhooks.length === 0}
		<div class="flex items-center justify-center py-12">
			<div
				class="h-8 w-8 animate-spin rounded-full border-4 border-primary-200 border-t-primary-600"
			></div>
		</div>
	{:else if webhooks.length === 0}
		<!-- Empty State -->
		<div class="rounded-lg border border-dashed border-secondary-300 p-8 text-center">
			<svg
				class="mx-auto h-12 w-12 text-secondary-400"
				fill="none"
				viewBox="0 0 24 24"
				stroke="currentColor"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="1.5"
					d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"
				/>
			</svg>
			<h4 class="mt-4 text-lg font-medium text-secondary-900">No webhooks configured</h4>
			<p class="mt-2 text-sm text-secondary-500">
				Create a webhook to receive notifications when events occur.
			</p>
		</div>
	{:else}
		<!-- Webhook List -->
		<div class="space-y-3">
			{#each webhooks as webhook (webhook.id)}
				<div class="card p-4">
					<div class="flex items-start justify-between gap-4">
						<div class="min-w-0 flex-1">
							<div class="flex items-center gap-2">
								<h4 class="font-medium text-secondary-900 truncate">
									{webhook.name}
								</h4>
								<span
									class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium {getStatusColor(
										webhook.status
									)}"
								>
									{webhook.status}
								</span>
							</div>
							<p class="mt-1 text-sm text-secondary-500 truncate font-mono">
								{webhook.url}
							</p>
							<div
								class="mt-2 flex flex-wrap items-center gap-2 text-xs text-secondary-400"
							>
								<span>Events: {webhook.events.join(', ')}</span>
								<span>|</span>
								<span>Success: {webhook.successCount}</span>
								<span>|</span>
								<span>Failed: {webhook.failureCount}</span>
							</div>
							{#if webhook.lastTriggeredAt}
								<p class="mt-1 text-xs text-secondary-400">
									Last triggered: {formatDate(webhook.lastTriggeredAt)}
								</p>
							{/if}
							{#if webhook.lastError}
								<p class="mt-1 text-xs text-red-500 truncate">
									Last error: {webhook.lastError}
								</p>
							{/if}
						</div>
						<div class="flex items-center gap-2 flex-shrink-0">
							<!-- Toggle Status -->
							<button
								class="rounded-lg p-2 text-secondary-500 hover:bg-secondary-100 hover:text-secondary-700 disabled:opacity-50"
								onclick={() => handleToggleStatus(webhook)}
								disabled={togglingId === webhook.id}
								title={webhook.status === 'active' ? 'Deactivate' : 'Activate'}
							>
								{#if togglingId === webhook.id}
									<svg
										class="h-5 w-5 animate-spin"
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
								{:else if webhook.status === 'active'}
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
											d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z"
										/>
									</svg>
								{:else}
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
											d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"
										/>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
										/>
									</svg>
								{/if}
							</button>

							<!-- Test -->
							<button
								class="rounded-lg p-2 text-secondary-500 hover:bg-secondary-100 hover:text-secondary-700 disabled:opacity-50"
								onclick={() => handleTest(webhook)}
								disabled={testingId === webhook.id || webhook.status !== 'active'}
								title="Test webhook"
							>
								{#if testingId === webhook.id}
									<svg
										class="h-5 w-5 animate-spin"
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
										class="h-5 w-5"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M13 10V3L4 14h7v7l9-11h-7z"
										/>
									</svg>
								{/if}
							</button>

							<!-- Edit -->
							<button
								class="rounded-lg p-2 text-secondary-500 hover:bg-secondary-100 hover:text-secondary-700"
								onclick={() => onEdit?.(webhook)}
								title="Edit webhook"
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
										d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
									/>
								</svg>
							</button>

							<!-- Delete -->
							<button
								class="rounded-lg p-2 text-red-500 hover:bg-red-50 hover:text-red-700 disabled:opacity-50"
								onclick={() => handleDelete(webhook)}
								disabled={deletingId === webhook.id}
								title="Delete webhook"
							>
								{#if deletingId === webhook.id}
									<svg
										class="h-5 w-5 animate-spin"
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
										class="h-5 w-5"
										fill="none"
										viewBox="0 0 24 24"
										stroke="currentColor"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											stroke-width="2"
											d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
										/>
									</svg>
								{/if}
							</button>
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
