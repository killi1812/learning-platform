export type CourseTerm = 'SPRING' | 'AUTUMN'
export type CourseRole = 'STUDENT' | 'TEACHER'

export interface Course {
  uuid: string
  createdAt: string
  updatedAt: string
  title: string
  description: string
  term: CourseTerm
  year: number
  teachers?: string[]
}

export interface CreateCoursePayload {
  title: string
  description?: string
  term: CourseTerm
  year: number
}

export interface UpdateCoursePayload {
  title: string
  description?: string
  term: CourseTerm
  year: number
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
