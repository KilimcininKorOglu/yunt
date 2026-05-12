<script lang="ts">
	import { getMailboxesApi } from '$lib/api';
	import type { Mailbox, MailboxCreateInput, MailboxUpdateInput } from '$lib/api';

	interface Props {
		onRefresh?: () => void;
	}

	const { onRefresh }: Props = $props();

	const mailboxesApi = getMailboxesApi();

	let mailboxes = $state<Mailbox[]>([]);
	let isLoading = $state(true);
	let error = $state<string | null>(null);
	let deletingId = $state<string | null>(null);
	let settingDefaultId = $state<string | null>(null);

	let showCreateForm = $state(false);
	let editingMailbox = $state<Mailbox | null>(null);
	let formData = $state({ name: '', address: '', description: '', retentionDays: 30, isCatchAll: false });
	let isSubmitting = $state(false);
	let formError = $state<string | null>(null);

	let nameError = $state<string | null>(null);
	let addressError = $state<string | null>(null);

	const isFormValid = $derived(formData.name.trim().length > 0 && formData.address.trim().length > 0 && !nameError && !addressError);

	$effect(() => { loadMailboxes(); });

	async function loadMailboxes(): Promise<void> {
		isLoading = true; error = null;
		try {
			const response = await mailboxesApi.list();
			mailboxes = response.items;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load mailboxes';
		} finally { isLoading = false; }
	}

	function validateName(): void {
		if (!formData.name.trim()) nameError = 'Name is required';
		else if (formData.name.trim().length < 2) nameError = 'Min 2 characters';
		else nameError = null;
	}

	function validateAddress(): void {
		if (!formData.address.trim()) addressError = 'Address is required';
		else if (!/^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+$/.test(formData.address.trim())) addressError = 'Invalid email format';
		else addressError = null;
	}

	function openCreateForm(): void {
		editingMailbox = null;
		formData = { name: '', address: '', description: '', retentionDays: 30, isCatchAll: false };
		nameError = null; addressError = null; formError = null;
		showCreateForm = true;
	}

	function openEditForm(mailbox: Mailbox): void {
		editingMailbox = mailbox;
		formData = { name: mailbox.name, address: mailbox.address, description: mailbox.description || '', retentionDays: mailbox.retentionDays, isCatchAll: mailbox.isCatchAll };
		nameError = null; addressError = null; formError = null;
		showCreateForm = true;
	}

	function closeForm(): void {
		showCreateForm = false; editingMailbox = null;
		formData = { name: '', address: '', description: '', retentionDays: 30, isCatchAll: false };
		nameError = null; addressError = null; formError = null;
	}

	async function handleSubmit(event: Event): Promise<void> {
		event.preventDefault();
		validateName();
		if (!editingMailbox) validateAddress();
		if (!isFormValid) return;

		isSubmitting = true; formError = null;
		try {
			if (editingMailbox) {
				await mailboxesApi.update(editingMailbox.id, {
					name: formData.name.trim(), description: formData.description.trim() || undefined, retentionDays: formData.retentionDays
				} as MailboxUpdateInput);
			} else {
				await mailboxesApi.create({
					name: formData.name.trim(), address: formData.address.trim(),
					description: formData.description.trim() || undefined, retentionDays: formData.retentionDays, isCatchAll: formData.isCatchAll
				} as MailboxCreateInput);
			}
			closeForm(); await loadMailboxes(); onRefresh?.();
		} catch (err) {
			formError = err instanceof Error ? err.message : 'Failed to save mailbox';
		} finally { isSubmitting = false; }
	}

	async function handleDelete(mailbox: Mailbox): Promise<void> {
		if (!confirm(`Delete mailbox "${mailbox.name}"? All messages will be permanently deleted.`)) return;
		deletingId = mailbox.id;
		try {
			await mailboxesApi.delete(mailbox.id);
			await loadMailboxes(); onRefresh?.();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete mailbox';
		} finally { deletingId = null; }
	}

	async function handleSetDefault(mailbox: Mailbox): Promise<void> {
		if (mailbox.isDefault) return;
		settingDefaultId = mailbox.id;
		try {
			await mailboxesApi.setDefault(mailbox.id);
			await loadMailboxes(); onRefresh?.();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to set default';
		} finally { settingDefaultId = null; }
	}

	function formatSize(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
	}
</script>

<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:10px;">
	<button type="button" class="hotmail-btn" onclick={loadMailboxes} disabled={isLoading}>
		{isLoading ? 'Loading...' : 'Refresh'}
	</button>
	<button type="button" class="hotmail-btn toolbar-btn-primary" onclick={openCreateForm} disabled={showCreateForm}>
		New Mailbox
	</button>
