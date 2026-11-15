export interface FileFormatInfo {
  fileName?: string
  contentType?: string
  sourceUrl?: string
}

export function formatBytes(bytes?: number): string {
  if (!bytes) {
    return ''
  }
  if (bytes < 1024) {
    return `${bytes} B`
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`
  }
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function isPdfMaterial(file?: FileFormatInfo): boolean {
  const name = file?.fileName?.toLowerCase() ?? ''
  return Boolean(file?.contentType?.includes('pdf') || name.endsWith('.pdf'))
}

export function isWordMaterial(file?: FileFormatInfo): boolean {
  const name = file?.fileName?.toLowerCase() ?? ''
  return Boolean(
    file?.contentType?.includes('word') || name.endsWith('.docx') || name.endsWith('.doc'),
  )
}

export function isPresentationMaterial(file?: FileFormatInfo): boolean {
  const name = file?.fileName?.toLowerCase() ?? ''
  return Boolean(
    file?.contentType?.includes('presentation') ||
    file?.contentType?.includes('powerpoint') ||
    name.endsWith('.pptx') ||
    name.endsWith('.ppt'),
  )
}

export function isImageMaterial(file?: FileFormatInfo): boolean {
  const name = file?.fileName?.toLowerCase() ?? ''
  return Boolean(
    file?.contentType?.startsWith('image/') ||
    ['.png', '.jpg', '.jpeg', '.gif', '.svg', '.webp'].some(ext => name.endsWith(ext)),
  )
}

export function isTextMaterial(file?: FileFormatInfo): boolean {
  const name = file?.fileName?.toLowerCase() ?? ''
  return Boolean(
    file?.contentType?.includes('text') ||
    ['.txt', '.md', '.json', '.go', '.ts'].some(ext => name.endsWith(ext)),
  )
}

export function getFileFormatIcon(file?: FileFormatInfo): string {
  const name = file?.fileName?.toLowerCase() ?? ''
  if (isPdfMaterial(file)) {
    return 'mdi-file-pdf-box'
  }
  if (isWordMaterial(file)) {
    return 'mdi-file-word-box'
  }
  if (isPresentationMaterial(file)) {
    return 'mdi-file-powerpoint-box'
  }
  if (isImageMaterial(file)) {
    return 'mdi-file-image'
  }
  if (name.endsWith('.md')) {
    return 'mdi-language-markdown'
  }
  if (isTextMaterial(file)) {
    return 'mdi-file-document-outline'
  }
  if (file?.sourceUrl) {
    return 'mdi-link-variant'
  }
  return 'mdi-file-outline'
}

export function getFileFormatColor(file?: FileFormatInfo): string {
  if (isPdfMaterial(file)) {
    return 'error'
  }
  if (isWordMaterial(file)) {
    return 'primary'
  }
  if (isPresentationMaterial(file)) {
    return 'warning'
  }
  if (isImageMaterial(file)) {
    return 'info'
  }
  if (isTextMaterial(file)) {
    return 'primary'
  }
  if (file?.sourceUrl) {
    return 'info'
  }
  return 'secondary'
}
