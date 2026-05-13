/**
 * API Type Definitions for Yunt Mail Server
 * These types mirror the Go domain models for type-safe API interactions.
 */

// ============================================================================
// Base Types
// ============================================================================

/** Unique identifier type (string-based) */
export type ID = string;

/** Email address with optional display name */
export interface EmailAddress {
	name?: string;
	address: string;
}

/** ISO8601 timestamp string */
export type Timestamp = string;

// ============================================================================
// User Types
// ============================================================================

/** User roles in the system */
export type UserRole = 'admin' | 'user' | 'viewer';

/** User account status */
export type UserStatus = 'active' | 'inactive' | 'pending';

/** User information (public view) */
export interface UserInfo {
	id: ID;
	username: string;
	email: string;
	displayName?: string;
	role: UserRole;
}

/** Full user profile */
export interface UserProfile {
	id: ID;
	username: string;
	email: string;
	displayName?: string;
	role: UserRole;
	status: UserStatus;
	avatarUrl?: string;
	lastLoginAt?: Timestamp;
	createdAt: Timestamp;
	updatedAt: Timestamp;
}

/** Input for creating a new user */
export interface UserCreateInput {
	username: string;
	email: string;
	password: string;
	displayName?: string;
	role?: UserRole;
}

/** Input for updating a user */
export interface UserUpdateInput {
	displayName?: string;
	email?: string;
	role?: UserRole;
	status?: UserStatus;
	avatarUrl?: string;
}

/** Input for changing password */
export interface ChangePasswordInput {
	currentPassword: string;
	newPassword: string;
}

/** Input for admin password update */
export interface PasswordUpdateInput {
	currentPassword?: string;
	newPassword: string;
}

/** User statistics */
export interface UserStats {
	total: number;
	active: number;
	inactive: number;
	pending: number;
	byRole: Record<UserRole, number>;
}

// ============================================================================
// Authentication Types
// ============================================================================

/** Login credentials */
export interface LoginInput {
	username: string;
	password: string;
}

/** Refresh token input */
export interface RefreshTokenInput {
	refreshToken: string;
}

/** JWT token pair */
export interface TokenPair {
	accessToken: string;
	refreshToken: string;
	accessTokenExpiresAt: Timestamp;
	refreshTokenExpiresAt: Timestamp;
	tokenType: string;
}

/** Authentication response */
export interface AuthResponse {
	user: UserInfo;
	tokens: TokenPair;
}

// ============================================================================
// Mailbox Types
// ============================================================================

/** Mailbox entity */
export interface Mailbox {
	id: ID;
	userId: ID;
	name: string;
	address: string;
	description?: string;
	isCatchAll: boolean;
	isDefault: boolean;
	messageCount: number;
	unreadCount: number;
	totalSize: number;
	retentionDays: number;
	createdAt: Timestamp;
	updatedAt: Timestamp;
}

/** Input for creating a mailbox */
export interface MailboxCreateInput {
	name: string;
	address: string;
	description?: string;
	isCatchAll?: boolean;
	isDefault?: boolean;
	retentionDays?: number;
}

/** Input for updating a mailbox */
export interface MailboxUpdateInput {
	name?: string;
	description?: string;
	isDefault?: boolean;
	retentionDays?: number;
}

/** Mailbox statistics */
export interface MailboxStats {
	totalMessages: number;
	unreadMessages: number;
	totalSize: number;
	oldestMessage?: Timestamp;
	newestMessage?: Timestamp;
}

// ============================================================================
// Message Types
// ============================================================================

/** Message read/unread status */
export type MessageStatus = 'read' | 'unread';

/** Content type constants */
export type ContentType = 'text/plain' | 'text/html' | 'multipart/mixed';

/** Full message entity */
export interface Message {
	id: ID;
	mailboxId: ID;
	messageId?: string;
	from: EmailAddress;
	to: EmailAddress[];
	cc?: EmailAddress[];
	bcc?: EmailAddress[];
	replyTo?: EmailAddress;
	subject: string;
	textBody?: string;
	htmlBody?: string;
	headers?: Record<string, string>;
	contentType: ContentType;
	size: number;
	attachmentCount: number;
	status: MessageStatus;
	isStarred: boolean;
	isSpam: boolean;
	inReplyTo?: string;
	references?: string[];
	receivedAt: Timestamp;
	sentAt?: Timestamp;
	createdAt: Timestamp;
	updatedAt: Timestamp;
}

/** Lightweight message summary for listings */
export interface MessageSummary {
	id: ID;
	mailboxId: ID;
	from: EmailAddress;
	subject: string;
	preview: string;
	status: MessageStatus;
	isStarred: boolean;
	hasAttachments: boolean;
	receivedAt: Timestamp;
}

/** Message filter options */
export interface MessageFilter {
	mailboxId?: ID;
	status?: MessageStatus;
	isStarred?: boolean;
	isSpam?: boolean;
	hasAttachments?: boolean;
	from?: string;
	to?: string;
	subject?: string;
	receivedAfter?: Timestamp;
	receivedBefore?: Timestamp;
}

/** Input for moving a message */
export interface MoveMessageInput {
	targetMailboxId: ID;
}

