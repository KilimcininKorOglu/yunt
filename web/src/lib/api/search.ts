/**
 * Search API Client
 * Handles simple and advanced message search operations
 */

import type { ApiClient } from './client';
import { getApiClient, buildQueryParams } from './client';
import type { ID, Message, AdvancedSearchInput, ListOptions, PaginatedData } from './types';

// ============================================================================
// Search API Class
// ============================================================================

export class SearchApi {
	private readonly client: ApiClient;

	constructor(client?: ApiClient) {
		this.client = client ?? getApiClient();
	}

	/**
	 * Simple text search across message subject and body
	 * @param query Search query (min 2 characters)
	 * @param options Additional options
	 * @returns Paginated search results
	 */
	async simple(
		query: string,
		options?: {
			mailboxId?: ID;
			page?: number;
			perPage?: number;
			sort?: string;
			order?: 'asc' | 'desc';
		}
	): Promise<PaginatedData<Message>> {
		const params = buildQueryParams({
			q: query,
			mailboxId: options?.mailboxId,
			page: options?.page,
			perPage: options?.perPage,
			sort: options?.sort,
			order: options?.order
		});

		return this.client.get<PaginatedData<Message>>('/api/v1/search', { params });
	}

	/**
	 * Advanced search with multiple criteria
	 * @param input Advanced search criteria
	 * @param listOptions Pagination and sorting options
	 * @returns Paginated search results
	 */
	async advanced(
		input: AdvancedSearchInput,
		listOptions?: ListOptions
	): Promise<PaginatedData<Message>> {
		const params = buildQueryParams({
			q: input.q,
			mailboxId: input.mailboxId,
			from: input.from,
			to: input.to,
			subject: input.subject,
			status: input.status,
			isStarred: input.isStarred,
			isSpam: input.isSpam,
			hasAttachments: input.hasAttachments,
			receivedAfter: input.receivedAfter,
			receivedBefore: input.receivedBefore,
			minSize: input.minSize,
			maxSize: input.maxSize,
			page: listOptions?.page,
			perPage: listOptions?.perPage ?? listOptions?.pageSize,
			sort: listOptions?.sort,
			order: listOptions?.order
		});

		return this.client.get<PaginatedData<Message>>('/api/v1/search/advanced', { params });
	}

	/**
	 * Search messages by sender
	 * @param from Sender address (partial match)
	 * @param options Additional options
	 * @returns Paginated search results
	 */
	async bySender(
		from: string,
		options?: ListOptions & { mailboxId?: ID }
	): Promise<PaginatedData<Message>> {
		return this.advanced({ from, mailboxId: options?.mailboxId }, options);
	}

	/**
	 * Search messages by recipient
	 * @param to Recipient address (partial match)
	 * @param options Additional options
	 * @returns Paginated search results
	 */
	async byRecipient(
		to: string,
		options?: ListOptions & { mailboxId?: ID }
	): Promise<PaginatedData<Message>> {
		return this.advanced({ to, mailboxId: options?.mailboxId }, options);
	}

	/**
	 * Search messages by subject
	 * @param subject Subject text (partial match)
	 * @param options Additional options
	 * @returns Paginated search results
	 */
	async bySubject(
		subject: string,
		options?: ListOptions & { mailboxId?: ID }
	): Promise<PaginatedData<Message>> {
		return this.advanced({ subject, mailboxId: options?.mailboxId }, options);
	}

	/**
	 * Search messages by date range
	 * @param receivedAfter Start of date range (RFC3339)
	 * @param receivedBefore End of date range (RFC3339)
	 * @param options Additional options
	 * @returns Paginated search results
	 */
	async byDateRange(
		receivedAfter: string,
		receivedBefore: string,
		options?: ListOptions & { mailboxId?: ID }
	): Promise<PaginatedData<Message>> {
		return this.advanced(
			{ receivedAfter, receivedBefore, mailboxId: options?.mailboxId },
			options
		);
	}

	/**
	 * Search messages with attachments
	 * @param options Additional options
	 * @returns Paginated search results
	 */
	async withAttachments(
		options?: ListOptions & { mailboxId?: ID }
	): Promise<PaginatedData<Message>> {
		return this.advanced({ hasAttachments: true, mailboxId: options?.mailboxId }, options);
	}

	/**
	 * Search starred messages
	 * @param options Additional options
	 * @returns Paginated search results
	 */
	async starred(options?: ListOptions & { mailboxId?: ID }): Promise<PaginatedData<Message>> {
		return this.advanced({ isStarred: true, mailboxId: options?.mailboxId }, options);
	}

	/**
	 * Search unread messages
	 * @param options Additional options
	 * @returns Paginated search results
	 */
	async unread(options?: ListOptions & { mailboxId?: ID }): Promise<PaginatedData<Message>> {
		return this.advanced({ status: 'unread', mailboxId: options?.mailboxId }, options);
	}

	/**
	 * Search spam messages
	 * @param options Additional options
	 * @returns Paginated search results
	 */
	async spam(options?: ListOptions & { mailboxId?: ID }): Promise<PaginatedData<Message>> {
		return this.advanced({ isSpam: true, mailboxId: options?.mailboxId }, options);
	}
}

// ============================================================================
// Singleton Instance
// ============================================================================

let searchApiInstance: SearchApi | null = null;

/**
 * Get the singleton Search API instance
 */
export function getSearchApi(): SearchApi {
	if (!searchApiInstance) {
		searchApiInstance = new SearchApi();
	}
	return searchApiInstance;
}

/**
 * Create a new Search API instance with custom client
 */
export function createSearchApi(client: ApiClient): SearchApi {
	return new SearchApi(client);
}
