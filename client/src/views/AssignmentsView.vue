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
        to="/courses"
        variant="elevated"
      >
        My Courses
      </v-btn>
    </div>

    <!-- Partial Course Fetch Warning -->
    <v-alert
      v-if="partialWarning"
      class="mb-4"
      color="warning"
      icon="mdi-alert-outline"
      rounded="lg"
      variant="tonal"
    >
      {{ partialWarning }}
    </v-alert>

    <!-- Loading State -->
    <div v-if="loading" class="d-flex justify-center py-12">
      <v-progress-circular color="primary" indeterminate size="64" />
    </div>

    <!-- Error State -->
    <v-alert
      v-else-if="loadError"
      class="mb-4"
      color="error"
      icon="mdi-alert-circle"
      rounded="lg"
      variant="tonal"
    >
      <div class="d-flex flex-wrap align-center justify-space-between ga-3">
        <span>{{ loadError }}</span>
        <v-btn color="error" size="small" variant="outlined" @click="loadAssignments">
          Retry
        </v-btn>
      </div>
    </v-alert>

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
      <v-btn color="primary" prepend-icon="mdi-book-open-outline" to="/courses" variant="elevated">
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
              prepend-icon="mdi-arrow-right"
              size="small"
              :to="`/courses/${group.course.uuid}`"
              variant="text"
            >
              Course
            </v-btn>
          </div>
        </div>

        <v-divider class="mb-4" />

        <div class="d-flex flex-column ga-2">
          <template v-for="assignment in group.assignments" :key="assignment.uuid">
            <HandinAssignmentItem
              v-if="assignment.type === 'HANDIN'"
              :assignment="assignment"
              :is-teacher="group.role === 'TEACHER'"
              show-module
              @delete="handleDeleteHandin(group.course.uuid, assignment)"
              @edit="openEditHandinDialog(group.course.uuid, assignment)"
              @submissions="openTeacherSubmissions(group.course.uuid, assignment)"
              @submit="openStudentSubmit(group.course.uuid, assignment)"
            />
            <QuizListItem
              v-else
              :assignment="assignment"
              :is-teacher="group.role === 'TEACHER'"
              show-module
              @delete="handleDeleteQuiz(group.course.uuid, assignment)"
              @edit="handleEditQuiz(group.course.uuid, assignment)"
              @results="handleResultsQuiz(group.course.uuid, assignment)"
              @take="handleTakeQuiz(group.course.uuid, assignment)"
            />
          </template>
        </div>
      </v-card>
    </div>

    <!-- Shared Edit Hand-in Assignment Dialog -->
    <HandinFormDialog
      v-model="handinDialogOpen"
      :assignment="editingHandin"
      :course-uuid="targetCourseUuid"
      :module-uuid="targetHandinModUuid"
      @saved="loadAssignments"
    />

    <!-- Student Submission Dialog -->
    <HandinSubmissionDialog
      v-model="studentSubmitDialogOpen"
      :assignment="selectedHandinForSubmit"
      :course-uuid="selectedCourseUuid"
      @submitted="loadAssignments"
    />

    <!-- Teacher Submissions Dialog -->
    <TeacherSubmissionsDialog
      v-model="teacherSubmissionsDialogOpen"
      :assignment="selectedHandinForTeacher"
      :course-uuid="selectedCourseUuid"
    />
  </v-container>
</template>

<script setup lang="ts">
import type { Course, CourseRole } from '@/types/course'
import type { Assignment } from '@/types/quiz'
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { getApiErrorMessage } from '@/api/client'
import { getMyCourses } from '@/api/course'
import { deleteQuiz, listAssignments } from '@/api/quiz'
import { deleteHandin } from '@/api/submission'
import HandinAssignmentItem from '@/components/HandinAssignmentItem.vue'
import HandinFormDialog from '@/components/HandinFormDialog.vue'
import HandinSubmissionDialog from '@/components/HandinSubmissionDialog.vue'
import QuizListItem from '@/components/QuizListItem.vue'
import TeacherSubmissionsDialog from '@/components/TeacherSubmissionsDialog.vue'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@/composables/useToast'

interface CourseAssignmentGroup {
  course: Course
  role: CourseRole
  assignments: Assignment[]
}

const router = useRouter()
const toast = useToast()
const { confirm } = useConfirm()

const loading = ref(false)
const loadError = ref<string | undefined>(undefined)
const partialWarning = ref<string | undefined>(undefined)
const courseAssignments = ref<CourseAssignmentGroup[]>([])
let fetchCounter = 0

