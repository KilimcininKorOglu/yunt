<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { StatCard, MessageList, StorageUsage, ActivityChart } from '$components';
	import { getSystemApi, getMessagesApi } from '$lib/api';
	import type { SystemStats, MessageSummary } from '$lib/api';
	import { authStore } from '$stores/auth';

	// API instances
	const systemApi = getSystemApi();
	const messagesApi = getMessagesApi();

	// State
	let loading = $state(true);
	let stats = $state<SystemStats | null>(null);
	let recentMessages = $state<MessageSummary[]>([]);
	let activityData = $state<{ label: string; received: number; sent?: number }[]>([]);
	let refreshInterval: ReturnType<typeof setInterval> | null = null;
	let lastRefresh = $state<Date | null>(null);

	// Auto-refresh interval in milliseconds (30 seconds)
	const autoRefreshMs = 30000;

	// Computed stats
	const totalMessages = $derived(stats?.messages?.total ?? 0);
	const unreadMessages = $derived(stats?.messages?.unread ?? 0);
	const totalSize = $derived(stats?.messages?.totalSize ?? 0);
	const totalMailboxes = $derived(stats?.mailboxes?.total ?? 0);
	const mailboxSize = $derived(stats?.mailboxes?.totalSize ?? 0);

	const todayMessages = $derived(stats?.messages?.todayCount ?? 0);
	const weekMessages = $derived(stats?.messages?.weekCount ?? 0);

	/**
	 * Fetch all dashboard data
	 */
	async function fetchDashboardData(): Promise<void> {
		try {
			loading = true;

			// Fetch stats and messages in parallel
			const [statsData, messagesData] = await Promise.all([
				systemApi.getStats(),
				messagesApi.list({ pageSize: 5, sort: 'receivedAt', order: 'desc' })
			]);

			stats = statsData;

			// Convert Message to MessageSummary format
			recentMessages = messagesData.items.map((msg) => ({
				id: msg.id,
				mailboxId: msg.mailboxId,
				from: msg.from,
				subject: msg.subject,
				preview: msg.textBody?.slice(0, 100) ?? '',
				status: msg.status,
				isStarred: msg.isStarred,
				hasAttachments: msg.attachmentCount > 0,
				receivedAt: msg.receivedAt
			}));

			activityData = mapActivityData();

			lastRefresh = new Date();
		} catch (error) {
			console.error('Failed to fetch dashboard data:', error);
		} finally {
			loading = false;
		}
	}

	/**
	 * Map daily counts from API to activity chart data
	 */
	function mapActivityData(): { label: string; received: number; sent?: number }[] {
		const dailyCounts = stats?.messages?.dailyCounts;
		if (!dailyCounts || dailyCounts.length === 0) return [];

		const days = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
		return dailyCounts.map((dc) => ({
			label: days[new Date(dc.date + 'T00:00:00').getDay()],
			received: dc.count,
			sent: undefined
		}));
	}

	/**
	 * Handle message click - navigate to message detail
	 */
	function handleMessageClick(message: MessageSummary): void {
		goto(`/messages/${message.id}`);
	}

	/**
	 * Manual refresh handler
	 */
	async function handleRefresh(): Promise<void> {
		await fetchDashboardData();
	}

	/**
	 * Format last refresh time
	 */
	function formatLastRefresh(date: Date | null): string {
		if (!date) return 'Never';
		return date.toLocaleTimeString(undefined, {
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit'
		});
	}

	// Lifecycle
	onMount(() => {
		fetchDashboardData();

		// Setup auto-refresh
		refreshInterval = setInterval(() => {
			fetchDashboardData();
		}, autoRefreshMs);
	});

	onDestroy(() => {
		if (refreshInterval) {
			clearInterval(refreshInterval);
		}
	});
</script>

<svelte:head>
	<title>Dashboard - Yunt Mail Server</title>
</svelte:head>

