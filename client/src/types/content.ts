export interface StaticPage {
  uuid: string
  moduleUuid?: string
  title: string
  position: number
  fileName?: string
  fileSize?: number
  contentType?: string
  description?: string
  sourceUrl?: string
  createdAt: string
  updatedAt: string
}

export interface UploadStaticPagePayload {
  file?: File
  title?: string
  description?: string
  sourceUrl?: string
}

export interface UpdateStaticPagePayload {
  title: string
  description?: string
}
