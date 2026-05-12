<script lang="ts">
	interface Props {
		usedBytes: number;
		totalBytes?: number;
		loading?: boolean;
	}

	const { usedBytes = 0, totalBytes = 0, loading = false }: Props = $props();

	const isUnlimited = $derived(totalBytes === 0);
	const usagePercent = $derived(isUnlimited ? 0 : Math.min((usedBytes / totalBytes) * 100, 100));

	function formatBytes(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
	}

	const usedFormatted = $derived(formatBytes(usedBytes));
</script>

<div class="info-box">
	<div class="info-box-header">Storage Usage</div>
	<div class="info-box-body">
		{#if loading}
			<div class="loading-skeleton" style="width:100%;height:8px;margin-bottom:6px;"></div>
		{:else}
			<div class="storage-track">
				<div class="storage-fill" style="width:{isUnlimited ? 0 : usagePercent}%"></div>
			</div>
			<div class="storage-label">
				<span>{usedFormatted} used</span>
				<span>{isUnlimited ? 'Unlimited' : usagePercent.toFixed(1) + '%'}</span>
			</div>
		{/if}
	</div>
</div>
