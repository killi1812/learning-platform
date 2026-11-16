/** Checks whether a string uses an allowed external web protocol (`http:` or `https:`). */
export function isSafeHttpUrl(url?: string): boolean {
  if (!url) {
    return false
  }
  try {
    const parsed = new URL(url.trim())
    return parsed.protocol === 'http:' || parsed.protocol === 'https:'
  } catch {
    return false
  }
}

/** Checks whether a string is strictly an internal relative application path. */
export function isInternalPath(target: string): boolean {
  if (!target || target !== target.trim()) {
    return false
  }
  if (!target.startsWith('/') || target.startsWith('//') || target.startsWith('/\\')) {
    return false
  }
  const hasControlChar = [...target].some(char => {
    const code = char.codePointAt(0) ?? 0
    return code < 32 || code === 127
  })
  if (hasControlChar) {
    return false
  }
  try {
    const parsed = new URL(target, 'http://localhost')
    return (
      parsed.origin === 'http://localhost' &&
      parsed.pathname.startsWith('/') &&
      !parsed.pathname.startsWith('//')
    )
  } catch {
    return false
  }
}
