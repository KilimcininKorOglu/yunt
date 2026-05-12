<script lang="ts">
	interface Props {
		data: { label: string; received: number; sent?: number }[];
		loading?: boolean;
		title?: string;
		periodLabel?: string;
	}

	const {
		data = [],
		loading = false,
		title = 'Message Activity',
		periodLabel = 'Last 7 days'
	}: Props = $props();

	const maxValue = $derived(Math.max(1, ...data.map((d) => d.received)));
	const totalReceived = $derived(data.reduce((sum, d) => sum + d.received, 0));
</script>

<div class="info-box">
	<div class="info-box-header">{title} <span style="font-weight:normal;color:var(--text-muted);">({periodLabel})</span></div>
	<div class="info-box-body">
		{#if loading}
			<div class="loading-skeleton" style="width:100%;height:80px;"></div>
		{:else if data.length === 0}
			<p style="color:var(--text-muted);">No activity data available</p>
		{:else}
			<div class="chart-bar-container">
				{#each data as point (point.label)}
					{@const pct = (point.received / maxValue) * 100}
					<div class="chart-bar-col">
						<div class="chart-bar" style="height:{Math.max(pct, 2)}%;" title="{point.label}: {point.received}"></div>
						<span class="chart-label">{point.label}</span>
					</div>
				{/each}
			</div>
			<div style="text-align:right;font-size:10px;color:var(--text-muted);margin-top:4px;">
				Total: {totalReceived} messages
			</div>
		{/if}
	</div>
</div>
