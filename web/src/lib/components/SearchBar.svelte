<script lang="ts">
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

<div class="search-bar">
	<input
		type="text"
		{disabled}
		value={inputValue}
		oninput={handleInput}
		onkeydown={handleKeydown}
		{placeholder}
		class="hotmail-input"
		style="flex:1;margin:0;"
	/>
	{#if inputValue}
		<button type="button" class="hotmail-btn" onclick={handleClear}>Clear</button>
	{/if}
</div>
