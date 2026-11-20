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
      <div v-if="overview.items.length > 0">
        <v-table density="comfortable" hover>
          <thead>
            <tr>
              <th class="text-left font-weight-bold">Deadline</th>
              <th class="text-left font-weight-bold">Assignment</th>
              <th class="text-center font-weight-bold">Status</th>
              <th class="text-center font-weight-bold">Score</th>
              <th class="text-right font-weight-bold">Feedback</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in sortedItems" :key="item.contentUuid">
              <td class="py-3 text-body-2 font-weight-medium text-no-wrap">
                {{ formatDateTime(item.deadline, 'No deadline') }}
              </td>
              <td class="py-3">
                <div class="d-flex align-center flex-wrap ga-1">
                  <v-icon
                    :icon="
                      item.type === 'QUIZ' ? 'mdi-help-circle-outline' : 'mdi-file-upload-outline'
                    "
                    size="small"
                  />
                  <span class="font-weight-bold text-high-emphasis">{{ item.title }}</span>
                  <v-chip v-if="item.isGroup" size="x-small" variant="tonal">
                    <v-icon icon="mdi-account-group" size="x-small" start />
                    {{ item.groupName || 'Group' }}
                  </v-chip>
                  <v-chip v-if="item.isLate" color="warning" size="x-small" variant="tonal">
                    Late
                  </v-chip>
                </div>
                <div class="text-caption text-medium-emphasis">{{ item.moduleTitle }}</div>
              </td>
              <td class="text-center">
                <v-chip :color="statusOf(item).color" size="small" variant="tonal">
                  {{ statusOf(item).label }}
                </v-chip>
              </td>
              <td class="text-center">
                <span
                  :class="
                    item.status === 'GRADED'
                      ? 'font-weight-bold text-high-emphasis'
                      : 'text-medium-emphasis'
                  "
                >
                  {{ item.status === 'GRADED' ? formatPoints(item.score ?? 0) : '-' }} /
                  {{ formatPoints(item.maxPoints) }}
                </span>
                <template v-if="item.status === 'GRADED'">
                  <span class="text-caption text-medium-emphasis">
                    · {{ Math.round(item.percent ?? 0) }}%
                  </span>
                  <v-icon
                    v-if="item.passed !== undefined"
                    :color="item.passed ? 'success' : 'error'"
                    :icon="item.passed ? 'mdi-check-circle' : 'mdi-close-circle'"
                    size="small"
                  />
                </template>
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
        </v-table>
      </div>

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
      return { label: 'Graded', color: 'success' }
    }
    case 'AWAITING_RELEASE': {
      return { label: 'Awaiting release', color: 'info' }
    }
    case 'SUBMITTED': {
      return { label: 'Submitted', color: 'info' }
    }
    default: {
      const deadline = parseValidDate(item.deadline)
      if (deadline && deadline.getTime() < Date.now()) {
        return { label: 'Missing', color: 'error' }
      }
      return { label: item.type === 'QUIZ' ? 'Not attempted' : 'Not submitted' }
    }
  }
}

function formatPoints(value: number): string {
  return Number.isInteger(value) ? String(value) : value.toFixed(1)
}

function openFeedback(item: CourseGradeItem): void {
  activeItem.value = item
  feedbackOpen.value = true
}
</script>
