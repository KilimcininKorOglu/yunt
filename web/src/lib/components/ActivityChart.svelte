<script lang="ts">
	/**
	 * ActivityChart Component
	 * Displays a message activity timeline chart using pure CSS/SVG (no external dependencies).
	 */

	interface Props {
		/** Activity data points */
		data: ActivityDataPoint[];
		/** Whether data is loading */
		loading?: boolean;
		/** Chart height in pixels */
		height?: number;
		/** Show x-axis labels */
		showLabels?: boolean;
		/** Chart title */
		title?: string;
		/** Time period label */
		periodLabel?: string;
	}

	interface ActivityDataPoint {
		/** Label for the data point (e.g., day name, date) */
		label: string;
		/** Number of messages received */
		received: number;
		/** Number of messages sent (optional) */
		sent?: number;
	}

	const {
		data = [],
		loading = false,
		height = 200,
		showLabels = true,
		title = 'Message Activity',
		periodLabel = 'Last 7 days'
	}: Props = $props();

	// Calculate chart dimensions
	const chartPadding = 40;
	const barWidth = 24;
	const barGap = 8;

	// Calculate max value for scaling
	const maxValue = $derived(
		Math.max(
			1,
			...data.map((d) => Math.max(d.received, d.sent ?? 0))
		)
	);

	// Calculate total messages
	const totalReceived = $derived(data.reduce((sum, d) => sum + d.received, 0));
	const totalSent = $derived(data.reduce((sum, d) => sum + (d.sent ?? 0), 0));

	// Scale a value to chart height
	function scaleY(value: number): number {
		const availableHeight = height - chartPadding;
		return availableHeight - (value / maxValue) * availableHeight;
	}

	// Generate y-axis labels
	const yAxisLabels = $derived(() => {
		const labels: number[] = [];
		const step = Math.ceil(maxValue / 4);
		for (let i = 0; i <= maxValue; i += step) {
			labels.push(i);
		}
		if (labels[labels.length - 1] < maxValue) {
			labels.push(Math.ceil(maxValue));
		}
		return labels;
	});
</script>

<div class="card p-5">
	<div class="flex items-center justify-between">
		<div>
			<h3 class="text-base font-semibold text-secondary-900">{title}</h3>
			<p class="mt-0.5 text-xs text-secondary-400">{periodLabel}</p>
		</div>
		{#if !loading}
			<div class="flex items-center gap-4">
				<div class="flex items-center gap-1.5">
					<span class="h-2.5 w-2.5 rounded-full bg-primary-500"></span>
					<span class="text-xs text-secondary-500">Received ({totalReceived})</span>
				</div>
				{#if totalSent > 0}
					<div class="flex items-center gap-1.5">
						<span class="h-2.5 w-2.5 rounded-full bg-green-500"></span>
						<span class="text-xs text-secondary-500">Sent ({totalSent})</span>
					</div>
				{/if}
			</div>
		{/if}
	</div>

	{#if loading}
		<div class="mt-4 flex items-end justify-between gap-2" style="height: {height}px;">
			{#each Array(7) as _, i (i)}
				<div class="flex-1">
					<div
						class="animate-pulse rounded-t bg-secondary-200"
						style="height: {Math.random() * 60 + 40}%;"
					></div>
				</div>
			{/each}
		</div>
		<div class="mt-3 flex justify-between">
			{#each Array(7) as _, i (i)}
				<div class="h-3 w-8 animate-pulse rounded bg-secondary-200"></div>
			{/each}
		</div>
	{:else if data.length === 0}
		<div class="flex items-center justify-center" style="height: {height}px;">
			<div class="text-center">
				<svg
					class="mx-auto h-12 w-12 text-secondary-300"
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="1.5"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z"
					/>
				</svg>
				<p class="mt-2 text-sm text-secondary-500">No activity data available</p>
			</div>
		</div>
	{:else}
		<!-- Chart Container -->
		<div class="relative mt-4" style="height: {height}px;">
			<!-- Y-axis grid lines -->
			<div class="absolute inset-0">
				{#each yAxisLabels() as label (label)}
					<div
						class="absolute left-0 right-0 border-t border-secondary-100"
						style="top: {scaleY(label)}px;"
					>
						<span class="absolute -left-1 -translate-x-full text-xs text-secondary-400">
							{label}
						</span>
					</div>
				{/each}
			</div>

			<!-- Bars -->
			<div
				class="absolute bottom-0 left-8 right-0 flex items-end justify-around"
				style="height: {height - chartPadding}px;"
			>
				{#each data as point, i (i)}
					<div class="group relative flex flex-col items-center" style="width: {barWidth * 2 + barGap}px;">
						<div class="flex items-end gap-0.5" style="height: 100%;">
							<!-- Received bar -->
							<div
								class="relative w-6 rounded-t bg-primary-500 transition-all duration-300 hover:bg-primary-600"
								style="height: {(point.received / maxValue) * 100}%;"
							>
								<!-- Tooltip -->
								<div
									class="absolute -top-8 left-1/2 hidden -translate-x-1/2 whitespace-nowrap rounded bg-secondary-800 px-2 py-1 text-xs text-white group-hover:block"
								>
									{point.received} received
								</div>
							</div>
							<!-- Sent bar (if exists) -->
							{#if point.sent !== undefined && point.sent > 0}
								<div
									class="w-6 rounded-t bg-green-500 transition-all duration-300 hover:bg-green-600"
									style="height: {(point.sent / maxValue) * 100}%;"
								></div>
							{/if}
						</div>
						{#if showLabels}
							<span class="mt-2 text-xs text-secondary-500">{point.label}</span>
						{/if}
					</div>
				{/each}
			</div>
		</div>
	{/if}
</div>
