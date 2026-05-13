/**
 * Authentication Store
 * Manages authentication state with Svelte 5 runes for reactivity.
 * Handles login, logout, token persistence, and automatic token refresh.
 */

import { getAuthApi, LocalStorageTokenStorage, getApiClient } from '$lib/api';
import type { UserInfo, TokenPair, LoginInput } from '$lib/api';

// ============================================================================
// Types
// ============================================================================

export interface AuthState {
	/** Current authenticated user or null if not authenticated */
	user: UserInfo | null;
	/** Whether authentication is currently being checked */
	isLoading: boolean;
	/** Whether user is authenticated */
	isAuthenticated: boolean;
	/** Error message from last auth operation */
	error: string | null;
}

// ============================================================================
// Auth Store Implementation
// ============================================================================

/**
 * Creates the authentication store with Svelte 5 runes.
 * This is a factory function that returns the store state and actions.
 */
function createAuthStore() {
	// State using Svelte 5 runes
	let user = $state<UserInfo | null>(null);
	let isLoading = $state(true);
	let error = $state<string | null>(null);

	// Derived state
	const isAuthenticated = $derived(user !== null);

	// Get API instances
	const authApi = getAuthApi();
	const apiClient = getApiClient();

	// Setup localStorage token storage for persistence
	const tokenStorage = new LocalStorageTokenStorage('yunt');
	apiClient.setTokenStorage(tokenStorage);

	/**
	 * Initialize auth state from stored tokens
	 * Should be called on app startup
	 */
	async function initialize(): Promise<void> {
		isLoading = true;
		error = null;

		try {
			// Check if we have stored tokens
			const accessToken = tokenStorage.getAccessToken();
			if (!accessToken) {
				user = null;
				return;
			}

			// Validate tokens by fetching current user
			const currentUser = await authApi.getCurrentUser();
			user = currentUser;
		} catch {
			// Token is invalid or expired, clear it
			tokenStorage.clearTokens();
			user = null;
		} finally {
			isLoading = false;
		}
	}

	/**
	 * Login with credentials
	 * @param input Login credentials (username and password)
	 * @returns Promise that resolves when login is complete
	 */
	async function login(input: LoginInput): Promise<void> {
		isLoading = true;
		error = null;

		try {
			const response = await authApi.login(input);
			user = response.user;
		} catch (err) {
			const message = err instanceof Error ? err.message : 'Login failed. Please try again.';
			error = message;
			throw err;
		} finally {
			isLoading = false;
		}
	}

	/**
	 * Logout current user
	 * Clears local tokens and invalidates server session
	 */
	async function logout(): Promise<void> {
		isLoading = true;
		error = null;

		try {
			await authApi.logout();
		} catch {
			// Ignore errors, we'll clear local state anyway
		} finally {
			user = null;
			isLoading = false;
		}
	}

	/**
	 * Logout from all sessions
	 * Invalidates all sessions for the current user
	 */
	async function logoutAll(): Promise<void> {
		isLoading = true;
		error = null;

		try {
			await authApi.logoutAll();
		} catch {
			// Ignore errors, we'll clear local state anyway
		} finally {
			user = null;
			isLoading = false;
		}
	}

	/**
	 * Refresh the current user data from the server
	 */
	async function refreshUser(): Promise<void> {
		if (!isAuthenticated) return;

		try {
			const currentUser = await authApi.getCurrentUser();
			user = currentUser;
		} catch {
			// If refresh fails, user might be logged out
			user = null;
			tokenStorage.clearTokens();
		}
	}

	/**
	 * Clear any error state
	 */
	function clearError(): void {
		error = null;
	}

	/**
	 * Set tokens directly (e.g., from external source)
	 * @param tokens Token pair to set
	 */
	function setTokens(tokens: TokenPair): void {
		tokenStorage.setTokens(tokens);
	}

	/**
	 * Get current access token
	 * @returns Access token or null
	 */
	function getAccessToken(): string | null {
		return tokenStorage.getAccessToken();
	}

	return {
		// State (getters for reactivity)
		get user() {
			return user;
		},
		get isLoading() {
			return isLoading;
		},
		get isAuthenticated() {
			return isAuthenticated;
		},
		get error() {
			return error;
		},

		// Actions
		initialize,
		login,
		logout,
		logoutAll,
		refreshUser,
		clearError,
		setTokens,
		getAccessToken
	};
}

// ============================================================================
// Singleton Instance
// ============================================================================

// Create singleton store instance
export const authStore = createAuthStore();

// Export type for the store
export type AuthStore = ReturnType<typeof createAuthStore>;
