import type { CourseRole } from './course'
import type { AssignmentType } from './quiz'

/** One assignment with a deadline, flattened with its course for dashboard lists. */
export interface DeadlineItem {
  uuid: string
  title: string
  type: AssignmentType
  courseUuid: string
  courseTitle: string
  role: CourseRole
  deadline: string
  attemptsUsed: number
  maxAttempts: number
  quizFinished: boolean
}

export type DeadlineUrgency = 'overdue' | 'soon' | 'later'

/** Teacher view: how many enrolled students handed in one assignment. */
export interface SubmissionActivityItem {
  uuid: string
  title: string
  courseUuid: string
  courseTitle: string
  deadline: string
  submitted: number
  late: number
  enrolled: number
}
