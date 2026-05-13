/**
 * Messages API Client
 * Handles message listing, retrieval, and operations
 */

import type { ApiClient } from './client';
import { getApiClient, buildQueryParams } from './client';
import type {
	ID,
	Message,
	MessageListFilter,
	MoveMessageInput,
	BulkIdsInput,
	BulkMoveInput,
	BulkOperationResponse,
	AttachmentSummary,
	Attachment,
	PaginatedData,
	SendMessageInput,
	SendMessageResult,
	DraftInput,
	DraftSaveResult,
	AttachmentUploadResult
} from './types';

// ============================================================================
// Messages API Class
// ============================================================================

export class MessagesApi {
	private readonly client: ApiClient;

	constructor(client?: ApiClient) {
		this.client = client ?? getApiClient();
	}

	/**
	 * List messages with optional filtering
	 * @param filter Filter and pagination options
	 * @returns Paginated list of messages
	 */
	async list(filter?: MessageListFilter): Promise<PaginatedData<Message>> {
		const params = buildQueryParams({
			mailboxId: filter?.mailboxId,
			status: filter?.status,
			isStarred: filter?.isStarred,
			isSpam: filter?.isSpam,
			hasAttachments: filter?.hasAttachments,
			from: filter?.from,
			to: filter?.to,
			subject: filter?.subject,
			receivedAfter: filter?.receivedAfter,
			receivedBefore: filter?.receivedBefore,
			page: filter?.page,
			perPage: filter?.perPage ?? filter?.pageSize,
			sort: filter?.sort,
			order: filter?.order
		});

		return this.client.get<PaginatedData<Message>>('/api/v1/messages', { params });
	}

	async listWithEtag(
		filter?: MessageListFilter,
		etag?: string | null
	): Promise<{ data: PaginatedData<Message> | null; etag: string | null; notModified: boolean }> {
		const params = buildQueryParams({
			mailboxId: filter?.mailboxId,
			status: filter?.status,
			isStarred: filter?.isStarred,
			isSpam: filter?.isSpam,
			hasAttachments: filter?.hasAttachments,
			from: filter?.from,
			to: filter?.to,
			subject: filter?.subject,
			receivedAfter: filter?.receivedAfter,
			receivedBefore: filter?.receivedBefore,
			page: filter?.page,
			perPage: filter?.perPage ?? filter?.pageSize,
			sort: filter?.sort,
			order: filter?.order
		});

		const headers: Record<string, string> = {};
		if (etag) headers['If-None-Match'] = etag;

		const response = await this.client.rawRequest('GET', '/api/v1/messages', undefined, {
			params,
			headers
		});

		if (response.status === 304) {
			return { data: null, etag: etag ?? null, notModified: true };
		}

		const json = await response.json();
		const newEtag = response.headers.get('ETag');
		return { data: json.data, etag: newEtag, notModified: false };
	}

	/**
	 * Search messages
	 * @param query Search query
	 * @param filter Additional filter options
	 * @returns Paginated search results
	 */
	async search(query: string, filter?: MessageListFilter): Promise<PaginatedData<Message>> {
		const params = buildQueryParams({
			q: query,
			mailboxId: filter?.mailboxId,
			page: filter?.page,
			perPage: filter?.perPage ?? filter?.pageSize,
			sort: filter?.sort,
			order: filter?.order
		});

		return this.client.get<PaginatedData<Message>>('/api/v1/messages/search', { params });
	}

	/**
	 * Get a message by ID
	 * @param id Message ID
	 * @returns Full message details
	 */
	async get(id: ID): Promise<Message> {
		return this.client.get<Message>(`/api/v1/messages/${id}`);
	}

	/**
	 * Get message HTML body
	 * @param id Message ID
	 * @returns HTML content
	 */
	async getHtml(id: ID): Promise<string> {
		const response = await this.client.rawRequest('GET', `/api/v1/messages/${id}/html`);
		if (!response.ok) {
			throw new Error('Failed to fetch HTML body');
		}
		return response.text();
	}

	/**
	 * Get message text body
	 * @param id Message ID
	 * @returns Plain text content
	 */
	async getText(id: ID): Promise<string> {
		const response = await this.client.rawRequest('GET', `/api/v1/messages/${id}/text`);
		if (!response.ok) {
			throw new Error('Failed to fetch text body');
		}
		return response.text();
	}

