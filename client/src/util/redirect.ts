import { isInternalPath } from './url'

/**
 * Validates and sanitizes a redirect target to prevent open redirect vulnerabilities.
 * Ensures the destination is strictly a relative, internal application path.
 */
export function getSafeRedirect(redirect: unknown, fallback = '/dashboard'): string {
  if (typeof redirect === 'string' && isInternalPath(redirect)) {
    return redirect
  }
  return fallback
}
