import type {
  CourseFinalGradeDto,
  CourseGradeOverview,
  GradeDto,
  PendingSubmissionDto,
  ReleaseResultDto,
  StudentFinalGradeRowDto,
  StudentGradeDto,
  UpsertGradePayload,
} from '@/types/grade'
import { parseDownloadFilename, triggerBlobDownload } from '@/util/download'
import { authApi, request } from './client'

const enc = encodeURIComponent

/** Grade or update an individual submission (Teachers & Admins) */
export function upsertGrade(
  courseUuid: string,
  contentUuid: string,
  submissionUuid: string,
  payload: UpsertGradePayload,
): Promise<GradeDto> {
  return request(
    authApi.put<GradeDto>(
      `/course/${enc(courseUuid)}/content/${enc(contentUuid)}/submission/${enc(submissionUuid)}/grade`,
      payload,
    ),
  )
}

/** Release all unreleased individual grades for an assignment (Teachers & Admins) */
export function releaseAssignmentGrades(
  courseUuid: string,
  contentUuid: string,
): Promise<ReleaseResultDto> {
  return request(
    authApi.post<ReleaseResultDto>(
      `/course/${enc(courseUuid)}/content/${enc(contentUuid)}/grade/release`,
    ),
  )
}

/** List pending ungraded individual submissions across courses taught by caller */
export function listPendingGrades(courseUuid?: string): Promise<PendingSubmissionDto[]> {
  return request(
    authApi.get<PendingSubmissionDto[]>('/grade/pending', {
      params: courseUuid ? { courseUuid } : undefined,
    }),
  )
}

/** List authenticated student's own grades and quiz results */
export function listMyGrades(courseUuid?: string): Promise<StudentGradeDto[]> {
  return request(
    authApi.get<StudentGradeDto[]>('/grade/my', {
      params: courseUuid ? { courseUuid } : undefined,
    }),
  )
}

/** List authenticated student's cumulative subject evaluation and final letter grades */
export function getMyFinalGrades(courseUuid?: string): Promise<CourseFinalGradeDto[]> {
  return request(
    authApi.get<CourseFinalGradeDto[]>('/grade/my/final', {
      params: courseUuid ? { courseUuid } : undefined,
    }),
  )
}

/** Get authenticated student's per-item grade overview and totals for one course */
export function getMyCourseGrades(courseUuid: string): Promise<CourseGradeOverview> {
  return request(authApi.get<CourseGradeOverview>(`/course/${enc(courseUuid)}/grade/my`))
}

/** List course roster with final cumulative points and letter grades (Teachers & Admins) */
export function getCourseRosterFinalGrades(courseUuid: string): Promise<StudentFinalGradeRowDto[]> {
  return request(authApi.get<StudentFinalGradeRowDto[]>(`/course/${enc(courseUuid)}/grade/roster`))
}

/** Download course gradebook CSV */
export async function exportCourseGrades(courseUuid: string, filename?: string): Promise<void> {
  const response = await request(
    authApi
      .get<Blob>(`/course/${enc(courseUuid)}/grade/export`, { responseType: 'blob' })
      .then(res => ({ data: res })),
    'Failed to export gradebook CSV.',
  )

  const contentType =
    typeof response.headers['content-type'] === 'string'
      ? response.headers['content-type']
      : 'text/csv'
  const blob =
    response.data instanceof Blob ? response.data : new Blob([response.data], { type: contentType })
  const fallbackName = `course-${courseUuid.slice(0, 8)}-gradebook.csv`
  const targetFilename =
    filename || parseDownloadFilename(response.headers['content-disposition'], fallbackName)
  triggerBlobDownload(blob, targetFilename)
}
