import { authApi, getApiErrorMessage } from './client'
import type {
  Course,
  CourseRole,
  CreateCoursePayload,
  UpdateCoursePayload,
  Enrollment,
  MyCourse,
  Module,
  CreateModulePayload,
  UpdateModulePayload,
  Group,
  CreateGroupPayload,
  BulkCreateGroupsPayload,
  UpdateGroupPayload,
} from '@/types/course'

async function wrapRequest<T>(promise: Promise<{ data: T }>): Promise<T> {
  try {
    const response = await promise
    return response.data
  } catch (err: unknown) {
    throw new Error(getApiErrorMessage(err))
  }
}

export async function listCourses(term?: string, year?: number): Promise<Course[]> {
  const params: Record<string, string> = {}
  if (term) params.term = term
  if (year) params.year = year.toString()
  return wrapRequest(authApi.get<Course[]>('/course', { params }))
}

export async function getCourse(uuid: string): Promise<Course> {
  return wrapRequest(authApi.get<Course>(`/course/${uuid}`))
}

export async function getMyCourses(): Promise<MyCourse[]> {
  return wrapRequest(authApi.get<MyCourse[]>('/course/my-course'))
}

export async function createCourse(data: CreateCoursePayload): Promise<Course> {
  return wrapRequest(authApi.post<Course>('/course', data))
}

export async function updateCourse(uuid: string, data: UpdateCoursePayload): Promise<Course> {
  return wrapRequest(authApi.put<Course>(`/course/${uuid}`, data))
}

export async function deleteCourse(uuid: string): Promise<void> {
  await wrapRequest(authApi.delete<void>(`/course/${uuid}`))
}

export async function selfEnroll(courseUuid: string): Promise<Enrollment> {
  return wrapRequest(authApi.post<Enrollment>(`/course/${courseUuid}/enroll`))
}

export async function withdraw(courseUuid: string): Promise<void> {
  await wrapRequest(authApi.delete<void>(`/course/${courseUuid}/enroll`))
}

export async function getCourseEnrollments(courseUuid: string): Promise<Enrollment[]> {
  return wrapRequest(authApi.get<Enrollment[]>(`/course/${courseUuid}/enrollments`))
}

export async function addEnrollment(
  courseUuid: string,
  data: { email: string; role?: CourseRole },
): Promise<Enrollment> {
  return wrapRequest(authApi.post<Enrollment>(`/course/${courseUuid}/enrollments`, data))
}

export async function removeEnrollment(courseUuid: string, userUuid: string): Promise<void> {
  await wrapRequest(authApi.delete<void>(`/course/${courseUuid}/enrollments/${userUuid}`))
}

export async function listModules(courseUuid: string): Promise<Module[]> {
  return wrapRequest(authApi.get<Module[]>(`/course/${courseUuid}/module`))
}

export async function getModule(courseUuid: string, moduleUuid: string): Promise<Module> {
  return wrapRequest(authApi.get<Module>(`/course/${courseUuid}/module/${moduleUuid}`))
}

export async function createModule(courseUuid: string, data: CreateModulePayload): Promise<Module> {
  return wrapRequest(authApi.post<Module>(`/course/${courseUuid}/module`, data))
}

export async function updateModule(
  courseUuid: string,
  moduleUuid: string,
  data: UpdateModulePayload,
): Promise<Module> {
  return wrapRequest(authApi.put<Module>(`/course/${courseUuid}/module/${moduleUuid}`, data))
}

export async function deleteModule(courseUuid: string, moduleUuid: string): Promise<void> {
  await wrapRequest(authApi.delete<void>(`/course/${courseUuid}/module/${moduleUuid}`))
}

export async function reorderModules(courseUuid: string, moduleUuids: string[]): Promise<void> {
  await wrapRequest(authApi.put<void>(`/course/${courseUuid}/module/reorder`, { moduleUuids }))
}

export async function listCourseGroups(courseUuid: string): Promise<Group[]> {
  return wrapRequest(authApi.get<Group[]>(`/course/${courseUuid}/group`))
}

export async function getGroup(groupUuid: string): Promise<Group> {
  return wrapRequest(authApi.get<Group>(`/group/${groupUuid}`))
}

export async function createGroup(data: CreateGroupPayload): Promise<Group> {
  return wrapRequest(authApi.post<Group>('/group', data))
}

export async function bulkCreateGroups(data: BulkCreateGroupsPayload): Promise<Group[]> {
  return wrapRequest(authApi.post<Group[]>('/group/bulk', data))
}

export async function updateGroup(groupUuid: string, data: UpdateGroupPayload): Promise<Group> {
  return wrapRequest(authApi.patch<Group>(`/group/${groupUuid}`, data))
}

export async function deleteGroup(groupUuid: string): Promise<void> {
  await wrapRequest(authApi.delete<void>(`/group/${groupUuid}`))
}

export async function joinGroup(groupUuid: string): Promise<Group> {
  return wrapRequest(authApi.post<Group>(`/group/join/${groupUuid}`))
}

export async function leaveGroup(groupUuid: string): Promise<Group> {
  return wrapRequest(authApi.patch<Group>(`/group/leave/${groupUuid}`))
}