	/**
	 * Get raw message (EML format)
	 * @param id Message ID
	 * @returns Raw message blob
	 */
	async getRaw(id: ID): Promise<Blob> {
		const response = await this.client.rawRequest('GET', `/api/v1/messages/${id}/raw`);
		if (!response.ok) {
			throw new Error('Failed to fetch raw message');
		}
		return response.blob();
	}

	/**
	 * Download raw message as EML file
	 * @param id Message ID
	 * @param filename Optional filename
	 */
	async downloadRaw(id: ID, filename = 'message.eml'): Promise<void> {
		const blob = await this.getRaw(id);
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = filename;
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
		URL.revokeObjectURL(url);
	}

	/**
	 * Delete a message
	 * @param id Message ID
	 */
	async delete(id: ID): Promise<void> {
		await this.client.delete<void>(`/api/v1/messages/${id}`);
	}

	/**
	 * Mark a message as read
	 * @param id Message ID
	 */
	async markAsRead(id: ID): Promise<void> {
		await this.client.put<void>(`/api/v1/messages/${id}/read`);
	}

	/**
	 * Mark a message as unread
	 * @param id Message ID
	 */
	async markAsUnread(id: ID): Promise<void> {
		await this.client.put<void>(`/api/v1/messages/${id}/unread`);
	}

	/**
	 * Star a message
	 * @param id Message ID
	 */
	async star(id: ID): Promise<void> {
		await this.client.put<void>(`/api/v1/messages/${id}/star`);
	}

	/**
	 * Unstar a message
	 * @param id Message ID
	 */
	async unstar(id: ID): Promise<void> {
		await this.client.put<void>(`/api/v1/messages/${id}/unstar`);
	}

	/**
	 * Mark a message as spam
	 * @param id Message ID
	 */
	async markAsSpam(id: ID): Promise<void> {
		await this.client.put<void>(`/api/v1/messages/${id}/spam`);
	}

	/**
	 * Mark a message as not spam
	 * @param id Message ID
	 */
	async markAsNotSpam(id: ID): Promise<void> {
		await this.client.put<void>(`/api/v1/messages/${id}/not-spam`);
	}

	/**
	 * Move a message to another mailbox
	 * @param id Message ID
	 * @param targetMailboxId Target mailbox ID
	 */
	async move(id: ID, targetMailboxId: ID): Promise<void> {
		const input: MoveMessageInput = { targetMailboxId };
		await this.client.put<void>(`/api/v1/messages/${id}/move`, input);
	}

	// ========================================================================
	// Attachment Operations
	// ========================================================================

	/**
	 * List attachments for a message
	 * @param messageId Message ID
	 * @returns List of attachment summaries
	 */
	async listAttachments(messageId: ID): Promise<AttachmentSummary[]> {
		return this.client.get<AttachmentSummary[]>(`/api/v1/messages/${messageId}/attachments`);
	}

	/**
	 * Get attachment metadata
	 * @param messageId Message ID
	 * @param attachmentId Attachment ID
	 * @returns Attachment metadata
	 */
	async getAttachment(messageId: ID, attachmentId: ID): Promise<Attachment> {
		return this.client.get<Attachment>(
			`/api/v1/messages/${messageId}/attachments/${attachmentId}`
		);
	}

	/**
	 * Download attachment
	 * @param messageId Message ID
	 * @param attachmentId Attachment ID
	 * @returns Attachment blob
	 */
	async downloadAttachment(messageId: ID, attachmentId: ID): Promise<Blob> {
		const response = await this.client.rawRequest(
			'GET',
			`/api/v1/messages/${messageId}/attachments/${attachmentId}/download`
		);
		if (!response.ok) {
			throw new Error('Failed to download attachment');
		}
		return response.blob();
	}

	/**
	 * Download and save attachment to file
	 * @param messageId Message ID
	 * @param attachmentId Attachment ID
	 * @param filename Filename to save as
	 */
	async saveAttachment(messageId: ID, attachmentId: ID, filename: string): Promise<void> {
		const blob = await this.downloadAttachment(messageId, attachmentId);
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = filename;
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
		URL.revokeObjectURL(url);
	}

