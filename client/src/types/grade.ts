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