</div>

{#if error}
	<div class="alert alert-error">
		{error}
		<button type="button" class="alert-close" onclick={() => (error = null)}>X</button>
	</div>
{/if}

{#if showCreateForm}
	<div class="info-box" style="margin-bottom:12px;">
		<div class="info-box-header">{editingMailbox ? 'Edit Mailbox' : 'Create Mailbox'}</div>
		<div class="info-box-body">
			{#if formError}
				<div class="alert alert-error" style="margin-bottom:8px;">{formError}</div>
			{/if}
			<form onsubmit={handleSubmit}>
				<table class="server-info-table"><tbody>
					<tr>
						<td class="lbl">Name *</td>
						<td>
							<input type="text" class="hotmail-input" bind:value={formData.name} onblur={validateName} disabled={isSubmitting} style="width:200px;" />
							{#if nameError}<br><span style="color:var(--error-red);font-size:10px;">{nameError}</span>{/if}
						</td>
					</tr>
					<tr>
						<td class="lbl">Address *</td>
						<td>
							<input type="email" class="hotmail-input" bind:value={formData.address} onblur={validateAddress} disabled={isSubmitting || !!editingMailbox} style="width:250px;font-family:monospace;font-size:10px;" />
							{#if addressError}<br><span style="color:var(--error-red);font-size:10px;">{addressError}</span>{/if}
							{#if editingMailbox}<br><span style="font-size:10px;color:var(--text-muted);">Cannot be changed</span>{/if}
						</td>
					</tr>
					<tr>
						<td class="lbl">Description</td>
						<td><input type="text" class="hotmail-input" bind:value={formData.description} disabled={isSubmitting} style="width:300px;" /></td>
					</tr>
					<tr>
						<td class="lbl">Retention (days)</td>
						<td><input type="number" class="hotmail-input" bind:value={formData.retentionDays} min="1" max="365" disabled={isSubmitting} style="width:80px;" /></td>
					</tr>
					{#if !editingMailbox}
						<tr>
							<td class="lbl">Catch-All</td>
							<td><label><input type="checkbox" bind:checked={formData.isCatchAll} disabled={isSubmitting} /> Receive all unmatched emails</label></td>
						</tr>
					{/if}
				</tbody></table>
				<div style="margin-top:8px;display:flex;gap:6px;">
					<button type="submit" class="hotmail-btn toolbar-btn-primary" disabled={isSubmitting || !isFormValid}>
						{isSubmitting ? 'Saving...' : (editingMailbox ? 'Update' : 'Create')}
					</button>
					<button type="button" class="hotmail-btn" onclick={closeForm} disabled={isSubmitting}>Cancel</button>
				</div>
			</form>
		</div>
	</div>
{/if}

{#if isLoading && mailboxes.length === 0}
	<div style="text-align:center;padding:20px;">
		<div class="loading-spinner"></div>
	</div>
{:else if mailboxes.length === 0}
	<p style="text-align:center;padding:20px;color:var(--text-muted);">No mailboxes configured.</p>
{:else}
	<table class="msg-table">
		<thead>
			<tr>
				<th>Name</th>
				<th>Address</th>
				<th>Messages</th>
				<th>Size</th>
				<th>Retention</th>
				<th>Default</th>
				<th style="text-align:right;">Actions</th>
			</tr>
		</thead>
		<tbody>
			{#each mailboxes as mailbox (mailbox.id)}
				<tr>
					<td><b>{mailbox.name}</b>{#if mailbox.isCatchAll}<br><span style="font-size:9px;color:var(--text-muted);">Catch-All</span>{/if}</td>
					<td style="font-family:monospace;font-size:10px;">{mailbox.address}</td>
					<td>{mailbox.messageCount} ({mailbox.unreadCount} unread)</td>
					<td>{formatSize(mailbox.totalSize)}</td>
					<td>{mailbox.retentionDays}d</td>
					<td>
						{#if mailbox.isDefault}
							<b>Default</b>
						{:else}
							<button type="button" class="hotmail-btn" onclick={() => handleSetDefault(mailbox)} disabled={settingDefaultId === mailbox.id} style="font-size:9px;">
								Set Default
							</button>
						{/if}
					</td>
					<td style="text-align:right;white-space:nowrap;">
						<button type="button" class="hotmail-btn" onclick={() => openEditForm(mailbox)}>Edit</button>
						<button type="button" class="hotmail-btn" style="color:var(--error-red);" onclick={() => handleDelete(mailbox)} disabled={deletingId === mailbox.id}>
							Delete
						</button>
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
{/if}
