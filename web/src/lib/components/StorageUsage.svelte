<script lang="ts">
	/**
	 * StorageUsage Component
	 * Displays a visual representation of storage usage with progress bar and details.
	 */

	interface Props {
		/** Current storage used in bytes */
		usedBytes: number;
		/** Total storage capacity in bytes (0 for unlimited) */
		totalBytes?: number;
		/** Whether data is loading */
		loading?: boolean;
		/** Show detailed breakdown */
		showBreakdown?: boolean;
		/** Optional breakdown of storage by category */
		breakdown?: StorageBreakdown[];
	}

	interface StorageBreakdown {
		label: string;
		bytes: number;
		color: string;
	}

	const {
		usedBytes = 0,
		totalBytes = 0,
		loading = false,
		showBreakdown = false,
		breakdown = []
	}: Props = $props();

	const isUnlimited = $derived(totalBytes === 0);
	const usagePercent = $derived(isUnlimited ? 0 : Math.min((usedBytes / totalBytes) * 100, 100));
	const usageLevel = $derived(
		usagePercent >= 90 ? 'critical' : usagePercent >= 70 ? 'warning' : 'normal'
	);

	function formatBytes(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
	}

	const usedFormatted = $derived(formatBytes(usedBytes));
	const totalFormatted = $derived(isUnlimited ? 'Unlimited' : formatBytes(totalBytes));
	const remainingBytes = $derived(isUnlimited ? 0 : Math.max(totalBytes - usedBytes, 0));
	const remainingFormatted = $derived(formatBytes(remainingBytes));

	const progressColor = $derived(
		usageLevel === 'critical'
			? 'bg-red-500'
			: usageLevel === 'warning'
				? 'bg-amber-500'
				: 'bg-primary-500'
	);

	const progressBgColor = $derived(
		usageLevel === 'critical'
			? 'bg-red-100'
			: usageLevel === 'warning'
				? 'bg-amber-100'
				: 'bg-primary-100'
	);
</script>

<div class="card p-5">
	<div class="flex items-center justify-between">
		<h3 class="text-base font-semibold text-secondary-900">Storage Usage</h3>
		{#if !isUnlimited && !loading}
			<span
				class={`text-sm font-medium ${
					usageLevel === 'critical'
						? 'text-red-600'
						: usageLevel === 'warning'
							? 'text-amber-600'
							: 'text-secondary-500'
				}`}
			>
				{usagePercent.toFixed(1)}%
			</span>
		{/if}
	</div>

	{#if loading}
		<div class="mt-4">
			<div class="h-3 w-full animate-pulse rounded-full bg-secondary-200"></div>
			<div class="mt-4 flex justify-between">
				<div class="h-4 w-20 animate-pulse rounded bg-secondary-200"></div>
				<div class="h-4 w-20 animate-pulse rounded bg-secondary-200"></div>
			</div>
		</div>
	{:else}
		<!-- Progress Bar -->
		<div class={`mt-4 h-3 w-full overflow-hidden rounded-full ${progressBgColor}`}>
			{#if !isUnlimited}
				<div
					class={`h-full rounded-full transition-all duration-500 ${progressColor}`}
					style="width: {usagePercent}%"
				></div>
			{:else}
				<div class="h-full w-full bg-secondary-200"></div>
			{/if}
		</div>

		<!-- Storage Details -->
		<div class="mt-4 flex items-center justify-between text-sm">
			<div>
				<span class="font-medium text-secondary-900">{usedFormatted}</span>
				<span class="text-secondary-400"> used</span>
			</div>
			{#if isUnlimited}
				<span class="text-secondary-400">Unlimited storage</span>
			{:else}
				<div class="text-right">
					<span class="font-medium text-secondary-900">{remainingFormatted}</span>
					<span class="text-secondary-400"> available</span>
				</div>
			{/if}
		</div>

		<!-- Capacity Info -->
		{#if !isUnlimited}
			<div class="mt-2 text-center text-xs text-secondary-400">
				Total capacity: {totalFormatted}
			</div>
		{/if}

		<!-- Warning Messages -->
		{#if usageLevel === 'critical' && !isUnlimited}
			<div class="mt-4 rounded-lg bg-red-50 p-3">
				<div class="flex items-start gap-2">
					<svg
						class="mt-0.5 h-4 w-4 flex-shrink-0 text-red-500"
						viewBox="0 0 24 24"
						fill="currentColor"
					>
						<path
							fill-rule="evenodd"
							d="M9.401 3.003c1.155-2 4.043-2 5.197 0l7.355 12.748c1.154 2-.29 4.5-2.599 4.5H4.645c-2.309 0-3.752-2.5-2.598-4.5L9.4 3.003zM12 8.25a.75.75 0 01.75.75v3.75a.75.75 0 01-1.5 0V9a.75.75 0 01.75-.75zm0 8.25a.75.75 0 100-1.5.75.75 0 000 1.5z"
							clip-rule="evenodd"
						/>
					</svg>
					<div>
						<p class="text-sm font-medium text-red-800">Storage almost full</p>
						<p class="mt-0.5 text-xs text-red-600">
							Consider deleting old messages or upgrading your plan.
						</p>
					</div>
				</div>
			</div>
		{:else if usageLevel === 'warning' && !isUnlimited}
			<div class="mt-4 rounded-lg bg-amber-50 p-3">
				<div class="flex items-start gap-2">
					<svg
						class="mt-0.5 h-4 w-4 flex-shrink-0 text-amber-500"
						viewBox="0 0 24 24"
						fill="currentColor"
					>
						<path
							fill-rule="evenodd"
							d="M9.401 3.003c1.155-2 4.043-2 5.197 0l7.355 12.748c1.154 2-.29 4.5-2.599 4.5H4.645c-2.309 0-3.752-2.5-2.598-4.5L9.4 3.003zM12 8.25a.75.75 0 01.75.75v3.75a.75.75 0 01-1.5 0V9a.75.75 0 01.75-.75zm0 8.25a.75.75 0 100-1.5.75.75 0 000 1.5z"
							clip-rule="evenodd"
						/>
					</svg>
					<div>
						<p class="text-sm font-medium text-amber-800">Storage filling up</p>
						<p class="mt-0.5 text-xs text-amber-600">
							You've used more than 70% of your storage.
						</p>
					</div>
				</div>
			</div>
		{/if}

		<!-- Storage Breakdown -->
		{#if showBreakdown && breakdown.length > 0}
			<div class="mt-4 border-t border-secondary-100 pt-4">
				<p class="mb-2 text-xs font-medium text-secondary-500">Storage Breakdown</p>
				<div class="space-y-2">
					{#each breakdown as item (item.label)}
						<div class="flex items-center justify-between">
							<div class="flex items-center gap-2">
								<span class={`h-2.5 w-2.5 rounded-full ${item.color}`}></span>
								<span class="text-sm text-secondary-600">{item.label}</span>
							</div>
							<span class="text-sm font-medium text-secondary-900">{formatBytes(item.bytes)}</span>
						</div>
					{/each}
				</div>
			</div>
		{/if}
	{/if}
</div>
