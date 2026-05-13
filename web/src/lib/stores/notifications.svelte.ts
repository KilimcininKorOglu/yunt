/**
 * Notifications Store
 * Manages notification state and user preferences for real-time updates.
 * Handles toast notifications and notification preferences with persistence.
 */

// ============================================================================
// Types
// ============================================================================

export type NotificationType = 'info' | 'success' | 'warning' | 'error';

export interface Toast {
	id: string;
	type: NotificationType;
	title: string;
	message?: string;
	duration?: number;
	dismissible?: boolean;
}

export interface NotificationPreferences {
	/** Whether notifications are enabled */
	enabled: boolean;
	/** Play sound on new message */
	playSound: boolean;
	/** Show desktop notifications (requires permission) */
	showDesktop: boolean;
	/** Show in-app toast notifications */
	showToast: boolean;
}

const DEFAULT_PREFERENCES: NotificationPreferences = {
	enabled: true,
	playSound: false,
	showDesktop: false,
	showToast: true
};

const STORAGE_KEY = 'yunt-notification-preferences';
const DEFAULT_TOAST_DURATION = 5000;

// ============================================================================
// Notifications Store Implementation
// ============================================================================

/**
 * Creates the notifications store with Svelte 5 runes.
 */
function createNotificationsStore() {
	// State using Svelte 5 runes
	let toasts = $state<Toast[]>([]);
	let preferences = $state<NotificationPreferences>(loadPreferences());
	let hasDesktopPermission = $state(false);

	// Check desktop notification permission on initialization
	if (typeof window !== 'undefined' && 'Notification' in window) {
		hasDesktopPermission = Notification.permission === 'granted';
	}

	/**
	 * Load preferences from localStorage
	 */
	function loadPreferences(): NotificationPreferences {
		if (typeof window === 'undefined') {
			return DEFAULT_PREFERENCES;
		}

		try {
			const stored = localStorage.getItem(STORAGE_KEY);
			if (stored) {
				return { ...DEFAULT_PREFERENCES, ...JSON.parse(stored) };
			}
		} catch {
			console.warn('Failed to load notification preferences');
		}

		return DEFAULT_PREFERENCES;
	}

	/**
	 * Save preferences to localStorage
	 */
	function savePreferences(): void {
		if (typeof window === 'undefined') return;

		try {
			localStorage.setItem(STORAGE_KEY, JSON.stringify(preferences));
		} catch {
			console.warn('Failed to save notification preferences');
		}
	}

	/**
	 * Generate unique ID for toast
	 */
	function generateId(): string {
		return `toast-${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
	}

	/**
	 * Add a toast notification
	 * @param toast Toast configuration
	 * @returns Toast ID for programmatic dismissal
	 */
	function addToast(toast: Omit<Toast, 'id'>): string {
		const id = generateId();
		const newToast: Toast = {
			id,
			dismissible: true,
			duration: DEFAULT_TOAST_DURATION,
			...toast
		};

		toasts = [...toasts, newToast];

		// Auto-dismiss after duration
		if (newToast.duration && newToast.duration > 0) {
			setTimeout(() => {
				dismissToast(id);
			}, newToast.duration);
		}

		return id;
	}

	/**
	 * Dismiss a toast notification
	 * @param id Toast ID to dismiss
	 */
	function dismissToast(id: string): void {
		toasts = toasts.filter((t) => t.id !== id);
	}

	/**
	 * Clear all toast notifications
	 */
	function clearAllToasts(): void {
		toasts = [];
	}

	/**
	 * Show info toast
	 */
	function info(title: string, message?: string): string {
		return addToast({ type: 'info', title, message });
	}

	/**
	 * Show success toast
	 */
	function success(title: string, message?: string): string {
		return addToast({ type: 'success', title, message });
	}

	/**
	 * Show warning toast
	 */
	function warning(title: string, message?: string): string {
		return addToast({ type: 'warning', title, message });
	}

	/**
	 * Show error toast
	 */
	function error(title: string, message?: string): string {
		return addToast({ type: 'error', title, message });
	}

	/**
	 * Show new message notification
	 * Respects user preferences and triggers appropriate notifications
	 * @param count Number of new messages
	 * @param fromAddress Sender address (optional, for single message)
	 * @param subject Email subject (optional, for single message)
	 */
	function notifyNewMessages(count: number, fromAddress?: string, subject?: string): void {
		if (!preferences.enabled) return;

		// Show in-app toast
		if (preferences.showToast) {
			const title = count === 1 ? 'New Message' : `${count} New Messages`;
			const message = count === 1 && fromAddress
				? `From: ${fromAddress}${subject ? ` - ${subject}` : ''}`
				: undefined;
			addToast({ type: 'info', title, message });
		}

		// Show desktop notification
		if (preferences.showDesktop && hasDesktopPermission) {
			showDesktopNotification(count, fromAddress, subject);
		}

		// Play sound
		if (preferences.playSound) {
			playNotificationSound();
		}
	}

	/**
	 * Show desktop notification
	 */
	function showDesktopNotification(count: number, fromAddress?: string, subject?: string): void {
		if (typeof window === 'undefined' || !('Notification' in window)) return;
		if (Notification.permission !== 'granted') return;

		const title = count === 1 ? 'New Message - Yunt' : `${count} New Messages - Yunt`;
		const body = count === 1 && fromAddress
			? `From: ${fromAddress}${subject ? `\n${subject}` : ''}`
			: `You have ${count} new message${count > 1 ? 's' : ''}`;

		new Notification(title, {
			body,
			icon: '/favicon.png',
			tag: 'yunt-new-message'
		});
	}

	/**
	 * Play notification sound
	 */
	function playNotificationSound(): void {
		if (typeof window === 'undefined') return;

		try {
			// Create a simple beep sound using Web Audio API
			const audioContext = new (window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext)();
			const oscillator = audioContext.createOscillator();
			const gainNode = audioContext.createGain();

			oscillator.connect(gainNode);
			gainNode.connect(audioContext.destination);

			oscillator.frequency.value = 800;
			oscillator.type = 'sine';
			gainNode.gain.value = 0.1;

			oscillator.start();
			oscillator.stop(audioContext.currentTime + 0.1);
		} catch {
			// Audio not supported or blocked
		}
	}

	/**
	 * Request desktop notification permission
	 */
	async function requestDesktopPermission(): Promise<boolean> {
		if (typeof window === 'undefined' || !('Notification' in window)) {
			return false;
		}

		if (Notification.permission === 'granted') {
			hasDesktopPermission = true;
			return true;
		}

		if (Notification.permission === 'denied') {
			return false;
		}

		const permission = await Notification.requestPermission();
		hasDesktopPermission = permission === 'granted';
		return hasDesktopPermission;
	}

	/**
	 * Update notification preferences
	 * @param newPrefs Partial preferences to update
	 */
	function updatePreferences(newPrefs: Partial<NotificationPreferences>): void {
		preferences = { ...preferences, ...newPrefs };
		savePreferences();
	}

	/**
	 * Enable all notifications
	 */
	function enableNotifications(): void {
		updatePreferences({ enabled: true });
	}

	/**
	 * Disable all notifications
	 */
	function disableNotifications(): void {
		updatePreferences({ enabled: false });
	}

	/**
	 * Toggle notifications enabled state
	 */
	function toggleNotifications(): void {
		updatePreferences({ enabled: !preferences.enabled });
	}

	return {
		// State getters
		get toasts() {
			return toasts;
		},
		get preferences() {
			return preferences;
		},
		get hasDesktopPermission() {
			return hasDesktopPermission;
		},

		// Toast actions
		addToast,
		dismissToast,
		clearAllToasts,
		info,
		success,
		warning,
		error,

		// Notification actions
		notifyNewMessages,
		requestDesktopPermission,

		// Preference actions
		updatePreferences,
		enableNotifications,
		disableNotifications,
		toggleNotifications
	};
}

// ============================================================================
// Singleton Instance
// ============================================================================

export const notificationsStore = createNotificationsStore();

export type NotificationsStore = ReturnType<typeof createNotificationsStore>;
