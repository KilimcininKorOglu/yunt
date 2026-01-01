<script lang="ts">
	/**
	 * StatCard Component
	 * Displays a statistics card with an icon, title, value, and optional trend indicator.
	 */

	/** Supported icon types */
	export type IconType = 'mail' | 'clock' | 'calendar' | 'week' | 'none';

	interface Props {
		/** Card title */
		title: string;
		/** Primary value to display */
		value: number | string;
		/** Optional subtitle or unit */
		subtitle?: string;
		/** Icon type to display */
		icon?: IconType;
		/** Icon background color class */
		iconBgClass?: string;
		/** Icon color class */
		iconColorClass?: string;
		/** Whether data is loading */
		loading?: boolean;
		/** Optional trend percentage (positive = up, negative = down) */
		trend?: number;
		/** Trend comparison label (e.g., "vs last week") */
		trendLabel?: string;
	}

	const {
		title,
		value,
		subtitle = '',
		icon = 'none',
		iconBgClass = 'bg-primary-100',
		iconColorClass = 'text-primary-600',
		loading = false,
		trend,
		trendLabel = ''
	}: Props = $props();

	const formattedValue = $derived(typeof value === 'number' ? value.toLocaleString() : value);
	const trendIsPositive = $derived(trend !== undefined && trend >= 0);
	const trendFormatted = $derived(
		trend !== undefined ? `${trendIsPositive ? '+' : ''}${trend.toFixed(1)}%` : ''
	);
</script>

<div class="card p-5 transition-shadow hover:shadow-md">
	<div class="flex items-start justify-between">
		<div class="flex-1">
			<p class="text-sm font-medium text-secondary-500">{title}</p>
			{#if loading}
				<div class="mt-2 h-8 w-20 animate-pulse rounded bg-secondary-200"></div>
			{:else}
				<p class="mt-1 text-2xl font-semibold text-secondary-900">{formattedValue}</p>
			{/if}
			{#if subtitle}
				<p class="mt-1 text-xs text-secondary-400">{subtitle}</p>
			{/if}
		</div>
		{#if icon !== 'none'}
			<div class={`flex h-10 w-10 items-center justify-center rounded-lg ${iconBgClass}`}>
				<svg
					class={`h-5 w-5 ${iconColorClass}`}
					viewBox="0 0 24 24"
					fill="none"
					stroke="currentColor"
					stroke-width="2"
					stroke-linecap="round"
					stroke-linejoin="round"
				>
					{#if icon === 'mail'}
						<path d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
					{:else if icon === 'clock'}
						<circle cx="12" cy="12" r="10" />
						<path d="M12 6v6l4 2" />
					{:else if icon === 'calendar'}
						<rect x="3" y="4" width="18" height="18" rx="2" ry="2" />
						<line x1="16" y1="2" x2="16" y2="6" />
						<line x1="8" y1="2" x2="8" y2="6" />
						<line x1="3" y1="10" x2="21" y2="10" />
					{:else if icon === 'week'}
						<path d="M8 2v4m8-4v4M3 10h18M5 4h14a2 2 0 012 2v14a2 2 0 01-2 2H5a2 2 0 01-2-2V6a2 2 0 012-2z" />
						<path d="M8 14h.01M12 14h.01M16 14h.01M8 18h.01M12 18h.01" />
					{/if}
				</svg>
			</div>
		{/if}
	</div>
	{#if trend !== undefined && !loading}
		<div class="mt-3 flex items-center gap-1.5">
			<span
				class={`flex items-center text-sm font-medium ${trendIsPositive ? 'text-green-600' : 'text-red-600'}`}
			>
				{#if trendIsPositive}
					<svg class="mr-0.5 h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
						<path
							fill-rule="evenodd"
							d="M5.293 9.707a1 1 0 010-1.414l4-4a1 1 0 011.414 0l4 4a1 1 0 01-1.414 1.414L11 7.414V15a1 1 0 11-2 0V7.414L6.707 9.707a1 1 0 01-1.414 0z"
							clip-rule="evenodd"
						/>
					</svg>
				{:else}
					<svg class="mr-0.5 h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
						<path
							fill-rule="evenodd"
							d="M14.707 10.293a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 111.414-1.414L9 12.586V5a1 1 0 012 0v7.586l2.293-2.293a1 1 0 011.414 0z"
							clip-rule="evenodd"
						/>
					</svg>
				{/if}
				{trendFormatted}
			</span>
			{#if trendLabel}
				<span class="text-sm text-secondary-400">{trendLabel}</span>
			{/if}
		</div>
	{/if}
</div>
