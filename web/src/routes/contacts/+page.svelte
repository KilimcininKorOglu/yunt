<script lang="ts">
	import { onMount } from 'svelte';
	import { authStore } from '$stores/auth.svelte';
	import { getMessagesApi } from '$lib/api';

	const messagesApi = getMessagesApi();

	let contacts = $state<{ name: string; email: string }[]>([]);
	let loading = $state(true);
	let selectedLetter = $state('');

	const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'.split('');

	const filteredContacts = $derived(
		selectedLetter
			? contacts.filter((c) => (c.name || c.email).toUpperCase().startsWith(selectedLetter))
			: contacts
	);

	onMount(async () => {
		if (!authStore.isAuthenticated) return;

		try {
			const response = await messagesApi.list({ pageSize: 100, sort: 'receivedAt', order: 'desc' });
			const seen = new Set<string>();
			const result: { name: string; email: string }[] = [];

			for (const msg of response.items) {
				const addr = msg.from.address;
				if (!seen.has(addr)) {
					seen.add(addr);
					result.push({ name: msg.from.name || '', email: addr });
				}
			}

			result.sort((a, b) => (a.name || a.email).localeCompare(b.name || b.email));
			contacts = result;
		} catch (err) {
			console.error('Failed to load contacts:', err);
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>Contacts - Yunt Mail</title>
</svelte:head>

<div class="options-area">
	<h2>Contacts</h2>
	<p style="font-size:11px;color:var(--text-muted);margin-bottom:8px;">
		Senders from your received messages.
	</p>

	<div class="alphabet-bar">
		<button
			type="button"
			class:active={selectedLetter === ''}
			onclick={() => (selectedLetter = '')}
			style="background:none;border:none;cursor:pointer;padding:1px 3px;font-size:10px;color:var(--msn-dark);{selectedLetter === '' ? 'font-weight:bold;background:var(--msn-dark);color:#fff;' : ''}"
		>All</button>
		{#each alphabet as letter}
			<button
				type="button"
				onclick={() => (selectedLetter = letter)}
				style="background:none;border:none;cursor:pointer;padding:1px 3px;font-size:10px;color:var(--msn-dark);{selectedLetter === letter ? 'font-weight:bold;background:var(--msn-dark);color:#fff;' : ''}"
			>{letter}</button>
		{/each}
	</div>

	{#if loading}
		<div style="text-align:center;padding:20px;">
			<div class="loading-spinner"></div>
			Loading contacts...
		</div>
	{:else if filteredContacts.length === 0}
		<p style="padding:20px;text-align:center;color:var(--text-muted);">No contacts found.</p>
	{:else}
		<table class="msg-table" style="margin-top:8px;">
			<thead>
				<tr>
					<th>Name</th>
					<th>E-Mail</th>
				</tr>
			</thead>
			<tbody>
				{#each filteredContacts as contact (contact.email)}
					<tr>
						<td>{contact.name || '-'}</td>
						<td><a href="mailto:{contact.email}">{contact.email}</a></td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</div>
