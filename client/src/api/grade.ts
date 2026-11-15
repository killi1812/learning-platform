import { authApi, getApiErrorMessage } from './client'
import type {
  UpsertGradePayload,
  GradeDto,
  ReleaseResultDto,
  PendingSubmissionDto,
  StudentGradeDto,
} from '@/types/grade'

async function wrapRequest<T>(promise: Promise<{ data: T }>): Promise<T> {
  try {
    const response = await promise
    return response.data
  } catch (err: unknown) {
    throw new Error(getApiErrorMessage(err))
  }
}

/** Grade or update an individual submission (Teachers & Admins) */
export function upsertGrade(
  courseUuid: string,
  contentUuid: string,
  submissionUuid: string,
  payload: UpsertGradePayload,
): Promise<GradeDto> {
  return wrapRequest(
    authApi.put<GradeDto>(
      `/course/${courseUuid}/content/${contentUuid}/submission/${submissionUuid}/grade`,
      payload,
    ),
  )
}

/** Release all unreleased individual grades for an assignment (Teachers & Admins) */
export function releaseAssignmentGrades(
  courseUuid: string,
  contentUuid: string,
): Promise<ReleaseResultDto> {
  return wrapRequest(
    authApi.post<ReleaseResultDto>(`/course/${courseUuid}/content/${contentUuid}/grade/release`),
  )
}

/** List pending ungraded individual submissions across courses taught by caller */
export function listPendingGrades(courseUuid?: string): Promise<PendingSubmissionDto[]> {
  const params: Record<string, string> = {}
  if (courseUuid) {
    params.courseUuid = courseUuid
  }
  return wrapRequest(authApi.get<PendingSubmissionDto[]>('/grade/pending', { params }))
}

/** List authenticated student's own grades and quiz results */
export function listMyGrades(courseUuid?: string): Promise<StudentGradeDto[]> {
  const params: Record<string, string> = {}
  if (courseUuid) {
    params.courseUuid = courseUuid
  }
  return wrapRequest(authApi.get<StudentGradeDto[]>('/grade/my', { params }))
}

/** Download course gradebook CSV */
export async function exportCourseGrades(courseUuid: string, filename?: string): Promise<void> {
  try {
    const response = await authApi.get<Blob>(`/course/${courseUuid}/grade/export`, {
      responseType: 'blob',
    })
    const contentType =
      typeof response.headers['content-type'] === 'string'
        ? response.headers['content-type']
        : 'text/csv'
    const blob =
      response.data instanceof Blob
        ? response.data
        : new Blob([response.data], { type: contentType })

    // If filename wasn't provided, try reading from Content-Disposition header
    let targetFilename = filename
    if (!targetFilename) {
      const disposition = response.headers['content-disposition']
      if (typeof disposition === 'string') {
        const match = disposition.match(/filename=["']?([^"']+)["']?/)
        if (match && match[1]) {
          targetFilename = match[1]
        }
      }
    }
    if (!targetFilename) {
      targetFilename = `course-${courseUuid.slice(0, 8)}-gradebook.csv`
    }

    const downloadUrl = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = downloadUrl
    link.download = targetFilename
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(downloadUrl)
  } catch (err: unknown) {
    throw new Error(getApiErrorMessage(err, 'Failed to export gradebook CSV.'))
  }
}
