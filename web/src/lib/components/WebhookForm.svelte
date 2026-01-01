<script lang="ts">
	import { getWebhooksApi } from '$lib/api';
	import type { Webhook, WebhookCreateInput, WebhookUpdateInput, WebhookEvent } from '$lib/api';

	interface Props {
		/** Webhook to edit, or undefined for create mode */
		webhook?: Webhook;
		/** Callback when form is submitted successfully */
		onSuccess?: (webhook: Webhook) => void;
		/** Callback when form is cancelled */
		onCancel?: () => void;
	}

	const { webhook, onSuccess, onCancel }: Props = $props();

	// API instance
	const webhooksApi = getWebhooksApi();

	// Form state
	let name = $state('');
	let url = $state('');
	let secret = $state('');
	let events = $state<WebhookEvent[]>([]);
	let maxRetries = $state(3);
	let timeoutSeconds = $state(30);
	let headers = $state<{ key: string; value: string }[]>([]);

	// Initialize form from webhook prop
	$effect(() => {
		if (webhook) {
			name = webhook.name;
			url = webhook.url;
			events = [...webhook.events];
			maxRetries = webhook.maxRetries;
			timeoutSeconds = webhook.timeoutSeconds;
			headers = webhook.headers
				? Object.entries(webhook.headers).map(([key, value]) => ({ key, value }))
				: [];
		} else {
			name = '';
			url = '';
			secret = '';
			events = [];
			maxRetries = 3;
			timeoutSeconds = 30;
			headers = [];
		}
	});

	// UI state
	let isSubmitting = $state(false);
	let formError = $state<string | null>(null);

	// Validation state
	let nameError = $state<string | null>(null);
	let urlError = $state<string | null>(null);
	let eventsError = $state<string | null>(null);

	// Edit mode
	const isEditing = $derived(!!webhook);

	// All available events
	const availableEvents: { value: WebhookEvent; label: string; description: string }[] = [
		{ value: 'message.received', label: 'Message Received', description: 'Triggered when a new email is received' },
		{ value: 'message.deleted', label: 'Message Deleted', description: 'Triggered when an email is deleted' },
		{ value: 'mailbox.created', label: 'Mailbox Created', description: 'Triggered when a new mailbox is created' },
		{ value: 'mailbox.deleted', label: 'Mailbox Deleted', description: 'Triggered when a mailbox is deleted' }
	];

	// Form validation
	const isFormValid = $derived(
		name.trim().length > 0 &&
		url.trim().length > 0 &&
		events.length > 0 &&
		!nameError &&
		!urlError &&
		!eventsError
	);

	function validateName(): void {
		if (!name.trim()) {
			nameError = 'Name is required';
		} else if (name.trim().length < 2) {
			nameError = 'Name must be at least 2 characters';
		} else if (name.trim().length > 100) {
			nameError = 'Name must be less than 100 characters';
		} else {
			nameError = null;
		}
	}

	function validateUrl(): void {
		if (!url.trim()) {
			urlError = 'URL is required';
		} else {
			try {
				const parsedUrl = new URL(url.trim());
				if (!['http:', 'https:'].includes(parsedUrl.protocol)) {
					urlError = 'URL must use HTTP or HTTPS protocol';
				} else {
					urlError = null;
				}
			} catch {
				urlError = 'Please enter a valid URL';
			}
		}
	}

	function validateEvents(): void {
		if (events.length === 0) {
			eventsError = 'Select at least one event';
		} else {
			eventsError = null;
		}
	}

	function toggleEvent(event: WebhookEvent): void {
		if (events.includes(event)) {
			events = events.filter((e) => e !== event);
		} else {
			events = [...events, event];
		}
		validateEvents();
	}

	function addHeader(): void {
		headers = [...headers, { key: '', value: '' }];
	}

	function removeHeader(index: number): void {
		headers = headers.filter((_, i) => i !== index);
	}

	function updateHeader(index: number, field: 'key' | 'value', value: string): void {
		headers = headers.map((h, i) => (i === index ? { ...h, [field]: value } : h));
	}

	async function handleSubmit(event: Event): Promise<void> {
		event.preventDefault();

		// Validate all fields
		validateName();
		validateUrl();
		validateEvents();

		if (!isFormValid) {
			return;
		}

		isSubmitting = true;
		formError = null;

		try {
			// Build headers object from array
			const headersObj: Record<string, string> = {};
			for (const header of headers) {
				if (header.key.trim() && header.value.trim()) {
					headersObj[header.key.trim()] = header.value.trim();
				}
			}

			let result: Webhook;

			if (isEditing && webhook) {
				const input: WebhookUpdateInput = {
					name: name.trim(),
					url: url.trim(),
					events,
					maxRetries,
					timeoutSeconds,
					headers: Object.keys(headersObj).length > 0 ? headersObj : undefined
				};

				// Only include secret if it was changed
				if (secret.trim()) {
					input.secret = secret.trim();
				}

				result = await webhooksApi.update(webhook.id, input);
			} else {
				const input: WebhookCreateInput = {
					name: name.trim(),
					url: url.trim(),
					events,
					maxRetries,
					timeoutSeconds,
					headers: Object.keys(headersObj).length > 0 ? headersObj : undefined
				};

				if (secret.trim()) {
					input.secret = secret.trim();
				}

				result = await webhooksApi.create(input);
			}

			onSuccess?.(result);
		} catch (err) {
			formError = err instanceof Error ? err.message : 'Failed to save webhook';
		} finally {
			isSubmitting = false;
		}
	}
