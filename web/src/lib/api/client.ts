/**
 * Base API Client for Yunt Mail Server
 * Handles authentication, token refresh, and HTTP requests
 */

import type { ApiResponse, TokenPair, RefreshTokenInput } from './types';
import { getCookie, setCookie, deleteCookie } from '$lib/utils/cookie';

// ============================================================================
// Types
// ============================================================================

/** HTTP methods supported by the client */
export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';

/** Request options for API calls */
export interface RequestOptions {
	/** Skip authentication header */
	skipAuth?: boolean;
	/** Custom headers */
	headers?: Record<string, string>;
	/** Query parameters */
	params?: Record<string, string | number | boolean | undefined>;
	/** Request timeout in milliseconds */
	timeout?: number;
	/** Custom signal for request cancellation */
	signal?: AbortSignal;
}

/** API client configuration */
export interface ApiClientConfig {
	/** Base URL for API requests */
	baseUrl: string;
	/** Callback when tokens are refreshed */
	onTokenRefresh?: (tokens: TokenPair) => void;
	/** Callback when authentication fails (401 after retry) */
	onAuthError?: () => void;
	/** Default request timeout in milliseconds */
	timeout?: number;
}

/** Token storage interface */
export interface TokenStorage {
	getAccessToken(): string | null;
	getRefreshToken(): string | null;
	setTokens(tokens: TokenPair): void;
	clearTokens(): void;
}

// ============================================================================
// Custom Errors
// ============================================================================

/** API error class with additional details */
export class ApiClientError extends Error {
	public readonly code: string;
	public readonly status: number;
	public readonly details?: unknown;

	constructor(message: string, code: string, status: number, details?: unknown) {
		super(message);
		this.name = 'ApiClientError';
		this.code = code;
		this.status = status;
		this.details = details;
	}

	/** Check if error is due to authentication failure */
	isAuthError(): boolean {
		return this.status === 401;
	}

	/** Check if error is due to forbidden access */
	isForbiddenError(): boolean {
		return this.status === 403;
	}

	/** Check if error is due to resource not found */
	isNotFoundError(): boolean {
		return this.status === 404;
	}

	/** Check if error is due to validation failure */
	isValidationError(): boolean {
		return this.code === 'VALIDATION_FAILED';
	}

	/** Check if error is a network error */
	isNetworkError(): boolean {
		return this.code === 'NETWORK_ERROR';
	}
}

/** Network error class */
export class NetworkError extends ApiClientError {
	constructor(message: string) {
		super(message, 'NETWORK_ERROR', 0);
		this.name = 'NetworkError';
	}
}

/** Timeout error class */
export class TimeoutError extends ApiClientError {
	constructor(message = 'Request timed out') {
		super(message, 'TIMEOUT_ERROR', 0);
		this.name = 'TimeoutError';
	}
}

// ============================================================================
// Token Storage Implementation
// ============================================================================

/** Memory-based token storage (default) */
export class MemoryTokenStorage implements TokenStorage {
	private accessToken: string | null = null;
	private refreshToken: string | null = null;

	getAccessToken(): string | null {
		return this.accessToken;
	}

	getRefreshToken(): string | null {
		return this.refreshToken;
	}

	setTokens(tokens: TokenPair): void {
		this.accessToken = tokens.accessToken;
		this.refreshToken = tokens.refreshToken;
	}

	clearTokens(): void {
		this.accessToken = null;
		this.refreshToken = null;
	}
}

/** Cookie-based token storage */
export class CookieTokenStorage implements TokenStorage {
	private readonly accessTokenKey: string;
	private readonly refreshTokenKey: string;

	constructor(prefix = 'yunt') {
		this.accessTokenKey = `${prefix}_access_token`;
		this.refreshTokenKey = `${prefix}_refresh_token`;
	}

	getAccessToken(): string | null {
		return getCookie(this.accessTokenKey);
	}

	getRefreshToken(): string | null {
		return getCookie(this.refreshTokenKey);
	}

	setTokens(tokens: TokenPair): void {
		setCookie(this.accessTokenKey, tokens.accessToken, 1);
		setCookie(this.refreshTokenKey, tokens.refreshToken, 7);
	}

	clearTokens(): void {
		deleteCookie(this.accessTokenKey);
		deleteCookie(this.refreshTokenKey);
	}
}

// ============================================================================
// API Client
// ============================================================================

export class ApiClient {
	private readonly config: Required<Omit<ApiClientConfig, 'onTokenRefresh' | 'onAuthError'>> &
		Pick<ApiClientConfig, 'onTokenRefresh' | 'onAuthError'>;
	private tokenStorage: TokenStorage;
	private refreshPromise: Promise<TokenPair> | null = null;