// Shared Hand-in edit dialog state
const handinDialogOpen = ref(false)
const targetCourseUuid = ref('')
const targetHandinModUuid = ref('')
const editingHandin = ref<Assignment | undefined>(undefined)

const studentSubmitDialogOpen = ref(false)
const selectedHandinForSubmit = ref<Assignment | undefined>(undefined)

const teacherSubmissionsDialogOpen = ref(false)
const selectedHandinForTeacher = ref<Assignment | undefined>(undefined)

const selectedCourseUuid = ref('')

async function loadAssignments(): Promise<void> {
  const requestId = ++fetchCounter
  loading.value = true
  loadError.value = undefined
  partialWarning.value = undefined

  try {
    const enrollments = await getMyCourses()
    if (requestId !== fetchCounter) {
      return
    }
    const results: CourseAssignmentGroup[] = []
    const failedCourses: string[] = []

    await Promise.all(
      enrollments.map(async enrollment => {
        try {
          const assignments = await listAssignments(enrollment.course.uuid)
          if (assignments.length > 0) {
            results.push({
              course: enrollment.course,
              role: enrollment.role,
              assignments,
            })
          }
        } catch {
          failedCourses.push(enrollment.course.title)
        }
      }),
    )

    if (requestId !== fetchCounter) {
      return
    }

    results.sort((a, b) => a.course.title.localeCompare(b.course.title))
    courseAssignments.value = results
    if (failedCourses.length > 0) {
      partialWarning.value = `Could not load assignments for: ${failedCourses.join(', ')}`
    }
  } catch (error: unknown) {
    if (requestId === fetchCounter) {
      courseAssignments.value = []
      loadError.value = getApiErrorMessage(error, 'Failed to load assignments.')
    }
  } finally {
    if (requestId === fetchCounter) {
      loading.value = false
    }
  }
}

function handleTakeQuiz(courseUuid: string, assignment: Assignment): void {
  void router.push({
    name: 'QuizTake',
    params: {
      uuid: courseUuid,
      moduleUuid: assignment.moduleUuid,
      quizUuid: assignment.uuid,
    },
  })
}

function handleEditQuiz(courseUuid: string, assignment: Assignment): void {
  void router.push({
    name: 'QuizEdit',
    params: {
      uuid: courseUuid,
      moduleUuid: assignment.moduleUuid,
      quizUuid: assignment.uuid,
    },
  })
}

function handleResultsQuiz(courseUuid: string, assignment: Assignment): void {
  void router.push({
    name: 'QuizResults',
    params: {
      uuid: courseUuid,
      moduleUuid: assignment.moduleUuid,
      quizUuid: assignment.uuid,
    },
  })
}

async function handleDeleteQuiz(courseUuid: string, assignment: Assignment): Promise<void> {
  if (
    !(await confirm({
      title: 'Delete Quiz',
      message: `Delete quiz "${assignment.title}"?`,
      confirmText: 'Delete',
      confirmColor: 'error',
    }))
  ) {
    return
  }
  try {
    await deleteQuiz(courseUuid, assignment.uuid)
    toast.success('Quiz deleted.')
    await loadAssignments()
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to delete quiz.'))
  }
}

function openEditHandinDialog(courseUuid: string, asgn: Assignment): void {
  targetCourseUuid.value = courseUuid
  targetHandinModUuid.value = asgn.moduleUuid
  editingHandin.value = asgn
  handinDialogOpen.value = true
}

async function handleDeleteHandin(courseUuid: string, assignment: Assignment): Promise<void> {
  if (
    !(await confirm({
      title: 'Delete Assignment',
      message: `Delete assignment "${assignment.title}"?`,
      confirmText: 'Delete',
      confirmColor: 'error',
    }))
  ) {
    return
  }
  try {
    await deleteHandin(courseUuid, assignment.moduleUuid, assignment.uuid)
    toast.success('Assignment deleted.')
    await loadAssignments()
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to delete assignment.'))
  }
}

function openStudentSubmit(courseUuid: string, asgn: Assignment): void {
  selectedCourseUuid.value = courseUuid
  selectedHandinForSubmit.value = asgn
  studentSubmitDialogOpen.value = true
}

function openTeacherSubmissions(courseUuid: string, asgn: Assignment): void {
  selectedCourseUuid.value = courseUuid
  selectedHandinForTeacher.value = asgn
  teacherSubmissionsDialogOpen.value = true
}

onMounted(() => {
  void loadAssignments()
})
</script>
