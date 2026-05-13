/**
 * SSE (Server-Sent Events) Service
 * Provides real-time updates from the server via SSE.
 * Falls back to polling when SSE is unavailable.
 */

import { messagesStore } from '$stores/messages.svelte';
import { notificationsStore } from '$stores/notifications.svelte';
import { pollingService } from './polling.svelte';

// ============================================================================
// Types
// ============================================================================

interface SSEEvent {
	event: string;
	mailboxId: string;
	messageId?: string;
	messageCount?: number;
	flags?: string[];
	timestamp: string;
}

// ============================================================================
// SSE Service Implementation
// ============================================================================

function createSSEService() {
	let eventSource: EventSource | null = null;
	let isConnected = $state(false);
	let reconnectAttempts = $state(0);
	let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
	let currentToken: string | null = null;

	const MAX_RECONNECT_ATTEMPTS = 5;
	const BASE_RECONNECT_DELAY = 3000;

	function getReconnectDelay(): number {
		const jitter = Math.random() * 2000;
		return Math.min(BASE_RECONNECT_DELAY * Math.pow(2, reconnectAttempts), 30000) + jitter;
	}

	function handleEvent(event: MessageEvent): void {
		try {
			const data: SSEEvent = JSON.parse(event.data);

			switch (data.event) {
				case 'message.new':
					notificationsStore.notifyNewMessages(1);
					messagesStore.refresh();
					break;
				case 'message.flags':
				case 'message.deleted':
					messagesStore.refresh();
					break;
				case 'mailbox.updated':
					messagesStore.loadMailboxes();
					break;
			}
		} catch {
			// Ignore parse errors (keepalive comments etc.)
		}
	}

	function connect(token: string): void {
		if (eventSource) {
			eventSource.close();
		}

		currentToken = token;
		const baseUrl = typeof window !== 'undefined' ? window.location.origin : '';
		const url = `${baseUrl}/api/v1/events/stream?token=${encodeURIComponent(token)}`;

		eventSource = new EventSource(url);

		eventSource.addEventListener('connected', () => {
			isConnected = true;
			reconnectAttempts = 0;
			pollingService.stop();
		});

		eventSource.addEventListener('message.new', handleEvent);
		eventSource.addEventListener('message.flags', handleEvent);
		eventSource.addEventListener('message.deleted', handleEvent);
		eventSource.addEventListener('mailbox.updated', handleEvent);

		eventSource.onerror = () => {
			isConnected = false;

			if (eventSource) {
				eventSource.close();
				eventSource = null;
			}

			if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS && currentToken) {
				reconnectAttempts++;
				const delay = getReconnectDelay();
				reconnectTimer = setTimeout(() => {
					if (currentToken) {
						connect(currentToken);
					}
				}, delay);
			} else {
				pollingService.start();
			}
		};
	}

	function start(token: string): void {
		if (typeof window === 'undefined' || typeof EventSource === 'undefined') {
			pollingService.start();
			return;
		}

		connect(token);
	}

	function stop(): void {
		currentToken = null;
		reconnectAttempts = 0;

		if (reconnectTimer) {
			clearTimeout(reconnectTimer);
			reconnectTimer = null;
		}

		if (eventSource) {
			eventSource.close();
			eventSource = null;
		}

		isConnected = false;
	}

	return {
		get isConnected() {
			return isConnected;
		},
		get reconnectAttempts() {
			return reconnectAttempts;
		},
		start,
		stop
	};
}

// ============================================================================
// Singleton Instance
// ============================================================================

export const sseService = createSSEService();

export type SSEService = ReturnType<typeof createSSEService>;
