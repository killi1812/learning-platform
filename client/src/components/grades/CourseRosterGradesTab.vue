<template>
  <v-card class="bg-surface pa-4 mb-4" elevation="1" rounded="lg">
    <div class="d-flex flex-wrap align-center justify-space-between ga-3 mb-4">
      <div class="d-flex align-center ga-2">
        <span class="text-h6 font-weight-bold text-high-emphasis">Course Final Grades Roster</span>
        <v-chip v-if="rosterStudents.length > 0" color="primary" size="small" variant="tonal">
          {{ rosterStudents.length }} students
        </v-chip>
      </div>

      <div class="d-flex align-center ga-3">
        <v-select
          v-model="selectedCourseUuid"
          class="roster-course-select"
          density="compact"
          hide-details
          :items="courseFilterOptions"
          label="Select Course"
          variant="outlined"
        />

        <v-btn
          v-if="selectedCourseUuid"
          color="primary"
          :loading="exporting"
          prepend-icon="mdi-file-delimited-outline"
          variant="outlined"
          @click="emit('export-selected')"
        >
          Export CSV
        </v-btn>
      </div>
    </div>

    <div v-if="loadingRoster" class="d-flex justify-center py-12">
      <v-progress-circular color="primary" indeterminate size="48" />
    </div>

    <div v-else-if="!selectedCourseUuid" class="py-8 text-center text-medium-emphasis">
      <v-icon class="mb-2 text-disabled" icon="mdi-book-search-outline" size="48" />
      <p class="text-body-1 font-weight-medium mb-1">Select a course</p>
      <p class="text-caption">
        Select a course above to view students' cumulative standing and final grades.
      </p>
    </div>

    <div v-else-if="rosterStudents.length === 0" class="py-8 text-center text-medium-emphasis">
      <v-icon class="mb-2 text-disabled" icon="mdi-account-off-outline" size="48" />
      <p class="text-body-1 font-weight-medium mb-1">No enrolled students</p>
      <p class="text-caption">There are currently no students enrolled in this course.</p>
    </div>

    <v-table v-else density="comfortable" hover>
      <thead>
        <tr>
          <th class="text-left font-weight-bold">Student</th>
          <th class="text-left font-weight-bold">Email</th>
          <th class="text-center font-weight-bold">Total Points Earned</th>
          <th class="text-center font-weight-bold">Max Possible Points</th>
          <th class="text-center font-weight-bold">Cumulative %</th>
          <th class="text-center font-weight-bold">Final Grade</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="st in rosterStudents" :key="st.userUuid">
          <td class="font-weight-medium text-high-emphasis">
            {{ st.userName || '—' }}
          </td>
          <td class="text-body-2 text-medium-emphasis">{{ st.userEmail }}</td>
          <td class="text-center font-weight-bold">{{ st.totalScore.toFixed(1) }}</td>
          <td class="text-center text-medium-emphasis">{{ st.totalMaxScore }}</td>
          <td class="text-center">
            <v-chip :color="st.percent >= 40 ? 'primary' : 'error'" size="small" variant="tonal">
              {{ st.percent.toFixed(1) }}%
            </v-chip>
          </td>
          <td class="text-center">
            <v-chip
              class="font-weight-bold px-3"
              :color="getGradeBadgeColor(st.finalGrade)"
              size="small"
              variant="flat"
            >
              {{ st.finalGrade }}
            </v-chip>
          </td>
        </tr>
      </tbody>
    </v-table>
  </v-card>
</template>

<script setup lang="ts">
import type { Course } from '@/types/course'
import type { StudentFinalGradeRowDto } from '@/types/grade'
import { computed } from 'vue'
import { getGradeBadgeColor } from '@/util/grading'

interface Props {
  gradingCourses: readonly Course[]
  rosterStudents: readonly StudentFinalGradeRowDto[]
  loadingRoster: boolean
  exporting: boolean
}

const props = defineProps<Props>()

const selectedCourseUuid = defineModel<string | undefined>('selectedCourseUuid')

const emit = defineEmits<{
  'export-selected': []
}>()

const courseFilterOptions = computed(() =>
  props.gradingCourses.map(c => ({
    title: c.title,
    value: c.uuid,
  })),
)
</script>

<style scoped>
.roster-course-select {
  min-width: 240px;
}
</style>
