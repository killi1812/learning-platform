<template>
  <v-container class="py-6" max-width="1400">
    <div class="d-flex flex-wrap align-center justify-space-between mb-6 ga-4">
      <div>
        <h1 class="text-h4 font-weight-bold text-high-emphasis my-0">Grades & Evaluations</h1>
        <p class="text-subtitle-1 text-medium-emphasis mb-0">
          Review academic performance, feedback, and grading tasks.
        </p>
      </div>

      <div class="d-flex align-center ga-3">
        <!-- Export Gradebook Menu (Teachers/Admins) -->
        <v-menu v-if="canGrade && teacherCourses.length > 0" location="bottom end">
          <template #activator="{ props: menuProps }">
            <v-btn
              v-bind="menuProps"
              color="primary"
              variant="outlined"
              prepend-icon="mdi-file-delimited-outline"
              :loading="exporting"
            >
              Export Gradebook (CSV)
            </v-btn>
          </template>
          <v-list density="compact">
            <v-list-subheader>Select Course</v-list-subheader>
            <v-list-item
              v-for="c in teacherCourses"
              :key="c.course.uuid"
              :title="c.course.title"
              :subtitle="`${c.course.term} ${c.course.year}`"
              prepend-icon="mdi-download"
              @click="handleExportCourse(c.course.uuid, c.course.title)"
            />
          </v-list>
        </v-menu>

        <v-btn
          color="primary"
          prepend-icon="mdi-refresh"
          variant="text"
          :loading="loading"
          @click="loadAll"
        >
          Refresh
        </v-btn>
      </div>
    </div>

    <!-- Role-Based Tabs (if user is both student and teacher, or admin) -->
    <v-tabs v-if="canGrade && isStudent" v-model="activeTab" class="mb-6" color="primary">
      <v-tab value="my-grades">
        <v-icon start icon="mdi-school-outline" />
        My Grades
        <v-chip
          v-if="myGrades.length > 0"
          class="ml-2"
          size="x-small"
          color="primary"
          variant="tonal"
        >
          {{ myGrades.length }}
        </v-chip>
      </v-tab>
      <v-tab value="pending">
        <v-icon start icon="mdi-clipboard-check-outline" />
        Pending Grading
        <v-chip
          v-if="pendingSubmissions.length > 0"
          class="ml-2"
          size="x-small"
          color="warning"
          variant="flat"
        >
          {{ pendingSubmissions.length }}
        </v-chip>
      </v-tab>
    </v-tabs>

    <!-- Loading State -->
    <div v-if="loading" class="d-flex justify-center py-12">
      <v-progress-circular color="primary" indeterminate size="64" />
    </div>

    <!-- STUDENT VIEW: My Grades -->
    <div v-else-if="activeTab === 'my-grades'">
      <v-card class="bg-surface pa-4 mb-4" elevation="1" rounded="lg">
        <div class="d-flex flex-wrap align-center justify-space-between ga-3 mb-4">
          <div class="text-h6 font-weight-bold text-high-emphasis">My Course Grades</div>
          <v-select
            v-if="studentCourses.length > 1"
            v-model="selectedStudentCourse"
            :items="studentCourseFilterOptions"
            density="compact"
            label="Filter Course"
            hide-details
            clearable
            variant="outlined"
            style="max-width: 280px"
          />
        </div>

        <div v-if="filteredMyGrades.length === 0" class="py-8 text-center text-medium-emphasis">
          <v-icon icon="mdi-certificate-outline" size="48" class="mb-2 text-disabled" />
          <p class="text-body-1 font-weight-medium mb-1">No released grades yet</p>
          <p class="text-caption">
            Grades for completed assignments and quizzes will appear here once released by
            instructors.
          </p>
        </div>

        <v-table v-else hover density="comfortable">
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
            <tr v-for="g in filteredMyGrades" :key="`${g.type}-${g.contentUuid}-${g.attempt}`">
              <td class="py-3">
                <div class="font-weight-bold text-high-emphasis">{{ g.courseName }}</div>
                <div class="text-caption text-medium-emphasis">{{ g.moduleTitle }}</div>
              </td>
              <td class="font-weight-medium">{{ g.assignmentTitle }}</td>
              <td class="text-center">
                <v-chip
                  size="x-small"
                  :color="g.type === 'QUIZ' ? 'info' : 'secondary'"
                  variant="tonal"
                >
                  {{ g.type === 'QUIZ' ? 'Quiz' : 'Hand-in' }}
                </v-chip>
              </td>
              <td class="text-center">
                <v-chip size="x-small" color="primary" variant="flat"> #{{ g.attempt }} </v-chip>
              </td>
              <td class="text-center font-weight-bold">
                {{ g.score.toFixed(1) }} / {{ g.maxPoints }}
              </td>
              <td class="text-center">
                <v-chip
                  size="x-small"
                  :color="g.percent >= 50 ? 'success' : 'error'"
                  variant="flat"
                >
                  {{ g.percent.toFixed(1) }}%
                </v-chip>
              </td>
              <td class="text-caption text-medium-emphasis">
                {{ formatDateTime(g.gradedAt) }}
              </td>
              <td class="text-right">
                <v-btn
                  v-if="g.feedback"
                  size="small"
                  variant="tonal"
                  color="primary"
                  prepend-icon="mdi-comment-text-outline"
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
    </div>

    <!-- TEACHER VIEW: Pending Grading Dashboard -->
    <div v-else-if="activeTab === 'pending'">
      <v-card class="bg-surface pa-4 mb-4" elevation="1" rounded="lg">
        <div class="d-flex flex-wrap align-center justify-space-between ga-3 mb-4">
          <div class="d-flex align-center ga-2">
            <span class="text-h6 font-weight-bold text-high-emphasis">Pending Grading Queue</span>
            <v-chip color="warning" size="small" variant="flat">
              {{ filteredPending.length }} pending
            </v-chip>
          </div>

          <v-select
            v-if="teacherCourses.length > 1"
            v-model="selectedTeacherCourse"
            :items="teacherCourseFilterOptions"
            density="compact"
            label="Filter Course"
            hide-details
            clearable
            variant="outlined"
            style="max-width: 280px"
          />
        </div>

        <div v-if="filteredPending.length === 0" class="py-8 text-center text-medium-emphasis">
          <v-icon icon="mdi-check-all" size="48" class="mb-2 text-success" />
          <p class="text-body-1 font-weight-medium mb-1">All caught up!</p>
          <p class="text-caption">No submissions are currently waiting to be graded.</p>
        </div>

        <v-table v-else hover density="comfortable">
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
                <v-chip size="x-small" color="primary" variant="flat"> #{{ item.attempt }} </v-chip>
              </td>
              <td class="text-caption text-medium-emphasis">
                {{ formatDateTime(item.submittedAt) }}
              </td>
              <td class="text-center">
                <v-chip size="x-small" :color="item.isLate ? 'warning' : 'success'" variant="tonal">
                  {{ item.isLate ? 'Late' : 'On Time' }}
                </v-chip>
              </td>
              <td class="text-right">
                <v-btn
                  color="primary"
                  size="small"
                  variant="elevated"
                  prepend-icon="mdi-check-decagram"
                  @click="openGradeModal(item)"
                >
                  Grade
                </v-btn>
              </td>
            </tr>
          </tbody>
        </v-table>
      </v-card>
    </div>

    <!-- Student Feedback Dialog -->
    <v-dialog v-model="feedbackDialogOpen" max-width="600">
      <v-card class="bg-surface pa-2" rounded="lg">
        <v-card-title
          class="text-h6 font-weight-bold pt-4 px-4 text-high-emphasis d-flex align-center justify-space-between"
        >
          <span>Feedback: {{ activeFeedbackGrade?.assignmentTitle }}</span>
          <v-btn icon="mdi-close" variant="text" size="small" @click="feedbackDialogOpen = false" />
        </v-card-title>
        <v-card-subtitle class="px-4 text-medium-emphasis d-flex align-center ga-2">
          <span>Attempt #{{ activeFeedbackGrade?.attempt }}</span>
          <span>·</span>
          <v-chip size="x-small" color="primary" variant="flat">
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

    <!-- Teacher Grading Dialog -->
    <GradeDialog
      v-model="gradeDialogOpen"
      :course-uuid="gradingItem?.courseUuid || ''"
      :content-uuid="gradingItem?.contentUuid || ''"
      :submission-uuid="gradingItem?.submissionUuid || ''"
      :assignment-title="gradingItem?.assignmentTitle"
      :student-email="gradingItem?.userEmail"
      :attempt="gradingItem?.attempt"
      :max-points="gradingItem?.maxPoints || 100"
      @saved="onPendingGradeSaved"
    />
  </v-container>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useToast } from '@/composables/useToast'
