<template>
  <div>
    <v-card class="bg-surface pa-4 mb-4" elevation="1" rounded="lg">
      <div class="d-flex flex-wrap align-center justify-space-between ga-3 mb-4">
        <div class="d-flex align-center ga-2">
          <span class="text-h6 font-weight-bold text-high-emphasis">Pending Grading Queue</span>
          <v-chip color="warning" size="small" variant="flat">
            {{ filteredPending.length }} pending
          </v-chip>
        </div>

        <v-select
          v-if="gradingCourses.length > 1"
          v-model="selectedCourseUuid"
          class="course-filter-select"
          clearable
          density="compact"
          hide-details
          :items="courseFilterOptions"
          label="Filter Course"
          variant="outlined"
        />
      </div>

      <div v-if="filteredPending.length === 0" class="py-8 text-center text-medium-emphasis">
        <v-icon class="mb-2 text-success" icon="mdi-check-all" size="48" />
        <p class="text-body-1 font-weight-medium mb-1">All caught up!</p>
        <p class="text-caption">No submissions are currently waiting to be graded.</p>
      </div>

      <v-table v-else density="comfortable" hover>
        <thead>
          <tr>
            <th class="text-left font-weight-bold">Course</th>
            <th class="text-left font-weight-bold">Assignment</th>
            <th class="text-left font-weight-bold">Student</th>
            <th class="text-center font-weight-bold">Attempt</th>
            <th class="text-left font-weight-bold">Submitted</th>
            <th class="text-center font-weight-bold">Timing</th>
            <th class="text-right font-weight-bold">Action</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="item in filteredPending" :key="item.submissionUuid">
            <td class="py-3">
              <div class="font-weight-bold text-high-emphasis">{{ item.courseName }}</div>
              <div class="text-caption text-medium-emphasis">{{ item.courseCode }}</div>
            </td>
            <td>
              <div class="font-weight-medium">{{ item.assignmentTitle }}</div>
              <div class="text-caption text-medium-emphasis">Max: {{ item.maxPoints }} pts</div>
            </td>
            <td>
              <div class="text-body-2">{{ item.userEmail }}</div>
              <div v-if="item.userName" class="text-caption text-medium-emphasis">
                {{ item.userName }}
              </div>
            </td>
            <td class="text-center">
              <v-chip color="primary" size="x-small" variant="flat"> #{{ item.attempt }} </v-chip>
            </td>
            <td class="text-caption text-medium-emphasis">
              {{ formatDateTime(item.submittedAt) }}
            </td>
            <td class="text-center">
              <v-chip :color="item.isLate ? 'warning' : 'success'" size="x-small" variant="tonal">
                {{ item.isLate ? 'Late' : 'On Time' }}
              </v-chip>
            </td>
            <td class="text-right">
              <v-btn
                color="primary"
                prepend-icon="mdi-check-decagram"
                size="small"
                variant="elevated"
                @click="openGradeModal(item)"
              >
                Grade
              </v-btn>
            </td>
          </tr>
        </tbody>
      </v-table>
    </v-card>

    <GradeDialog
      v-model="gradeDialogOpen"
      :assignment-title="gradingItem?.assignmentTitle"
      :attempt="gradingItem?.attempt"
      :content-uuid="gradingItem?.contentUuid || ''"
      :course-uuid="gradingItem?.courseUuid || ''"
      :max-points="gradingItem?.maxPoints || 100"
      :student-email="gradingItem?.userEmail"
      :submission-uuid="gradingItem?.submissionUuid || ''"
      @saved="onGradeSaved"
    />
  </div>
</template>

<script setup lang="ts">
import type { Course } from '@/types/course'
import type { PendingSubmissionDto } from '@/types/grade'
import { computed, ref } from 'vue'
import GradeDialog from '@/components/GradeDialog.vue'
import { formatDateTime } from '@/util/dateTime'

interface Props {
  gradingCourses: readonly Course[]
  pendingSubmissions: readonly PendingSubmissionDto[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  graded: [submissionUuid: string]
}>()

const selectedCourseUuid = ref<string | undefined>(undefined)
const gradeDialogOpen = ref(false)
const gradingItem = ref<PendingSubmissionDto | undefined>(undefined)

const courseFilterOptions = computed(() =>
  props.gradingCourses.map(c => ({
    title: c.title,
    value: c.uuid,
  })),
)

const filteredPending = computed(() => {
  if (!selectedCourseUuid.value) {
    return props.pendingSubmissions
  }
  return props.pendingSubmissions.filter(p => p.courseUuid === selectedCourseUuid.value)
})

function openGradeModal(item: PendingSubmissionDto): void {
  gradingItem.value = item
  gradeDialogOpen.value = true
}

function onGradeSaved(): void {
  if (!gradingItem.value) {
    return
  }
  emit('graded', gradingItem.value.submissionUuid)
}
</script>

<style scoped>
.course-filter-select {
  max-width: 280px;
}
</style>
