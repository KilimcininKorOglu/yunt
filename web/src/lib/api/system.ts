/**
 * System API Client
 * Handles system management and administrative operations
 */

import type { ApiClient } from './client';
import { getApiClient } from './client';
import type {
	VersionInfo,
	SystemStats,
	SystemInfo,
	DeleteAllMessagesResponse,
	CleanupRequest,
	CleanupResponse
} from './types';

// ============================================================================
// System API Class
// ============================================================================

export class SystemApi {
	private readonly client: ApiClient;

	constructor(client?: ApiClient) {
		this.client = client ?? getApiClient();
	}

	/**
	 * Get version information
	 * No authentication required
	 * @returns Version information
	 */
	async getVersion(): Promise<VersionInfo> {
		return this.client.get<VersionInfo>('/api/v1/system/version', { skipAuth: true });
	}

	/**
	 * Get system statistics
	 * Requires authentication
	 * @returns System statistics
	 */
	async getStats(): Promise<SystemStats> {
		return this.client.get<SystemStats>('/api/v1/stats');
	}

	/**
	 * Get detailed system information (admin only)
	 * Includes configuration and runtime info
	 * @returns Detailed system information
	 */
	async getSystemInfo(): Promise<SystemInfo> {
		return this.client.get<SystemInfo>('/api/v1/system/info');
	}

	/**
	 * Delete all messages (admin only)
	 * Warning: This permanently deletes all messages from all mailboxes
	 * @returns Delete result with count
	 */
	async deleteAllMessages(): Promise<DeleteAllMessagesResponse> {
		return this.client.delete<DeleteAllMessagesResponse>('/api/v1/system/messages');
	}

	/**
	 * Run cleanup operations (admin only)
	 * @param options Cleanup options
	 * @returns Cleanup result
	 */
	async cleanup(options: CleanupRequest): Promise<CleanupResponse> {
		return this.client.post<CleanupResponse>('/api/v1/system/cleanup', options);
	}

	/**
	 * Delete old messages (admin only)
	 * Convenience method for cleanup with deleteOldMessages
	 * @param olderThanDays Delete messages older than this many days
	 * @returns Cleanup result
	 */
	async deleteOldMessages(olderThanDays: number): Promise<CleanupResponse> {
		return this.cleanup({ deleteOldMessages: olderThanDays });
	}

	/**
	 * Delete all spam messages (admin only)
	 * @returns Cleanup result
	 */
	async deleteSpam(): Promise<CleanupResponse> {
		return this.cleanup({ deleteSpam: true });
	}

	/**
	 * Recalculate all mailbox statistics (admin only)
	 * @returns Cleanup result
	 */
	async recalculateStats(): Promise<CleanupResponse> {
		return this.cleanup({ recalculateStats: true });
	}

	/**
	 * Perform full cleanup (admin only)
	 * Deletes old messages, spam, and recalculates stats
	 * @param olderThanDays Delete messages older than this many days
	 * @returns Cleanup result
	 */
	async fullCleanup(olderThanDays = 30): Promise<CleanupResponse> {
		return this.cleanup({
			deleteOldMessages: olderThanDays,
			deleteSpam: true,
			recalculateStats: true
		});
	}
}

// ============================================================================
// Singleton Instance
// ============================================================================

let systemApiInstance: SystemApi | null = null;

/**
 * Get the singleton System API instance
 */
export function getSystemApi(): SystemApi {
	if (!systemApiInstance) {
		systemApiInstance = new SystemApi();
	}
	return systemApiInstance;
}

/**
 * Create a new System API instance with custom client
 */
export function createSystemApi(client: ApiClient): SystemApi {
	return new SystemApi(client);
}
