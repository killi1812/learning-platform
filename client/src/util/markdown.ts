import { marked } from 'marked'
import DOMPurify from 'dompurify'

/**
 * Parses markdown into sanitized HTML safe for direct rendering.
 */
export function renderMarkdown(src?: string): string {
  if (!src || !src.trim()) {
    return ''
  }
  const rawHtml = marked.parse(src, {
    async: false,
    gfm: true,
    breaks: true,
  }) as string
  return DOMPurify.sanitize(rawHtml)
}
