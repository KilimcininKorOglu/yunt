<script lang="ts">
	/**
	 * Toast Component
	 * Displays toast notifications in a stacked format.
	 * Supports info, success, warning, and error types with animations.
	 */

	import { notificationsStore, type NotificationType } from '$stores/notifications';

	// Type configuration for styling
	const typeConfig: Record<NotificationType, { bgClass: string; iconPath: string; iconClass: string }> = {
		info: {
			bgClass: 'bg-blue-50 border-blue-200',
			iconClass: 'text-blue-500',
			iconPath: 'M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z'
		},
		success: {
			bgClass: 'bg-green-50 border-green-200',
			iconClass: 'text-green-500',
			iconPath: 'M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z'
		},
		warning: {
			bgClass: 'bg-yellow-50 border-yellow-200',
			iconClass: 'text-yellow-500',
			iconPath: 'M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z'
		},
		error: {
			bgClass: 'bg-red-50 border-red-200',
			iconClass: 'text-red-500',
			iconPath: 'M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z'
		}
	};

	function handleDismiss(id: string): void {
		notificationsStore.dismissToast(id);
	}
</script>

<!-- Toast Container -->
<div
	class="pointer-events-none fixed bottom-0 right-0 z-50 flex flex-col items-end gap-3 p-4"
	role="region"
	aria-label="Notifications"
>
	{#each notificationsStore.toasts as toast (toast.id)}
		{@const config = typeConfig[toast.type]}
		<div
			class="pointer-events-auto flex w-full max-w-sm animate-slide-in items-start gap-3 rounded-lg border p-4 shadow-lg {config.bgClass}"
			role="alert"
			aria-live="polite"
		>
			<!-- Icon -->
			<div class="flex-shrink-0">
				<svg class="h-5 w-5 {config.iconClass}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={config.iconPath} />
				</svg>
			</div>

			<!-- Content -->
			<div class="min-w-0 flex-1">
				<p class="text-sm font-medium text-secondary-900">{toast.title}</p>
				{#if toast.message}
					<p class="mt-1 text-sm text-secondary-600">{toast.message}</p>
				{/if}
			</div>

			<!-- Dismiss Button -->
			{#if toast.dismissible}
				<button
					type="button"
					class="flex-shrink-0 rounded-md p-1 text-secondary-400 hover:bg-secondary-200 hover:text-secondary-500 focus:outline-none focus:ring-2 focus:ring-secondary-400"
					onclick={() => handleDismiss(toast.id)}
					aria-label="Dismiss notification"
				>
					<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			{/if}
		</div>
	{/each}
</div>

<style>
	@keyframes slideIn {
		from {
			transform: translateX(100%);
			opacity: 0;
		}
		to {
			transform: translateX(0);
			opacity: 1;
		}
	}

	.animate-slide-in {
		animation: slideIn 0.3s ease-out;
	}
</style>