	constructor(config: ApiClientConfig, tokenStorage?: TokenStorage) {
		this.config = {
			baseUrl: config.baseUrl.replace(/\/$/, ''), // Remove trailing slash
			timeout: config.timeout ?? 30000,
			onTokenRefresh: config.onTokenRefresh,
			onAuthError: config.onAuthError
		};
		this.tokenStorage = tokenStorage ?? new MemoryTokenStorage();
	}

	// ========================================================================
	// Token Management
	// ========================================================================

	/** Set the token storage implementation */
	setTokenStorage(storage: TokenStorage): void {
		this.tokenStorage = storage;
	}

	/** Get the current access token */
	getAccessToken(): string | null {
		return this.tokenStorage.getAccessToken();
	}

	/** Get the current refresh token */
	getRefreshToken(): string | null {
		return this.tokenStorage.getRefreshToken();
	}

	/** Set tokens after successful login */
	setTokens(tokens: TokenPair): void {
		this.tokenStorage.setTokens(tokens);
	}

	/** Clear all tokens (logout) */
	clearTokens(): void {
		this.tokenStorage.clearTokens();
	}

	/** Check if user is authenticated */
	isAuthenticated(): boolean {
		return !!this.tokenStorage.getAccessToken();
	}

	// ========================================================================
	// HTTP Methods
	// ========================================================================

	/** Make a GET request */
	async get<T>(path: string, options?: RequestOptions): Promise<T> {
		return this.request<T>('GET', path, undefined, options);
	}

