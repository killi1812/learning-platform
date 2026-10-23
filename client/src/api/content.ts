import { authApi, getApiErrorMessage } from './client'
import type { StaticPage, UploadStaticPagePayload, UpdateStaticPagePayload } from '@/types/content'

async function wrapRequest<T>(promise: Promise<{ data: T }>): Promise<T> {
  try {
    const response = await promise
    return response.data
  } catch (err: unknown) {
    throw new Error(getApiErrorMessage(err))
  }
}

export async function listStaticPages(
  courseUuid: string,
  moduleUuid: string,
): Promise<StaticPage[]> {
  return wrapRequest(
    authApi.get<StaticPage[]>(`/course/${courseUuid}/module/${moduleUuid}/static-page`),
  )
}

export async function getStaticPage(
  courseUuid: string,
  moduleUuid: string,
  pageUuid: string,
): Promise<StaticPage> {
  return wrapRequest(
    authApi.get<StaticPage>(`/course/${courseUuid}/module/${moduleUuid}/static-page/${pageUuid}`),
  )
}

export async function uploadStaticPage(
  courseUuid: string,
  moduleUuid: string,
  payload: UploadStaticPagePayload,
): Promise<StaticPage> {
  const formData = new FormData()
  if (payload.file) formData.append('file', payload.file)
  if (payload.title) formData.append('title', payload.title)
  if (payload.description) formData.append('description', payload.description)
  if (payload.sourceUrl) formData.append('sourceUrl', payload.sourceUrl)

  return wrapRequest(
    authApi.post<StaticPage>(`/course/${courseUuid}/module/${moduleUuid}/static-page`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
    }),
  )
}

export async function getStaticPagePreviewBlob(
  courseUuid: string,
  moduleUuid: string,
  pageUuid: string,
): Promise<Blob> {
  try {
    const response = await authApi.get<Blob>(
      `/course/${courseUuid}/module/${moduleUuid}/static-page/${pageUuid}/preview`,
      {
        responseType: 'blob',
      },
    )
    return response.data
  } catch (err: unknown) {
    throw new Error(getApiErrorMessage(err))
  }
}

export async function updateStaticPage(
  courseUuid: string,
  moduleUuid: string,
  pageUuid: string,
  payload: UpdateStaticPagePayload,
): Promise<StaticPage> {
  return wrapRequest(
    authApi.put<StaticPage>(
      `/course/${courseUuid}/module/${moduleUuid}/static-page/${pageUuid}`,
      payload,
    ),
  )
}

export async function deleteStaticPage(
  courseUuid: string,
  moduleUuid: string,
  pageUuid: string,
): Promise<void> {
  await wrapRequest(
    authApi.delete<void>(`/course/${courseUuid}/module/${moduleUuid}/static-page/${pageUuid}`),
  )
}
