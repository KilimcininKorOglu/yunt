/**
 * Authentication API Client
 * Handles login, logout, token refresh, and current user operations
 */

import type { ApiClient } from './client';
import { getApiClient } from './client';
import type { LoginInput, AuthResponse, UserInfo, TokenPair } from './types';

// ============================================================================
// Auth API Class
// ============================================================================

export class AuthApi {
	private readonly client: ApiClient;

	constructor(client?: ApiClient) {
		this.client = client ?? getApiClient();
	}

	/**
	 * Login with username and password
	 * @param input Login credentials
	 * @returns Authentication response with user info and tokens
	 */
	async login(input: LoginInput): Promise<AuthResponse> {
		const response = await this.client.post<AuthResponse>('/api/v1/auth/login', input, {
			skipAuth: true
		});

		// Store tokens after successful login
		if (response.tokens) {
			this.client.setTokens(response.tokens);
		}

		return response;
	}

	/**
	 * Refresh access token using refresh token
	 * @param refreshToken The refresh token
	 * @returns New token pair
	 */
	async refreshToken(refreshToken: string): Promise<{ tokens: TokenPair }> {
		const response = await this.client.post<{ tokens: TokenPair }>(
			'/api/v1/auth/refresh',
			{ refreshToken },
			{ skipAuth: true }
		);

		// Store new tokens
		if (response.tokens) {
			this.client.setTokens(response.tokens);
		}

		return response;
	}

	/**
	 * Logout current session
	 * Invalidates the current session on the server
	 */
	async logout(): Promise<void> {
		try {
			await this.client.post<void>('/api/v1/auth/logout');
		} finally {
			// Always clear local tokens, even if server request fails
			this.client.clearTokens();
		}
	}

	/**
	 * Logout from all sessions
	 * Invalidates all sessions for the current user
	 */
	async logoutAll(): Promise<void> {
		try {
			await this.client.post<void>('/api/v1/auth/logout-all');
		} finally {
			// Always clear local tokens
			this.client.clearTokens();
		}
	}

	/**
	 * Get current authenticated user info
	 * @returns Current user information
	 */
	async getCurrentUser(): Promise<UserInfo> {
		return this.client.get<UserInfo>('/api/v1/auth/me');
	}

	/**
	 * Check if user is authenticated
	 * @returns True if user has valid tokens
	 */
	isAuthenticated(): boolean {
		return this.client.isAuthenticated();
	}

	/**
	 * Get current access token
	 * @returns Access token or null
	 */
	getAccessToken(): string | null {
		return this.client.getAccessToken();
	}

	/**
	 * Get current refresh token
	 * @returns Refresh token or null
	 */
	getRefreshToken(): string | null {
		return this.client.getRefreshToken();
	}

	/**
	 * Clear stored tokens
	 */
	clearTokens(): void {
		this.client.clearTokens();
	}

	/**
	 * Set tokens directly (e.g., from stored state)
	 * @param tokens Token pair to set
	 */
	setTokens(tokens: TokenPair): void {
		this.client.setTokens(tokens);
	}
}

// ============================================================================
// Singleton Instance
// ============================================================================

let authApiInstance: AuthApi | null = null;

/**
 * Get the singleton Auth API instance
 */
export function getAuthApi(): AuthApi {
	if (!authApiInstance) {
		authApiInstance = new AuthApi();
	}
	return authApiInstance;
}

/**
 * Create a new Auth API instance with custom client
 */
export function createAuthApi(client: ApiClient): AuthApi {
	return new AuthApi(client);
}
