import type { GradingScale } from './course'

export interface UpsertGradePayload {
  score: number
  feedback?: string
  isVisibleToStudent: boolean
}

export interface GradeDto {
  score: number
  maxPoints: number
  feedback?: string
  isVisibleToStudent: boolean
  gradedByEmail?: string
  gradedAt: string
  releasedAt?: string
}

export interface ReleaseResultDto {
  released: number
}

export interface PendingSubmissionDto {
  submissionUuid: string
  courseUuid: string
  contentUuid: string
  courseCode: string
  courseName: string
  assignmentTitle: string
  userEmail: string
  userName?: string
  isGroup?: boolean
  groupName?: string
  memberEmails?: string[]
  attempt: number
  isLate: boolean
  submittedAt: string
  deadline?: string
  maxPoints: number
}

export type StudentGradeType = 'HANDIN' | 'QUIZ'

export interface StudentGradeDto {
  type: StudentGradeType
  courseUuid: string
  contentUuid: string
  courseName: string
  courseCode: string
  moduleTitle: string
  assignmentTitle: string
  isGroup?: boolean
  groupName?: string
  attempt: number
  score: number
  maxPoints: number
  percent: number
  feedback?: string
  gradedAt: string
}

export interface CourseFinalGradeDto {
  courseUuid: string
  courseName: string
  courseCode: string
  totalScore: number
  totalMaxScore: number
  percent: number
  finalGrade: string
  gradingScale: GradingScale
  /** True when every counted item is graded or past its deadline. finalGrade only applies then. */
  finished: boolean
  /** Points of items that are graded or past their deadline. */
  progressScore: number
  progressMaxScore: number
}

export interface StudentFinalGradeRowDto {
  userUuid: string
  userEmail: string
  userName?: string
  totalScore: number
  totalMaxScore: number
  percent: number
  finalGrade: string
}

export type CourseGradeStatus = 'NOT_SUBMITTED' | 'SUBMITTED' | 'AWAITING_RELEASE' | 'GRADED'

/** Score, percent, feedback and gradedAt are only present when status is GRADED. */
export interface CourseGradeItem {
  type: StudentGradeType
  contentUuid: string
  title: string
  moduleUuid: string
  moduleTitle: string
  deadline?: string
  status: CourseGradeStatus
  submittedAt?: string
  score?: number
  maxPoints: number
  percent?: number
  isLate: boolean
  passed?: boolean
  attempt: number
  maxAttempts?: number
  isGroup: boolean
  groupName?: string
  feedback?: string
  gradedAt?: string
}

export interface CourseGradeOverview {
  courseUuid: string
  courseName: string
  courseCode: string
  totalScore: number
  totalMaxScore: number
  percent: number
  gradedPercent: number
  finalGrade: string
  gradingScale: GradingScale
  /** True when every counted item is graded or past its deadline. finalGrade only applies then. */
  finished: boolean
  /** Points of items that are graded or past their deadline. */
  progressScore: number
  progressMaxScore: number
  items: CourseGradeItem[]
}
