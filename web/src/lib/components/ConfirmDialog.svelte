<script lang="ts">
	/**
	 * ConfirmDialog Component
	 * A reusable confirmation dialog for destructive actions.
	 * Uses a modal overlay with accessible keyboard handling.
	 */

	interface Props {
		/** Whether the dialog is visible */
		open: boolean;
		/** Dialog title */
		title?: string;
		/** Dialog message/description */
		message: string;
		/** Text for confirm button */
		confirmText?: string;
		/** Text for cancel button */
		cancelText?: string;
		/** Visual variant for the confirm button */
		variant?: 'danger' | 'primary' | 'secondary';
		/** Whether the confirm action is in progress */
		isLoading?: boolean;
		/** Callback when confirm is clicked */
		onConfirm: () => void | Promise<void>;
		/** Callback when cancel is clicked or dialog is dismissed */
		onCancel: () => void;
	}

	const {
		open,
		title = 'Confirm Action',
		message,
		confirmText = 'Confirm',
		cancelText = 'Cancel',
		variant = 'danger',
		isLoading = false,
		onConfirm,
		onCancel
	}: Props = $props();

	// Handle keyboard events
	function handleKeydown(event: KeyboardEvent): void {
		if (event.key === 'Escape' && !isLoading) {
			onCancel();
		}
	}

	// Handle backdrop click
	function handleBackdropClick(event: MouseEvent): void {
		if (event.target === event.currentTarget && !isLoading) {
			onCancel();
		}
	}

	// Get button class based on variant
	function getConfirmButtonClass(): string {
		switch (variant) {
			case 'danger':
				return 'btn-danger';
			case 'primary':
				return 'btn-primary';
			case 'secondary':
				return 'btn-secondary';
			default:
				return 'btn-danger';
		}
	}
</script>

<svelte:window on:keydown={handleKeydown} />

{#if open}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- Backdrop - keyboard handling is done via svelte:window -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
		role="dialog"
		aria-modal="true"
		aria-labelledby="dialog-title"
		aria-describedby="dialog-description"
		tabindex="-1"
		onclick={handleBackdropClick}
	>
		<!-- Dialog box -->
		<div class="card w-full max-w-md p-6 shadow-xl">
			<!-- Header -->
			<div class="mb-4 flex items-start gap-4">
				<!-- Icon -->
				{#if variant === 'danger'}
					<div
						class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full bg-red-100"
					>
						<svg
							class="h-6 w-6 text-red-600"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
							/>
						</svg>
					</div>
				{:else}
					<div
						class="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full bg-primary-100"
					>
						<svg
							class="h-6 w-6 text-primary-600"
							fill="none"
							viewBox="0 0 24 24"
							stroke="currentColor"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								stroke-width="2"
								d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
					</div>
				{/if}

				<!-- Title and message -->
				<div class="flex-1">
					<h3 id="dialog-title" class="text-lg font-semibold text-secondary-900">
						{title}
					</h3>
					<p id="dialog-description" class="mt-1 text-sm text-secondary-600">
						{message}
					</p>
				</div>
			</div>

			<!-- Actions -->
			<div class="flex justify-end gap-3">
				<button type="button" class="btn-secondary" disabled={isLoading} onclick={onCancel}>
					{cancelText}
				</button>
				<button
					type="button"
					class={getConfirmButtonClass()}
					disabled={isLoading}
					onclick={onConfirm}
				>
					{#if isLoading}
						<span class="flex items-center gap-2">
							<svg class="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
								<circle
									class="opacity-25"
									cx="12"
									cy="12"
									r="10"
									stroke="currentColor"
									stroke-width="4"
								></circle>
								<path
									class="opacity-75"
									fill="currentColor"
									d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
								></path>
							</svg>
							Processing...
						</span>
					{:else}
						{confirmText}
					{/if}
				</button>
			</div>
		</div>
	</div>
{/if}
