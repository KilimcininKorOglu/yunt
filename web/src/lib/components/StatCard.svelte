<script lang="ts">
	export type IconType = 'mail' | 'clock' | 'calendar' | 'week' | 'none';

	interface Props {
		title: string;
		value: number | string;
		subtitle?: string;
		icon?: IconType;
		loading?: boolean;
	}

	const {
		title,
		value,
		subtitle = '',
		icon = 'none',
		loading = false
	}: Props = $props();

	const formattedValue = $derived(typeof value === 'number' ? value.toLocaleString() : value);

	function getEmoji(i: IconType): string {
		switch (i) {
			case 'mail': return '📧';
			case 'clock': return '📬';
			case 'calendar': return '📅';
			case 'week': return '📊';
			default: return '';
		}
	}
</script>

<div class="stat-card">
	{#if icon !== 'none'}
		<span class="stat-icon">{getEmoji(icon)}</span>
	{/if}
	<div class="stat-info">
		<div class="stat-title">{title}</div>
		{#if loading}
			<div class="loading-skeleton" style="width:50px;height:16px;"></div>
		{:else}
			<div class="stat-value">{formattedValue}</div>
		{/if}
		{#if subtitle}
			<div class="stat-subtitle">{subtitle}</div>
		{/if}
	</div>
</div>