/** Input for bulk operations with IDs */
export interface BulkIdsInput {
	ids: ID[];
}

/** Input for bulk move operation */
export interface BulkMoveInput {
	ids: ID[];
	targetMailboxId: ID;
}

/** Result of bulk operations */
export interface BulkOperationResponse {
	succeeded: number;
	failed: number;
	errors?: Record<string, string>;
}

// ============================================================================
// Attachment Types
// ============================================================================

/** Attachment disposition */
export type AttachmentDisposition = 'attachment' | 'inline';

/** Full attachment entity */
export interface Attachment {
	id: ID;
	messageId: ID;
	filename: string;
	contentType: string;
	size: number;
	contentId?: string;
	disposition: AttachmentDisposition;
	checksum?: string;
	isInline: boolean;
	createdAt: Timestamp;
}

/** Lightweight attachment summary */
export interface AttachmentSummary {
	id: ID;
	filename: string;
	contentType: string;
	size: number;
	sizeFormatted: string;
	isInline: boolean;
}

/** Attachment filter options */
export interface AttachmentFilter {
	messageId?: ID;
	isInline?: boolean;
	contentType?: string;
}

// ============================================================================
// Webhook Types
// ============================================================================

/** Webhook event types */
export type WebhookEvent =
	| 'message.received'
	| 'message.deleted'
	| 'mailbox.created'
	| 'mailbox.deleted';

/** Webhook status */
export type WebhookStatus = 'active' | 'inactive' | 'failed';

/** Webhook entity */
export interface Webhook {
	id: ID;
	userId: ID;
	name: string;
	url: string;
	events: WebhookEvent[];
	status: WebhookStatus;
	headers?: Record<string, string>;
	retryCount: number;
	maxRetries: number;
	timeoutSeconds: number;
	lastTriggeredAt?: Timestamp;
	lastSuccessAt?: Timestamp;
	lastFailureAt?: Timestamp;
	lastError?: string;
	successCount: number;
	failureCount: number;
	createdAt: Timestamp;
	updatedAt: Timestamp;
}

/** Input for creating a webhook */
export interface WebhookCreateInput {
	name: string;
	url: string;
	secret?: string;
	events: WebhookEvent[];
	headers?: Record<string, string>;
	maxRetries?: number;
	timeoutSeconds?: number;
}

/** Input for updating a webhook */
export interface WebhookUpdateInput {
	name?: string;
	url?: string;
	secret?: string;
	events?: WebhookEvent[];
	status?: WebhookStatus;
	headers?: Record<string, string>;
	maxRetries?: number;
	timeoutSeconds?: number;
}

/** Webhook delivery record */
export interface WebhookDelivery {
	id: ID;
	webhookId: ID;
	event: WebhookEvent;
	payload: string;
	statusCode: number;
	response?: string;
	error?: string;
	success: boolean;
	duration: number;
	attemptNumber: number;
	createdAt: Timestamp;
}

/** Webhook delivery statistics */
export interface WebhookDeliveryStats {
	totalDeliveries: number;
	successCount: number;
	failureCount: number;
	averageDuration: number;
	lastDeliveryAt?: Timestamp;
}

// ============================================================================
// Search Types
// ============================================================================

/** Advanced search input */
export interface AdvancedSearchInput {
	q?: string;
	mailboxId?: ID;
	from?: string;
	to?: string;
	subject?: string;
	status?: MessageStatus;
	isStarred?: boolean;
	isSpam?: boolean;
	hasAttachments?: boolean;
	receivedAfter?: Timestamp;
	receivedBefore?: Timestamp;
	minSize?: number;
	maxSize?: number;
}

// ============================================================================
// System Types
// ============================================================================

/** Version information */
export interface VersionInfo {
	version: string;
	goVersion: string;
	os: string;
	arch: string;
}

/** System statistics */
export interface SystemStats {
	users: {
		total: number;
		active: number;
		inactive: number;
		pending: number;
	};
	mailboxes: {
		total: number;
		totalSize: number;
	};
	messages: {
		total: number;
		unread: number;
		totalSize: number;
		todayCount: number;
		weekCount: number;
		relayedTotal: number;
		dailyCounts: { date: string; count: number }[];
		hourlyCounts: { hour: string; count: number }[];
	};
	uptime: number;
	timestamp: Timestamp;
}

/** Runtime information */
export interface RuntimeInfo {
	goVersion: string;
	numCpu: number;
	numGoroutine: number;
	memoryUsage: number;
	heapAlloc: number;
	heapSys: number;
	gcPauseTotal: number;
}

/** System information (admin only) */
export interface SystemInfo {
	version: string;
	uptime: number;
	startTime: Timestamp;
	runtime: RuntimeInfo;
	config: SystemConfigInfo;
	stats?: SystemStats;
}

