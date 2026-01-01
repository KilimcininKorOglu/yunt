<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { authStore } from '$stores/auth';
	import { notificationsStore } from '$stores/notifications';
	import { pollingService } from '$lib/services/polling';
	import WebhookList from '$components/WebhookList.svelte';
	import WebhookForm from '$components/WebhookForm.svelte';
	import MailboxSettings from '$components/MailboxSettings.svelte';
	import type { Webhook } from '$lib/api';

	// Tab definitions
	type Tab = 'general' | 'notifications' | 'webhooks' | 'mailboxes';

	const tabs: { id: Tab; label: string; icon: string }[] = [
		{ id: 'general', label: 'General', icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z' },
		{ id: 'notifications', label: 'Notifications', icon: 'M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9' },
		{ id: 'webhooks', label: 'Webhooks', icon: 'M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1' },
		{ id: 'mailboxes', label: 'Mailboxes', icon: 'M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z' }
	];

	// Active tab state
	let activeTab = $state<Tab>('general');

	// Webhook form state
	let showWebhookForm = $state(false);
	let editingWebhook = $state<Webhook | undefined>(undefined);

	// Get tab from URL hash on mount
	$effect(() => {
		const hash = $page.url.hash.replace('#', '') as Tab;
		if (hash && tabs.some((t) => t.id === hash)) {
			activeTab = hash;
		}
	});

	function setActiveTab(tab: Tab): void {
		activeTab = tab;
		// Update URL hash without navigation
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

	// Notification settings handlers
	function handleToggleNotifications(): void {
		notificationsStore.toggleNotifications();
	}

	function handleToggleToast(): void {
		notificationsStore.updatePreferences({ showToast: !notificationsStore.preferences.showToast });
	}

	function handleToggleSound(): void {
		notificationsStore.updatePreferences({ playSound: !notificationsStore.preferences.playSound });
	}

	async function handleToggleDesktop(): Promise<void> {
		if (!notificationsStore.preferences.showDesktop) {
			// Request permission if enabling
			const granted = await notificationsStore.requestDesktopPermission();
			if (granted) {
				notificationsStore.updatePreferences({ showDesktop: true });
			}
		} else {
			notificationsStore.updatePreferences({ showDesktop: false });
		}
	}

	function handlePollingIntervalChange(event: Event): void {
		const select = event.target as HTMLSelectElement;
		const interval = parseInt(select.value, 10);
		pollingService.setPollingInterval(interval);
	}

	function handleTestNotification(): void {
		notificationsStore.info('Test Notification', 'This is a test notification to verify your settings.');
	}

	// Polling interval options (in milliseconds)
	const pollingIntervals = [
		{ value: 15000, label: '15 seconds' },
		{ value: 30000, label: '30 seconds' },
		{ value: 60000, label: '1 minute' },
		{ value: 120000, label: '2 minutes' },
		{ value: 300000, label: '5 minutes' }
	];
</script>

<svelte:head>
	<title>Settings - Yunt</title>
</svelte:head>

<main class="min-h-screen">
	<!-- Header -->
	<header class="border-b border-secondary-200 bg-white">
		<div class="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
			<div class="flex h-16 items-center justify-between">
				<div class="flex items-center gap-4">
					<a href="/" class="text-2xl font-bold text-primary-600">Yunt</a>
					<span class="text-secondary-300">/</span>
					<h1 class="text-lg font-medium text-secondary-900">Settings</h1>
				</div>
				<div class="flex items-center gap-4">
					{#if authStore.user}
						<span class="text-sm text-secondary-500">
							{authStore.user.displayName || authStore.user.username}
						</span>
					{/if}
					<button
						class="btn-secondary text-sm"
						onclick={handleLogout}
					>
						Logout
					</button>
				</div>
			</div>
		</div>
	</header>

	<div class="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
		<div class="lg:flex lg:gap-8">
			<!-- Sidebar Tabs -->
			<nav class="mb-6 lg:mb-0 lg:w-64 lg:flex-shrink-0">
				<div class="card overflow-hidden">
					<ul class="divide-y divide-secondary-200">
						{#each tabs as tab (tab.id)}
							<li>
								<button
									class="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors {activeTab === tab.id ? 'bg-primary-50 text-primary-700' : 'text-secondary-700 hover:bg-secondary-50'}"
									onclick={() => setActiveTab(tab.id)}
								>
									<svg class="h-5 w-5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={tab.icon} />
									</svg>
									<span class="font-medium">{tab.label}</span>
								</button>
							</li>
						{/each}
					</ul>
				</div>
			</nav>

			<!-- Content Area -->
			<div class="flex-1 min-w-0">
				{#if activeTab === 'general'}
					<!-- General Settings -->
					<div class="card p-6">
						<h2 class="text-xl font-semibold text-secondary-900">General Settings</h2>
						<p class="mt-1 text-sm text-secondary-500">Manage your account and preferences.</p>

						<div class="mt-6 space-y-6">
							<!-- User Profile Section -->
							<div class="border-b border-secondary-200 pb-6">
								<h3 class="text-lg font-medium text-secondary-900">Profile</h3>
								<div class="mt-4 space-y-4">
									{#if authStore.user}
										<div class="grid gap-4 sm:grid-cols-2">
											<div>
												<span class="block text-sm font-medium text-secondary-700">Username</span>
												<p class="mt-1 text-secondary-900">{authStore.user.username}</p>
											</div>
											<div>
												<span class="block text-sm font-medium text-secondary-700">Email</span>
												<p class="mt-1 text-secondary-900">{authStore.user.email}</p>
											</div>
											<div>
												<span class="block text-sm font-medium text-secondary-700">Role</span>
												<p class="mt-1 text-secondary-900 capitalize">{authStore.user.role}</p>
											</div>
											{#if authStore.user.displayName}
												<div>
													<span class="block text-sm font-medium text-secondary-700">Display Name</span>
													<p class="mt-1 text-secondary-900">{authStore.user.displayName}</p>
												</div>
											{/if}
										</div>
									{/if}
								</div>
							</div>

							<!-- Server Information -->
							<div class="border-b border-secondary-200 pb-6">
								<h3 class="text-lg font-medium text-secondary-900">Server Information</h3>
								<div class="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
									<div class="rounded-lg bg-secondary-50 p-4">
										<div class="text-sm font-medium text-secondary-500">SMTP Server</div>
										<div class="mt-1 font-mono text-secondary-900">localhost:1025</div>
									</div>
									<div class="rounded-lg bg-secondary-50 p-4">
										<div class="text-sm font-medium text-secondary-500">IMAP Server</div>
										<div class="mt-1 font-mono text-secondary-900">localhost:1143</div>
									</div>
									<div class="rounded-lg bg-secondary-50 p-4">
										<div class="text-sm font-medium text-secondary-500">Web UI</div>
										<div class="mt-1 font-mono text-secondary-900">localhost:8025</div>
									</div>
								</div>
							</div>

							<!-- Danger Zone -->
							<div>
								<h3 class="text-lg font-medium text-red-600">Danger Zone</h3>
								<p class="mt-1 text-sm text-secondary-500">Irreversible actions that require extra caution.</p>
								<div class="mt-4 space-y-3">
									<button class="btn-danger">
										Delete All Messages
									</button>
								</div>
							</div>
						</div>
					</div>
				{:else if activeTab === 'notifications'}
					<!-- Notifications Tab -->
					<div class="card p-6">
						<h2 class="text-xl font-semibold text-secondary-900">Notification Settings</h2>
						<p class="mt-1 text-sm text-secondary-500">Configure how you receive notifications for new messages.</p>

						<div class="mt-6 space-y-6">
							<!-- Master Toggle -->
							<div class="border-b border-secondary-200 pb-6">
								<div class="flex items-center justify-between">
									<div>
										<h3 class="text-lg font-medium text-secondary-900">Enable Notifications</h3>
										<p class="mt-1 text-sm text-secondary-500">Receive notifications when new messages arrive.</p>
									</div>
									<button
										type="button"
										role="switch"
										aria-checked={notificationsStore.preferences.enabled}
										aria-label="Toggle notifications"
										onclick={handleToggleNotifications}
										class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 {notificationsStore.preferences.enabled ? 'bg-primary-600' : 'bg-secondary-200'}"
									>
										<span
											aria-hidden="true"
											class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {notificationsStore.preferences.enabled ? 'translate-x-5' : 'translate-x-0'}"
										></span>
									</button>
								</div>
							</div>

							<!-- Notification Types -->
							<div class="border-b border-secondary-200 pb-6">
								<h3 class="text-lg font-medium text-secondary-900">Notification Types</h3>
								<p class="mt-1 text-sm text-secondary-500">Choose how you want to be notified.</p>
								<div class="mt-4 space-y-4">
									<!-- Toast Notifications -->
									<div class="flex items-center justify-between">
										<div class="flex items-center gap-3">
											<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100">
												<svg class="h-5 w-5 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
												</svg>
											</div>
											<div>
												<p class="font-medium text-secondary-900">In-app Toast</p>
												<p class="text-sm text-secondary-500">Show pop-up notifications in the app</p>
											</div>
										</div>
										<button
											type="button"
											role="switch"
											aria-checked={notificationsStore.preferences.showToast}
											aria-label="Toggle in-app toast notifications"
											onclick={handleToggleToast}
											disabled={!notificationsStore.preferences.enabled}
											class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 {notificationsStore.preferences.showToast && notificationsStore.preferences.enabled ? 'bg-primary-600' : 'bg-secondary-200'}"
										>
											<span
												aria-hidden="true"
												class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {notificationsStore.preferences.showToast ? 'translate-x-5' : 'translate-x-0'}"
											></span>
										</button>
									</div>

									<!-- Desktop Notifications -->
									<div class="flex items-center justify-between">
										<div class="flex items-center gap-3">
											<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-purple-100">
												<svg class="h-5 w-5 text-purple-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
												</svg>
											</div>
											<div>
												<p class="font-medium text-secondary-900">Desktop Notifications</p>
												<p class="text-sm text-secondary-500">Show system notifications (requires permission)</p>
											</div>
										</div>
										<button
											type="button"
											role="switch"
											aria-checked={notificationsStore.preferences.showDesktop}
											aria-label="Toggle desktop notifications"
											onclick={handleToggleDesktop}
											disabled={!notificationsStore.preferences.enabled}
											class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 {notificationsStore.preferences.showDesktop && notificationsStore.preferences.enabled ? 'bg-primary-600' : 'bg-secondary-200'}"
										>
											<span
												aria-hidden="true"
												class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {notificationsStore.preferences.showDesktop ? 'translate-x-5' : 'translate-x-0'}"
											></span>
										</button>
									</div>

									<!-- Sound -->
									<div class="flex items-center justify-between">
										<div class="flex items-center gap-3">
											<div class="flex h-10 w-10 items-center justify-center rounded-lg bg-green-100">
												<svg class="h-5 w-5 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.536 8.464a5 5 0 010 7.072m2.828-9.9a9 9 0 010 12.728M5.586 15H4a1 1 0 01-1-1v-4a1 1 0 011-1h1.586l4.707-4.707C10.923 3.663 12 4.109 12 5v14c0 .891-1.077 1.337-1.707.707L5.586 15z" />
												</svg>
											</div>
											<div>
												<p class="font-medium text-secondary-900">Sound</p>
												<p class="text-sm text-secondary-500">Play a sound when new messages arrive</p>
											</div>
										</div>
										<button
											type="button"
											role="switch"
											aria-checked={notificationsStore.preferences.playSound}
											aria-label="Toggle notification sound"
											onclick={handleToggleSound}
											disabled={!notificationsStore.preferences.enabled}
											class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 {notificationsStore.preferences.playSound && notificationsStore.preferences.enabled ? 'bg-primary-600' : 'bg-secondary-200'}"
										>
											<span
												aria-hidden="true"
												class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {notificationsStore.preferences.playSound ? 'translate-x-5' : 'translate-x-0'}"
											></span>
										</button>
									</div>
								</div>
							</div>

							<!-- Polling Settings -->
							<div class="border-b border-secondary-200 pb-6">
								<h3 class="text-lg font-medium text-secondary-900">Update Frequency</h3>
								<p class="mt-1 text-sm text-secondary-500">How often to check for new messages.</p>
								<div class="mt-4">
									<label for="polling-interval" class="sr-only">Polling interval</label>
									<select
										id="polling-interval"
										class="block w-full max-w-xs rounded-md border-secondary-300 shadow-sm focus:border-primary-500 focus:ring-primary-500 sm:text-sm"
										value={pollingService.config.interval}
										onchange={handlePollingIntervalChange}
									>
										{#each pollingIntervals as option (option.value)}
											<option value={option.value}>{option.label}</option>
										{/each}
									</select>
								</div>
								<div class="mt-3">
									<p class="text-sm text-secondary-500">
										Status:
										{#if pollingService.isActive}
											<span class="font-medium text-green-600">Active</span>
											{#if pollingService.isPaused}
												<span class="text-secondary-400">(paused - tab inactive)</span>
											{/if}
										{:else}
											<span class="font-medium text-secondary-400">Inactive</span>
										{/if}
									</p>
								</div>
							</div>

							<!-- Test Notification -->
							<div>
								<h3 class="text-lg font-medium text-secondary-900">Test Notifications</h3>
								<p class="mt-1 text-sm text-secondary-500">Send a test notification to verify your settings.</p>
								<div class="mt-4">
									<button
										type="button"
										class="btn-secondary"
										onclick={handleTestNotification}
										disabled={!notificationsStore.preferences.enabled}
									>
										<svg class="mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
										</svg>
										Send Test Notification
									</button>
								</div>
							</div>
						</div>
					</div>
				{:else if activeTab === 'webhooks'}
					<!-- Webhooks Tab -->
					<div class="space-y-6">
						{#if showWebhookForm}
							<!-- Webhook Form -->
							<div class="card p-6">
								<h2 class="mb-6 text-xl font-semibold text-secondary-900">
									{editingWebhook ? 'Edit Webhook' : 'Create Webhook'}
								</h2>
								<WebhookForm
									webhook={editingWebhook}
									onSuccess={handleWebhookSuccess}
									onCancel={closeWebhookForm}
								/>
							</div>
						{:else}
							<!-- Webhook List Header -->
							<div class="card p-6">
								<div class="flex items-center justify-between mb-6">
									<div>
										<h2 class="text-xl font-semibold text-secondary-900">Webhooks</h2>
										<p class="mt-1 text-sm text-secondary-500">
											Configure webhooks to receive notifications when events occur.
										</p>
									</div>
									<button
										class="btn-primary"
										onclick={() => openWebhookForm()}
									>
										<svg class="mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
										</svg>
										New Webhook
									</button>
								</div>
								<WebhookList onEdit={(webhook) => openWebhookForm(webhook)} />
							</div>
						{/if}
					</div>
				{:else if activeTab === 'mailboxes'}
					<!-- Mailboxes Tab -->
					<div class="card p-6">
						<div class="mb-6">
							<h2 class="text-xl font-semibold text-secondary-900">Mailboxes</h2>
							<p class="mt-1 text-sm text-secondary-500">
								Manage your mailboxes for receiving and organizing emails.
							</p>
						</div>
						<MailboxSettings />
					</div>
				{/if}
			</div>
		</div>
	</div>
</main>
