function sanitizeFilename(name: string, fallback: string): string {
  const leaf = name.split(/[/\\]/).pop() ?? ''
  const cleaned = [...leaf]
    .filter(char => {
      const code = char.codePointAt(0) ?? 0
      return code >= 32 && code !== 127
    })
    .join('')
    .trim()
  return cleaned || fallback
}

/** Extracts a safe filename from an HTTP Content-Disposition header (supporting RFC 5987). */
export function parseDownloadFilename(contentDisposition: unknown, fallback: string): string {
  if (typeof contentDisposition !== 'string' || !contentDisposition.trim()) {
    return fallback
  }

  const utf8Match = contentDisposition.match(/filename\*\s*=\s*UTF-8''([^;]+)/i)
  if (utf8Match?.[1]) {
    try {
      return sanitizeFilename(decodeURIComponent(utf8Match[1].trim()), fallback)
    } catch {
      // Fall through to standard filename parameter parsing
    }
  }

  const quotedMatch = contentDisposition.match(/filename\s*=\s*"([^"]+)"/i)
  if (quotedMatch?.[1]) {
    return sanitizeFilename(quotedMatch[1], fallback)
  }

  const plainMatch = contentDisposition.match(/filename\s*=\s*([^;\s]+)/i)
  if (plainMatch?.[1]) {
    return sanitizeFilename(plainMatch[1], fallback)
  }

  return fallback
}

/** Triggers a browser file download for a Blob and cleans up the object URL safely. */
export function triggerBlobDownload(blob: Blob, filename: string): void {
  const downloadUrl = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = downloadUrl
  link.download = sanitizeFilename(filename, 'download')
  document.body.append(link)
  try {
    link.click()
  } finally {
    link.remove()
    window.setTimeout(() => {
      window.URL.revokeObjectURL(downloadUrl)
    }, 1000)
  }
}
