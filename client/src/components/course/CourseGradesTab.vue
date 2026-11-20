<template>
  <div>
    <h2 class="text-h5 font-weight-bold text-high-emphasis mb-4">Grades</h2>

    <div v-if="loading && !overview" class="d-flex justify-center py-8">
      <v-progress-circular indeterminate />
    </div>

    <v-alert v-else-if="loadFailed" class="mb-4" type="error" variant="tonal">
      <div class="d-flex align-center justify-space-between ga-2">
        <span>Could not load your grades.</span>
        <v-btn size="small" variant="text" @click="loadOverview">Retry</v-btn>
      </div>
    </v-alert>

    <template v-else-if="overview">
      <v-table v-if="overview.items.length > 0" class="grades-table" density="comfortable">
        <thead>
          <tr>
            <th class="text-left font-weight-bold">Name</th>
            <th class="text-left font-weight-bold">Deadline</th>
            <th class="text-left font-weight-bold">Submitted</th>
            <th class="text-left font-weight-bold">Status</th>
            <th class="text-right font-weight-bold">Result</th>
            <th class="text-right font-weight-bold">Feedback</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in sortedItems" :key="item.contentUuid">
            <td class="py-3">
              <div class="d-flex align-start ga-2">
                <v-icon
                  class="mt-1"
                  :icon="
                    item.type === 'QUIZ' ? 'mdi-help-circle-outline' : 'mdi-file-upload-outline'
                  "
                  size="small"
                />
                <div>
                  <div class="font-weight-bold text-high-emphasis">{{ item.title }}</div>
                  <div class="text-caption text-medium-emphasis">
                    {{ item.type === 'QUIZ' ? 'Quiz' : 'Hand-in' }} · {{ item.moduleTitle }}
                    <template v-if="item.isGroup"> · {{ item.groupName || 'Group' }}</template>
                  </div>
                </div>
              </div>
            </td>
            <td class="text-body-2 text-no-wrap">
              {{ formatDateTime(item.deadline, 'No deadline') }}
            </td>
            <td class="text-body-2 text-no-wrap">
              {{ formatDateTime(item.submittedAt, '-') }}
            </td>
            <td>
              <v-chip :color="statusOf(item).color" size="small" variant="tonal">
                {{ statusOf(item).label }}
              </v-chip>
            </td>
            <td class="text-right text-no-wrap">
              <span
                :class="
                  item.status === 'GRADED'
                    ? 'font-weight-bold text-high-emphasis'
                    : 'text-medium-emphasis'
                "
              >
                {{ scoreLabel(item) }} / {{ formatPoints(item.maxPoints) }}
              </span>
              <v-icon
                v-if="item.status === 'GRADED' && item.passed !== undefined"
                :color="item.passed ? 'success' : 'error'"
                :icon="item.passed ? 'mdi-check-circle' : 'mdi-close-circle'"
                size="small"
              />
            </td>
            <td class="text-right">
              <v-btn
                v-if="item.feedback"
                color="primary"
                prepend-icon="mdi-comment-text-outline"
                size="small"
                variant="tonal"
                @click="openFeedback(item)"
              >
                View
              </v-btn>
              <span v-else class="text-caption text-disabled font-italic">None</span>
            </td>
          </tr>
        </tbody>
        <tfoot>
          <tr class="grades-table__total">
            <td class="py-4 text-h6 text-high-emphasis" colspan="3">
              Total
              <div class="text-caption text-medium-emphasis">
                Counts graded items and items past their deadline. The grade shows when the course
                is finished.
              </div>
            </td>
            <td>
              <div v-if="overview.finished" class="d-flex align-center ga-3">
                <span class="text-h5 font-weight-bold text-high-emphasis">
                  {{ formatPoints(overview.percent) }}%
                </span>
                <v-avatar
                  class="font-weight-bold elevation-1"
                  :color="getGradeBadgeColor(overview.finalGrade)"
                  size="36"
                >
                  {{ overview.finalGrade }}
                </v-avatar>
              </div>
              <v-chip v-else color="info" size="small" variant="tonal">In progress</v-chip>
            </td>
            <td class="text-right text-h6 font-weight-bold text-high-emphasis text-no-wrap">
              {{ formatPoints(overview.progressScore) }} /
              {{ formatPoints(overview.progressMaxScore) }}
            </td>
            <td />
          </tr>
        </tfoot>
      </v-table>

      <v-card v-else class="pa-6 bg-surface mb-6 text-center" elevation="1" rounded="lg">
        <v-icon class="mb-3 text-medium-emphasis" icon="mdi-school-outline" size="48" />
        <p class="text-body-1 text-medium-emphasis mb-0">
          No quizzes or hand-ins in this course yet.
        </p>
      </v-card>
    </template>

    <v-dialog v-model="feedbackOpen" max-width="600">
      <v-card class="bg-surface pa-2" rounded="lg">
        <v-card-title
          class="text-h6 font-weight-bold pt-4 px-4 text-high-emphasis d-flex align-center justify-space-between"
        >
          <span>Feedback: {{ activeItem?.title }}</span>
          <v-btn
            aria-label="Close feedback"
            icon="mdi-close"
            size="small"
            variant="text"
            @click="feedbackOpen = false"
          />
        </v-card-title>
        <v-card-text class="px-4 pt-3">
          <div class="pa-4 rounded border bg-surface-variant">
            <MarkdownView :source="activeItem?.feedback" />
          </div>
        </v-card-text>
        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="primary" variant="text" @click="feedbackOpen = false">Close</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import type { CourseGradeItem, CourseGradeOverview } from '@/types/grade'
