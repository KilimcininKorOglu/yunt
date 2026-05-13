/**
 * Yunt API Client Library
 *
 * A TypeScript client library for the Yunt Mail Server REST API.
 * Provides type-safe access to all API endpoints with automatic
 * authentication handling and token refresh.
 *
 * @example Basic usage
 * ```typescript
 * import { initApiClient, getAuthApi, getMailboxesApi } from '$lib/api';
 *
 * // Initialize the client
 * initApiClient({ baseUrl: 'http://localhost:8025' });
 *
 * // Login
 * const auth = getAuthApi();
 * await auth.login({ username: 'admin', password: 'secret' });
 *
 * // Use other APIs
 * const mailboxes = getMailboxesApi();
 * const list = await mailboxes.list();
 * ```
 */

// ============================================================================
// Core Client
// ============================================================================

export {
	ApiClient,
	ApiClientError,
	NetworkError,
	TimeoutError,
	MemoryTokenStorage,
	CookieTokenStorage,
	createApiClient,
	getApiClient,
	initApiClient,
	buildQueryParams,
	type HttpMethod,
	type RequestOptions,
	type ApiClientConfig,
	type TokenStorage
} from './client';

// ============================================================================
// API Modules
// ============================================================================

export { AuthApi, getAuthApi, createAuthApi } from './auth';

export { UsersApi, getUsersApi, createUsersApi } from './users';

export { MailboxesApi, getMailboxesApi, createMailboxesApi } from './mailboxes';

export { MessagesApi, getMessagesApi, createMessagesApi } from './messages';

export { AttachmentsApi, getAttachmentsApi, createAttachmentsApi } from './attachments';

export { SearchApi, getSearchApi, createSearchApi } from './search';

export { WebhooksApi, getWebhooksApi, createWebhooksApi } from './webhooks';

export { SystemApi, getSystemApi, createSystemApi } from './system';

// ============================================================================
// Types
// ============================================================================

export type {
	// Base types
	ID,
	EmailAddress,
	Timestamp,

	// User types
	UserRole,
	UserStatus,
	UserInfo,
	UserProfile,
	UserCreateInput,
	UserUpdateInput,
	ChangePasswordInput,
	PasswordUpdateInput,
	UserStats,

	// Auth types
	LoginInput,
	RefreshTokenInput,
	TokenPair,
	AuthResponse,

	// Mailbox types
	Mailbox,
	MailboxCreateInput,
	MailboxUpdateInput,
	MailboxStats,

	// Message types
	MessageStatus,
	ContentType,
	Message,
	MessageSummary,
	MessageFilter,
	MoveMessageInput,
	BulkIdsInput,
	BulkMoveInput,
	BulkOperationResponse,

	// Attachment types
	AttachmentDisposition,
	Attachment,
	AttachmentSummary,
	AttachmentFilter,

	// Webhook types
	WebhookEvent,
	WebhookStatus,
	Webhook,
	WebhookCreateInput,
	WebhookUpdateInput,
	WebhookDelivery,
	WebhookDeliveryStats,

	// Search types
	AdvancedSearchInput,

	// System types
	VersionInfo,
	SystemStats,
	RuntimeInfo,
	SystemInfo,
	SystemConfigInfo,
	DeleteAllMessagesResponse,
	CleanupRequest,
	CleanupResponse,

	// API response types
	ApiResponse,
	ApiError,
	ResponseMeta,
	PaginationInfo,
	PaginatedData,
	PaginatedResponse,

	// List/Filter options
	SortOrder,
	ListOptions,
	UserListFilter,
	MailboxListFilter,
	MessageListFilter,
	WebhookListFilter,
	AttachmentListFilter,

	// Validation types
	ValidationError,
	ValidationErrors
} from './types';

export { ErrorCodes, type ErrorCode } from './types';
