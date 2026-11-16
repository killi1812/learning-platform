import type {
  AddEnrollmentPayload,
  BulkCreateGroupsPayload,
  Course,
  CourseTerm,
  CreateCoursePayload,
  CreateGroupPayload,
  CreateModulePayload,
  Enrollment,
  Group,
  Module,
  MyCourse,
  UpdateCoursePayload,
  UpdateGroupPayload,
  UpdateModulePayload,
} from '@/types/course'
import { authApi, request } from './client'

const enc = encodeURIComponent

export function listCourses(term?: CourseTerm, year?: number): Promise<Course[]> {
  const params: Record<string, string> = {}
  if (term) {
    params.term = term
  }
  if (year !== undefined) {
    params.year = year.toString()
  }
  return request(authApi.get<Course[]>('/course', { params }))
}

export function getCourse(uuid: string): Promise<Course> {
  return request(authApi.get<Course>(`/course/${enc(uuid)}`))
}

export function getMyCourses(): Promise<MyCourse[]> {
  return request(authApi.get<MyCourse[]>('/course/my-course'))
}

export function createCourse(data: CreateCoursePayload): Promise<Course> {
  return request(authApi.post<Course>('/course', data))
}

export function updateCourse(uuid: string, data: UpdateCoursePayload): Promise<Course> {
  return request(authApi.put<Course>(`/course/${enc(uuid)}`, data))
}

export async function deleteCourse(uuid: string): Promise<void> {
  await request(authApi.delete<void>(`/course/${enc(uuid)}`))
}

export function selfEnroll(courseUuid: string): Promise<Enrollment> {
  return request(authApi.post<Enrollment>(`/course/${enc(courseUuid)}/enroll`))
}

export async function withdraw(courseUuid: string): Promise<void> {
  await request(authApi.delete<void>(`/course/${enc(courseUuid)}/enroll`))
}

export function getCourseEnrollments(courseUuid: string): Promise<Enrollment[]> {
  return request(authApi.get<Enrollment[]>(`/course/${enc(courseUuid)}/enrollments`))
}

export function addEnrollment(courseUuid: string, data: AddEnrollmentPayload): Promise<Enrollment> {
  return request(authApi.post<Enrollment>(`/course/${enc(courseUuid)}/enrollments`, data))
}

export async function removeEnrollment(courseUuid: string, userUuid: string): Promise<void> {
  await request(authApi.delete<void>(`/course/${enc(courseUuid)}/enrollments/${enc(userUuid)}`))
}

export function listModules(courseUuid: string): Promise<Module[]> {
  return request(authApi.get<Module[]>(`/course/${enc(courseUuid)}/module`))
}

export function getModule(courseUuid: string, moduleUuid: string): Promise<Module> {
  return request(authApi.get<Module>(`/course/${enc(courseUuid)}/module/${enc(moduleUuid)}`))
}

export function createModule(courseUuid: string, data: CreateModulePayload): Promise<Module> {
  return request(authApi.post<Module>(`/course/${enc(courseUuid)}/module`, data))
}

export function updateModule(
  courseUuid: string,
  moduleUuid: string,
  data: UpdateModulePayload,
): Promise<Module> {
  return request(authApi.put<Module>(`/course/${enc(courseUuid)}/module/${enc(moduleUuid)}`, data))
}

export async function deleteModule(courseUuid: string, moduleUuid: string): Promise<void> {
  await request(authApi.delete<void>(`/course/${enc(courseUuid)}/module/${enc(moduleUuid)}`))
}

export async function reorderModules(courseUuid: string, moduleUuids: string[]): Promise<void> {
  await request(authApi.put<void>(`/course/${enc(courseUuid)}/module/reorder`, { moduleUuids }))
}

export function listCourseGroups(courseUuid: string): Promise<Group[]> {
  return request(authApi.get<Group[]>(`/course/${enc(courseUuid)}/group`))
}

export function getGroup(groupUuid: string): Promise<Group> {
  return request(authApi.get<Group>(`/group/${enc(groupUuid)}`))
}

export function createGroup(data: CreateGroupPayload): Promise<Group> {
  return request(authApi.post<Group>('/group', data))
}

export function bulkCreateGroups(data: BulkCreateGroupsPayload): Promise<Group[]> {
  return request(authApi.post<Group[]>('/group/bulk', data))
}

export function updateGroup(groupUuid: string, data: UpdateGroupPayload): Promise<Group> {
  return request(authApi.patch<Group>(`/group/${enc(groupUuid)}`, data))
}

export async function deleteGroup(groupUuid: string): Promise<void> {
  await request(authApi.delete<void>(`/group/${enc(groupUuid)}`))
}

export function joinGroup(groupUuid: string): Promise<Group> {
  return request(authApi.post<Group>(`/group/join/${enc(groupUuid)}`))
}

export function leaveGroup(groupUuid: string): Promise<Group> {
  return request(authApi.patch<Group>(`/group/leave/${enc(groupUuid)}`))
}
