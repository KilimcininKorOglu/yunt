<script lang="ts">
	/**
	 * SearchBar Component
	 * Provides search functionality for filtering messages.
	 */

	interface Props {
		value?: string;
		placeholder?: string;
		onSearch: (query: string) => void;
		onClear?: () => void;
		disabled?: boolean;
	}

	const {
		value = '',
		placeholder = 'Search messages...',
		onSearch,
		onClear,
		disabled = false
	}: Props = $props();

	let internalValue = $state('');
	let debounceTimer: ReturnType<typeof setTimeout> | null = null;
	let lastExternalValue = '';

	// Derived current value - prefer internal state unless external changed
	const inputValue = $derived.by(() => {
		if (value !== lastExternalValue) {
			lastExternalValue = value;
			internalValue = value;
		}
		return internalValue;
	});

	function handleInput(event: Event): void {
		const target = event.target as HTMLInputElement;
		internalValue = target.value;

		// Debounce search
		if (debounceTimer) {
			clearTimeout(debounceTimer);
		}

		debounceTimer = setTimeout(() => {
			onSearch(internalValue);
		}, 300);
	}

	function handleKeydown(event: KeyboardEvent): void {
		if (event.key === 'Enter') {
			if (debounceTimer) {
				clearTimeout(debounceTimer);
			}
			onSearch(internalValue);
		} else if (event.key === 'Escape') {
			handleClear();
		}
	}

	function handleClear(): void {
		internalValue = '';
		if (debounceTimer) {
			clearTimeout(debounceTimer);
		}
		onSearch('');
		onClear?.();
	}
</script>

<div class="relative flex-1">
	<div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
		<svg
			class="h-5 w-5 text-secondary-400"
			fill="none"
			viewBox="0 0 24 24"
			stroke="currentColor"
			aria-hidden="true"
		>
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				stroke-width="2"
				d="M21 21l-5.197-5.197m0 0A7.5 7.5 0 105.196 5.196a7.5 7.5 0 0010.607 10.607z"
			/>
		</svg>
	</div>
	<input
		type="text"
		{disabled}
		value={inputValue}
		oninput={handleInput}
		onkeydown={handleKeydown}
		{placeholder}
		class="input pl-10 pr-10"
		aria-label="Search messages"
	/>
	{#if inputValue}
		<button
			type="button"
			onclick={handleClear}
			class="absolute inset-y-0 right-0 flex items-center pr-3 text-secondary-400 hover:text-secondary-600"
			aria-label="Clear search"
		>
			<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="2"
					d="M6 18L18 6M6 6l12 12"
				/>
			</svg>
		</button>
	{/if}
</div>
