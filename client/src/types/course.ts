export type CourseTerm = 'SPRING' | 'AUTUMN'
export type CourseRole = 'STUDENT' | 'TEACHER'

export interface GradingScale {
  scaleA: number
  scaleB: number
  scaleC: number
  scaleD: number
  scaleE: number
}

export const DEFAULT_GRADING_SCALE: GradingScale = {
  scaleA: 90,
  scaleB: 80,
  scaleC: 60,
  scaleD: 50,
  scaleE: 40,
}

export function isValidGradingScale(s: GradingScale): boolean {
  const values = [s.scaleA, s.scaleB, s.scaleC, s.scaleD, s.scaleE]
  if (values.some(v => typeof v !== 'number' || !Number.isFinite(v))) {
    return false
  }
  return (
    s.scaleA <= 100 &&
    s.scaleA > s.scaleB &&
    s.scaleB > s.scaleC &&
    s.scaleC > s.scaleD &&
    s.scaleD > s.scaleE &&
    s.scaleE > 0
  )
}

export interface Course {
  uuid: string
  createdAt: string
  updatedAt: string
  title: string
  description: string
  term: CourseTerm
  year: number
  gradingScale?: GradingScale
  teachers?: string[]
}

export interface CreateCoursePayload {
  title: string
  description?: string
  term: CourseTerm
  year: number
  gradingScale?: GradingScale
}

export interface UpdateCoursePayload {
  title: string
  description?: string
  term: CourseTerm
  year: number
  gradingScale?: GradingScale
}

export interface Enrollment {
  userUuid?: string
  username?: string
  userEmail: string
  courseUuid?: string
  role: CourseRole
  enrolledAt: string
}

export interface MyCourse {
  course: Course
  role: CourseRole
  enrolledAt: string
}

export interface Module {
  uuid: string
  courseUuid?: string
  title: string
  position: number
  dependsOnUuid?: string
  dependsOn?: Module
}

export interface CreateModulePayload {
  title: string
  position?: number
  dependsOnUuid?: string
}

export interface UpdateModulePayload {
  title: string
  position?: number
  dependsOnUuid?: string
}

export interface GroupMember {
  uuid: string
  username: string
  email: string
}

export interface Group {
  uuid: string
  courseUuid: string
  name: string
  maxMembers: number
  members: GroupMember[]
}

export interface CreateGroupPayload {
  courseUuid: string
  name: string
  maxMembers: number
}

export interface BulkCreateGroupsPayload {
  courseUuid: string
  prefix?: string
  count: number
  maxMembers: number
}

export interface UpdateGroupPayload {
  name: string
  maxMembers: number
}
