<script lang="ts">
	interface Props {
		open: boolean;
		title?: string;
		message: string;
		confirmText?: string;
		cancelText?: string;
		variant?: 'danger' | 'primary' | 'secondary';
		isLoading?: boolean;
		onConfirm: () => void | Promise<void>;
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

	function handleKeydown(event: KeyboardEvent): void {
		if (event.key === 'Escape' && !isLoading) {
			onCancel();
		}
	}

	function handleBackdropClick(event: MouseEvent): void {
		if (event.target === event.currentTarget && !isLoading) {
			onCancel();
		}
	}
</script>

<svelte:window on:keydown={handleKeydown} />

{#if open}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<div class="dialog-overlay" role="dialog" aria-modal="true" tabindex="-1" onclick={handleBackdropClick}>
		<div class="dialog-box">
			<div class="dialog-titlebar">
				<span>{title}</span>
				<button type="button" class="close-btn" onclick={onCancel} disabled={isLoading}>X</button>
			</div>
			<div class="dialog-body">
				<p>{message}</p>
			</div>
			<div class="dialog-footer">
				<button type="button" class="hotmail-btn" onclick={onCancel} disabled={isLoading}>
					{cancelText}
				</button>
				{#if variant === 'danger'}
					<button type="button" class="hotmail-btn" style="background:#cc3333;color:#fff;border-color:#cc3333;" onclick={onConfirm} disabled={isLoading}>
						{isLoading ? 'Processing...' : confirmText}
					</button>
				{:else}
					<button type="button" class="hotmail-btn toolbar-btn-primary" onclick={onConfirm} disabled={isLoading}>
						{isLoading ? 'Processing...' : confirmText}
					</button>
				{/if}
			</div>
		</div>
	</div>
{/if}
