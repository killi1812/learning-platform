import type { Group, MyCourse } from '@/types/course'
import type { DeadlineItem, SubmissionActivityItem } from '@/types/dashboard'
import type { CourseFinalGradeDto, PendingSubmissionDto } from '@/types/grade'
import type { Assignment } from '@/types/quiz'
import type { Submission } from '@/types/submission'
import type { Ref } from 'vue'
import { computed, onBeforeUnmount, ref } from 'vue'
import { getApiErrorMessage } from '@/api/client'
import { getCourseEnrollments, listCourseGroups } from '@/api/course'
import { getMyFinalGrades, listPendingGrades } from '@/api/grade'
import { listAssignments } from '@/api/quiz'
import { listAssignmentSubmissions, listMySubmissions } from '@/api/submission'
import { useAuth } from '@/composables/useAuth'
import { toDeadlineItems } from '@/util/dashboard'

const ACTIVITY_WINDOW_MS = 14 * 24 * 3_600_000
const MAX_CHECKED_HANDINS = 8
const MAX_ACTIVITY_ITEMS = 6

interface Section<T> {
  data: Ref<T>
  loading: Ref<boolean>
  error: Ref<string | undefined>
}

function createSection<T>(initial: T): Section<T> {
  return { data: ref(initial) as Ref<T>, loading: ref(false), error: ref(undefined) }
}

/** Distinct people covered by a list of submissions (group submissions count every member). */
function distinctSubmitters(submissions: readonly Submission[], onlyLate = false): number {
  const people = new Set<string>()
  for (const submission of submissions) {
    if (onlyLate && !submission.isLate) {
      continue
    }
    const emails = submission.isGroup ? (submission.memberEmails ?? []) : [submission.userEmail]
    for (const email of emails) {
      if (email) {
        people.add(email)
      }
    }
  }
  return people.size
}

