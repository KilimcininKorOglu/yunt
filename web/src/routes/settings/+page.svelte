<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { authStore } from '$stores/auth';
	import WebhookList from '$components/WebhookList.svelte';
	import WebhookForm from '$components/WebhookForm.svelte';
	import MailboxSettings from '$components/MailboxSettings.svelte';
	import type { Webhook } from '$lib/api';

	// Tab definitions
	type Tab = 'general' | 'webhooks' | 'mailboxes';

	const tabs: { id: Tab; label: string; icon: string }[] = [
		{ id: 'general', label: 'General', icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z' },
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
