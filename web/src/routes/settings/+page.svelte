<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { authStore } from '$stores/auth.svelte';
	import { notificationsStore } from '$stores/notifications.svelte';
	import { pollingService } from '$lib/services/polling.svelte';
	import WebhookList from '$components/WebhookList.svelte';
	import WebhookForm from '$components/WebhookForm.svelte';
	import MailboxSettings from '$components/MailboxSettings.svelte';
	import type { Webhook } from '$lib/api';

	type Tab = 'general' | 'notifications' | 'webhooks' | 'mailboxes';

	const tabs: { id: Tab; label: string }[] = [
		{ id: 'general', label: 'General' },
		{ id: 'webhooks', label: 'Webhooks' },
		{ id: 'mailboxes', label: 'Mailboxes' }
	];

	let activeTab = $state<Tab>('general');

	let showWebhookForm = $state(false);
	let editingWebhook = $state<Webhook | undefined>(undefined);

	$effect(() => {
		const hash = $page.url.hash.replace('#', '') as Tab;
		if (hash && tabs.some((t) => t.id === hash)) {
			activeTab = hash;
		}
	});

	function setActiveTab(tab: Tab): void {
		activeTab = tab;
		const url = new URL(window.location.href);
		url.hash = tab;
		window.history.replaceState({}, '', url.toString());
	}

	function openWebhookForm(webhook?: Webhook): void {
		editingWebhook = webhook;
		showWebhookForm = true;
	}

	function closeWebhookForm(): void {
		showWebhookForm = false;
		editingWebhook = undefined;
	}

	function handleWebhookSuccess(): void {
		closeWebhookForm();
	}

	async function handleLogout(): Promise<void> {
		await authStore.logout();
		await goto('/login');
	}

	function handleToggleToast(): void {
		notificationsStore.updatePreferences({ showToast: !notificationsStore.preferences.showToast });
	}

	function handlePollingIntervalChange(event: Event): void {
		const select = event.target as HTMLSelectElement;
		const interval = parseInt(select.value, 10);
		pollingService.setPollingInterval(interval);
	}

	function handleTestNotification(): void {
		notificationsStore.info('Test Notification', 'This is a test notification.');
	}

	const pollingIntervals = [
		{ value: 15000, label: '15 seconds' },
		{ value: 30000, label: '30 seconds' },
		{ value: 60000, label: '1 minute' },
		{ value: 120000, label: '2 minutes' },
		{ value: 300000, label: '5 minutes' }
	];
</script>

<svelte:head>
	<title>Options - Yunt Mail</title>
</svelte:head>

<div class="inbox-layout">
	<aside class="sidebar">
		<div class="sidebar-header">
			<span class="sidebar-title">Options</span>
		</div>
		<div class="folder-list">
			{#each tabs as tab (tab.id)}
				<button
					type="button"
					class="folder-item"
					class:active={activeTab === tab.id}
					onclick={() => setActiveTab(tab.id)}
				>
					<span class="folder-name">{tab.label}</span>
				</button>
			{/each}
		</div>
	</aside>

	<div class="inbox-content">
		<div class="options-area">
			{#if activeTab === 'general'}
				<h2>General Settings</h2>

				<div class="opt-section">
					<h3>Profile</h3>
					{#if authStore.user}
						<table class="server-info-table"><tbody>
							<tr><td class="lbl">Username</td><td>{authStore.user.username}</td></tr>
							<tr><td class="lbl">Email</td><td>{authStore.user.email}</td></tr>
							<tr><td class="lbl">Role</td><td>{authStore.user.role}</td></tr>
							{#if authStore.user.displayName}
								<tr><td class="lbl">Display Name</td><td>{authStore.user.displayName}</td></tr>
							{/if}
						</tbody></table>
					{/if}
				</div>

				<div class="opt-section">
					<h3>Server Information</h3>
					<table class="server-info-table"><tbody>
						<tr><td class="lbl">SMTP Server</td><td><code>localhost:1025</code></td></tr>
						<tr><td class="lbl">IMAP Server</td><td><code>localhost:1143</code></td></tr>
						<tr><td class="lbl">Web UI</td><td><code>localhost:8025</code></td></tr>
					</tbody></table>
				</div>

				<div class="opt-section">
					<h3>Notifications</h3>
					<table class="server-info-table"><tbody>
						<tr>
							<td class="lbl">Toast Notifications</td>
							<td>
								<label>
									<input type="checkbox" checked={notificationsStore.preferences.showToast} onchange={handleToggleToast} />
									Show toast on new messages
								</label>
							</td>
						</tr>
						<tr>
							<td class="lbl">Polling Interval</td>
							<td>
								<select class="hotmail-select" onchange={handlePollingIntervalChange}>
									{#each pollingIntervals as opt (opt.value)}
										<option value={opt.value}>{opt.label}</option>
									{/each}
								</select>
							</td>
						</tr>
						<tr>
							<td class="lbl">Test</td>
							<td>
								<button type="button" class="hotmail-btn" onclick={handleTestNotification}>Send Test Notification</button>
							</td>
						</tr>
					</tbody></table>
				</div>

				<div class="opt-section">
					<h3 style="color:var(--error-red);">Danger Zone</h3>
					<button type="button" class="hotmail-btn" style="color:var(--error-red);border-color:var(--error-red);" onclick={handleLogout}>
						Sign Out
					</button>
				</div>
			{:else if activeTab === 'webhooks'}
				<h2>Webhooks</h2>
				{#if !showWebhookForm}
					<button type="button" class="hotmail-btn" style="margin-bottom:10px;" onclick={() => openWebhookForm()}>
						New Webhook
					</button>
				{/if}

				{#if showWebhookForm}
					<div class="info-box" style="margin-bottom:12px;">
						<div class="info-box-header">{editingWebhook ? 'Edit Webhook' : 'Create Webhook'}</div>
						<div class="info-box-body">
							<WebhookForm webhook={editingWebhook} onSuccess={handleWebhookSuccess} onCancel={closeWebhookForm} />
						</div>
					</div>
				{/if}

				<WebhookList onEdit={(webhook) => openWebhookForm(webhook)} />
			{:else if activeTab === 'mailboxes'}
				<h2>Mailboxes</h2>
				<MailboxSettings />
			{/if}
		</div>
	</div>
</div>
