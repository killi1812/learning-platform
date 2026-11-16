<template>
  <div>
    <v-card class="bg-surface pa-4 mb-4" elevation="1" rounded="lg">
      <div class="d-flex flex-wrap align-center justify-space-between ga-3 mb-4">
        <div class="text-h6 font-weight-bold text-high-emphasis">My Course Grades</div>
        <v-select
          v-if="studentCourses.length > 1"
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

      <!-- Final Subject Grade Summary Cards -->
      <div v-if="filteredFinalGrades.length > 0" class="mb-6">
        <v-row dense>
          <v-col
            v-for="fg in filteredFinalGrades"
            :key="fg.courseUuid"
            cols="12"
            :md="filteredFinalGrades.length > 1 ? 6 : 12"
          >
            <v-card border class="bg-surface-variant pa-4" elevation="0" rounded="lg">
              <div class="d-flex align-center justify-space-between flex-wrap ga-2">
                <div>
                  <div class="text-caption text-medium-emphasis text-uppercase font-weight-bold">
                    Final Subject Evaluation · {{ fg.courseCode }}
                  </div>
                  <div class="text-h6 font-weight-bold text-high-emphasis">
                    {{ fg.courseName }}
                  </div>
                </div>

                <div class="d-flex align-center ga-3">
                  <div class="text-right">
                    <div class="text-h5 font-weight-bold text-high-emphasis">
                      {{ fg.totalScore.toFixed(1) }} / {{ fg.totalMaxScore }} pts
                    </div>
                    <div class="text-caption text-medium-emphasis">
                      {{ fg.percent.toFixed(1) }}% Cumulative
                    </div>
                  </div>

                  <v-avatar
                    class="text-h5 font-weight-bold elevation-1"
                    :color="getGradeBadgeColor(fg.finalGrade)"
                    size="48"
                  >
                    {{ fg.finalGrade }}
                  </v-avatar>
                </div>
              </div>

              <v-divider class="my-3" />

              <div class="d-flex align-center text-caption text-medium-emphasis">
                <v-icon class="mr-1" icon="mdi-scale-balance" size="16" />
                <span>
                  Scale: A &ge; {{ fg.gradingScale.scaleA }}%, B &ge; {{ fg.gradingScale.scaleB }}%,
                  C &ge; {{ fg.gradingScale.scaleC }}%, D &ge; {{ fg.gradingScale.scaleD }}%, E &ge;
                  {{ fg.gradingScale.scaleE }}% (F &lt; {{ fg.gradingScale.scaleE }}%)
                </span>
              </div>
            </v-card>
          </v-col>
        </v-row>
      </div>

      <div v-if="filteredGrades.length === 0" class="py-8 text-center text-medium-emphasis">
        <v-icon class="mb-2 text-disabled" icon="mdi-certificate-outline" size="48" />
        <p class="text-body-1 font-weight-medium mb-1">No released grades yet</p>
        <p class="text-caption">
          Grades for completed assignments and quizzes will appear here once released by
          instructors.
        </p>
      </div>

      <v-table v-else density="comfortable" hover>
        <thead>
          <tr>
            <th class="text-left font-weight-bold">Course / Module</th>
            <th class="text-left font-weight-bold">Assignment</th>
            <th class="text-center font-weight-bold">Type</th>
            <th class="text-center font-weight-bold">Attempt</th>
            <th class="text-center font-weight-bold">Score</th>
            <th class="text-center font-weight-bold">Percentage</th>
            <th class="text-left font-weight-bold">Graded Date</th>
            <th class="text-right font-weight-bold">Feedback</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="g in filteredGrades" :key="`${g.type}-${g.contentUuid}-${g.attempt}`">
            <td class="py-3">
              <div class="font-weight-bold text-high-emphasis">{{ g.courseName }}</div>
              <div class="text-caption text-medium-emphasis">{{ g.moduleTitle }}</div>
            </td>
            <td class="font-weight-medium">{{ g.assignmentTitle }}</td>
            <td class="text-center">
              <v-chip
                :color="g.type === 'QUIZ' ? 'info' : 'secondary'"
                size="x-small"
                variant="tonal"
              >
                {{ g.type === 'QUIZ' ? 'Quiz' : 'Hand-in' }}
              </v-chip>
            </td>
            <td class="text-center">
              <v-chip color="primary" size="x-small" variant="flat"> #{{ g.attempt }} </v-chip>
            </td>
            <td class="text-center font-weight-bold">
              {{ g.score.toFixed(1) }} / {{ g.maxPoints }}
            </td>
            <td class="text-center">
              <v-chip :color="g.percent >= 50 ? 'success' : 'error'" size="x-small" variant="flat">
                {{ g.percent.toFixed(1) }}%
              </v-chip>
            </td>
            <td class="text-caption text-medium-emphasis">
              {{ formatDateTime(g.gradedAt) }}
            </td>
            <td class="text-right">
              <v-btn
                v-if="g.feedback"
                color="primary"
                prepend-icon="mdi-comment-text-outline"
                size="small"
                variant="tonal"
                @click="openFeedbackModal(g)"
              >
                View
              </v-btn>
              <span v-else class="text-caption text-disabled font-italic">None</span>
            </td>
          </tr>
        </tbody>
      </v-table>
    </v-card>

    <!-- Student Feedback Dialog -->
    <v-dialog v-model="feedbackDialogOpen" max-width="600">
      <v-card class="bg-surface pa-2" rounded="lg">
        <v-card-title
          class="text-h6 font-weight-bold pt-4 px-4 text-high-emphasis d-flex align-center justify-space-between"
        >
          <span>Feedback: {{ activeFeedbackGrade?.assignmentTitle }}</span>
          <v-btn
            aria-label="Close feedback"
            icon="mdi-close"
            size="small"
            variant="text"
            @click="feedbackDialogOpen = false"
          />
        </v-card-title>
        <v-card-subtitle class="px-4 text-medium-emphasis d-flex align-center ga-2">
          <span>Attempt #{{ activeFeedbackGrade?.attempt }}</span>
          <span>&middot;</span>
          <v-chip color="primary" size="x-small" variant="flat">
            {{ activeFeedbackGrade?.score.toFixed(1) }} / {{ activeFeedbackGrade?.maxPoints }} pts
            ({{ activeFeedbackGrade?.percent.toFixed(1) }}%)
          </v-chip>
        </v-card-subtitle>
        <v-card-text class="px-4 pt-3">
          <div class="pa-4 rounded border bg-surface-variant">
            <MarkdownView :source="activeFeedbackGrade?.feedback" />
          </div>
        </v-card-text>
        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="primary" variant="text" @click="feedbackDialogOpen = false"> Close </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import type { MyCourse } from '@/types/course'