<div class="min-h-screen">
	<!-- Header -->
	<header class="border-b border-secondary-200 bg-white">
		<div class="mx-auto max-w-7xl px-4 py-4 sm:px-6 lg:px-8">
			<div class="flex items-center justify-between">
				<div class="flex items-center gap-4">
					<h1 class="text-2xl font-bold text-primary-600">Yunt</h1>
					<span class="hidden text-sm text-secondary-400 sm:inline"
						>Development Mail Server</span
					>
				</div>
				<div class="flex items-center gap-4">
					<!-- Refresh Button -->
					<button
						onclick={handleRefresh}
						disabled={loading}
						class="flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-secondary-600 transition-colors hover:bg-secondary-100 disabled:opacity-50"
					>
						<svg
							class={`h-4 w-4 ${loading ? 'animate-spin' : ''}`}
							viewBox="0 0 24 24"
							fill="none"
							stroke="currentColor"
							stroke-width="2"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
							/>
						</svg>
						<span class="hidden sm:inline">Refresh</span>
					</button>
					<!-- User Menu -->
					{#if authStore.user}
						<div class="flex items-center gap-2">
							<div
								class="flex h-8 w-8 items-center justify-center rounded-full bg-primary-100 text-primary-700"
							>
								<span class="text-sm font-medium">
									{authStore.user.username.charAt(0).toUpperCase()}
								</span>
							</div>
							<span class="hidden text-sm font-medium text-secondary-700 sm:inline">
								{authStore.user.username}
							</span>
						</div>
					{/if}
				</div>
			</div>
		</div>
	</header>

	<!-- Main Content -->
	<main class="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
		<!-- Page Title -->
		<div class="mb-6 flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
			<div>
				<h2 class="text-xl font-semibold text-secondary-900">Dashboard</h2>
				<p class="text-sm text-secondary-500">Overview of your mail server activity</p>
			</div>
			{#if lastRefresh}
				<p class="text-xs text-secondary-400">
					Last updated: {formatLastRefresh(lastRefresh)}
				</p>
			{/if}
		</div>

		<!-- Stats Grid -->
		<div class="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<StatCard
				title="Total Messages"
				value={totalMessages}
				icon="mail"
				iconBgClass="bg-primary-100"
				iconColorClass="text-primary-600"
				{loading}
			/>
			<StatCard
				title="Unread Messages"
				value={unreadMessages}
				icon="clock"
				iconBgClass="bg-amber-100"
				iconColorClass="text-amber-600"
				{loading}
			/>
			<StatCard
				title="Today"
				value={todayMessages}
				subtitle="Messages received"
				icon="calendar"
				iconBgClass="bg-green-100"
				iconColorClass="text-green-600"
				{loading}
			/>
			<StatCard
				title="This Week"
				value={weekMessages}
				subtitle="Messages received"
				icon="week"
				iconBgClass="bg-purple-100"
				iconColorClass="text-purple-600"
				{loading}
			/>
		</div>

		<!-- Two Column Layout -->
		<div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
			<!-- Left Column - Charts and Activity -->
			<div class="space-y-6 lg:col-span-2">
				<!-- Activity Chart -->
				<ActivityChart
					data={activityData}
					{loading}
					title="Message Activity"
					periodLabel="Last 7 days"
				/>

				<!-- Server Info Card -->
				<div class="card p-5">
					<h3 class="text-base font-semibold text-secondary-900">Server Information</h3>
					<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-3">
						<div class="rounded-lg bg-secondary-50 p-4 text-center">
							<p class="text-xs font-medium text-secondary-500">SMTP Server</p>
							<p class="mt-1 font-mono text-sm text-secondary-700">localhost:1025</p>
						</div>
						<div class="rounded-lg bg-secondary-50 p-4 text-center">
							<p class="text-xs font-medium text-secondary-500">IMAP Server</p>
							<p class="mt-1 font-mono text-sm text-secondary-700">localhost:1143</p>
						</div>
						<div class="rounded-lg bg-secondary-50 p-4 text-center">
							<p class="text-xs font-medium text-secondary-500">Web UI</p>
							<p class="mt-1 font-mono text-sm text-secondary-700">localhost:8025</p>
						</div>
					</div>
					<div
						class="mt-4 grid grid-cols-2 gap-4 border-t border-secondary-100 pt-4 sm:grid-cols-4"
					>
						<div>
							<p class="text-xs text-secondary-400">Total Mailboxes</p>
							{#if loading}
								<div
									class="mt-1 h-5 w-12 animate-pulse rounded bg-secondary-200"
								></div>
							{:else}
								<p class="mt-1 text-lg font-semibold text-secondary-900">
									{totalMailboxes}
								</p>
							{/if}
						</div>
						<div>
							<p class="text-xs text-secondary-400">Active Users</p>
							{#if loading}
								<div
									class="mt-1 h-5 w-12 animate-pulse rounded bg-secondary-200"
								></div>
							{:else}
								<p class="mt-1 text-lg font-semibold text-secondary-900">
									{stats?.users?.active ?? 0}
								</p>
							{/if}
						</div>
						<div>
							<p class="text-xs text-secondary-400">Uptime</p>
							{#if loading}
								<div
									class="mt-1 h-5 w-16 animate-pulse rounded bg-secondary-200"
								></div>
							{:else}
								<p class="mt-1 text-lg font-semibold text-secondary-900">
									{stats?.uptime ? Math.floor(stats.uptime / 3600) + 'h' : '0h'}
								</p>
							{/if}
						</div>
						<div>
							<p class="text-xs text-secondary-400">Messages/Hour</p>
							{#if loading}
								<div
									class="mt-1 h-5 w-12 animate-pulse rounded bg-secondary-200"
								></div>
							{:else}
								<p class="mt-1 text-lg font-semibold text-secondary-900">
									{stats?.uptime && stats.uptime > 0
										? Math.round(totalMessages / (stats.uptime / 3600))
										: 0}
								</p>
							{/if}
						</div>
					</div>
				</div>
			</div>

			<!-- Right Column - Messages and Storage -->
			<div class="space-y-6">
				<!-- Storage Usage -->
				<StorageUsage usedBytes={totalSize || mailboxSize} totalBytes={0} {loading} />

				<!-- Recent Messages -->
				<MessageList
					messages={recentMessages}
					{loading}
					maxItems={5}
					onMessageClick={handleMessageClick}
				/>
			</div>
		</div>
	</main>
</div>