import { useAuth } from '@/composables/useAuth'
import { getApiErrorMessage } from '@/api/client'
import { getMyCourses } from '@/api/course'
import { listMyGrades, listPendingGrades, exportCourseGrades } from '@/api/grade'
import MarkdownView from '@/components/MarkdownView.vue'
import GradeDialog from '@/components/GradeDialog.vue'
import type { MyCourse } from '@/types/course'
import type { StudentGradeDto, PendingSubmissionDto } from '@/types/grade'

const toast = useToast()
const { currentUser } = useAuth()

const loading = ref(true)
const exporting = ref(false)
const myCourses = ref<MyCourse[]>([])
const myGrades = ref<StudentGradeDto[]>([])
const pendingSubmissions = ref<PendingSubmissionDto[]>([])

const activeTab = ref<'my-grades' | 'pending'>('my-grades')
const selectedStudentCourse = ref<string | null>(null)
const selectedTeacherCourse = ref<string | null>(null)

// Feedback modal
const feedbackDialogOpen = ref(false)
const activeFeedbackGrade = ref<StudentGradeDto | null>(null)

// Grading modal
const gradeDialogOpen = ref(false)
const gradingItem = ref<PendingSubmissionDto | null>(null)

const isStudent = computed(() => myCourses.value.some(c => c.role === 'STUDENT'))
const isTeacher = computed(() => myCourses.value.some(c => c.role === 'TEACHER'))
const isAdmin = computed(() => {
  const r = currentUser.value?.role?.toLowerCase()
  return r === 'admin' || r === 'superadmin'
})
const canGrade = computed(() => isTeacher.value || isAdmin.value)

