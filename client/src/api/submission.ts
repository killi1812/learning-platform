import { authApi, getApiErrorMessage } from './client'
import type {
  UploadTicket,
  SubmissionAccepted,
  Submission,
  CreateHandinPayload,
  UpdateHandinPayload,
  HandinAssignment,
} from '@/types/submission'

async function wrapRequest<T>(promise: Promise<{ data: T }>): Promise<T> {
  try {
    const response = await promise
    return response.data
  } catch (err: unknown) {
    throw new Error(getApiErrorMessage(err))
  }
}

const basePath = (courseUuid: string, contentUuid: string) =>
  `/course/${courseUuid}/content/${contentUuid}/submission`

/** Phase 1: Reserve ticket and stamp submission receipt timestamp immediately */
export function initFileSubmission(courseUuid: string, contentUuid: string): Promise<UploadTicket> {
  return wrapRequest(authApi.post<UploadTicket>(`${basePath(courseUuid, contentUuid)}/file/init`))
}

/** Phase 2: Stream upload deliverable with progress watchdog */
export function uploadSubmissionFile(
  courseUuid: string,
  contentUuid: string,
  ticketUuid: string,
  file: File,
  onProgress?: (pct: number, loaded: number, total: number) => void,
): Promise<SubmissionAccepted> {
  const formData = new FormData()
  formData.append('file', file)

  return wrapRequest(
    authApi.post<SubmissionAccepted>(
      `${basePath(courseUuid, contentUuid)}/file/upload?ticket=${encodeURIComponent(ticketUuid)}`,
      formData,
      {
        headers: { 'Content-Type': 'multipart/form-data' },
        timeout: 0, // Disable client wall-clock timeout so active streaming can continue
        onUploadProgress: progressEvent => {
          if (progressEvent.total && onProgress) {
            const pct = Math.round((progressEvent.loaded * 100) / progressEvent.total)
            onProgress(pct, progressEvent.loaded, progressEvent.total)
          }
        },
      },
    ),
  )
}

/** Full Two-Phase submission helper: Inits ticket first, then streams file */
export async function submitFileTwoPhase(
  courseUuid: string,
  contentUuid: string,
  file: File,
  onProgress?: (pct: number, loaded: number, total: number) => void,
): Promise<SubmissionAccepted> {
  const ticket = await initFileSubmission(courseUuid, contentUuid)
  return uploadSubmissionFile(courseUuid, contentUuid, ticket.ticketUuid, file, onProgress)
}

/** Submit deliverable via Git repository or external URL */
export function submitUrl(
  courseUuid: string,
  contentUuid: string,
  url: string,
): Promise<SubmissionAccepted> {
  return wrapRequest(
    authApi.post<SubmissionAccepted>(basePath(courseUuid, contentUuid), {
      payloadType: 'URL',
      payloadUrl: url,
    }),
  )
}

/** Student view of own submissions */
export function listMySubmissions(courseUuid: string, contentUuid: string): Promise<Submission[]> {
  return wrapRequest(authApi.get<Submission[]>(`${basePath(courseUuid, contentUuid)}/my`))
}

/** Instructor view of all submissions */
export function listAssignmentSubmissions(
  courseUuid: string,
  contentUuid: string,
): Promise<Submission[]> {
  return wrapRequest(authApi.get<Submission[]>(basePath(courseUuid, contentUuid)))
}

/** Download submitted file */
export async function downloadSubmissionFile(
  courseUuid: string,
  contentUuid: string,
  submissionUuid: string,
  fileName?: string,
): Promise<void> {
  try {
    const response = await authApi.get<Blob>(
      `${basePath(courseUuid, contentUuid)}/${submissionUuid}/file`,
      { responseType: 'blob' },
    )
    const contentType =
      typeof response.headers['content-type'] === 'string'
        ? response.headers['content-type']
        : 'application/octet-stream'
    const blob =
      response.data instanceof Blob
        ? response.data
        : new Blob([response.data], { type: contentType })
    const downloadUrl = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = downloadUrl
    link.download = fileName || 'submitted_file'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(downloadUrl)
  } catch (err: unknown) {
    throw new Error(getApiErrorMessage(err))
  }
}

/** Hand-in assignment management (Teachers) */
export function createHandin(
  courseUuid: string,
  moduleUuid: string,
  payload: CreateHandinPayload,
): Promise<HandinAssignment> {
  return wrapRequest(
    authApi.post<HandinAssignment>(
      `/course/${courseUuid}/module/${moduleUuid}/assignment`,
      payload,
    ),
  )
}

export function updateHandin(
  courseUuid: string,
  moduleUuid: string,
  assignmentUuid: string,
  payload: UpdateHandinPayload,
): Promise<HandinAssignment> {
  return wrapRequest(
    authApi.put<HandinAssignment>(
      `/course/${courseUuid}/module/${moduleUuid}/assignment/${assignmentUuid}`,
      payload,
    ),
  )
}

export function deleteHandin(
  courseUuid: string,
  moduleUuid: string,
  assignmentUuid: string,
): Promise<void> {
  return wrapRequest(
    authApi.delete<void>(`/course/${courseUuid}/module/${moduleUuid}/assignment/${assignmentUuid}`),
  )
}

export function listHandins(courseUuid: string, moduleUuid: string): Promise<HandinAssignment[]> {
  return wrapRequest(
    authApi.get<HandinAssignment[]>(`/course/${courseUuid}/module/${moduleUuid}/assignment`),
  )
}
