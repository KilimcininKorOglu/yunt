<script lang="ts">
	/**
	 * FilterControls Component
	 * Provides filter options for messages (unread, starred).
	 */

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

	// Derived state for active filter count
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

<div class="flex items-center gap-2">
	<!-- Unread Filter -->
	<button
		type="button"
		{disabled}
		onclick={toggleUnread}
		class="inline-flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors {filters.status ===
		'unread'
			? 'border-primary-500 bg-primary-50 text-primary-700'
			: 'border-secondary-300 bg-white text-secondary-700 hover:bg-secondary-50'}"
		aria-pressed={filters.status === 'unread'}
	>
		<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				stroke-width="2"
				d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
			/>
		</svg>
		Unread
	</button>

	<!-- Starred Filter -->
	<button
		type="button"
		{disabled}
		onclick={toggleStarred}
		class="inline-flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors {filters.isStarred ===
		true
			? 'border-yellow-500 bg-yellow-50 text-yellow-700'
			: 'border-secondary-300 bg-white text-secondary-700 hover:bg-secondary-50'}"
		aria-pressed={filters.isStarred === true}
	>
		<svg
			class="h-4 w-4"
			fill={filters.isStarred ? 'currentColor' : 'none'}
			viewBox="0 0 24 24"
			stroke="currentColor"
		>
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				stroke-width="2"
				d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
			/>
		</svg>
		Starred
	</button>

	<!-- Clear Filters -->
	{#if activeFilterCount > 0}
		<button
			type="button"
			{disabled}
			onclick={onClearFilters}
			class="ml-1 inline-flex items-center gap-1 text-sm text-secondary-500 hover:text-secondary-700"
		>
			<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M6 18L18 6M6 6l12 12"
				/>
			</svg>
			Clear ({activeFilterCount})
		</button>
	{/if}
</div>