/** System configuration info (non-sensitive) */
export interface SystemConfigInfo {
	server: {
		name: string;
		domain: string;
		gracefulTimeout: string;
	};
	smtp: {
		enabled: boolean;
		host: string;
		port: number;
		tlsEnabled: boolean;
		maxMessageSize: number;
		maxRecipients: number;
		authRequired: boolean;
		allowRelay: boolean;
	};
	imap: {
		enabled: boolean;
		host: string;
		port: number;
		tlsEnabled: boolean;
	};
	api: {
		enabled: boolean;
		host: string;
		port: number;
		tlsEnabled: boolean;
		enableSwagger: boolean;
		corsOrigins: string[];
		rateLimit: number;
	};
	database: {
		driver: string;
		host: string;
		port: number;
		name: string;
		maxOpenConns: number;
		maxIdleConns: number;
		autoMigrate: boolean;
	};
	storage: {
		type: string;
		maxMailboxSize: number;
		retentionDays: number;
	};
}

/** Delete all messages response */
export interface DeleteAllMessagesResponse {
	deleted: number;
	message: string;
}

/** Cleanup request */
export interface CleanupRequest {
	deleteOldMessages?: number;
	deleteSpam?: boolean;
	recalculateStats?: boolean;
}

/** Cleanup response */
export interface CleanupResponse {
	messagesDeleted: number;
	spamDeleted: number;
	statsRecalculated: number;
	message: string;
}

// ============================================================================
// API Response Types
// ============================================================================

/** Standard API response wrapper */
export interface ApiResponse<T = unknown> {
	success: boolean;
	data?: T;
	error?: ApiError;
	meta?: ResponseMeta;
}

/** API error details */
export interface ApiError {
	code: string;
	message: string;
	details?: unknown;
}

/** Response metadata */
export interface ResponseMeta {
	timestamp: Timestamp;
	requestId?: string;
}

/** Pagination information */
export interface PaginationInfo {
	page: number;
	pageSize: number;
	totalItems: number;
	totalPages: number;
	hasNext: boolean;
	hasPrev: boolean;
}

/** Paginated response data */
export interface PaginatedData<T> {
	items: T[];
	pagination: PaginationInfo;
}

/** Paginated API response */
export type PaginatedResponse<T> = ApiResponse<PaginatedData<T>>;

// ============================================================================
// Send / Draft / Signature
// ============================================================================

/** Input for sending a message */
export interface SendMessageInput {
	fromMailboxId: string;
	to: string[];
	cc?: string[];
	bcc?: string[];
	subject: string;
	textBody: string;
	htmlBody?: string;
}

/** Result of sending a message */
export interface SendMessageResult {
	success: boolean;
	messageId: string;
	recipients: string[];
	failedRecipients?: string[];
}

/** Input for saving a draft */
export interface DraftInput {
	fromMailboxId?: string;
	to?: string[];
	cc?: string[];
	bcc?: string[];
	subject?: string;
	textBody?: string;
	htmlBody?: string;
}

/** Result of saving a draft */
export interface DraftSaveResult {
	id: string;
	mailboxId: string;
}

/** Attachment summary returned from upload */
export interface AttachmentUploadResult {
	id: string;
	filename: string;
	contentType: string;
	size: number;
}

// ============================================================================
// List/Filter Options
// ============================================================================

/** Sort order */
export type SortOrder = 'asc' | 'desc';

/** Common list options */
export interface ListOptions {
	page?: number;
	pageSize?: number;
	perPage?: number;
	sort?: string;
	order?: SortOrder;
}

/** User list filter */
export interface UserListFilter extends ListOptions {
	status?: UserStatus;
	role?: UserRole;
	search?: string;
}

/** Mailbox list filter */
export interface MailboxListFilter extends ListOptions {
	sort?:
		| 'name'
		| 'address'
		| 'messageCount'
		| 'unreadCount'
		| 'totalSize'
		| 'createdAt'
		| 'updatedAt';
}

/** Message list filter */
export interface MessageListFilter extends ListOptions, MessageFilter {
	sort?: 'receivedAt' | 'sentAt' | 'subject' | 'from' | 'size' | 'status' | 'createdAt';
}

/** Webhook list filter */
export interface WebhookListFilter extends ListOptions {
	status?: WebhookStatus;
	event?: WebhookEvent;
	search?: string;
}

/** Attachment list filter */
export interface AttachmentListFilter extends ListOptions, AttachmentFilter {
	sort?: 'filename' | 'size' | 'contentType' | 'createdAt';
}

// ============================================================================
// Error Codes
// ============================================================================

/** API error codes */
export const ErrorCodes = {
	BAD_REQUEST: 'BAD_REQUEST',
	UNAUTHORIZED: 'UNAUTHORIZED',
	FORBIDDEN: 'FORBIDDEN',
	NOT_FOUND: 'NOT_FOUND',
	CONFLICT: 'CONFLICT',
	VALIDATION_FAILED: 'VALIDATION_FAILED',
	INTERNAL_SERVER_ERROR: 'INTERNAL_SERVER_ERROR',
	SERVICE_UNAVAILABLE: 'SERVICE_UNAVAILABLE'
} as const;

export type ErrorCode = (typeof ErrorCodes)[keyof typeof ErrorCodes];

// ============================================================================
// Validation Error Types
// ============================================================================

/** Field-level validation error */
export interface ValidationError {
	field: string;
	message: string;
}

/** Validation errors collection */
export type ValidationErrors = ValidationError[];
