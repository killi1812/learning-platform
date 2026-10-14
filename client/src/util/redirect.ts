/**
 * Validates and sanitizes a redirect target to prevent open redirect vulnerabilities.
 * Ensures the destination is strictly a relative, internal application path.
 */
export function getSafeRedirect(redirect: unknown, fallback = '/dashboard'): string {
  if (
    typeof redirect === 'string' &&
    redirect.startsWith('/') &&
    !redirect.startsWith('//') &&
    !redirect.includes('://')
  ) {
    return redirect
  }
  return fallback
}
