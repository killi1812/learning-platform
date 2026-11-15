export type CourseTerm = 'SPRING' | 'AUTUMN'
export type CourseRole = 'STUDENT' | 'TEACHER'

export interface GradingScale {
  scaleA: number
  scaleB: number
  scaleC: number
  scaleD: number
  scaleE: number
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

export type UpdateCoursePayload = CreateCoursePayload

export interface AddEnrollmentPayload {
  email: string
  role?: CourseRole
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

export type UpdateModulePayload = CreateModulePayload

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
