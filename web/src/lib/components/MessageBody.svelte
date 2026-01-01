<script lang="ts">
	/**
	 * MessageBody Component
	 * Displays message content with tabs for HTML, Text, Headers, and Raw views.
	 * Implements safe HTML rendering with XSS protection using sandboxed iframe.
	 */

	import type { Message } from '$lib/api/types';
	import { getMessagesApi } from '$lib/api';

	interface Props {
		/** The message object */
		message: Message;
		/** Map of inline attachment content IDs to data URLs */
		inlineAttachments?: Map<string, string>;
	}

	const { message, inlineAttachments = new Map() }: Props = $props();

	// Tab state
	type TabId = 'html' | 'text' | 'headers' | 'raw';
	let activeTab = $state<TabId>('html');

	// Loading states
	let rawLoading = $state(false);
	let rawContent = $state<string | null>(null);
	let rawError = $state<string | null>(null);

	const messagesApi = getMessagesApi();

	// Determine which tab should be default based on content availability
	$effect(() => {
		if (message.htmlBody) {
			activeTab = 'html';
		} else if (message.textBody) {
			activeTab = 'text';
		}
	});

	/**
	 * Sanitize HTML content for safe rendering
	 * This provides defense-in-depth alongside the sandboxed iframe
	 */
	function sanitizeHtml(html: string): string {
		// Replace inline image references with data URLs
		let sanitized = html;

		// Replace cid: references with actual attachment data URLs
		for (const [contentId, dataUrl] of inlineAttachments) {
			// Handle both cid:xxx and src="cid:xxx" formats
			const cidPattern = new RegExp(
				`cid:${contentId.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}`,
				'gi'
			);
			sanitized = sanitized.replace(cidPattern, dataUrl);
		}

		// Remove dangerous elements and attributes
		// This is additional protection on top of iframe sandboxing
		sanitized = sanitized
			// Remove script tags and their content
			.replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
			// Remove event handlers (onclick, onerror, etc.)
			.replace(/\s+on\w+\s*=\s*["'][^"']*["']/gi, '')
			.replace(/\s+on\w+\s*=\s*[^\s>]+/gi, '')
			// Remove javascript: URLs
			.replace(/javascript\s*:/gi, 'javascript-blocked:')
			// Remove data: URLs for scripts (but allow images)
			.replace(/data\s*:\s*text\/html/gi, 'data-blocked:text/html')
			// Remove base tags that could redirect relative URLs
			.replace(/<base\b[^>]*>/gi, '')
			// Remove form elements that could be used for phishing
			.replace(/<form\b[^>]*>.*?<\/form>/gis, '')
			.replace(/<input\b[^>]*>/gi, '')
			.replace(/<button\b[^>]*>.*?<\/button>/gis, '')
			// Remove meta refresh redirects
			.replace(/<meta\s+http-equiv\s*=\s*["']?refresh["']?[^>]*>/gi, '');

		return sanitized;
	}

	/**
	 * Generate srcDoc content for the sandboxed iframe
	 */
	function generateIframeSrcDoc(html: string): string {
		const sanitized = sanitizeHtml(html);
		return `<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<meta http-equiv="Content-Security-Policy" content="default-src 'none'; img-src data: https: http:; style-src 'unsafe-inline';">
	<style>
		* { box-sizing: border-box; }
		body {
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
			font-size: 14px;
			line-height: 1.6;
			color: #1e293b;
			margin: 0;
			padding: 16px;
			background: white;
			word-wrap: break-word;
			overflow-wrap: break-word;
		}
		a { color: #0ea5e9; }
		img { max-width: 100%; height: auto; }
		table { border-collapse: collapse; max-width: 100%; }
		blockquote {
			border-left: 3px solid #e2e8f0;
			margin: 1em 0;
			padding-left: 1em;
			color: #64748b;
		}
		pre, code {
			background: #f1f5f9;
			border-radius: 4px;
			font-family: monospace;
			font-size: 13px;
		}
		pre { padding: 12px; overflow-x: auto; }
		code { padding: 2px 4px; }
	</style>
</head>
<body>${sanitized}</body>
</html>`;
	}

	/**
	 * Format plain text with basic whitespace preservation and link detection
	 */
	function formatPlainText(text: string): string {
		// Escape HTML entities
		let escaped = text
			.replace(/&/g, '&amp;')
			.replace(/</g, '&lt;')
			.replace(/>/g, '&gt;')
			.replace(/"/g, '&quot;')
			.replace(/'/g, '&#039;');

		// Convert URLs to links
		const urlPattern = /https?:\/\/[^\s<>"{}|\\^`[\]]+/g;
		escaped = escaped.replace(urlPattern, (url) => {
			return `<a href="${url}" target="_blank" rel="noopener noreferrer" class="text-primary-600 hover:underline">${url}</a>`;
		});

		// Convert email addresses to mailto links
		const emailPattern = /[\w.-]+@[\w.-]+\.[a-z]{2,}/gi;
		escaped = escaped.replace(emailPattern, (email) => {
			return `<a href="mailto:${email}" class="text-primary-600 hover:underline">${email}</a>`;
		});

		return escaped;
	}

	/**
	 * Load raw message content
	 */
	async function loadRawContent(): Promise<void> {
		if (rawContent !== null || rawLoading) return;

		rawLoading = true;
		rawError = null;

		try {
			const blob = await messagesApi.getRaw(message.id);
			rawContent = await blob.text();
		} catch (err) {
			rawError = err instanceof Error ? err.message : 'Failed to load raw content';
		} finally {
			rawLoading = false;
		}
	}

	/**
	 * Download raw message as EML file
	 */
	async function downloadRaw(): Promise<void> {
		const filename = `${message.subject || 'message'}.eml`.replace(/[/\\?%*:|"<>]/g, '-');
		await messagesApi.downloadRaw(message.id, filename);
	}

	// Load raw content when tab is selected
	$effect(() => {
		if (activeTab === 'raw') {
			loadRawContent();
		}
	});

	// Tab definitions
	const tabs: { id: TabId; label: string; icon: string; available: boolean }[] = $derived([
		{
			id: 'html',
			label: 'HTML',
			icon: 'M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z',
			available: !!message.htmlBody
		},
		{
			id: 'text',
			label: 'Text',
			icon: 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z',
			available: !!message.textBody
		},
		{
			id: 'headers',
			label: 'Headers',
			icon: 'M4 6h16M4 10h16M4 14h16M4 18h16',
			available: true
		},
		{
			id: 'raw',
			label: 'Raw',
			icon: 'M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4',
			available: true
		}
	]);
</script>

<div class="flex flex-1 flex-col overflow-hidden">
	<!-- Tab navigation -->
	<div class="flex items-center gap-1 border-b border-secondary-200 bg-secondary-50 px-4">
		{#each tabs as tab (tab.id)}
			<button
				type="button"
				onclick={() => (activeTab = tab.id)}
				disabled={!tab.available}
				class="flex items-center gap-2 border-b-2 px-3 py-3 text-sm font-medium transition-colors {activeTab ===
				tab.id
					? 'border-primary-600 text-primary-600'
					: 'border-transparent text-secondary-600 hover:border-secondary-300 hover:text-secondary-900'} {!tab.available
					? 'cursor-not-allowed opacity-50'
					: ''}"
			>
				<svg
					class="h-4 w-4"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d={tab.icon} />
				</svg>
				{tab.label}
			</button>
		{/each}

		<!-- Spacer -->
		<div class="flex-1"></div>

		<!-- Download raw button -->
		{#if activeTab === 'raw'}
			<button
				type="button"
				onclick={downloadRaw}
				class="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-medium text-secondary-600 transition-colors hover:bg-secondary-200 hover:text-secondary-900"
			>
				<svg
					class="h-4 w-4"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
					stroke-width="2"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
					/>
				</svg>
				Download
			</button>
		{/if}
	</div>

	<!-- Tab content -->
	<div class="flex-1 overflow-auto">
		{#if activeTab === 'html'}
			{#if message.htmlBody}
				<!-- Sandboxed iframe for HTML content -->
				<iframe
					title="Email content"
					srcdoc={generateIframeSrcDoc(message.htmlBody)}
					sandbox="allow-same-origin"
					class="h-full w-full border-none"
					style="min-height: 400px;"
				></iframe>
			{:else}
				<div class="flex h-full items-center justify-center text-secondary-500">
					<p>No HTML content available</p>
				</div>
			{/if}
		{:else if activeTab === 'text'}
			{#if message.textBody}
				<div class="p-6">
					<!-- eslint-disable-next-line svelte/no-at-html-tags -- formatPlainText sanitizes content first then adds safe link markup -->
					<pre
						class="whitespace-pre-wrap break-words font-mono text-sm text-secondary-800">{@html formatPlainText(
							message.textBody
						)}</pre>
				</div>
			{:else}
				<div class="flex h-full items-center justify-center text-secondary-500">
					<p>No plain text content available</p>
				</div>
			{/if}
		{:else if activeTab === 'headers'}
			<div class="p-6">
				{#if message.headers && Object.keys(message.headers).length > 0}
					<div class="overflow-x-auto">
						<table class="w-full text-sm">
							<tbody class="divide-y divide-secondary-100">
								{#each Object.entries(message.headers) as [key, value] (key)}
									<tr>
										<td
											class="whitespace-nowrap py-2 pr-4 font-medium text-secondary-700"
											>{key}</td
										>
										<td
											class="break-all py-2 font-mono text-xs text-secondary-600"
											>{value}</td
										>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{:else}
					<p class="text-secondary-500">No headers available</p>
				{/if}
			</div>
		{:else if activeTab === 'raw'}
			<div class="p-6">
				{#if rawLoading}
					<div class="flex items-center justify-center py-12">
						<div class="flex items-center gap-3">
							<div
								class="h-5 w-5 animate-spin rounded-full border-2 border-primary-200 border-t-primary-600"
							></div>
							<span class="text-secondary-600">Loading raw content...</span>
						</div>
					</div>
				{:else if rawError}
					<div class="rounded-lg border border-red-200 bg-red-50 p-4 text-red-700">
						<p class="font-medium">Error loading raw content</p>
						<p class="mt-1 text-sm">{rawError}</p>
					</div>
				{:else if rawContent}
					<pre
						class="max-h-[600px] overflow-auto whitespace-pre-wrap break-all rounded-lg border border-secondary-200 bg-secondary-50 p-4 font-mono text-xs text-secondary-700">{rawContent}</pre>
				{/if}
			</div>
		{/if}
	</div>
</div>
