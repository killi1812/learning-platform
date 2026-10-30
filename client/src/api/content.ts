import axios from 'axios'
import { authApi, getApiErrorMessage } from './client'
import type { FileUpload, UploadFilePayload, UpdateFilePayload } from '@/types/content'

async function wrapRequest<T>(promise: Promise<{ data: T }>): Promise<T> {
  try {
    const response = await promise
    return response.data
  } catch (err: unknown) {
    throw new Error(getApiErrorMessage(err))
  }
}

export async function listFiles(courseUuid: string, moduleUuid: string): Promise<FileUpload[]> {
  return wrapRequest(authApi.get<FileUpload[]>(`/course/${courseUuid}/module/${moduleUuid}/file`))
}

export async function getFile(
  courseUuid: string,
  moduleUuid: string,
  fileUuid: string,
): Promise<FileUpload> {
  return wrapRequest(
    authApi.get<FileUpload>(`/course/${courseUuid}/module/${moduleUuid}/file/${fileUuid}`),
  )
}

export async function uploadFile(
  courseUuid: string,
  moduleUuid: string,
  payload: UploadFilePayload,
): Promise<FileUpload> {
  const formData = new FormData()
  if (payload.file) formData.append('file', payload.file)
  if (payload.title) formData.append('title', payload.title)
  if (payload.description) formData.append('description', payload.description)
  if (payload.sourceUrl) formData.append('sourceUrl', payload.sourceUrl)

  return wrapRequest(
    authApi.post<FileUpload>(`/course/${courseUuid}/module/${moduleUuid}/file`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }),
  )
}

export async function getFilePreviewBlob(
  courseUuid: string,
  moduleUuid: string,
  fileUuid: string,
): Promise<Blob> {
  try {
    const response = await authApi.get<Blob>(
      `/course/${courseUuid}/module/${moduleUuid}/file/${fileUuid}/preview`,
      {
        responseType: 'blob',
      },
    )
    return response.data
  } catch (err: unknown) {
    if (axios.isAxiosError(err) && err.response?.data instanceof Blob) {
      try {
        const text = await err.response.data.text()
        const parsed = JSON.parse(text)
        if (parsed && typeof parsed.error === 'string') {
          throw new Error(parsed.error)
        }
      } catch (parseErr: unknown) {
        if (parseErr instanceof Error && parseErr.message && !parseErr.message.includes('JSON')) {
          throw parseErr
        }
      }
    }
    throw new Error(getApiErrorMessage(err))
  }
}

export async function updateFile(
  courseUuid: string,
  moduleUuid: string,
  fileUuid: string,
  payload: UpdateFilePayload,
): Promise<FileUpload> {
  return wrapRequest(
    authApi.put<FileUpload>(`/course/${courseUuid}/module/${moduleUuid}/file/${fileUuid}`, payload),
  )
}

export async function deleteFile(
  courseUuid: string,
  moduleUuid: string,
  fileUuid: string,
): Promise<void> {
  await wrapRequest(
    authApi.delete<void>(`/course/${courseUuid}/module/${moduleUuid}/file/${fileUuid}`),
  )
}
