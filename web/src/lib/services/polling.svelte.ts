/**
 * Polling Service
 * Handles real-time updates for messages using polling.
 * Automatically pauses when tab is inactive and respects user preferences.
 */

import { getMessagesApi } from '$lib/api';
import { messagesStore } from '$stores/messages.svelte';
import { notificationsStore } from '$stores/notifications.svelte';
import type { Message } from '$lib/api/types';

// ============================================================================
// Types
// ============================================================================

export interface PollingConfig {
	/** Polling interval in milliseconds (default: 30000 = 30 seconds) */
	interval: number;
	/** Whether to pause polling when tab is inactive (default: true) */
	pauseWhenHidden: boolean;
	/** Whether to refresh mailbox counts (default: true) */
	refreshMailboxes: boolean;
}

export interface PollingState {
	/** Whether polling is active */
	isActive: boolean;
	/** Whether polling is currently paused (due to tab visibility) */
	isPaused: boolean;
	/** Last successful poll timestamp */
	lastPollAt: number | null;
	/** Last known message timestamp for comparison */
	lastMessageTimestamp: string | null;
	/** Error from last poll attempt */
	lastError: string | null;
}

const DEFAULT_CONFIG: PollingConfig = {
	interval: 30000,
	pauseWhenHidden: true,
	refreshMailboxes: true
};

const STORAGE_KEY = 'yunt-polling-config';

// ============================================================================
// Polling Service Implementation
// ============================================================================

/**
 * Creates the polling service with Svelte 5 runes.
 */