export function useDashboardData(courses: Ref<MyCourse[]>) {
  const { currentUser } = useAuth()

  const assignments = createSection<DeadlineItem[]>([])
  const progress = createSection<CourseFinalGradeDto[]>([])
  const groups = createSection<{ courseTitle: string; group: Group }[]>([])
  const pending = createSection<PendingSubmissionDto[]>([])
  const submittedUuids = createSection<Set<string>>(new Set())
  const activity = createSection<SubmissionActivityItem[]>([])

  const studentCourses = computed(() => courses.value.filter(c => c.role === 'STUDENT'))
  const teacherCourses = computed(() => courses.value.filter(c => c.role === 'TEACHER'))

  let isUnmounted = false
  let requestCounter = 0

  onBeforeUnmount(() => {
    isUnmounted = true
  })

  /** Runs one section load. A stale or unmounted call writes nothing. */
  async function run<T>(
    section: Section<T>,
    requestId: number,
    fallback: string,
    load: () => Promise<T>,
  ): Promise<void> {
    section.loading.value = true
    section.error.value = undefined
    try {
      const result = await load()
      if (!isUnmounted && requestId === requestCounter) {
        section.data.value = result
      }
    } catch (error: unknown) {
      if (!isUnmounted && requestId === requestCounter) {
        section.error.value = getApiErrorMessage(error, fallback)
      }
    } finally {
      if (!isUnmounted && requestId === requestCounter) {
        section.loading.value = false
      }
    }
  }

  async function loadAssignments(): Promise<DeadlineItem[]> {
    const mine = courses.value
    const results = await Promise.allSettled(mine.map(c => listAssignments(c.course.uuid)))
    const entries: { course: MyCourse; assignments: Assignment[] }[] = []
    for (const [index, result] of results.entries()) {
      const course = mine[index]
      if (course && result.status === 'fulfilled') {
        entries.push({ course, assignments: result.value })
      }
    }
    if (mine.length > 0 && entries.length === 0) {
      throw new Error('Failed to load assignments.')
    }
    return toDeadlineItems(entries)
  }

  async function loadSubmitted(items: readonly DeadlineItem[]): Promise<Set<string>> {
    const now = Date.now()
    const upcoming = items
      .filter(i => i.role === 'STUDENT' && i.type === 'HANDIN' && Date.parse(i.deadline) >= now)
      .slice(0, MAX_CHECKED_HANDINS)
    const results = await Promise.allSettled(
      upcoming.map(i => listMySubmissions(i.courseUuid, i.uuid)),
    )
    const done = new Set<string>()
    for (const [index, result] of results.entries()) {
      const item = upcoming[index]
      if (item && result.status === 'fulfilled' && result.value.length > 0) {
        done.add(item.uuid)
      }
    }
    return done
  }

  async function loadGroups(): Promise<{ courseTitle: string; group: Group }[]> {
    const userUuid = currentUser.value?.userUuid
    if (!userUuid) {
      return []
    }
    const results = await Promise.allSettled(
      studentCourses.value.map(c => listCourseGroups(c.course.uuid)),
    )
    const mine: { courseTitle: string; group: Group }[] = []
    for (const [index, result] of results.entries()) {
      const course = studentCourses.value[index]
      if (!course || result.status !== 'fulfilled') {
        continue
      }
      for (const group of result.value) {
        if (group.members.some(member => member.uuid === userUuid)) {
          mine.push({ courseTitle: course.course.title, group })
        }
      }
    }
    return mine
  }

  async function loadActivity(items: readonly DeadlineItem[]): Promise<SubmissionActivityItem[]> {
    const now = Date.now()
    const teaching = new Set(teacherCourses.value.map(c => c.course.uuid))
    const candidates = items.filter(
      i =>
        teaching.has(i.courseUuid) &&
        i.type === 'HANDIN' &&
        Math.abs(Date.parse(i.deadline) - now) <= ACTIVITY_WINDOW_MS,
    )
    candidates.sort(
      (a, b) => Math.abs(Date.parse(a.deadline) - now) - Math.abs(Date.parse(b.deadline) - now),
    )
    const nearby = candidates.slice(0, MAX_ACTIVITY_ITEMS)

    const courseUuids = [...new Set(nearby.map(i => i.courseUuid))]
    const enrollmentResults = await Promise.allSettled(
      courseUuids.map(uuid => getCourseEnrollments(uuid)),
    )
    const enrolledByCourse = new Map<string, number>()
    for (const [index, result] of enrollmentResults.entries()) {
      const uuid = courseUuids[index]
      if (uuid && result.status === 'fulfilled') {
        enrolledByCourse.set(uuid, result.value.filter(e => e.role === 'STUDENT').length)
      }
    }

    const submissionResults = await Promise.allSettled(
      nearby.map(i => listAssignmentSubmissions(i.courseUuid, i.uuid)),
    )
    const rows: SubmissionActivityItem[] = []
    for (const [index, result] of submissionResults.entries()) {
      const item = nearby[index]
      if (!item || result.status !== 'fulfilled') {
        continue
      }
      rows.push({
        uuid: item.uuid,
        title: item.title,
        courseUuid: item.courseUuid,
        courseTitle: item.courseTitle,
        deadline: item.deadline,
        submitted: distinctSubmitters(result.value),
        late: distinctSubmitters(result.value, true),
        enrolled: enrolledByCourse.get(item.courseUuid) ?? 0,
      })
    }
    rows.sort((a, b) => Date.parse(a.deadline) - Date.parse(b.deadline))
    return rows
  }

  async function load(): Promise<void> {
    const requestId = ++requestCounter
    await run(assignments, requestId, 'Failed to load deadlines.', loadAssignments)
    if (isUnmounted || requestId !== requestCounter) {
      return
    }
    const items = assignments.data.value
    const tasks: Promise<void>[] = []
    if (studentCourses.value.length > 0) {
      tasks.push(
        run(progress, requestId, 'Failed to load course progress.', () => getMyFinalGrades()),
        run(groups, requestId, 'Failed to load groups.', loadGroups),
        run(submittedUuids, requestId, 'Failed to check submissions.', () => loadSubmitted(items)),
      )
    }
    if (teacherCourses.value.length > 0) {
      tasks.push(
        run(pending, requestId, 'Failed to load grading queue.', () => listPendingGrades()),
        run(activity, requestId, 'Failed to load submission activity.', () => loadActivity(items)),
      )
    }
    await Promise.all(tasks)
  }

  return {
    assignments,
    progress,
    groups,
    pending,
    submittedUuids,
    activity,
    studentCourses,
    teacherCourses,
    load,
  }
}
