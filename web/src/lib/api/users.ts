/**
 * Users API Client
 * Handles user management operations
 */

import type { ApiClient } from './client';
import { getApiClient, buildQueryParams } from './client';
import type {
	ID,
	UserInfo,
	UserProfile,
	UserCreateInput,
	UserUpdateInput,
	PasswordUpdateInput,
	ChangePasswordInput,
	UserStats,
	UserListFilter,
	UserRole,
	UserStatus,
	PaginatedData
} from './types';

// ============================================================================
// Users API Class
// ============================================================================

export class UsersApi {
	private readonly client: ApiClient;

	constructor(client?: ApiClient) {
		this.client = client ?? getApiClient();
	}

	/**
	 * List all users (admin only)
	 * @param filter Filter and pagination options
	 * @returns Paginated list of user profiles
	 */
	async list(filter?: UserListFilter): Promise<PaginatedData<UserProfile>> {
		const params = buildQueryParams({
			page: filter?.page,
			pageSize: filter?.pageSize,
			status: filter?.status,
			role: filter?.role,
			search: filter?.search
		});

		const raw = await this.client.get<Record<string, unknown>>('/api/v1/users', { params });
		const users = (raw as Record<string, unknown>).users as UserProfile[] ?? [];
		const total = (raw as Record<string, unknown>).total as number ?? 0;
		const page = (raw as Record<string, unknown>).page as number ?? 1;
		const pageSize2 = (raw as Record<string, unknown>).pageSize as number ?? 20;
		const totalPages = (raw as Record<string, unknown>).totalPages as number ?? 1;
		return {
			items: users,
			pagination: {
				page,
				pageSize: pageSize2,
				totalItems: total,
				totalPages,
				hasNext: page < totalPages,
				hasPrev: page > 1
			}
		};
	}

	/**
	 * Get a user by ID
	 * @param id User ID
	 * @returns User profile
	 */
	async get(id: ID): Promise<UserProfile> {
		return this.client.get<UserProfile>(`/api/v1/users/${id}`);
	}

	/**
	 * Create a new user (admin only)
	 * @param input User creation data
	 * @returns Created user info
	 */
	async create(input: UserCreateInput): Promise<UserInfo> {
		return this.client.post<UserInfo>('/api/v1/users', input);
	}

	/**
	 * Update a user
	 * @param id User ID
	 * @param input Update data
	 * @returns Updated user profile
	 */
	async update(id: ID, input: UserUpdateInput): Promise<UserProfile> {
		return this.client.put<UserProfile>(`/api/v1/users/${id}`, input);
	}

	/**
	 * Delete a user (admin only)
	 * @param id User ID
	 */
	async delete(id: ID): Promise<void> {
		await this.client.delete<void>(`/api/v1/users/${id}`);
	}

	/**
	 * Update a user's password
	 * @param id User ID
	 * @param input Password update data
	 */
	async updatePassword(id: ID, input: PasswordUpdateInput): Promise<void> {
		await this.client.put<void>(`/api/v1/users/${id}/password`, input);
	}

	/**
	 * Update a user's role (admin only)
	 * @param id User ID
	 * @param role New role
	 */
	async updateRole(id: ID, role: UserRole): Promise<void> {
		await this.client.put<void>(`/api/v1/users/${id}/role`, { role });
	}

	/**
	 * Update a user's status (admin only)
	 * @param id User ID
	 * @param status New status
	 */
	async updateStatus(id: ID, status: UserStatus): Promise<void> {
		await this.client.put<void>(`/api/v1/users/${id}/status`, { status });
	}

	/**
	 * Get user statistics (admin only)
	 * @returns User statistics
	 */
	async getStats(): Promise<UserStats> {
		return this.client.get<UserStats>('/api/v1/users/stats');
	}

	/**
	 * Get current user's profile
	 * @returns Current user profile
	 */
	async getMyProfile(): Promise<UserProfile> {
		return this.client.get<UserProfile>('/api/v1/users/me/profile');
	}

	/**
	 * Update current user's profile
	 * @param input Profile update data
	 * @returns Updated profile
	 */
	async updateMyProfile(input: UserUpdateInput): Promise<UserProfile> {
		return this.client.put<UserProfile>('/api/v1/users/me/profile', input);
	}

	/**
	 * Change current user's password
	 * @param input Password change data
	 */
	async changeMyPassword(input: ChangePasswordInput): Promise<void> {
		await this.client.put<void>('/api/v1/users/me/password', input);
	}
}

// ============================================================================
// Singleton Instance
// ============================================================================

let usersApiInstance: UsersApi | null = null;

/**
 * Get the singleton Users API instance
 */
export function getUsersApi(): UsersApi {
	if (!usersApiInstance) {
		usersApiInstance = new UsersApi();
	}
	return usersApiInstance;
}

/**
 * Create a new Users API instance with custom client
 */
export function createUsersApi(client: ApiClient): UsersApi {
	return new UsersApi(client);
}