function createPollingService() {
	// State using Svelte 5 runes
	let config = $state<PollingConfig>(loadConfig());
	let isActive = $state(false);
	let isPaused = $state(false);
	let lastPollAt = $state<number | null>(null);
	let lastMessageTimestamp = $state<string | null>(null);
	let lastError = $state<string | null>(null);
	let lastEtag = $state<string | null>(null);

	// Internal state
	let pollingTimer: ReturnType<typeof setInterval> | null = null;
	let isPolling = false;

	// API instances
	const messagesApi = getMessagesApi();

	function loadConfig(): PollingConfig {
		if (typeof document === 'undefined') return DEFAULT_CONFIG;
		try {
			const match = document.cookie.match(new RegExp('(?:^|; )' + STORAGE_KEY + '=([^;]*)'));
			if (match) return { ...DEFAULT_CONFIG, ...JSON.parse(decodeURIComponent(match[1])) };
		} catch { /* ignore */ }
		return DEFAULT_CONFIG;
	}

	function saveConfig(): void {
		if (typeof document === 'undefined') return;
		try {
			const expires = new Date(Date.now() + 365 * 864e5).toUTCString();
			document.cookie = `${STORAGE_KEY}=${encodeURIComponent(JSON.stringify(config))}; expires=${expires}; path=/; SameSite=Lax`;
		} catch { /* ignore */ }
	}

	/**
	 * Setup visibility change handler
	 */
	function setupVisibilityHandler(): void {
		if (typeof document === 'undefined') return;

		document.addEventListener('visibilitychange', handleVisibilityChange);
	}

	/**
	 * Remove visibility change handler
	 */
	function removeVisibilityHandler(): void {
		if (typeof document === 'undefined') return;

		document.removeEventListener('visibilitychange', handleVisibilityChange);
	}

	/**
	 * Handle document visibility change
	 */
	function handleVisibilityChange(): void {
		if (!config.pauseWhenHidden) return;

		if (document.hidden) {
			pausePolling();
		} else {
			resumePolling();
		}
	}

	/**
	 * Check for new messages
	 */
	async function checkForNewMessages(): Promise<void> {
		if (isPolling) return;
		isPolling = true;
		lastError = null;

		try {
			const result = await messagesApi.listWithEtag(
				{ page: 1, pageSize: 10, sort: 'receivedAt', order: 'desc' },
				lastEtag
			);

			lastPollAt = Date.now();

			if (result.notModified) {
				return;
			}

			lastEtag = result.etag;
			const latestMessages = result.data?.items ?? [];

			if (lastMessageTimestamp && latestMessages.length > 0) {
				const newMessages = latestMessages.filter(
					(msg) => msg.receivedAt > lastMessageTimestamp!
				);

				if (newMessages.length > 0) {
					if (newMessages.length === 1) {
						const msg = newMessages[0];
						notificationsStore.notifyNewMessages(
							1,
							msg.from.address,
							msg.subject
						);
					} else {
						notificationsStore.notifyNewMessages(newMessages.length);
					}

					await messagesStore.refresh();
				}
			}

			if (latestMessages.length > 0) {
				lastMessageTimestamp = latestMessages[0].receivedAt;
			}

			if (config.refreshMailboxes) {
				await messagesStore.loadMailboxes();
			}
		} catch (err) {
			lastError = err instanceof Error ? err.message : 'Failed to check for new messages';
			console.error('Polling error:', lastError);
		} finally {
			isPolling = false;
		}
	}

	/**
	 * Start the polling timer
	 */
	function startPollingTimer(): void {
		if (pollingTimer) return;

		pollingTimer = setInterval(() => {
			if (!isPaused) {
				checkForNewMessages();
			}
		}, config.interval);
	}

	/**
	 * Stop the polling timer
	 */
	function stopPollingTimer(): void {
		if (pollingTimer) {
			clearInterval(pollingTimer);
			pollingTimer = null;
		}
	}

	/**
	 * Start polling for new messages
	 */
	function start(): void {
		if (isActive) return;

		isActive = true;
		isPaused = false;

		// Setup visibility handler
		setupVisibilityHandler();

		// Check visibility state
		if (typeof document !== 'undefined' && document.hidden && config.pauseWhenHidden) {
			isPaused = true;
		}

		// Do initial poll
		checkForNewMessages();

		// Start timer
		startPollingTimer();
	}

	/**
	 * Stop polling completely
	 */
	function stop(): void {
		if (!isActive) return;

		isActive = false;
		isPaused = false;

		// Stop timer
		stopPollingTimer();

		// Remove visibility handler
		removeVisibilityHandler();

		// Clear state
		lastMessageTimestamp = null;
		lastEtag = null;
	}

	/**
	 * Pause polling (e.g., when tab becomes inactive)
	 */
	function pausePolling(): void {
		if (!isActive || isPaused) return;
		isPaused = true;
	}

	/**
	 * Resume polling (e.g., when tab becomes active)
	 */
	function resumePolling(): void {
		if (!isActive || !isPaused) return;
		isPaused = false;

		// Do immediate poll on resume
		checkForNewMessages();
	}

	/**
	 * Force an immediate poll
	 */
	function pollNow(): void {
		if (!isActive) return;
		checkForNewMessages();
	}

	/**
	 * Update polling configuration
	 * @param newConfig Partial config to update
	 */
	function updateConfig(newConfig: Partial<PollingConfig>): void {
		const wasActive = isActive;

		// Stop current polling
		if (wasActive) {
			stop();
		}

		// Update config
		config = { ...config, ...newConfig };
		saveConfig();

		// Restart if it was active
		if (wasActive) {
			start();
		}
	}

	/**
	 * Set polling interval
	 * @param intervalMs Interval in milliseconds
	 */
	function setPollingInterval(intervalMs: number): void {
		updateConfig({ interval: intervalMs });
	}

	/**
	 * Enable or disable pause when hidden
	 * @param enabled Whether to pause when tab is hidden
	 */
	function setPauseWhenHidden(enabled: boolean): void {
		updateConfig({ pauseWhenHidden: enabled });
	}

	/**
	 * Enable or disable mailbox refresh
	 * @param enabled Whether to refresh mailbox counts on each poll
	 */
	function setRefreshMailboxes(enabled: boolean): void {
		updateConfig({ refreshMailboxes: enabled });
	}

	/**
	 * Reset last message timestamp (forces re-check on next poll)
	 */
	function resetLastTimestamp(): void {
		lastMessageTimestamp = null;
	}

	/**
	 * Get time until next poll in milliseconds
	 */
	function getTimeUntilNextPoll(): number | null {
		if (!isActive || isPaused || !lastPollAt) return null;
		const elapsed = Date.now() - lastPollAt;
		return Math.max(0, config.interval - elapsed);
	}

	return {
		// State getters
		get config() {
			return config;
		},
		get isActive() {
			return isActive;
		},
		get isPaused() {
			return isPaused;
		},
		get lastPollAt() {
			return lastPollAt;
		},
		get lastError() {
			return lastError;
		},

		// Actions
		start,
		stop,
		pausePolling,
		resumePolling,
		pollNow,
		updateConfig,
		setPollingInterval,
		setPauseWhenHidden,
		setRefreshMailboxes,
		resetLastTimestamp,
		getTimeUntilNextPoll
	};
}

// ============================================================================
// Singleton Instance
// ============================================================================

export const pollingService = createPollingService();

export type PollingService = ReturnType<typeof createPollingService>;