</script>

<form onsubmit={handleSubmit} class="space-y-6">
	<!-- Form Error -->
	{#if formError}
		<div class="rounded-lg border border-red-200 bg-red-50 p-4" role="alert">
			<div class="flex items-start gap-3">
				<svg class="mt-0.5 h-5 w-5 flex-shrink-0 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
				</svg>
				<p class="text-sm text-red-700">{formError}</p>
			</div>
		</div>
	{/if}

	<!-- Name Field -->
	<div>
		<label for="name" class="mb-2 block text-sm font-medium text-secondary-700">
			Name <span class="text-red-500">*</span>
		</label>
		<input
			type="text"
			id="name"
			bind:value={name}
			onblur={validateName}
			class="input"
			class:border-red-500={nameError}
			placeholder="My Webhook"
			disabled={isSubmitting}
		/>
		{#if nameError}
			<p class="mt-1 text-sm text-red-500">{nameError}</p>
		{/if}
	</div>

	<!-- URL Field -->
	<div>
		<label for="url" class="mb-2 block text-sm font-medium text-secondary-700">
			URL <span class="text-red-500">*</span>
		</label>
		<input
			type="url"
			id="url"
			bind:value={url}
			onblur={validateUrl}
			class="input font-mono text-sm"
			class:border-red-500={urlError}
			placeholder="https://example.com/webhook"
			disabled={isSubmitting}
		/>
		{#if urlError}
			<p class="mt-1 text-sm text-red-500">{urlError}</p>
		{/if}
	</div>

	<!-- Secret Field -->
	<div>
		<label for="secret" class="mb-2 block text-sm font-medium text-secondary-700">
			Secret {#if isEditing}<span class="text-secondary-400">(leave blank to keep current)</span>{/if}
		</label>
		<input
			type="password"
			id="secret"
			bind:value={secret}
			class="input font-mono text-sm"
			placeholder={isEditing ? '••••••••' : 'Optional signing secret'}
			disabled={isSubmitting}
		/>
		<p class="mt-1 text-xs text-secondary-500">
			Used to sign webhook payloads for verification
		</p>
	</div>

	<!-- Events Selection -->
	<fieldset>
		<legend class="mb-2 block text-sm font-medium text-secondary-700">
			Events <span class="text-red-500">*</span>
		</legend>
		<div class="space-y-2">
			{#each availableEvents as event (event.value)}
				<label class="flex items-start gap-3 rounded-lg border border-secondary-200 p-3 cursor-pointer hover:bg-secondary-50 transition-colors {events.includes(event.value) ? 'border-primary-500 bg-primary-50' : ''}">
					<input
						type="checkbox"
						checked={events.includes(event.value)}
						onchange={() => toggleEvent(event.value)}
						class="mt-0.5 h-4 w-4 rounded border-secondary-300 text-primary-600 focus:ring-primary-500"
						disabled={isSubmitting}
					/>
					<div>
						<div class="font-medium text-secondary-900">{event.label}</div>
						<div class="text-sm text-secondary-500">{event.description}</div>
					</div>
				</label>
			{/each}
		</div>
		{#if eventsError}
			<p class="mt-1 text-sm text-red-500">{eventsError}</p>
		{/if}
	</fieldset>

	<!-- Advanced Settings -->
	<details class="group">
		<summary class="cursor-pointer list-none">
			<div class="flex items-center gap-2 text-sm font-medium text-secondary-700">
				<svg class="h-4 w-4 transition-transform group-open:rotate-90" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				Advanced Settings
			</div>
		</summary>

		<div class="mt-4 space-y-4 pl-6">
			<!-- Max Retries -->
			<div>
				<label for="maxRetries" class="mb-2 block text-sm font-medium text-secondary-700">
					Max Retries
				</label>
				<input
					type="number"
					id="maxRetries"
					bind:value={maxRetries}
					min="0"
					max="10"
					class="input w-24"
					disabled={isSubmitting}
				/>
				<p class="mt-1 text-xs text-secondary-500">
					Number of retry attempts on failure (0-10)
				</p>
			</div>

			<!-- Timeout -->
			<div>
				<label for="timeoutSeconds" class="mb-2 block text-sm font-medium text-secondary-700">
					Timeout (seconds)
				</label>
				<input
					type="number"
					id="timeoutSeconds"
					bind:value={timeoutSeconds}
					min="5"
					max="60"
					class="input w-24"
					disabled={isSubmitting}
				/>
				<p class="mt-1 text-xs text-secondary-500">
					Request timeout in seconds (5-60)
				</p>
			</div>

			<!-- Custom Headers -->
			<div>
				<div class="mb-2 flex items-center justify-between">
					<span class="text-sm font-medium text-secondary-700">Custom Headers</span>
					<button
						type="button"
						class="text-sm text-primary-600 hover:text-primary-700"
						onclick={addHeader}
						disabled={isSubmitting}
					>
						+ Add Header
					</button>
				</div>
				<div class="space-y-2">
					{#each headers as header, index (index)}
						<div class="flex items-center gap-2">
							<input
								type="text"
								value={header.key}
								oninput={(e) => updateHeader(index, 'key', e.currentTarget.value)}
								class="input flex-1"
								placeholder="Header name"
								disabled={isSubmitting}
							/>
							<input
								type="text"
								value={header.value}
								oninput={(e) => updateHeader(index, 'value', e.currentTarget.value)}
								class="input flex-1"
								placeholder="Header value"
								disabled={isSubmitting}
							/>
							<button
								type="button"
								class="rounded-lg p-2 text-red-500 hover:bg-red-50 hover:text-red-700"
								onclick={() => removeHeader(index)}
								disabled={isSubmitting}
								aria-label="Remove header"
							>
								<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
								</svg>
							</button>
						</div>
					{/each}
				</div>
				{#if headers.length === 0}
					<p class="text-sm text-secondary-400">No custom headers configured</p>
				{/if}
			</div>
		</div>
	</details>

	<!-- Form Actions -->
	<div class="flex items-center justify-end gap-3 pt-4 border-t border-secondary-200">
		<button
			type="button"
			class="btn-secondary"
			onclick={onCancel}
			disabled={isSubmitting}
		>
			Cancel
		</button>
		<button
			type="submit"
			class="btn-primary"
			disabled={isSubmitting || !isFormValid}
		>
			{#if isSubmitting}
				<span class="flex items-center gap-2">
					<svg class="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
					</svg>
					Saving...
				</span>
			{:else}
				{isEditing ? 'Update Webhook' : 'Create Webhook'}
			{/if}
		</button>
	</div>
</form>