const studentCourses = computed(() => myCourses.value.filter(c => c.role === 'STUDENT'))
const teacherCourses = computed(() => myCourses.value.filter(c => c.role === 'TEACHER'))

const studentCourseFilterOptions = computed(() =>
  studentCourses.value.map(c => ({
    title: c.course.title,
    value: c.course.uuid,
  })),
)

const teacherCourseFilterOptions = computed(() =>
  teacherCourses.value.map(c => ({
    title: c.course.title,
    value: c.course.uuid,
  })),
)

const filteredMyGrades = computed(() => {
  if (!selectedStudentCourse.value) return myGrades.value
  return myGrades.value.filter(g => g.courseUuid === selectedStudentCourse.value)
})

const filteredPending = computed(() => {
  if (!selectedTeacherCourse.value) return pendingSubmissions.value
  return pendingSubmissions.value.filter(p => p.courseUuid === selectedTeacherCourse.value)
})

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString()
}

function openFeedbackModal(g: StudentGradeDto) {
  activeFeedbackGrade.value = g
  feedbackDialogOpen.value = true
}

function openGradeModal(item: PendingSubmissionDto) {
  gradingItem.value = item
  gradeDialogOpen.value = true
}

function onPendingGradeSaved() {
  if (!gradingItem.value) return
  const finishedUuid = gradingItem.value.submissionUuid
  pendingSubmissions.value = pendingSubmissions.value.filter(p => p.submissionUuid !== finishedUuid)
}

async function handleExportCourse(courseUuid: string, courseTitle: string) {
  exporting.value = true
  try {
    const filename = `${courseTitle.toLowerCase().replace(/[^a-z0-9]+/g, '-')}-gradebook.csv`
    await exportCourseGrades(courseUuid, filename)
    toast.success('Gradebook exported successfully!')
  } catch (err: unknown) {
    toast.error(getApiErrorMessage(err, 'Failed to export gradebook CSV.'))
  } finally {
    exporting.value = false
  }
}

async function loadAll() {
  loading.value = true
  try {
    const courses = await getMyCourses()
    myCourses.value = courses

    const hasTeacherRole = courses.some(c => c.role === 'TEACHER') || isAdmin.value
    const hasStudentRole = courses.some(c => c.role === 'STUDENT')

    // Determine initial active tab based on role
    if (hasTeacherRole && !hasStudentRole) {
      activeTab.value = 'pending'
    } else {
      activeTab.value = 'my-grades'
    }

    const tasks: Promise<void>[] = []

    if (hasStudentRole) {
      tasks.push(
        listMyGrades().then(g => {
          myGrades.value = g
        }),
      )
    }

    if (hasTeacherRole) {
      tasks.push(
        listPendingGrades().then(p => {
          pendingSubmissions.value = p
        }),
      )
    }

    await Promise.all(tasks)
  } catch (err: unknown) {
    toast.error(getApiErrorMessage(err, 'Failed to load grades.'))
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadAll()
})
</script>
