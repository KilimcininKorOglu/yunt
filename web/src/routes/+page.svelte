<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { StatCard, MessageList, StorageUsage, ActivityChart } from '$components';
	import { getSystemApi, getMessagesApi } from '$lib/api';
	import type { SystemStats, MessageSummary } from '$lib/api';
	import { authStore } from '$stores/auth';

	const systemApi = getSystemApi();
	const messagesApi = getMessagesApi();

	let loading = $state(true);
	let stats = $state<SystemStats | null>(null);
	let recentMessages = $state<MessageSummary[]>([]);
	let activityData = $state<{ label: string; received: number; sent?: number }[]>([]);
	let refreshInterval: ReturnType<typeof setInterval> | null = null;
	let lastRefresh = $state<Date | null>(null);

	const autoRefreshMs = 30000;

	const totalMessages = $derived(stats?.messages?.total ?? 0);
	const unreadMessages = $derived(stats?.messages?.unread ?? 0);
	const totalSize = $derived(stats?.messages?.totalSize ?? 0);
	const totalMailboxes = $derived(stats?.mailboxes?.total ?? 0);
	const mailboxSize = $derived(stats?.mailboxes?.totalSize ?? 0);
	const todayMessages = $derived(stats?.messages?.todayCount ?? 0);
	const weekMessages = $derived(stats?.messages?.weekCount ?? 0);

	async function fetchDashboardData(): Promise<void> {
		try {
			loading = true;

			const [statsData, messagesData] = await Promise.all([
				systemApi.getStats(),
				messagesApi.list({ pageSize: 5, sort: 'receivedAt', order: 'desc' })
			]);

			stats = statsData;

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

	function mapActivityData(): { label: string; received: number; sent?: number }[] {
		const hourlyCounts = stats?.messages?.hourlyCounts;
		if (!hourlyCounts || hourlyCounts.length === 0) return [];

		return hourlyCounts.map((hc) => {
			const hour = hc.hour.split(' ')[1] ?? hc.hour;
			return { label: hour, received: hc.count, sent: undefined };
		});
	}

	function handleMessageClick(message: MessageSummary): void {
		goto(`/message/${message.id}`);
	}

	async function handleRefresh(): Promise<void> {
		await fetchDashboardData();
	}

	function formatLastRefresh(date: Date | null): string {
		if (!date) return 'Never';
		return date.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', second: '2-digit' });
	}

	onMount(() => {
		fetchDashboardData();
		refreshInterval = setInterval(() => { fetchDashboardData(); }, autoRefreshMs);
	});

	onDestroy(() => {
		if (refreshInterval) clearInterval(refreshInterval);
	});
</script>

<svelte:head>
	<title>Today - Yunt Mail</title>
</svelte:head>

<div class="today-page">
	<div class="today-header">
		<h2>Today on Yunt Mail</h2>
		<div class="today-actions">
			{#if lastRefresh}
				<span style="font-size:10px;color:var(--text-muted);">Last updated: {formatLastRefresh(lastRefresh)}</span>
			{/if}
			<button type="button" class="hotmail-btn" onclick={handleRefresh} disabled={loading}>
				{loading ? 'Loading...' : 'Refresh'}
			</button>
		</div>
	</div>

	<div class="today-stats">
		<StatCard title="Total Messages" value={totalMessages} icon="mail" {loading} />
		<StatCard title="Unread" value={unreadMessages} icon="clock" {loading} />
		<StatCard title="Today" value={todayMessages} subtitle="received" icon="calendar" {loading} />
		<StatCard title="This Week" value={weekMessages} subtitle="received" icon="week" {loading} />
	</div>

	<div class="today-grid">
		<div class="today-left">
			<ActivityChart data={activityData} {loading} title="Message Activity" periodLabel="Last 24 hours" />

			<div class="info-box">
				<div class="info-box-header">Server Information</div>
				<div class="info-box-body">
					<table class="server-info-table"><tbody>
						<tr>
							<td class="lbl">SMTP Server</td>
							<td><code>localhost:1025</code></td>
						</tr>
						<tr>
							<td class="lbl">IMAP Server</td>
							<td><code>localhost:1143</code></td>
						</tr>
						<tr>
							<td class="lbl">Web UI</td>
							<td><code>localhost:8025</code></td>
						</tr>
						<tr>
							<td class="lbl">Mailboxes</td>
							<td>{loading ? '...' : totalMailboxes}</td>
						</tr>
						<tr>
							<td class="lbl">Active Users</td>
							<td>{loading ? '...' : (stats?.users?.active ?? 0)}</td>
						</tr>
						<tr>
							<td class="lbl">Uptime</td>
							<td>{loading ? '...' : (stats?.uptime ? Math.floor(stats.uptime / 3600) + 'h' : '0h')}</td>
						</tr>
					</tbody></table>
				</div>
			</div>
		</div>

		<div class="today-right">
			<StorageUsage usedBytes={totalSize || mailboxSize} totalBytes={0} {loading} />
			<MessageList messages={recentMessages} {loading} maxItems={5} onMessageClick={handleMessageClick} />
		</div>
	</div>
</div>
