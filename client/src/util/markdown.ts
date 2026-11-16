import DOMPurify from 'dompurify'
import { marked } from 'marked'

let hookRegistered = false

function ensureAnchorSecurityHook(): void {
  if (hookRegistered) {
    return
  }
  hookRegistered = true
  DOMPurify.addHook('afterSanitizeAttributes', node => {
    if (node.tagName === 'A' && node.getAttribute('target') === '_blank') {
      node.setAttribute('rel', 'noopener noreferrer')
    }
  })
}

/**
 * Parses markdown into sanitized HTML safe for direct rendering.
 */
export function renderMarkdown(src?: string): string {
  if (!src || !src.trim()) {
    return ''
  }
  ensureAnchorSecurityHook()
  const rawHtml = marked.parse(src, {
    async: false,
    gfm: true,
    breaks: true,
  })
  return DOMPurify.sanitize(rawHtml, {
    FORBID_TAGS: ['style', 'form', 'input', 'textarea', 'select', 'button'],
    RETURN_DOM: false,
    RETURN_DOM_FRAGMENT: false,
  })
}
