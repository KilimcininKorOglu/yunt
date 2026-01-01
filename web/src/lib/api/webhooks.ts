/**
 * Webhooks API Client
 * Handles webhook CRUD operations, testing, and delivery history
 */

import type { ApiClient } from './client';
import { getApiClient, buildQueryParams } from './client';
import type {
	ID,
	Webhook,
	WebhookCreateInput,
	WebhookUpdateInput,
	WebhookDelivery,
	WebhookDeliveryStats,
	WebhookListFilter,
	ListOptions,
	PaginatedData
} from './types';

// ============================================================================
// Webhooks API Class
// ============================================================================

export class WebhooksApi {
	private readonly client: ApiClient;

	constructor(client?: ApiClient) {
		this.client = client ?? getApiClient();
	}

	/**
	 * List all webhooks for the authenticated user
	 * @param filter Filter and pagination options
	 * @returns Paginated list of webhooks
	 */
	async list(filter?: WebhookListFilter): Promise<PaginatedData<Webhook>> {
		const params = buildQueryParams({
			page: filter?.page,
			pageSize: filter?.pageSize,
			status: filter?.status,
			event: filter?.event,
			search: filter?.search
		});

		return this.client.get<PaginatedData<Webhook>>('/api/v1/webhooks', { params });
	}

	/**
	 * Get a webhook by ID
	 * @param id Webhook ID
	 * @returns Webhook details
	 */
	async get(id: ID): Promise<Webhook> {
		return this.client.get<Webhook>(`/api/v1/webhooks/${id}`);
	}

	/**
	 * Create a new webhook
	 * @param input Webhook creation data
	 * @returns Created webhook
	 */
	async create(input: WebhookCreateInput): Promise<Webhook> {
		return this.client.post<Webhook>('/api/v1/webhooks', input);
	}

	/**
	 * Update a webhook
	 * @param id Webhook ID
	 * @param input Update data
	 * @returns Updated webhook
	 */
	async update(id: ID, input: WebhookUpdateInput): Promise<Webhook> {
		return this.client.put<Webhook>(`/api/v1/webhooks/${id}`, input);
	}

	/**
	 * Partially update a webhook
	 * @param id Webhook ID
	 * @param input Partial update data
	 * @returns Updated webhook
	 */
	async patch(id: ID, input: Partial<WebhookUpdateInput>): Promise<Webhook> {
		return this.client.patch<Webhook>(`/api/v1/webhooks/${id}`, input);
	}

	/**
	 * Delete a webhook
	 * @param id Webhook ID
	 */
	async delete(id: ID): Promise<void> {
		await this.client.delete<void>(`/api/v1/webhooks/${id}`);
	}

	/**
	 * Test a webhook by sending a test payload
	 * @param id Webhook ID
	 * @returns Test delivery result
	 */
	async test(id: ID): Promise<WebhookDelivery> {
		return this.client.post<WebhookDelivery>(`/api/v1/webhooks/${id}/test`);
	}

	/**
	 * Activate a webhook
	 * @param id Webhook ID
	 * @returns Updated webhook
	 */
	async activate(id: ID): Promise<Webhook> {
		return this.client.post<Webhook>(`/api/v1/webhooks/${id}/activate`);
	}

	/**
	 * Deactivate a webhook
	 * @param id Webhook ID
	 * @returns Updated webhook
	 */
	async deactivate(id: ID): Promise<Webhook> {
		return this.client.post<Webhook>(`/api/v1/webhooks/${id}/deactivate`);
	}

	/**
	 * List delivery history for a webhook
	 * @param id Webhook ID
	 * @param options Pagination options
	 * @returns Paginated delivery history
	 */
	async listDeliveries(id: ID, options?: ListOptions): Promise<PaginatedData<WebhookDelivery>> {
		const params = buildQueryParams({
			page: options?.page,
			pageSize: options?.pageSize
		});

		return this.client.get<PaginatedData<WebhookDelivery>>(
			`/api/v1/webhooks/${id}/deliveries`,
			{ params }
		);
	}

	/**
	 * Get delivery statistics for a webhook
	 * @param id Webhook ID
	 * @returns Delivery statistics
	 */
	async getDeliveryStats(id: ID): Promise<WebhookDeliveryStats> {
		return this.client.get<WebhookDeliveryStats>(`/api/v1/webhooks/${id}/stats`);
	}
}

// ============================================================================
// Singleton Instance
// ============================================================================

let webhooksApiInstance: WebhooksApi | null = null;

/**
 * Get the singleton Webhooks API instance
 */
export function getWebhooksApi(): WebhooksApi {
	if (!webhooksApiInstance) {
		webhooksApiInstance = new WebhooksApi();
	}
	return webhooksApiInstance;
}

/**
 * Create a new Webhooks API instance with custom client
 */
export function createWebhooksApi(client: ApiClient): WebhooksApi {
	return new WebhooksApi(client);
}
