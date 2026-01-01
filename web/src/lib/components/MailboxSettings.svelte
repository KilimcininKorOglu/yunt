<script lang="ts">
	import { getMailboxesApi } from '$lib/api';
	import type { Mailbox, MailboxCreateInput, MailboxUpdateInput } from '$lib/api';

	interface Props {
		/** Callback when mailbox list changes */
		onRefresh?: () => void;
	}

	const { onRefresh }: Props = $props();

	// API instance
	const mailboxesApi = getMailboxesApi();

	// State
	let mailboxes = $state<Mailbox[]>([]);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let deletingId = $state<string | null>(null);
	let settingDefaultId = $state<string | null>(null);

	// Form state
	let showCreateForm = $state(false);
	let editingMailbox = $state<Mailbox | null>(null);
	let formData = $state({
		name: '',
		address: '',
		description: '',
		retentionDays: 30,
		isCatchAll: false
	});
	let isSubmitting = $state(false);
	let formError = $state<string | null>(null);

	// Validation state
	let nameError = $state<string | null>(null);
	let addressError = $state<string | null>(null);

	// Form validation
	const isFormValid = $derived(
		formData.name.trim().length > 0 &&
		formData.address.trim().length > 0 &&
		!nameError &&
		!addressError
	);

	// Load mailboxes on mount
	$effect(() => {
		loadMailboxes();
	});

	async function loadMailboxes(): Promise<void> {
		isLoading = true;
		error = null;

		try {
			const response = await mailboxesApi.list();
			mailboxes = response.items;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load mailboxes';
		} finally {
			isLoading = false;
		}
	}

	function validateName(): void {
		if (!formData.name.trim()) {
			nameError = 'Name is required';
		} else if (formData.name.trim().length < 2) {
			nameError = 'Name must be at least 2 characters';
		} else if (formData.name.trim().length > 100) {
			nameError = 'Name must be less than 100 characters';
		} else {
			nameError = null;
		}
	}

	function validateAddress(): void {
		if (!formData.address.trim()) {
			addressError = 'Address is required';
		} else if (!/^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+$/.test(formData.address.trim())) {
			addressError = 'Please enter a valid email address format';
		} else {
			addressError = null;
		}
	}

	function openCreateForm(): void {
		editingMailbox = null;
		formData = {
			name: '',
			address: '',
			description: '',
			retentionDays: 30,
			isCatchAll: false
		};
		nameError = null;
		addressError = null;
		formError = null;
		showCreateForm = true;
	}

	function openEditForm(mailbox: Mailbox): void {
		editingMailbox = mailbox;
		formData = {
			name: mailbox.name,
			address: mailbox.address,
			description: mailbox.description || '',
			retentionDays: mailbox.retentionDays,
			isCatchAll: mailbox.isCatchAll
		};
		nameError = null;
		addressError = null;
		formError = null;
		showCreateForm = true;
	}

	function closeForm(): void {
		showCreateForm = false;
		editingMailbox = null;
		formData = {
			name: '',
			address: '',
			description: '',
			retentionDays: 30,
			isCatchAll: false
		};
		nameError = null;
		addressError = null;
		formError = null;
	}

	async function handleSubmit(event: Event): Promise<void> {
		event.preventDefault();

		validateName();
		if (!editingMailbox) {
			validateAddress();
		}

		if (!isFormValid) {
			return;
		}

		isSubmitting = true;
		formError = null;

		try {
			if (editingMailbox) {
				const input: MailboxUpdateInput = {
					name: formData.name.trim(),
					description: formData.description.trim() || undefined,
					retentionDays: formData.retentionDays
				};
				await mailboxesApi.update(editingMailbox.id, input);
			} else {
				const input: MailboxCreateInput = {
					name: formData.name.trim(),
					address: formData.address.trim(),
					description: formData.description.trim() || undefined,
					retentionDays: formData.retentionDays,
					isCatchAll: formData.isCatchAll
				};
				await mailboxesApi.create(input);
			}

			closeForm();
			await loadMailboxes();
			onRefresh?.();
		} catch (err) {
			formError = err instanceof Error ? err.message : 'Failed to save mailbox';
		} finally {
			isSubmitting = false;
		}
	}

	async function handleDelete(mailbox: Mailbox): Promise<void> {
		if (!confirm(`Are you sure you want to delete the mailbox "${mailbox.name}"? All messages will be permanently deleted.`)) {
			return;
		}

		deletingId = mailbox.id;

		try {
			await mailboxesApi.delete(mailbox.id);
			await loadMailboxes();
			onRefresh?.();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete mailbox';
		} finally {
			deletingId = null;
		}
	}

	async function handleSetDefault(mailbox: Mailbox): Promise<void> {
		if (mailbox.isDefault) return;

		settingDefaultId = mailbox.id;

		try {
			await mailboxesApi.setDefault(mailbox.id);
			await loadMailboxes();
			onRefresh?.();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to set default mailbox';
		} finally {
			settingDefaultId = null;
		}
	}

	function formatSize(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
	}
