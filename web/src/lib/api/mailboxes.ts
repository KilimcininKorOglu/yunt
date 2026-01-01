/**
 * Mailboxes API Client
 * Handles mailbox CRUD operations and statistics
 */

import type { ApiClient } from './client';
import { getApiClient, buildQueryParams } from './client';
import type {
	ID,
	Mailbox,
	MailboxCreateInput,
	MailboxUpdateInput,
	MailboxStats,
	MailboxListFilter,
	PaginatedData
} from './types';

// ============================================================================
// Mailboxes API Class
// ============================================================================

export class MailboxesApi {
	private readonly client: ApiClient;

	constructor(client?: ApiClient) {
		this.client = client ?? getApiClient();
	}

	/**
	 * List all mailboxes for the authenticated user
	 * @param filter Filter and pagination options
	 * @returns Paginated list of mailboxes
	 */
	async list(filter?: MailboxListFilter): Promise<PaginatedData<Mailbox>> {
		const params = buildQueryParams({
			page: filter?.page,
			perPage: filter?.perPage ?? filter?.pageSize,
			sort: filter?.sort,
			order: filter?.order
		});

		return this.client.get<PaginatedData<Mailbox>>('/api/v1/mailboxes', { params });
	}

	/**
	 * Get a mailbox by ID
	 * @param id Mailbox ID
	 * @returns Mailbox details
	 */
	async get(id: ID): Promise<Mailbox> {
		return this.client.get<Mailbox>(`/api/v1/mailboxes/${id}`);
	}

	/**
	 * Create a new mailbox
	 * @param input Mailbox creation data
	 * @returns Created mailbox
	 */
	async create(input: MailboxCreateInput): Promise<Mailbox> {
		return this.client.post<Mailbox>('/api/v1/mailboxes', input);
	}

	/**
	 * Update a mailbox
	 * @param id Mailbox ID
	 * @param input Update data
	 * @returns Updated mailbox
	 */
	async update(id: ID, input: MailboxUpdateInput): Promise<Mailbox> {
		return this.client.put<Mailbox>(`/api/v1/mailboxes/${id}`, input);
	}

	/**
	 * Delete a mailbox
	 * System mailboxes cannot be deleted
	 * @param id Mailbox ID
	 */
	async delete(id: ID): Promise<void> {
		await this.client.delete<void>(`/api/v1/mailboxes/${id}`);
	}

	/**
	 * Get statistics for a specific mailbox
	 * @param id Mailbox ID
	 * @returns Mailbox statistics
	 */
	async getStats(id: ID): Promise<MailboxStats> {
		return this.client.get<MailboxStats>(`/api/v1/mailboxes/${id}/stats`);
	}

	/**
	 * Get aggregated statistics for all user's mailboxes
	 * @returns Aggregated mailbox statistics
	 */
	async getUserStats(): Promise<MailboxStats> {
		return this.client.get<MailboxStats>('/api/v1/mailboxes/stats');
	}

	/**
	 * Set a mailbox as the default
	 * @param id Mailbox ID
	 */
	async setDefault(id: ID): Promise<void> {
		await this.client.post<void>(`/api/v1/mailboxes/${id}/default`);
	}
}

// ============================================================================
// Singleton Instance
// ============================================================================

let mailboxesApiInstance: MailboxesApi | null = null;

/**
 * Get the singleton Mailboxes API instance
 */
export function getMailboxesApi(): MailboxesApi {
	if (!mailboxesApiInstance) {
		mailboxesApiInstance = new MailboxesApi();
	}
	return mailboxesApiInstance;
}

/**
 * Create a new Mailboxes API instance with custom client
 */
export function createMailboxesApi(client: ApiClient): MailboxesApi {
	return new MailboxesApi(client);
}
