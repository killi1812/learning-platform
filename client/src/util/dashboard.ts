import type { MyCourse } from '@/types/course'
import type { DeadlineItem, DeadlineUrgency } from '@/types/dashboard'
import type { Assignment } from '@/types/quiz'
import { parseValidDate } from '@/util/dateTime'

const HOUR_MS = 3_600_000
const DAY_MS = 24 * HOUR_MS
const SOON_THRESHOLD_MS = 2 * DAY_MS

/** Flattens per-course assignments into deadline items, earliest deadline first. Skips unpublished and undated ones. */
export function toDeadlineItems(
  entries: readonly { course: MyCourse; assignments: readonly Assignment[] }[],
): DeadlineItem[] {
  const items: DeadlineItem[] = []
  for (const { course, assignments } of entries) {
    for (const assignment of assignments) {
      const deadline = assignment.deadline
      if (!assignment.published || deadline === undefined || !parseValidDate(deadline)) {
        continue
      }
      items.push({
        uuid: assignment.uuid,
        title: assignment.title,
        type: assignment.type,
        courseUuid: course.course.uuid,
        courseTitle: course.course.title,
        role: course.role,
        deadline,
        attemptsUsed: assignment.attemptsUsed,
        maxAttempts: assignment.maxAttempts,
        quizFinished: assignment.myResult !== undefined,
      })
    }
  }
  items.sort((a, b) => Date.parse(a.deadline) - Date.parse(b.deadline))
  return items
}

export function deadlineUrgency(iso: string, now: Date): DeadlineUrgency {
  const diff = Date.parse(iso) - now.getTime()
  if (diff < 0) {
    return 'overdue'
  }
  return diff <= SOON_THRESHOLD_MS ? 'soon' : 'later'
}

/** Short relative label: "in 3 d", "in 5 h", "2 d ago". */
export function formatTimeLeft(iso: string, now: Date): string {
  const diff = Date.parse(iso) - now.getTime()
  const abs = Math.abs(diff)
  const days = Math.floor(abs / DAY_MS)
  const value = days >= 1 ? `${days} d` : `${Math.max(1, Math.floor(abs / HOUR_MS))} h`
  return diff < 0 ? `${value} ago` : `in ${value}`
}