</script>

<div class="space-y-4">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<h3 class="text-lg font-semibold text-secondary-900">Mailboxes</h3>
		<div class="flex items-center gap-2">
			<button
				class="btn-secondary text-sm"
				onclick={loadMailboxes}
				disabled={isLoading}
			>
				{#if isLoading}
					<svg class="mr-2 h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
					</svg>
				{/if}
				Refresh
			</button>
			<button
				class="btn-primary text-sm"
				onclick={openCreateForm}
				disabled={showCreateForm}
			>
				<svg class="mr-2 h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
				</svg>
				New Mailbox
			</button>
		</div>
	</div>

	<!-- Error Alert -->
	{#if error}
		<div class="rounded-lg border border-red-200 bg-red-50 p-4" role="alert">
			<div class="flex items-start gap-3">
				<svg class="mt-0.5 h-5 w-5 flex-shrink-0 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
				</svg>
				<p class="text-sm text-red-700">{error}</p>
				<button class="ml-auto text-red-500 hover:text-red-700" onclick={() => (error = null)} aria-label="Dismiss error">
					<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>
		</div>
	{/if}

	<!-- Create/Edit Form -->
	{#if showCreateForm}
		<div class="card p-4">
			<h4 class="mb-4 font-medium text-secondary-900">
				{editingMailbox ? 'Edit Mailbox' : 'Create New Mailbox'}
			</h4>

			{#if formError}
				<div class="mb-4 rounded-lg border border-red-200 bg-red-50 p-3" role="alert">
					<p class="text-sm text-red-700">{formError}</p>
				</div>
			{/if}

			<form onsubmit={handleSubmit} class="space-y-4">
				<div class="grid gap-4 sm:grid-cols-2">
					<!-- Name Field -->
					<div>
						<label for="mailbox-name" class="mb-1 block text-sm font-medium text-secondary-700">
							Name <span class="text-red-500">*</span>
						</label>
						<input
							type="text"
							id="mailbox-name"
							bind:value={formData.name}
							onblur={validateName}
							class="input"
							class:border-red-500={nameError}
							placeholder="My Mailbox"
							disabled={isSubmitting}
						/>
						{#if nameError}
							<p class="mt-1 text-sm text-red-500">{nameError}</p>
						{/if}
					</div>

					<!-- Address Field -->
					<div>
						<label for="mailbox-address" class="mb-1 block text-sm font-medium text-secondary-700">
							Address <span class="text-red-500">*</span>
						</label>
						<input
							type="email"
							id="mailbox-address"
							bind:value={formData.address}
							onblur={validateAddress}
							class="input font-mono text-sm"
							class:border-red-500={addressError}
							placeholder="mailbox@example.com"
							disabled={isSubmitting || !!editingMailbox}
						/>
						{#if addressError}
							<p class="mt-1 text-sm text-red-500">{addressError}</p>
						{:else if editingMailbox}
							<p class="mt-1 text-xs text-secondary-400">Address cannot be changed after creation</p>
						{/if}
					</div>
				</div>

				<!-- Description Field -->
				<div>
					<label for="mailbox-description" class="mb-1 block text-sm font-medium text-secondary-700">
						Description
					</label>
					<textarea
						id="mailbox-description"
						bind:value={formData.description}
						class="input min-h-[80px] resize-y"
						placeholder="Optional description for this mailbox"
						disabled={isSubmitting}
					></textarea>
				</div>

				<div class="grid gap-4 sm:grid-cols-2">
					<!-- Retention Days -->
					<div>
						<label for="retention-days" class="mb-1 block text-sm font-medium text-secondary-700">
							Retention (days)
						</label>
						<input
							type="number"
							id="retention-days"
							bind:value={formData.retentionDays}
							min="1"
							max="365"
							class="input w-32"
							disabled={isSubmitting}
						/>
						<p class="mt-1 text-xs text-secondary-500">
							Messages older than this will be auto-deleted
						</p>
					</div>

					<!-- Catch-All -->
					{#if !editingMailbox}
						<div class="flex items-start pt-6">
							<label class="flex items-center gap-2 cursor-pointer">
								<input
									type="checkbox"
									bind:checked={formData.isCatchAll}
									class="h-4 w-4 rounded border-secondary-300 text-primary-600 focus:ring-primary-500"
									disabled={isSubmitting}
								/>
								<span class="text-sm text-secondary-700">Catch-all mailbox</span>
							</label>
						</div>
					{/if}
				</div>

				<!-- Form Actions -->
				<div class="flex items-center justify-end gap-3 pt-2">
					<button
						type="button"
						class="btn-secondary"
						onclick={closeForm}
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
							{editingMailbox ? 'Update' : 'Create'} Mailbox
						{/if}
					</button>
				</div>
			</form>
		</div>
	{/if}

	<!-- Loading State -->
	{#if isLoading && mailboxes.length === 0}
		<div class="flex items-center justify-center py-12">
			<div class="h-8 w-8 animate-spin rounded-full border-4 border-primary-200 border-t-primary-600"></div>
		</div>
	{:else if mailboxes.length === 0}
		<!-- Empty State -->
		<div class="rounded-lg border border-dashed border-secondary-300 p-8 text-center">
			<svg class="mx-auto h-12 w-12 text-secondary-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
			</svg>
			<h4 class="mt-4 text-lg font-medium text-secondary-900">No mailboxes</h4>
			<p class="mt-2 text-sm text-secondary-500">
				Create a mailbox to start receiving emails.
			</p>
			<button
				class="btn-primary mt-4"
				onclick={openCreateForm}
			>
				Create Mailbox
			</button>
		</div>
	{:else}
		<!-- Mailbox List -->
		<div class="space-y-3">
			{#each mailboxes as mailbox (mailbox.id)}
				<div class="card p-4">
					<div class="flex items-start justify-between gap-4">
						<div class="min-w-0 flex-1">
							<div class="flex items-center gap-2">
								<h4 class="font-medium text-secondary-900 truncate">{mailbox.name}</h4>
								{#if mailbox.isDefault}
									<span class="inline-flex items-center rounded-full bg-primary-100 px-2 py-0.5 text-xs font-medium text-primary-700">
										Default
									</span>
								{/if}
								{#if mailbox.isCatchAll}
									<span class="inline-flex items-center rounded-full bg-yellow-100 px-2 py-0.5 text-xs font-medium text-yellow-700">
										Catch-all
									</span>
								{/if}
							</div>
							<p class="mt-1 text-sm text-secondary-500 truncate font-mono">{mailbox.address}</p>
							{#if mailbox.description}
								<p class="mt-1 text-sm text-secondary-400 truncate">{mailbox.description}</p>
							{/if}
							<div class="mt-2 flex flex-wrap items-center gap-3 text-xs text-secondary-400">
								<span class="flex items-center gap-1">
									<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
									</svg>
									{mailbox.messageCount} messages
								</span>
								{#if mailbox.unreadCount > 0}
									<span class="flex items-center gap-1 text-primary-600">
										<svg class="h-4 w-4" fill="currentColor" viewBox="0 0 24 24">
											<circle cx="12" cy="12" r="4" />
										</svg>
										{mailbox.unreadCount} unread
									</span>
								{/if}
								<span class="flex items-center gap-1">
									<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
									</svg>
									{formatSize(mailbox.totalSize)}
								</span>
								<span class="flex items-center gap-1">
									<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
									</svg>
									{mailbox.retentionDays} days retention
								</span>
							</div>
						</div>
						<div class="flex items-center gap-2 flex-shrink-0">
							<!-- Set Default -->
							{#if !mailbox.isDefault}
								<button
									class="rounded-lg p-2 text-secondary-500 hover:bg-secondary-100 hover:text-secondary-700 disabled:opacity-50"
									onclick={() => handleSetDefault(mailbox)}
									disabled={settingDefaultId === mailbox.id}
									title="Set as default"
								>
									{#if settingDefaultId === mailbox.id}
										<svg class="h-5 w-5 animate-spin" fill="none" viewBox="0 0 24 24">
											<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
											<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
										</svg>
									{:else}
										<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
										</svg>
									{/if}
								</button>
							{/if}

							<!-- Edit -->
							<button
								class="rounded-lg p-2 text-secondary-500 hover:bg-secondary-100 hover:text-secondary-700"
								onclick={() => openEditForm(mailbox)}
								title="Edit mailbox"
							>
								<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
								</svg>
							</button>

							<!-- Delete -->
							<button
								class="rounded-lg p-2 text-red-500 hover:bg-red-50 hover:text-red-700 disabled:opacity-50"
								onclick={() => handleDelete(mailbox)}
								disabled={deletingId === mailbox.id || mailbox.isDefault}
								title={mailbox.isDefault ? 'Cannot delete default mailbox' : 'Delete mailbox'}
							>
								{#if deletingId === mailbox.id}
									<svg class="h-5 w-5 animate-spin" fill="none" viewBox="0 0 24 24">
										<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
										<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
									</svg>
								{:else}
									<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
									</svg>
								{/if}
							</button>
						</div>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