	/** Make a POST request */
	async post<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
		return this.request<T>('POST', path, body, options);
	}

	/** Make a PUT request */
	async put<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
		return this.request<T>('PUT', path, body, options);
	}

	/** Make a PATCH request */
	async patch<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
		return this.request<T>('PATCH', path, body, options);
	}

	/** Make a DELETE request */
	async delete<T>(path: string, options?: RequestOptions): Promise<T> {
		return this.request<T>('DELETE', path, undefined, options);
	}

	/** Make a raw request and return the Response object */
	async rawRequest(
		method: HttpMethod,
		path: string,
		body?: unknown,
		options?: RequestOptions
	): Promise<Response> {
		const url = this.buildUrl(path, options?.params);
		const headers = this.buildHeaders(options);
		if (body instanceof FormData) {
			delete headers['Content-Type'];
		}
		const timeout = options?.timeout ?? this.config.timeout;

		const controller = new AbortController();
		const timeoutId = setTimeout(() => controller.abort(), timeout);
		const signal = options?.signal
			? this.combineSignals(options.signal, controller.signal)
			: controller.signal;

		try {
			const response = await fetch(url, {
				method,
				headers,
				body: body instanceof FormData ? body : body !== undefined ? JSON.stringify(body) : undefined,
				signal
			});

			clearTimeout(timeoutId);
			return response;
		} catch (error) {
			clearTimeout(timeoutId);
			if (error instanceof Error) {
				if (error.name === 'AbortError') {
					throw new TimeoutError();
				}
				throw new NetworkError(error.message);
			}
			throw new NetworkError('An unknown error occurred');
		}
	}

	// ========================================================================
	// Private Methods
	// ========================================================================

	/** Main request method with retry logic */
	private async request<T>(
		method: HttpMethod,
		path: string,
		body?: unknown,
		options?: RequestOptions,
		isRetry = false
	): Promise<T> {
		const response = await this.rawRequest(method, path, body, options);

		// Handle 401 Unauthorized - attempt token refresh
		if (response.status === 401 && !options?.skipAuth && !isRetry) {
			const refreshed = await this.attemptTokenRefresh();
			if (refreshed) {
				// Retry the request with the new token
				return this.request<T>(method, path, body, options, true);
			}
			// Token refresh failed, notify and throw
			this.config.onAuthError?.();
		}

		// Handle 204 No Content
		if (response.status === 204) {
			return undefined as T;
		}

		// Parse response
		const result = await this.parseResponse<T>(response);

		// Check for API-level errors
		if (!response.ok) {
			const apiError = (result as ApiResponse).error;
			throw new ApiClientError(
				apiError?.message ?? 'Request failed',
				apiError?.code ?? 'UNKNOWN_ERROR',
				response.status,
				apiError?.details
			);
		}

		// Return data from successful response
		const apiResponse = result as ApiResponse<T>;
		if (apiResponse.success === false) {
			throw new ApiClientError(
				apiResponse.error?.message ?? 'Request failed',
				apiResponse.error?.code ?? 'UNKNOWN_ERROR',
				response.status,
				apiResponse.error?.details
			);
		}

		return apiResponse.data as T;
	}

	/** Attempt to refresh the access token */
	private async attemptTokenRefresh(): Promise<boolean> {
		const refreshToken = this.tokenStorage.getRefreshToken();
		if (!refreshToken) {
			return false;
		}

		// Prevent concurrent refresh attempts
		if (this.refreshPromise) {
			try {
				await this.refreshPromise;
				return true;
			} catch {
				return false;
			}
		}

		this.refreshPromise = this.doRefreshToken(refreshToken);

		try {
			const tokens = await this.refreshPromise;
			this.tokenStorage.setTokens(tokens);
			this.config.onTokenRefresh?.(tokens);
			return true;
		} catch {
			// Refresh failed, clear tokens
			this.tokenStorage.clearTokens();
			return false;
		} finally {
			this.refreshPromise = null;
		}
	}

	/** Perform the token refresh request */
	private async doRefreshToken(refreshToken: string): Promise<TokenPair> {
		const url = this.buildUrl('/api/v1/auth/refresh');
		const input: RefreshTokenInput = { refreshToken };

		const response = await fetch(url, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json'
			},
			body: JSON.stringify(input)
		});

		if (!response.ok) {
			throw new Error('Token refresh failed');
		}

		const result = (await response.json()) as ApiResponse<{ tokens: TokenPair }>;
		if (!result.success || !result.data?.tokens) {
			throw new Error('Invalid refresh response');
		}

		return result.data.tokens;
	}

	/** Build the full URL with query parameters */
	private buildUrl(
		path: string,
		params?: Record<string, string | number | boolean | undefined>
	): string {
		const url = new URL(path.startsWith('/') ? path : `/${path}`, this.config.baseUrl);

		if (params) {
			Object.entries(params).forEach(([key, value]) => {
				if (value !== undefined) {
					url.searchParams.append(key, String(value));
				}
			});
		}

		return url.toString();
	}

	/** Build request headers */
	private buildHeaders(options?: RequestOptions): Record<string, string> {
		const headers: Record<string, string> = {
			'Content-Type': 'application/json',
			...(options?.headers ?? {})
		};

		// Add authorization header if authenticated and not skipping auth
		if (!options?.skipAuth) {
			const accessToken = this.tokenStorage.getAccessToken();
			if (accessToken) {
				headers['Authorization'] = `Bearer ${accessToken}`;
			}
		}

		return headers;
	}

	/** Parse response body */
	private async parseResponse<T>(response: Response): Promise<T | ApiResponse<T>> {
		const contentType = response.headers.get('Content-Type') ?? '';

		if (contentType.includes('application/json')) {
			return response.json();
		}

		// Handle non-JSON responses
		const text = await response.text();
		return { success: response.ok, data: text as unknown as T } as ApiResponse<T>;
	}

	/** Combine multiple abort signals */
	private combineSignals(signal1: AbortSignal, signal2: AbortSignal): AbortSignal {
		const controller = new AbortController();

		const abort = () => controller.abort();

		if (signal1.aborted || signal2.aborted) {
			controller.abort();
		} else {
			signal1.addEventListener('abort', abort, { once: true });
			signal2.addEventListener('abort', abort, { once: true });
		}

		return controller.signal;
	}
}

// ============================================================================
// Utility Functions
// ============================================================================

/** Build query string from filter options */
export function buildQueryParams(
	options: Record<string, unknown>
): Record<string, string | number | boolean | undefined> {
	const params: Record<string, string | number | boolean | undefined> = {};

	for (const [key, value] of Object.entries(options)) {
		if (value === undefined || value === null || value === '') {
			continue;
		}
		if (typeof value === 'boolean') {
			params[key] = value;
		} else if (typeof value === 'number') {
			params[key] = value;
		} else if (typeof value === 'string') {
			params[key] = value;
		}
	}

	return params;
}

/** Create a default API client instance */
export function createApiClient(baseUrl: string, options?: Partial<ApiClientConfig>): ApiClient {
	return new ApiClient({
		baseUrl,
		...options
	});
}

// ============================================================================
// Singleton Instance
// ============================================================================

let defaultClient: ApiClient | null = null;

/** Get or create the default API client */
export function getApiClient(): ApiClient {
	if (!defaultClient) {
		// Default to relative URL for same-origin API
		defaultClient = createApiClient(
			typeof window !== 'undefined' ? window.location.origin : ''
		);
	}
	return defaultClient;
}

/** Initialize the default API client */
export function initApiClient(config: ApiClientConfig, tokenStorage?: TokenStorage): ApiClient {
	defaultClient = new ApiClient(config, tokenStorage);
	return defaultClient;
}
