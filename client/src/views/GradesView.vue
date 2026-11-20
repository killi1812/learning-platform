<template>
  <v-container class="py-6" max-width="1400">
    <div class="d-flex flex-wrap align-center justify-space-between mb-6 ga-4">
      <div>
        <h1 class="text-h4 font-weight-bold text-high-emphasis my-0">Grades &amp; Evaluations</h1>
        <p class="text-subtitle-1 text-medium-emphasis mb-0">
          Your total result for each course, and grading tasks.
        </p>
      </div>

      <div class="d-flex align-center ga-3">
        <!-- Export Gradebook Menu (Teachers/Admins) -->
        <v-menu v-if="canGrade && gradingCourses.length > 0" location="bottom end">
          <template #activator="{ props: menuProps }">
            <v-btn
              v-bind="menuProps"
              color="primary"
              :loading="exporting"
              prepend-icon="mdi-file-delimited-outline"
              variant="outlined"
            >
              Export Gradebook (CSV)
            </v-btn>
          </template>
          <v-list density="compact">
            <v-list-subheader>Select Course</v-list-subheader>
            <v-list-item
              v-for="c in gradingCourses"
              :key="c.uuid"
              prepend-icon="mdi-download"
              :subtitle="`${c.term} ${c.year}`"
              :title="c.title"
              @click="handleExportCourse(c.uuid, c.title)"
            />
          </v-list>
        </v-menu>
      </div>
    </div>

    <!-- Role-Based Tabs -->
    <v-tabs v-if="canGrade" v-model="activeTab" class="mb-6" color="primary">
      <v-tab v-if="isStudent" value="my-grades">
        <v-icon icon="mdi-school-outline" start />
        My Grades
      </v-tab>
      <v-tab value="pending">
        <v-icon icon="mdi-clipboard-check-outline" start />
        Pending Grading
        <v-chip
          v-if="pendingSubmissions.length > 0"
          class="ml-2"
          color="warning"
          size="x-small"
          variant="flat"
        >
          {{ pendingSubmissions.length }}
        </v-chip>
      </v-tab>
      <v-tab value="roster">
        <v-icon icon="mdi-account-group-outline" start />
        Course Gradebook &amp; Final Grades
      </v-tab>
    </v-tabs>

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
        <v-btn color="error" size="small" variant="outlined" @click="loadAll"> Retry </v-btn>
      </div>
    </v-alert>

    <!-- Tabbed Content (Teachers/Admins) -->
    <v-tabs-window v-else-if="canGrade" v-model="activeTab">
      <v-tabs-window-item v-if="isStudent" value="my-grades">
        <StudentGradesTab :my-final-grades="myFinalGrades" />
      </v-tabs-window-item>

      <v-tabs-window-item value="pending">
        <PendingGradingTab
          :grading-courses="gradingCourses"
          :pending-submissions="pendingSubmissions"
          @graded="onPendingGradeSaved"
        />
      </v-tabs-window-item>

      <v-tabs-window-item value="roster">
        <CourseRosterGradesTab
          v-model:selected-course-uuid="selectedRosterCourse"
          :exporting="exporting"
          :grading-courses="gradingCourses"
          :loading-roster="loadingRoster"
          :roster-students="rosterStudents"
          @export-selected="handleExportSelectedRosterCourse"
          @update:selected-course-uuid="fetchRoster"
        />
      </v-tabs-window-item>
    </v-tabs-window>

    <!-- Pure Student View -->
    <StudentGradesTab v-else :my-final-grades="myFinalGrades" />
  </v-container>
</template>

<script setup lang="ts">
import type { Course, MyCourse } from '@/types/course'
import type {
  CourseFinalGradeDto,
  PendingSubmissionDto,
  StudentFinalGradeRowDto,
} from '@/types/grade'
import { computed, onMounted, ref } from 'vue'
import { getApiErrorMessage } from '@/api/client'
import { getMyCourses, listCourses } from '@/api/course'
import {
  exportCourseGrades,
  getCourseRosterFinalGrades,
  getMyFinalGrades,
  listPendingGrades,
} from '@/api/grade'
import CourseRosterGradesTab from '@/components/grades/CourseRosterGradesTab.vue'
import PendingGradingTab from '@/components/grades/PendingGradingTab.vue'
import StudentGradesTab from '@/components/grades/StudentGradesTab.vue'
import { useAuth } from '@/composables/useAuth'
import { useToast } from '@/composables/useToast'

const toast = useToast()
const { isAdminOrSuperadmin } = useAuth()