	// ========================================================================
	// Bulk Operations
	// ========================================================================

	/**
	 * Mark multiple messages as read
	 * @param ids Message IDs
	 * @returns Bulk operation result
	 */
	async bulkMarkAsRead(ids: ID[]): Promise<BulkOperationResponse> {
		const input: BulkIdsInput = { ids };
		return this.client.post<BulkOperationResponse>('/api/v1/messages/bulk/read', input);
	}

	/**
	 * Mark multiple messages as unread
	 * @param ids Message IDs
	 * @returns Bulk operation result
	 */
	async bulkMarkAsUnread(ids: ID[]): Promise<BulkOperationResponse> {
		const input: BulkIdsInput = { ids };
		return this.client.post<BulkOperationResponse>('/api/v1/messages/bulk/unread', input);
	}

	/**
	 * Delete multiple messages
	 * @param ids Message IDs
	 * @returns Bulk operation result
	 */
	async bulkDelete(ids: ID[]): Promise<BulkOperationResponse> {
		const input: BulkIdsInput = { ids };
		return this.client.post<BulkOperationResponse>('/api/v1/messages/bulk/delete', input);
	}

	/**
	 * Move multiple messages to another mailbox
	 * @param ids Message IDs
	 * @param targetMailboxId Target mailbox ID
	 * @returns Bulk operation result
	 */
	async bulkMove(ids: ID[], targetMailboxId: ID): Promise<BulkOperationResponse> {
		const input: BulkMoveInput = { ids, targetMailboxId };
		return this.client.post<BulkOperationResponse>('/api/v1/messages/bulk/move', input);
	}

	/**
	 * Star multiple messages
	 * @param ids Message IDs
	 * @returns Bulk operation result
	 */
	async bulkStar(ids: ID[]): Promise<BulkOperationResponse> {
		const input: BulkIdsInput = { ids };
		return this.client.post<BulkOperationResponse>('/api/v1/messages/bulk/star', input);
	}

	/**
	 * Unstar multiple messages
	 * @param ids Message IDs
	 * @returns Bulk operation result
	 */
	async bulkUnstar(ids: ID[]): Promise<BulkOperationResponse> {
		const input: BulkIdsInput = { ids };
		return this.client.post<BulkOperationResponse>('/api/v1/messages/bulk/unstar', input);
	}

	// ========================================================================
	// Send & Draft Operations
	// ========================================================================

	async send(input: SendMessageInput): Promise<SendMessageResult> {
		return this.client.post<SendMessageResult>('/api/v1/messages/send', input);
	}

	async saveDraft(input: DraftInput): Promise<DraftSaveResult> {
		return this.client.post<DraftSaveResult>('/api/v1/messages/draft', input);
	}

	async updateDraft(id: ID, input: DraftInput): Promise<void> {
		await this.client.put<void>(`/api/v1/messages/draft/${id}`, input);
	}

	async sendDraft(id: ID): Promise<SendMessageResult> {
		return this.client.post<SendMessageResult>(`/api/v1/messages/draft/${id}/send`);
	}

	async uploadDraftAttachment(draftId: ID, file: File): Promise<AttachmentUploadResult> {
		const formData = new FormData();
		formData.append('file', file);
		return this.client.post<AttachmentUploadResult>(
			`/api/v1/messages/draft/${draftId}/attachments`,
			formData
		);
	}

	async deleteDraftAttachment(draftId: ID, attachmentId: ID): Promise<void> {
		await this.client.delete<void>(`/api/v1/messages/draft/${draftId}/attachments/${attachmentId}`);
	}
}

// ============================================================================
// Singleton Instance
// ============================================================================

let messagesApiInstance: MessagesApi | null = null;

/**
 * Get the singleton Messages API instance
 */
export function getMessagesApi(): MessagesApi {
	if (!messagesApiInstance) {
		messagesApiInstance = new MessagesApi();
	}
	return messagesApiInstance;
}

/**
 * Create a new Messages API instance with custom client
 */
export function createMessagesApi(client: ApiClient): MessagesApi {
	return new MessagesApi(client);
}
