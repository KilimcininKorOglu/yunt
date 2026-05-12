<script lang="ts">
	import type { Message } from '$lib/api/types';
	import { getMessagesApi } from '$lib/api';

	interface Props {
		message: Message;
		inlineAttachments?: Map<string, string>;
	}

	const { message, inlineAttachments = new Map() }: Props = $props();

	type TabId = 'html' | 'text' | 'headers' | 'raw';
	let activeTab = $state<TabId>('html');

	let rawLoading = $state(false);
	let rawContent = $state<string | null>(null);
	let rawError = $state<string | null>(null);

	const messagesApi = getMessagesApi();

	$effect(() => {
		if (message.htmlBody) {
			activeTab = 'html';
		} else if (message.textBody) {
			activeTab = 'text';
		}
	});

	function sanitizeHtml(html: string): string {
		let sanitized = html;

		for (const [contentId, dataUrl] of inlineAttachments) {
			const cidPattern = new RegExp(
				`cid:${contentId.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}`,
				'gi'
			);
			sanitized = sanitized.replace(cidPattern, dataUrl);
		}

		sanitized = sanitized
			.replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
			.replace(/\s+on\w+\s*=\s*["'][^"']*["']/gi, '')
			.replace(/\s+on\w+\s*=\s*[^\s>]+/gi, '')
			.replace(/javascript\s*:/gi, 'javascript-blocked:')
			.replace(/data\s*:\s*text\/html/gi, 'data-blocked:text/html')
			.replace(/<base\b[^>]*>/gi, '')
			.replace(/<form\b[^>]*>.*?<\/form>/gis, '')
			.replace(/<input\b[^>]*>/gi, '')
			.replace(/<button\b[^>]*>.*?<\/button>/gis, '')
			.replace(/<meta\s+http-equiv\s*=\s*["']?refresh["']?[^>]*>/gi, '');

		return sanitized;
	}

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
	font-family: Arial, sans-serif;
	font-size: 13px;
	line-height: 1.6;
	color: #333;
	margin: 0;
	padding: 16px;
	background: white;
	word-wrap: break-word;
}
a { color: #0033cc; }
img { max-width: 100%; height: auto; }
table { border-collapse: collapse; max-width: 100%; }
blockquote { border-left: 2px solid #8aafcc; margin: 1em 0; padding-left: 1em; color: #666; }
pre, code { background: #f0f4f8; font-family: monospace; font-size: 12px; }
pre { padding: 8px; overflow-x: auto; }
</style>
</head>
<body>${sanitized}</body>
</html>`;
	}

	function formatPlainText(text: string): string {
		let escaped = text
			.replace(/&/g, '&amp;')
			.replace(/</g, '&lt;')
			.replace(/>/g, '&gt;')
			.replace(/"/g, '&quot;')
			.replace(/'/g, '&#039;');

		const urlPattern = /https?:\/\/[^\s<>"{}|\\^`[\]]+/g;
		escaped = escaped.replace(urlPattern, (url) => {
			return `<a href="${url}" target="_blank" rel="noopener noreferrer">${url}</a>`;
		});

		const emailPattern = /[\w.-]+@[\w.-]+\.[a-z]{2,}/gi;
		escaped = escaped.replace(emailPattern, (email) => {
			return `<a href="mailto:${email}">${email}</a>`;
		});

		return escaped;
	}

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

	async function downloadRaw(): Promise<void> {
		const filename = `${message.subject || 'message'}.eml`.replace(/[/\\?%*:|"<>]/g, '-');
		await messagesApi.downloadRaw(message.id, filename);
	}

	$effect(() => {
		if (activeTab === 'raw') {
			loadRawContent();
		}
	});
</script>

<div class="body-tabs">
	<div class="body-tab-bar">
		<button type="button" class="body-tab" class:active={activeTab === 'html'} disabled={!message.htmlBody} onclick={() => (activeTab = 'html')}>HTML</button>
		<button type="button" class="body-tab" class:active={activeTab === 'text'} disabled={!message.textBody} onclick={() => (activeTab = 'text')}>Text</button>
		<button type="button" class="body-tab" class:active={activeTab === 'headers'} onclick={() => (activeTab = 'headers')}>Headers</button>
		<button type="button" class="body-tab" class:active={activeTab === 'raw'} onclick={() => (activeTab = 'raw')}>Raw</button>
		{#if activeTab === 'raw'}
			<button type="button" class="hotmail-btn" style="margin-left:auto;font-size:10px;" onclick={downloadRaw}>Download .eml</button>
		{/if}
	</div>

	<div class="body-content">
		{#if activeTab === 'html'}
			{#if message.htmlBody}
				<iframe
					title="Email content"
					srcdoc={generateIframeSrcDoc(message.htmlBody)}
					sandbox="allow-same-origin"
					style="width:100%;border:none;min-height:400px;height:100%;"
				></iframe>
			{:else}
				<div class="body-empty">No HTML content available</div>
			{/if}
		{:else if activeTab === 'text'}
			{#if message.textBody}
				<div class="read-body">
					<!-- eslint-disable-next-line svelte/no-at-html-tags -->
					<pre style="white-space:pre-wrap;word-wrap:break-word;font-family:monospace;font-size:12px;">{@html formatPlainText(message.textBody)}</pre>
				</div>
			{:else}
				<div class="body-empty">No plain text content available</div>
			{/if}
		{:else if activeTab === 'headers'}
			<div class="read-body">
				{#if message.headers && Object.keys(message.headers).length > 0}
					<table class="msg-table" style="font-size:11px;"><tbody>
						{#each Object.entries(message.headers) as [key, value] (key)}
							<tr>
								<td style="font-weight:bold;padding:2px 8px 2px 0;vertical-align:top;white-space:nowrap;color:var(--text-label);">{key}</td>
								<td style="padding:2px 0;word-break:break-all;font-family:monospace;font-size:10px;">{value}</td>
							</tr>
						{/each}
					</tbody></table>
				{:else}
					<p>No headers available</p>
				{/if}
			</div>
		{:else if activeTab === 'raw'}
			<div class="read-body">
				{#if rawLoading}
					<div style="text-align:center;padding:20px;">
						<div class="loading-spinner"></div>
						Loading raw content...
					</div>
				{:else if rawError}
					<div class="alert alert-error">{rawError}</div>
				{:else if rawContent}
					<pre style="white-space:pre-wrap;word-break:break-all;font-family:monospace;font-size:10px;background:#f8fafe;padding:10px;border:1px solid var(--border-light);max-height:500px;overflow:auto;">{rawContent}</pre>
				{/if}
			</div>
		{/if}
	</div>
</div>
