<script lang="ts">
	import type { MessageStatus } from '$lib/api/types';

	interface Filters {
		status?: MessageStatus;
		isStarred?: boolean;
	}

	interface Props {
		filters: Filters;
		onFilterChange: (filters: Partial<Filters>) => void;
		onClearFilters: () => void;
		disabled?: boolean;
	}

	const { filters, onFilterChange, onClearFilters, disabled = false }: Props = $props();

	const activeFilterCount = $derived(
		(filters.status ? 1 : 0) + (filters.isStarred !== undefined ? 1 : 0)
	);

	function toggleUnread(): void {
		if (filters.status === 'unread') {
			onFilterChange({ status: undefined });
		} else {
			onFilterChange({ status: 'unread' });
		}
	}

	function toggleStarred(): void {
		if (filters.isStarred === true) {
			onFilterChange({ isStarred: undefined });
		} else {
			onFilterChange({ isStarred: true });
		}
	}
</script>

<div class="filter-controls">
	<span class="filter-label">Show:</span>
	<button
		type="button"
		{disabled}
		onclick={toggleUnread}
		class="hotmail-btn"
		class:active={filters.status === 'unread'}
	>
		Unread
	</button>
	<button
		type="button"
		{disabled}
		onclick={toggleStarred}
		class="hotmail-btn"
		class:active={filters.isStarred === true}
	>
		Starred
	</button>
	{#if activeFilterCount > 0}
		<button type="button" {disabled} onclick={onClearFilters} class="hotmail-btn">
			Clear ({activeFilterCount})
		</button>
	{/if}
</div>
