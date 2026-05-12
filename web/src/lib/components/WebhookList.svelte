<script lang="ts">
	import { getWebhooksApi } from '$lib/api';
	import type { Webhook, WebhookDelivery, WebhookStatus } from '$lib/api';

	interface Props {
		onEdit?: (webhook: Webhook) => void;
		onRefresh?: () => void;
	}

	const { onEdit, onRefresh }: Props = $props();

	const webhooksApi = getWebhooksApi();

	let webhooks = $state<Webhook[]>([]);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let testingId = $state<string | null>(null);
	let deletingId = $state<string | null>(null);
	let togglingId = $state<string | null>(null);
	let testResult = $state<{ webhookId: string; success: boolean; message: string } | null>(null);
	let expandedId = $state<string | null>(null);
	let deliveries = $state<WebhookDelivery[]>([]);
	let deliveriesLoading = $state(false);

	$effect(() => { loadWebhooks(); });

	async function loadWebhooks(): Promise<void> {
		isLoading = true; error = null;
		try {
			const response = await webhooksApi.list();
			webhooks = response.items;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load webhooks';
		} finally { isLoading = false; }
	}

	async function handleTest(webhook: Webhook): Promise<void> {
		testingId = webhook.id; testResult = null;
		try {
			const delivery = await webhooksApi.test(webhook.id);
			testResult = { webhookId: webhook.id, success: delivery.success,
				message: delivery.success ? `Test successful (${delivery.statusCode})` : `Test failed: ${delivery.error || 'Unknown error'}` };
			await loadWebhooks();
		} catch (err) {
			testResult = { webhookId: webhook.id, success: false, message: err instanceof Error ? err.message : 'Test failed' };
		} finally { testingId = null; }
	}

	async function handleDelete(webhook: Webhook): Promise<void> {
		if (!confirm(`Delete webhook "${webhook.name}"?`)) return;
		deletingId = webhook.id;
		try {
			await webhooksApi.delete(webhook.id);
			await loadWebhooks(); onRefresh?.();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete webhook';
		} finally { deletingId = null; }
	}

	async function handleToggleStatus(webhook: Webhook): Promise<void> {
		togglingId = webhook.id;
		try {
			if (webhook.status === 'active') await webhooksApi.deactivate(webhook.id);
			else await webhooksApi.activate(webhook.id);
			await loadWebhooks(); onRefresh?.();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to update status';
		} finally { togglingId = null; }
	}

	function formatDate(timestamp: string | undefined): string {
		if (!timestamp) return 'Never';
		return new Date(timestamp).toLocaleString();
	}

	async function toggleDeliveries(webhook: Webhook): Promise<void> {
		if (expandedId === webhook.id) { expandedId = null; return; }
		expandedId = webhook.id; deliveriesLoading = true;
		try {
			const response = await webhooksApi.listDeliveries(webhook.id, { pageSize: 10 });
			deliveries = response.items;
		} catch { deliveries = []; }
		finally { deliveriesLoading = false; }
	}
</script>

{#if error}
	<div class="alert alert-error">
		{error}
		<button type="button" class="alert-close" onclick={() => (error = null)}>X</button>
	</div>
{/if}

{#if testResult}
	<div class="alert {testResult.success ? 'alert-success' : 'alert-error'}" style="margin-bottom:8px;">
		{testResult.message}
		<button type="button" class="alert-close" onclick={() => (testResult = null)}>X</button>
	</div>
{/if}

{#if isLoading && webhooks.length === 0}
	<div style="text-align:center;padding:20px;">
		<div class="loading-spinner"></div>
		Loading webhooks...
	</div>
{:else if webhooks.length === 0}
	<p style="padding:20px;text-align:center;color:var(--text-muted);">
		No webhooks configured. Create one to receive notifications.
	</p>
{:else}
	<table class="msg-table">
		<thead>
			<tr>
				<th>Name</th>
				<th>URL</th>
				<th>Events</th>
				<th>Status</th>
				<th>Success/Fail</th>
				<th>Last Triggered</th>
				<th style="text-align:right;">Actions</th>
			</tr>
		</thead>
		<tbody>
			{#each webhooks as webhook (webhook.id)}
				<tr>
					<td><b>{webhook.name}</b></td>
					<td style="font-family:monospace;font-size:10px;max-width:200px;overflow:hidden;text-overflow:ellipsis;">{webhook.url}</td>
					<td style="font-size:10px;">{webhook.events.join(', ')}</td>
					<td>{webhook.status}</td>
					<td>{webhook.successCount}/{webhook.failureCount}</td>
					<td style="font-size:10px;">{formatDate(webhook.lastTriggeredAt)}</td>
					<td style="text-align:right;white-space:nowrap;">
						<button type="button" class="hotmail-btn" onclick={() => handleToggleStatus(webhook)} disabled={togglingId === webhook.id}>
							{webhook.status === 'active' ? 'Pause' : 'Activate'}
						</button>
						<button type="button" class="hotmail-btn" onclick={() => handleTest(webhook)} disabled={testingId === webhook.id || webhook.status !== 'active'}>
							Test
						</button>
						<button type="button" class="hotmail-btn" onclick={() => onEdit?.(webhook)}>Edit</button>
						<button type="button" class="hotmail-btn" style="color:var(--error-red);" onclick={() => handleDelete(webhook)} disabled={deletingId === webhook.id}>
							Delete
						</button>
					</td>
				</tr>
				<tr>
					<td colspan="7" style="padding:0;">
						<button type="button" style="font-size:10px;color:var(--link-blue);background:none;border:none;cursor:pointer;padding:2px 5px;" onclick={() => toggleDeliveries(webhook)}>
							{expandedId === webhook.id ? '▼' : '▶'} Delivery History
						</button>
						{#if expandedId === webhook.id}
							<div style="padding:4px 5px 8px;">
								{#if deliveriesLoading}
									<div class="loading-spinner" style="width:12px;height:12px;"></div>
								{:else if deliveries.length === 0}
									<span style="font-size:10px;color:var(--text-muted);">No deliveries yet.</span>
								{:else}
									{#each deliveries as delivery (delivery.id)}
										<div style="font-size:10px;padding:2px 0;display:flex;gap:8px;">
											<span style="color:{delivery.success ? 'var(--success-green)' : 'var(--error-red)'};">{delivery.success ? 'OK' : 'FAIL'}</span>
											<span>{delivery.event}</span>
											<span>{delivery.statusCode}</span>
											<span>{delivery.duration}ms</span>
											{#if delivery.error}<span style="color:var(--error-red);">{delivery.error}</span>{/if}
											<span style="margin-left:auto;color:var(--text-muted);">{formatDate(delivery.createdAt)}</span>
										</div>
									{/each}
								{/if}
							</div>
						{/if}
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
{/if}
