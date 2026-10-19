export type CourseTerm = 'SPRING' | 'AUTUMN'
export type CourseRole = 'STUDENT' | 'TEACHER'

export interface Course {
  id: number
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
  id: number
  userId: number
  username?: string
  userEmail: string
  courseId: number
  courseUuid?: string
  role: CourseRole
  enrolledAt: string
}

export interface MyCourse {
  course: Course
  role: CourseRole
  enrolledAt: string
}

export type ContentType = 'STATIC_PAGE' | 'QUIZ' | 'ASSIGNMENT' | 'CODING_EXERCISE'

export interface ModuleContentItem {
  id: number
  moduleId: number
  title: string
  position: number
  type: ContentType
  visibleFrom?: string
  visibleUntil?: string
}

export interface Module {
  id: number
  courseId: number
  title: string
  position: number
  dependsOnId?: number
  dependsOn?: Module
  contents?: ModuleContentItem[]
}

export interface CreateModulePayload {
  title: string
  position?: number
  dependsOnId?: number
}

export interface UpdateModulePayload {
  title: string
  position?: number
  dependsOnId?: number
}
