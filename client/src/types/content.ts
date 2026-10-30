export interface FileUpload {
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

export interface UploadFilePayload {
  file?: File
  title?: string
  description?: string
  sourceUrl?: string
}

export interface UpdateFilePayload {
  title: string
  description?: string
}
