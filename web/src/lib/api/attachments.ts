/**
 * Attachments API Client
 * Handles direct attachment operations without message context
 */

import type { ApiClient } from './client';
import { getApiClient, buildQueryParams } from './client';
import type {
	ID,
	Attachment,
	AttachmentSummary,
	AttachmentListFilter,
	PaginatedData
} from './types';

// ============================================================================
// Attachments API Class
// ============================================================================

export class AttachmentsApi {
	private readonly client: ApiClient;

	constructor(client?: ApiClient) {
		this.client = client ?? getApiClient();
	}

	/**
	 * List attachments with optional filtering
	 * @param filter Filter and pagination options
	 * @returns Paginated list of attachment summaries
	 */
	async list(filter?: AttachmentListFilter): Promise<PaginatedData<AttachmentSummary>> {
		const params = buildQueryParams({
			messageId: filter?.messageId,
			isInline: filter?.isInline,
			contentType: filter?.contentType,
			page: filter?.page,
			perPage: filter?.perPage ?? filter?.pageSize,
			sort: filter?.sort,
			order: filter?.order
		});

		return this.client.get<PaginatedData<AttachmentSummary>>('/api/v1/attachments', { params });
	}

	/**
	 * Get attachment metadata by ID
	 * @param id Attachment ID
	 * @returns Attachment metadata
	 */
	async get(id: ID): Promise<Attachment> {
		return this.client.get<Attachment>(`/api/v1/attachments/${id}`);
	}

	/**
	 * Download attachment content
	 * @param id Attachment ID
	 * @param inline Whether to display inline instead of download
	 * @returns Attachment blob
	 */
	async download(id: ID, inline = false): Promise<Blob> {
		const params = inline ? { inline: 'true' } : undefined;
		const response = await this.client.rawRequest(
			'GET',
			`/api/v1/attachments/${id}/download`,
			undefined,
			{ params }
		);
		if (!response.ok) {
			throw new Error('Failed to download attachment');
		}
		return response.blob();
	}

	/**
	 * Download and save attachment to file
	 * @param id Attachment ID
	 * @param filename Filename to save as
	 */
	async saveToFile(id: ID, filename: string): Promise<void> {
		const blob = await this.download(id);
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
	 * Get attachment URL for inline display
	 * @param id Attachment ID
	 * @returns Object URL for the attachment (remember to revoke when done)
	 */
	async getObjectUrl(id: ID): Promise<string> {
		const blob = await this.download(id, true);
		return URL.createObjectURL(blob);
	}

	/**
	 * Get data URL for attachment (base64 encoded)
	 * @param id Attachment ID
	 * @returns Data URL string
	 */
	async getDataUrl(id: ID): Promise<string> {
		const blob = await this.download(id, true);
		return new Promise((resolve, reject) => {
			const reader = new FileReader();
			reader.onloadend = () => resolve(reader.result as string);
			reader.onerror = reject;
			reader.readAsDataURL(blob);
		});
	}
}

// ============================================================================
// Singleton Instance
// ============================================================================

let attachmentsApiInstance: AttachmentsApi | null = null;

/**
 * Get the singleton Attachments API instance
 */
export function getAttachmentsApi(): AttachmentsApi {
	if (!attachmentsApiInstance) {
		attachmentsApiInstance = new AttachmentsApi();
	}
	return attachmentsApiInstance;
}

/**
 * Create a new Attachments API instance with custom client
 */
export function createAttachmentsApi(client: ApiClient): AttachmentsApi {
	return new AttachmentsApi(client);
}
