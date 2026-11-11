<template>
  <v-container class="py-6" max-width="1400">
    <div class="d-flex flex-wrap align-center justify-space-between mb-6 ga-4">
      <div>
        <h1 class="text-h4 font-weight-bold text-high-emphasis my-0">My Assignments</h1>
        <p class="text-subtitle-1 text-medium-emphasis mb-0">
          All quizzes, exercises, and hand-ins grouped by course.
        </p>
      </div>
      <v-btn
        color="primary"
        prepend-icon="mdi-book-open-outline"
        size="large"
        variant="elevated"
        to="/courses"
      >
        My Courses
      </v-btn>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="d-flex justify-center py-12">
      <v-progress-circular color="primary" indeterminate size="64" />
    </div>

    <!-- Empty State -->
    <v-card
      v-else-if="courseAssignments.length === 0"
      class="pa-12 text-center bg-surface-variant"
      elevation="0"
      rounded="lg"
    >
      <v-icon class="mb-4 text-medium-emphasis" icon="mdi-clipboard-text-outline" size="64" />
      <h3 class="text-h6 font-weight-medium text-high-emphasis">No assignments found</h3>
      <p class="text-body-2 text-medium-emphasis mb-6">
        There are currently no assignments or quizzes published in your enrolled courses.
      </p>
      <v-btn color="primary" prepend-icon="mdi-book-open-outline" variant="elevated" to="/courses">
        Go to My Courses
      </v-btn>
    </v-card>

    <!-- Grouped Assignments by Course -->
    <div v-else class="d-flex flex-column ga-6">
      <v-card
        v-for="group in courseAssignments"
        :key="group.course.uuid"
        class="bg-surface pa-6"
        elevation="2"
        rounded="lg"
      >
        <div class="d-flex flex-wrap align-center justify-space-between mb-4 ga-2">
          <div class="d-flex align-center ga-3">
            <v-avatar color="primary" size="40" variant="tonal">
              <v-icon icon="mdi-book-open-variant" size="small" />
            </v-avatar>
            <div>
              <h2 class="text-h6 font-weight-bold text-high-emphasis my-0">
                {{ group.course.title }}
              </h2>
              <span class="text-caption text-medium-emphasis">
                {{ group.course.term }} {{ group.course.year }}
              </span>
            </div>
          </div>

          <div class="d-flex align-center ga-2">
            <v-chip
              :color="group.role === 'TEACHER' ? 'primary' : 'secondary'"
              size="small"
              variant="tonal"
            >
              {{ group.role === 'TEACHER' ? 'Teaching' : 'Enrolled' }}
            </v-chip>
            <v-btn
              color="primary"
              size="small"
              variant="text"
              prepend-icon="mdi-arrow-right"
              :to="`/courses/${group.course.uuid}`"
            >
              Course
            </v-btn>
          </div>
        </div>

        <v-divider class="mb-4" />

        <div class="d-flex flex-column ga-2">
          <QuizListItem
            v-for="assignment in group.assignments"
            :key="assignment.uuid"
            :assignment="assignment"
            :is-teacher="group.role === 'TEACHER'"
            show-module
            @take="handleTakeQuiz(group.course.uuid, assignment)"
            @edit="handleEditQuiz(group.course.uuid, assignment)"
            @results="handleResultsQuiz(group.course.uuid, assignment)"
            @delete="handleDeleteQuiz(group.course.uuid, assignment)"
          />
        </div>
      </v-card>
    </div>
  </v-container>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getMyCourses } from '@/api/course'
import { listAssignments, deleteQuiz } from '@/api/quiz'
import { useToast } from '@/composables/useToast'
import QuizListItem from '@/components/QuizListItem.vue'
import type { Course, CourseRole } from '@/types/course'
import type { Assignment } from '@/types/quiz'

interface CourseAssignmentGroup {
  course: Course
  role: CourseRole
  assignments: Assignment[]
}

const router = useRouter()
const toast = useToast()

const loading = ref(false)
const courseAssignments = ref<CourseAssignmentGroup[]>([])

async function loadAssignments() {
  loading.value = true
  try {
    const enrollments = await getMyCourses()
    const results: CourseAssignmentGroup[] = []

    const fetchPromises = enrollments.map(async enrollment => {
      try {
        const assignments = await listAssignments(enrollment.course.uuid)
        if (assignments && assignments.length > 0) {
          results.push({
            course: enrollment.course,
            role: enrollment.role,
            assignments,
          })
        }
      } catch {
        // If an assignment call fails for one course, keep the others
      }
    })

    await Promise.all(fetchPromises)

    // Sort by course title for consistent presentation
    results.sort((a, b) => a.course.title.localeCompare(b.course.title))
    courseAssignments.value = results
  } catch (err: unknown) {
    courseAssignments.value = []
    toast.error(err instanceof Error ? err.message : 'Failed to load assignments.')
  } finally {
    loading.value = false
  }
}

function handleTakeQuiz(courseUuid: string, assignment: Assignment) {
  router.push({
    name: 'QuizTake',
    params: {
      uuid: courseUuid,
      moduleUuid: assignment.moduleUuid,
      quizUuid: assignment.uuid,
    },
  })
}

function handleEditQuiz(courseUuid: string, assignment: Assignment) {
  router.push({
    name: 'QuizEdit',
    params: {
      uuid: courseUuid,
      moduleUuid: assignment.moduleUuid,
      quizUuid: assignment.uuid,
    },
  })
}

function handleResultsQuiz(courseUuid: string, assignment: Assignment) {
  router.push({
    name: 'QuizResults',
    params: {
      uuid: courseUuid,
      moduleUuid: assignment.moduleUuid,
      quizUuid: assignment.uuid,
    },
  })
}

async function handleDeleteQuiz(courseUuid: string, assignment: Assignment) {
  if (!confirm(`Delete quiz "${assignment.title}"?`)) return
  try {
    await deleteQuiz(courseUuid, assignment.uuid)
    toast.success('Quiz deleted.')
    await loadAssignments()
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to delete quiz.')
  }
}

onMounted(() => {
  loadAssignments()
})
</script>
