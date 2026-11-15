import type { FileUpload, UpdateFilePayload, UploadFilePayload } from '@/types/content'
import { authApi, request } from './client'

function filesBasePath(courseUuid: string, moduleUuid: string) {
  return `/course/${encodeURIComponent(courseUuid)}/module/${encodeURIComponent(moduleUuid)}/file`
}

export function listFiles(courseUuid: string, moduleUuid: string): Promise<FileUpload[]> {
  return request(authApi.get<FileUpload[]>(filesBasePath(courseUuid, moduleUuid)))
}

export function getFile(
  courseUuid: string,
  moduleUuid: string,
  fileUuid: string,
  signal?: AbortSignal,
): Promise<FileUpload> {
  return request(
    authApi.get<FileUpload>(
      `${filesBasePath(courseUuid, moduleUuid)}/${encodeURIComponent(fileUuid)}`,
      { signal },
    ),
  )
}

export function uploadFile(
  courseUuid: string,
  moduleUuid: string,
  payload: UploadFilePayload,
): Promise<FileUpload> {
  const formData = new FormData()
  if (payload.file) {
    formData.append('file', payload.file)
  }
  if (payload.title) {
    formData.append('title', payload.title)
  }
  if (payload.description) {
    formData.append('description', payload.description)
  }
  if (payload.sourceUrl) {
    formData.append('sourceUrl', payload.sourceUrl)
  }

  return request(
    authApi.post<FileUpload>(filesBasePath(courseUuid, moduleUuid), formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }),
  )
}

export function getFilePreviewBlob(
  courseUuid: string,
  moduleUuid: string,
  fileUuid: string,
  signal?: AbortSignal,
): Promise<Blob> {
  return request(
    authApi.get<Blob>(
      `${filesBasePath(courseUuid, moduleUuid)}/${encodeURIComponent(fileUuid)}/preview`,
      {
        responseType: 'blob',
        signal,
      },
    ),
  )
}

export function updateFile(
  courseUuid: string,
  moduleUuid: string,
  fileUuid: string,
  payload: UpdateFilePayload,
): Promise<FileUpload> {
  return request(
    authApi.put<FileUpload>(
      `${filesBasePath(courseUuid, moduleUuid)}/${encodeURIComponent(fileUuid)}`,
      payload,
    ),
  )
}

export async function deleteFile(
  courseUuid: string,
  moduleUuid: string,
  fileUuid: string,
): Promise<void> {
  await request(
    authApi.delete<void>(
      `${filesBasePath(courseUuid, moduleUuid)}/${encodeURIComponent(fileUuid)}`,
    ),
  )
}