const loading = ref(true)
const loadError = ref<string | undefined>(undefined)
const exporting = ref(false)
const myCourses = ref<MyCourse[]>([])
const allCourses = ref<Course[]>([])
const myFinalGrades = ref<CourseFinalGradeDto[]>([])
const pendingSubmissions = ref<PendingSubmissionDto[]>([])
const rosterStudents = ref<StudentFinalGradeRowDto[]>([])
const loadingRoster = ref(false)

const activeTab = ref<'my-grades' | 'pending' | 'roster'>('my-grades')
const selectedRosterCourse = ref<string | undefined>(undefined)

let loadCounter = 0
let rosterCounter = 0

const isStudent = computed(() => myCourses.value.some(c => c.role === 'STUDENT'))
const isTeacher = computed(() => myCourses.value.some(c => c.role === 'TEACHER'))
const canGrade = computed(() => isTeacher.value || isAdminOrSuperadmin.value)

const gradingCourses = computed<Course[]>(() => {
  if (isAdminOrSuperadmin.value) {
    return allCourses.value
  }
  return myCourses.value.filter(c => c.role === 'TEACHER').map(c => c.course)
})

function onPendingGradeSaved(finishedUuid: string): void {
  pendingSubmissions.value = pendingSubmissions.value.filter(p => p.submissionUuid !== finishedUuid)
  if (selectedRosterCourse.value) {
    void fetchRoster(selectedRosterCourse.value)
  }
}

async function handleExportCourse(courseUuid: string, courseTitle: string): Promise<void> {
  exporting.value = true
  try {
    const filename = `${courseTitle.toLowerCase().replace(/[^a-z0-9]+/g, '-')}-gradebook.csv`
    await exportCourseGrades(courseUuid, filename)
    toast.success('Gradebook exported successfully!')
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to export gradebook CSV.'))
  } finally {
    exporting.value = false
  }
}

async function fetchRoster(courseUuid: string | undefined): Promise<void> {
  const requestId = ++rosterCounter
  if (!courseUuid) {
    rosterStudents.value = []
    return
  }
  loadingRoster.value = true
  try {
    const rows = await getCourseRosterFinalGrades(courseUuid)
    if (requestId === rosterCounter) {
      rosterStudents.value = rows
    }
  } catch (error: unknown) {
    if (requestId === rosterCounter) {
      toast.error(getApiErrorMessage(error, 'Failed to load course grade roster.'))
    }
  } finally {
    if (requestId === rosterCounter) {
      loadingRoster.value = false
    }
  }
}

async function handleExportSelectedRosterCourse(): Promise<void> {
  if (!selectedRosterCourse.value) {
    return
  }
  const crs = gradingCourses.value.find(c => c.uuid === selectedRosterCourse.value)
  const title = crs ? crs.title : 'course'
  await handleExportCourse(selectedRosterCourse.value, title)
}

async function loadAll(): Promise<void> {
  const requestId = ++loadCounter
  loading.value = true
  loadError.value = undefined

  try {
    const [courses, catalog] = await Promise.all([
      getMyCourses(),
      isAdminOrSuperadmin.value ? listCourses() : Promise.resolve([] as Course[]),
    ])
    if (requestId !== loadCounter) {
      return
    }

    myCourses.value = courses
    allCourses.value = catalog

    const hasTeacherRole = courses.some(c => c.role === 'TEACHER') || isAdminOrSuperadmin.value
    const hasStudentRole = courses.some(c => c.role === 'STUDENT')

    activeTab.value = hasTeacherRole && !hasStudentRole ? 'pending' : 'my-grades'

    const firstGradingCourse = gradingCourses.value[0]
    if (hasTeacherRole && firstGradingCourse && !selectedRosterCourse.value) {
      selectedRosterCourse.value = firstGradingCourse.uuid
    }

    const tasks: Promise<void>[] = []

    if (hasStudentRole) {
      tasks.push(
        getMyFinalGrades().then(fg => {
          if (requestId === loadCounter) {
            myFinalGrades.value = fg
          }
        }),
      )
    }

    if (hasTeacherRole) {
      tasks.push(
        listPendingGrades().then(p => {
          if (requestId === loadCounter) {
            pendingSubmissions.value = p
          }
        }),
      )
      if (selectedRosterCourse.value) {
        tasks.push(fetchRoster(selectedRosterCourse.value))
      }
    }

    await Promise.all(tasks)
  } catch (error: unknown) {
    if (requestId === loadCounter) {
      loadError.value = getApiErrorMessage(error, 'Failed to load grades.')
    }
  } finally {
    if (requestId === loadCounter) {
      loading.value = false
    }
  }
}

onMounted(() => {
  void loadAll()
})
</script>