import { computed, ref, watch } from 'vue'
import { getApiErrorMessage } from '@/api/client'
import { getMyCourseGrades } from '@/api/grade'
import MarkdownView from '@/components/MarkdownView.vue'
import { useToast } from '@/composables/useToast'
import { formatDateTime, parseValidDate } from '@/util/dateTime'
import { getGradeBadgeColor } from '@/util/grading'

interface Props {
  courseUuid: string
}

const props = defineProps<Props>()
const toast = useToast()

const overview = ref<CourseGradeOverview | undefined>(undefined)
const loading = ref(false)
const loadFailed = ref(false)
const feedbackOpen = ref(false)
const activeItem = ref<CourseGradeItem | undefined>(undefined)

// Ignore responses from a previous course or an earlier request.
let loadCounter = 0

// Earliest deadline first. Items without a deadline go last and keep course order (stable sort).
const sortedItems = computed(() => {
  const items = overview.value?.items ?? []
  const time = (item: CourseGradeItem) =>
    parseValidDate(item.deadline)?.getTime() ?? Number.POSITIVE_INFINITY
  const sorted = [...items]
  sorted.sort((a, b) => {
    const diff = time(a) - time(b)
    return Number.isNaN(diff) ? 0 : diff
  })
  return sorted
})

function isPastDeadline(item: CourseGradeItem): boolean {
  const deadline = parseValidDate(item.deadline)
  return deadline !== undefined && deadline.getTime() < Date.now()
}

// Missing work past its deadline scores 0. Anything else without a visible grade shows "-".
function scoreLabel(item: CourseGradeItem): string {
  if (item.status === 'GRADED') return formatPoints(item.score ?? 0)
  if (item.status === 'NOT_SUBMITTED' && isPastDeadline(item)) return formatPoints(0)
  return '-'
}

async function loadOverview() {
  const requestId = ++loadCounter
  loading.value = true
  loadFailed.value = false
  try {
    const result = await getMyCourseGrades(props.courseUuid)
    if (requestId !== loadCounter) return
    overview.value = result
  } catch (error) {
    if (requestId !== loadCounter) return
    loadFailed.value = true
    toast.error(getApiErrorMessage(error, 'Failed to load grades.'))
  } finally {
    if (requestId === loadCounter) loading.value = false
  }
}

watch(
  () => props.courseUuid,
  () => {
    overview.value = undefined
    void loadOverview()
  },
  { immediate: true },
)

function statusOf(item: CourseGradeItem): { label: string; color?: string } {
  switch (item.status) {
    case 'GRADED': {
      return { label: item.isLate ? 'Graded (late)' : 'Graded', color: 'success' }
    }
    case 'AWAITING_RELEASE': {
      return { label: 'Awaiting release', color: 'info' }
    }
    case 'SUBMITTED': {
      return { label: item.isLate ? 'Submitted (late)' : 'Submitted', color: 'info' }
    }
    default: {
      if (isPastDeadline(item)) {
        return { label: 'Missing', color: 'error' }
      }
      return { label: item.type === 'QUIZ' ? 'Not attempted' : 'Not submitted' }
    }
  }
}

const numberFormat = new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 })

function formatPoints(value: number): string {
  return numberFormat.format(value)
}

function openFeedback(item: CourseGradeItem): void {
  activeItem.value = item
  feedbackOpen.value = true
}
</script>

<style scoped>
.grades-table__total td {
  border-top: 2px solid rgba(var(--v-border-color), var(--v-border-opacity));
}
</style>