import type { CourseFinalGradeDto, StudentGradeDto } from '@/types/grade'
import { computed, ref } from 'vue'
import MarkdownView from '@/components/MarkdownView.vue'
import { formatDateTime } from '@/util/dateTime'
import { getGradeBadgeColor } from '@/util/grading'

interface Props {
  studentCourses: readonly MyCourse[]
  myGrades: readonly StudentGradeDto[]
  myFinalGrades: readonly CourseFinalGradeDto[]
}

const props = defineProps<Props>()

const selectedCourseUuid = ref<string | undefined>(undefined)
const feedbackDialogOpen = ref(false)
const activeFeedbackGrade = ref<StudentGradeDto | undefined>(undefined)

const courseFilterOptions = computed(() =>
  props.studentCourses.map(c => ({
    title: c.course.title,
    value: c.course.uuid,
  })),
)

const filteredGrades = computed(() => {
  if (!selectedCourseUuid.value) {
    return props.myGrades
  }
  return props.myGrades.filter(g => g.courseUuid === selectedCourseUuid.value)
})

const filteredFinalGrades = computed(() => {
  if (!selectedCourseUuid.value) {
    return props.myFinalGrades
  }
  return props.myFinalGrades.filter(g => g.courseUuid === selectedCourseUuid.value)
})

function openFeedbackModal(grade: StudentGradeDto): void {
  activeFeedbackGrade.value = grade
  feedbackDialogOpen.value = true
}
</script>

<style scoped>
.course-filter-select {
  max-width: 280px;
}
</style>
