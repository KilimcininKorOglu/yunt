/**
 * Authentication Guards
 * Route protection utilities for authenticated and public pages.
 */

import { goto } from '$app/navigation';
import { authStore } from '$stores/auth.svelte';

// ============================================================================
// Types
// ============================================================================

export interface GuardOptions {
	/** URL to redirect to if guard fails */
	redirectTo?: string;
	/** Whether to preserve the original URL for redirect after login */
	preserveUrl?: boolean;
}

export interface GuardResult {
	/** Whether access is allowed */
	allowed: boolean;
	/** URL to redirect to if not allowed */
	redirectUrl?: string;
}

// ============================================================================
// Guard Functions
// ============================================================================

/**
 * Check if user is authenticated
 * Use this guard for protected routes that require login
 *
 * @param options Guard options
 * @returns Guard result indicating if access is allowed
 */
export function requireAuth(options: GuardOptions = {}): GuardResult {
	const { redirectTo = '/login', preserveUrl = true } = options;

	// Wait for auth to be initialized
	if (authStore.isLoading) {
		return { allowed: true }; // Allow during loading, layout will handle
	}

	if (!authStore.isAuthenticated) {
		// Build redirect URL with return path if needed
		let url = redirectTo;
		if (preserveUrl && typeof window !== 'undefined') {
			const currentPath = window.location.pathname;
			if (currentPath !== redirectTo && currentPath !== '/') {
				url = `${redirectTo}?redirect=${encodeURIComponent(currentPath)}`;
			}
		}
		return { allowed: false, redirectUrl: url };
	}

	return { allowed: true };
}

/**
 * Check if user is a guest (not authenticated)
 * Use this guard for public-only routes like login page
 *
 * @param options Guard options
 * @returns Guard result indicating if access is allowed
 */
export function requireGuest(options: GuardOptions = {}): GuardResult {
	const { redirectTo = '/' } = options;

	// Wait for auth to be initialized
	if (authStore.isLoading) {
		return { allowed: true }; // Allow during loading
	}

	if (authStore.isAuthenticated) {
		return { allowed: false, redirectUrl: redirectTo };
	}

	return { allowed: true };
}

/**
 * Check if user has a specific role
 *
 * @param role Required role
 * @param options Guard options
 * @returns Guard result indicating if access is allowed
 */
export function requireRole(
	role: 'admin' | 'user' | 'viewer',
	options: GuardOptions = {}
): GuardResult {
	const { redirectTo = '/' } = options;

	// First check authentication
	const authResult = requireAuth(options);
	if (!authResult.allowed) {
		return authResult;
	}

	// Check role
	const user = authStore.user;
	if (!user || user.role !== role) {
		return { allowed: false, redirectUrl: redirectTo };
	}

	return { allowed: true };
}

/**
 * Check if user has admin role
 *
 * @param options Guard options
 * @returns Guard result indicating if access is allowed
 */
export function requireAdmin(options: GuardOptions = {}): GuardResult {
	return requireRole('admin', options);
}

// ============================================================================
// Navigation Guards
// ============================================================================

/**
 * Navigate with authentication guard
 * Redirects to login if not authenticated, otherwise navigates to target
 *
 * @param targetUrl Target URL to navigate to
 * @param options Guard options
 */
export async function navigateProtected(
	targetUrl: string,
	options: GuardOptions = {}
): Promise<void> {
	const result = requireAuth(options);

	if (result.allowed) {
		await goto(targetUrl);
	} else if (result.redirectUrl) {
		await goto(result.redirectUrl);
	}
}

/**
 * Handle guard result by redirecting if needed
 *
 * @param result Guard result to handle
 * @returns Promise that resolves when navigation is complete (if any)
 */
export async function handleGuardResult(result: GuardResult): Promise<boolean> {
	if (!result.allowed && result.redirectUrl) {
		await goto(result.redirectUrl);
		return false;
	}
	return result.allowed;
}

// ============================================================================
// Route List Configuration
// ============================================================================

/** Routes that don't require authentication */
export const publicRoutes: string[] = [
	'/login',
	'/register',
	'/forgot-password',
	'/reset-password'
];

/** Routes that are only accessible to guests (non-authenticated users) */
export const guestOnlyRoutes: string[] = ['/login', '/register', '/forgot-password'];

/**
 * Check if a path is a public route
 *
 * @param path Path to check
 * @returns True if path is public
 */
export function isPublicRoute(path: string): boolean {
	return publicRoutes.some((route) => path === route || path.startsWith(`${route}/`));
}

/**
 * Check if a path is a guest-only route
 *
 * @param path Path to check
 * @returns True if path is guest-only
 */
export function isGuestOnlyRoute(path: string): boolean {
	return guestOnlyRoutes.some((route) => path === route || path.startsWith(`${route}/`));
}
