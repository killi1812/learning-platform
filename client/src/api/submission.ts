import type {
  CreateHandinPayload,
  HandinAssignment,
  Submission,
  SubmissionAccepted,
  UpdateHandinPayload,
  UploadTicket,
} from '@/types/submission'
import { parseDownloadFilename, triggerBlobDownload } from '@/util/download'
import { authApi, request } from './client'

const enc = encodeURIComponent
function basePath(courseUuid: string, contentUuid: string) {
  return `/course/${enc(courseUuid)}/content/${enc(contentUuid)}/submission`
}

/** Phase 1: Reserve ticket and stamp submission receipt timestamp immediately */
export function initFileSubmission(courseUuid: string, contentUuid: string): Promise<UploadTicket> {
  return request(authApi.post<UploadTicket>(`${basePath(courseUuid, contentUuid)}/file/init`))
}

/** Phase 2: Stream upload deliverable with progress watchdog */
export function uploadSubmissionFile(
  courseUuid: string,
  contentUuid: string,
  ticketUuid: string,
  file: File,
  onProgress?: (pct: number, loaded: number, total: number) => void,
  signal?: AbortSignal,
): Promise<SubmissionAccepted> {
  const formData = new FormData()
  formData.append('file', file)

  return request(
    authApi.post<SubmissionAccepted>(
      `${basePath(courseUuid, contentUuid)}/file/upload?ticket=${enc(ticketUuid)}`,
      formData,
      {
        headers: { 'Content-Type': 'multipart/form-data' },
        timeout: 0, // Disable client wall-clock timeout so active streaming can continue
        signal,
        onUploadProgress: progressEvent => {
          if (progressEvent.total && onProgress) {
            const rawPct = Math.round((progressEvent.loaded * 100) / progressEvent.total)
            const pct = Math.min(100, Math.max(0, rawPct))
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
  signal?: AbortSignal,
): Promise<SubmissionAccepted> {
  const ticket = await initFileSubmission(courseUuid, contentUuid)
  return uploadSubmissionFile(courseUuid, contentUuid, ticket.ticketUuid, file, onProgress, signal)
}

/** Submit deliverable via Git repository or external URL */
export function submitUrl(
  courseUuid: string,
  contentUuid: string,
  url: string,
): Promise<SubmissionAccepted> {
  return request(
    authApi.post<SubmissionAccepted>(basePath(courseUuid, contentUuid), {
      payloadType: 'URL',
      payloadUrl: url,
    }),
  )
}

/** Student view of own submissions */
export function listMySubmissions(
  courseUuid: string,
  contentUuid: string,
  signal?: AbortSignal,
): Promise<Submission[]> {
  return request(authApi.get<Submission[]>(`${basePath(courseUuid, contentUuid)}/my`, { signal }))
}

/** Instructor view of all submissions */
export function listAssignmentSubmissions(
  courseUuid: string,
  contentUuid: string,
  signal?: AbortSignal,
): Promise<Submission[]> {
  return request(authApi.get<Submission[]>(basePath(courseUuid, contentUuid), { signal }))
}

/** Download submitted file */
export async function downloadSubmissionFile(
  courseUuid: string,
  contentUuid: string,
  submissionUuid: string,
  fileName?: string,
): Promise<void> {
  const response = await request(
    authApi
      .get<Blob>(`${basePath(courseUuid, contentUuid)}/${enc(submissionUuid)}/file`, {
        responseType: 'blob',
      })
      .then(res => ({ data: res })),
    'Failed to download submitted file.',
  )
  const contentType =
    typeof response.headers['content-type'] === 'string'
      ? response.headers['content-type']
      : 'application/octet-stream'
  const blob =
    response.data instanceof Blob ? response.data : new Blob([response.data], { type: contentType })
  const targetFilename =
    fileName || parseDownloadFilename(response.headers['content-disposition'], 'submitted_file')
  triggerBlobDownload(blob, targetFilename)
}

/** Hand-in assignment management (Teachers) */
export function createHandin(
  courseUuid: string,
  moduleUuid: string,
  payload: CreateHandinPayload,
): Promise<HandinAssignment> {
  return request(
    authApi.post<HandinAssignment>(
      `/course/${enc(courseUuid)}/module/${enc(moduleUuid)}/assignment`,
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
  return request(
    authApi.put<HandinAssignment>(
      `/course/${enc(courseUuid)}/module/${enc(moduleUuid)}/assignment/${enc(assignmentUuid)}`,
      payload,
    ),
  )
}

export function deleteHandin(
  courseUuid: string,
  moduleUuid: string,
  assignmentUuid: string,
): Promise<void> {
  return request(
    authApi.delete<void>(
      `/course/${enc(courseUuid)}/module/${enc(moduleUuid)}/assignment/${enc(assignmentUuid)}`,
    ),
  )
}

export function listHandins(courseUuid: string, moduleUuid: string): Promise<HandinAssignment[]> {
  return request(
    authApi.get<HandinAssignment[]>(
      `/course/${enc(courseUuid)}/module/${enc(moduleUuid)}/assignment`,
    ),
  )
}
