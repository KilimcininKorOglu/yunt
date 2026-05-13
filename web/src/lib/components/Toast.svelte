<script lang="ts">
	import { notificationsStore, type NotificationType } from '$stores/notifications.svelte';

	function getToastClass(type: NotificationType): string {
		switch (type) {
			case 'success': return 'toast-success';
			case 'error': return 'toast-error';
			case 'warning': return 'toast-warning';
			default: return 'toast-info';
		}
	}

	function handleDismiss(id: string): void {
		notificationsStore.dismissToast(id);
	}
</script>

<div class="toast-container" role="region" aria-label="Notifications">
	{#each notificationsStore.toasts as toast (toast.id)}
		<div class="toast {getToastClass(toast.type)}" role="alert">
			<div>
				<b>{toast.title}</b>
				{#if toast.message}
					<span> — {toast.message}</span>
				{/if}
			</div>
			{#if toast.dismissible}
				<button type="button" class="toast-close" onclick={() => handleDismiss(toast.id)}>X</button>
			{/if}
		</div>
	{/each}
</div>
