<template>
  <v-container class="pt-2 pb-6" max-width="1200">
    <!-- Loading State -->
    <div v-if="loading" class="d-flex justify-center py-12">
      <v-progress-circular color="primary" indeterminate size="64" />
    </div>

    <!-- Error State -->
    <v-alert
      v-else-if="loadError"
      class="my-4"
      color="error"
      icon="mdi-alert-circle"
      rounded="lg"
      variant="tonal"
    >
      <div class="d-flex flex-wrap align-center justify-space-between ga-3">
        <span>{{ loadError }}</span>
        <v-btn color="error" size="small" variant="outlined" @click="fetchCourseDetails">
          Retry
        </v-btn>
      </div>
    </v-alert>

    <div v-else-if="course">
      <!-- Course Title -->
      <h1 class="text-h4 font-weight-bold text-high-emphasis mt-0 mb-4">
        {{ course.title }}
      </h1>

      <!-- Tabs Navigation -->
      <v-tabs v-model="activeTab" class="mb-6" color="primary">
        <v-tab value="overview">
          <v-icon icon="mdi-information-outline" start />
          Overview
        </v-tab>
        <v-tab v-if="isEnrolled" value="modules">
          <v-icon icon="mdi-view-module" start />
          Course Modules ({{ modules.length }})
        </v-tab>
        <v-tab v-if="isTeacher" value="participants">
          <v-icon icon="mdi-account-group" start />
          Participants ({{ enrollments.length }})
        </v-tab>
        <v-tab v-if="isEnrolled" value="assignments">
          <v-icon icon="mdi-clipboard-text-outline" start />
          Assignments
        </v-tab>
        <v-tab v-if="isEnrolled && !isTeacher" value="grades">
          <v-icon icon="mdi-school-outline" start />
          Grades
        </v-tab>
        <v-tab v-if="isEnrolled" value="groups">
          <v-icon icon="mdi-account-multiple" start />
          Groups ({{ groups.length }})
        </v-tab>
        <v-tab v-if="isTeacher" value="settings">
          <v-icon icon="mdi-cog" start />
          Course Settings
        </v-tab>
      </v-tabs>

      <v-tabs-window v-model="activeTab">
        <v-tabs-window-item value="overview">
          <CourseOverviewTab
            :course="course"
            :enrolling="enrolling"
            :is-enrolled="isEnrolled"
            :user-role="userRole"
            @enroll="handleSelfEnroll"
          />
        </v-tabs-window-item>

        <v-tabs-window-item v-if="isEnrolled" value="modules">
          <CourseModulesTab
            :assignments="assignments"
            :is-teacher="isTeacher"
            :locked-modules="lockedModules"
            :module-pages="modulePages"
            :modules="modules"
            :reordering-modules="reorderingModules"
            @add-quiz="handleAddQuiz"
            @create-handin="openCreateHandinDialog"
            @create-module="openCreateModuleDialog"
            @delete-handin="handleDeleteHandin"
            @delete-material="handleDeleteMaterial"
            @delete-module="handleDeleteModule"
            @delete-quiz="handleDeleteQuiz"
            @edit-handin="openEditHandinDialog"
            @edit-material="openEditMaterialDialog"
            @edit-module="openEditModuleDialog"
            @edit-quiz="editQuiz"
            @move-module="moveModule"
            @results-quiz="openQuizResults"
            @submissions-handin="openTeacherSubmissions"
            @submit-handin="openStudentSubmit"
            @take-quiz="takeQuiz"
            @upload-material="openUploadDialog"
            @view-material="viewMaterial"
          />
        </v-tabs-window-item>

        <v-tabs-window-item v-if="isTeacher" value="participants">
          <CourseParticipantsTab
            :deleting-user-uuid="deletingEnrollmentUserUuid"
            :enrollments="enrollments"
            @add-participant="addEnrollmentDialogOpen = true"
            @remove-participant="handleRemoveEnrollment"
          />
        </v-tabs-window-item>

        <v-tabs-window-item v-if="isEnrolled" value="assignments">
          <CourseAssignmentsTab
            :assignments="assignments"
            :is-teacher="isTeacher"
            @delete-handin="handleDeleteHandin"
            @delete-quiz="handleDeleteQuiz"
            @edit-handin="openEditHandinDialog"
            @edit-quiz="editQuiz"
            @results-quiz="openQuizResults"
            @submissions-handin="openTeacherSubmissions"
            @submit-handin="openStudentSubmit"
            @take-quiz="takeQuiz"
          />
        </v-tabs-window-item>

        <v-tabs-window-item v-if="isEnrolled && !isTeacher" value="grades">
          <CourseGradesTab :course-uuid="resolvedCourseUuid" />
        </v-tabs-window-item>

        <v-tabs-window-item v-if="isEnrolled" value="groups">
          <CourseGroupsTab
            :action-loading="actionLoading"
            :current-user="currentUser"
            :groups="groups"
            :is-teacher="isTeacher"
            :loading="loadingGroups"
            :my-group="myGroup"
            @bulk-create="bulkDialogOpen = true"
            @create-group="openCreateGroupDialog"
            @delete-group="handleDeleteGroup"
            @edit-group="openEditGroupDialog"
            @join-group="handleJoinGroup"
            @leave-group="handleLeaveGroup"
          />
        </v-tabs-window-item>

        <v-tabs-window-item v-if="isTeacher" value="settings">
          <CourseSettingsTab
            :course="course"
            :deleting="deletingCourse"
            :saving="savingCourse"
            @delete-course="handleDeleteCourse"
            @save-course="handleUpdateCourse"
          />
        </v-tabs-window-item>
      </v-tabs-window>
    </div>

    <!-- Dialogs -->
    <ModuleFormDialog
      v-model="moduleDialogOpen"
      :course-uuid="resolvedCourseUuid"
      :module="editingModule"
      :modules="modules"
      @saved="refreshModules"
    />

    <AddParticipantDialog
      v-model="addEnrollmentDialogOpen"
      :course-uuid="resolvedCourseUuid"
      @saved="refreshEnrollmentsAndCourse"
    />

    <MaterialUploadDialog
      v-model="uploadDialogOpen"
      :course-uuid="resolvedCourseUuid"
      :module-uuid="targetModuleUuid"
      @saved="refreshModuleMaterials"
    />

    <MaterialEditDialog
      v-model="editMaterialDialogOpen"
      :course-uuid="resolvedCourseUuid"
      :module-uuid="editMaterialTargetModUuid"
      :page="editingMaterialPage"
      @saved="refreshModuleMaterials"
    />

    <HandinFormDialog
      v-model="handinDialogOpen"
      :assignment="editingHandin"
      :course-uuid="resolvedCourseUuid"
      :module-uuid="targetHandinModUuid"
      @saved="loadAssignments"
    />

    <HandinSubmissionDialog
      v-model="studentSubmitDialogOpen"
      :assignment="selectedHandinForSubmit"
      :course-uuid="resolvedCourseUuid"
      @submitted="loadAssignments"
    />

    <TeacherSubmissionsDialog
      v-model="teacherSubmissionsDialogOpen"
      :assignment="selectedHandinForTeacher"
      :course-uuid="resolvedCourseUuid"
    />

    <GroupFormDialog
      v-model="groupDialogOpen"
      :course-uuid="resolvedCourseUuid"
      :default-name="`Group ${groups.length + 1}`"
      :group="editingGroup"
      @saved="loadGroups"
    />

    <BulkGroupsDialog
      v-model="bulkDialogOpen"
      :course-uuid="resolvedCourseUuid"
      @saved="loadGroups"
    />
  </v-container>
</template>

<script lang="ts" setup>
import type { FileUpload } from '@/types/content'
import type {
  Course,
  CourseRole,
  Enrollment,
  Group,
  Module,
  UpdateCoursePayload,
} from '@/types/course'
import type { Assignment } from '@/types/quiz'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ApiError, getApiErrorMessage } from '@/api/client'
import { deleteFile, listFiles } from '@/api/content'
import {
  deleteCourse,
  deleteGroup,
  deleteModule,
  getCourse,
  getCourseEnrollments,
  getMyCourses,
  joinGroup,
  leaveGroup,
  listCourseGroups,
  listModules,
  removeEnrollment,
  reorderModules,
  selfEnroll,
  updateCourse,
} from '@/api/course'
import { createQuiz, deleteQuiz, listAssignments } from '@/api/quiz'
import { deleteHandin } from '@/api/submission'
import AddParticipantDialog from '@/components/course/AddParticipantDialog.vue'
import BulkGroupsDialog from '@/components/course/BulkGroupsDialog.vue'
import CourseAssignmentsTab from '@/components/course/CourseAssignmentsTab.vue'
import CourseGradesTab from '@/components/course/CourseGradesTab.vue'
import CourseGroupsTab from '@/components/course/CourseGroupsTab.vue'
import CourseModulesTab from '@/components/course/CourseModulesTab.vue'
import CourseOverviewTab from '@/components/course/CourseOverviewTab.vue'
import CourseParticipantsTab from '@/components/course/CourseParticipantsTab.vue'
import CourseSettingsTab from '@/components/course/CourseSettingsTab.vue'
import GroupFormDialog from '@/components/course/GroupFormDialog.vue'
import MaterialEditDialog from '@/components/course/MaterialEditDialog.vue'
import MaterialUploadDialog from '@/components/course/MaterialUploadDialog.vue'
import ModuleFormDialog from '@/components/course/ModuleFormDialog.vue'
import HandinFormDialog from '@/components/HandinFormDialog.vue'
import HandinSubmissionDialog from '@/components/HandinSubmissionDialog.vue'
import TeacherSubmissionsDialog from '@/components/TeacherSubmissionsDialog.vue'
import { useAuth } from '@/composables/useAuth'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@/composables/useToast'
import { findUserGroup } from '@/util/group'

type CourseTab =
  'overview' | 'modules' | 'participants' | 'assignments' | 'grades' | 'groups' | 'settings'
const VALID_TABS: readonly CourseTab[] = [
  'overview',
  'modules',
  'participants',
  'assignments',
  'grades',
  'groups',
  'settings',
]

const props = defineProps<{
  courseUuid?: string
}>()

const emit = defineEmits<{
  'course-deleted': []
}>()

const route = useRoute()
const router = useRouter()
const { isAuthenticated, isAdminOrSuperadmin, currentUser } = useAuth()
const toast = useToast()
const { confirm } = useConfirm()

const resolvedCourseUuid = computed(() => props.courseUuid || String(route.params.uuid || ''))

const activeTab = ref<CourseTab>('modules')
const loading = ref(false)
const loadError = ref<string | undefined>(undefined)
const course = ref<Course | undefined>(undefined)
const modules = ref<Module[]>([])
const enrollments = ref<Enrollment[]>([])
const userRole = ref<CourseRole | undefined>(undefined)

const isTeacher = computed(() => userRole.value === 'TEACHER' || isAdminOrSuperadmin.value)
const isEnrolled = computed(() => Boolean(userRole.value) || isAdminOrSuperadmin.value)
const enrolling = ref(false)

let detailCounter = 0

function parseTabQuery(raw: unknown): CourseTab | undefined {
  const val = Array.isArray(raw) ? raw[0] : raw
  if (typeof val !== 'string') {
    return undefined
  }
  return VALID_TABS.find(t => t === val)
}

function canAccessTab(tab: CourseTab): boolean {
  if (tab === 'overview') {
    return true
  }
  if (tab === 'participants' || tab === 'settings') {
    return isTeacher.value
  }
  if (tab === 'grades') {
    return isEnrolled.value && !isTeacher.value
  }
  return isEnrolled.value
}

watch(activeTab, newTab => {
  if (loading.value || !course.value) {
    return
  }
  if (!canAccessTab(newTab)) {
    activeTab.value = 'overview'
    toast.info('You must be enrolled in this course to access course content.')
    return
  }
  if (parseTabQuery(route.query.tab) !== newTab) {
    void router.replace({ query: { ...route.query, tab: newTab } })
  }
})

watch(
  () => route.query.tab,
  nextRaw => {
    if (loading.value || !course.value) {
      return
    }
    const parsed = parseTabQuery(nextRaw)
    if (parsed && canAccessTab(parsed) && parsed !== activeTab.value) {
      activeTab.value = parsed
    }
  },
)

// Settings actions
const savingCourse = ref(false)
const deletingCourse = ref(false)

// Module management & reorder lock
const reorderingModules = ref(false)
const moduleDialogOpen = ref(false)
const editingModule = ref<Module | undefined>(undefined)

// Participant actions
const addEnrollmentDialogOpen = ref(false)
const deletingEnrollmentUserUuid = ref<string | undefined>(undefined)

// Groups state
const groups = ref<Group[]>([])
const loadingGroups = ref(false)
const groupDialogOpen = ref(false)
const editingGroup = ref<Group | undefined>(undefined)
const bulkDialogOpen = ref(false)
const actionLoading = ref<Record<string, boolean>>({})

const myGroup = computed(() => findUserGroup(groups.value, currentUser.value))

// Course materials state
const modulePages = ref<Record<string, FileUpload[]>>({})
const lockedModules = ref<Record<string, boolean>>({})
const assignments = ref<Assignment[]>([])
const uploadDialogOpen = ref(false)
const targetModuleUuid = ref('')

const editMaterialDialogOpen = ref(false)
const editMaterialTargetModUuid = ref('')
const editingMaterialPage = ref<FileUpload | undefined>(undefined)

// Hand-in assignment create/edit dialog
const handinDialogOpen = ref(false)
const targetHandinModUuid = ref('')
const editingHandin = ref<Assignment | undefined>(undefined)

// Student submit / Teacher submissions dialogs
const studentSubmitDialogOpen = ref(false)
const selectedHandinForSubmit = ref<Assignment | undefined>(undefined)
const teacherSubmissionsDialogOpen = ref(false)
const selectedHandinForTeacher = ref<Assignment | undefined>(undefined)

async function loadAssignments(): Promise<void> {
  if (!resolvedCourseUuid.value) {
    return
  }
  try {
    assignments.value = await listAssignments(resolvedCourseUuid.value)
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to load assignments.'))
  }
}

async function handleAddQuiz(moduleUuid: string): Promise<void> {
  try {
    const quiz = await createQuiz(resolvedCourseUuid.value, moduleUuid, 'New quiz')
    editQuiz({ uuid: quiz.uuid, moduleUuid })
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to create quiz.'))
  }
}

function quizRoute(quiz: Pick<Assignment, 'uuid' | 'moduleUuid'>, page: string): string {
  return `/courses/${resolvedCourseUuid.value}/modules/${quiz.moduleUuid}/quizzes/${quiz.uuid}/${page}`
}

function editQuiz(quiz: Pick<Assignment, 'uuid' | 'moduleUuid'>): void {
  void router.push(quizRoute(quiz, 'edit'))
}

function openQuizResults(quiz: Assignment): void {
  void router.push(quizRoute(quiz, 'results'))
}

function takeQuiz(quiz: Assignment): void {
  void router.push(quizRoute(quiz, 'take'))
}

async function handleDeleteQuiz(quiz: Assignment): Promise<void> {
  if (
    !(await confirm({
      title: 'Delete Quiz',
      message: `Delete quiz "${quiz.title}" and all student attempts?`,
      confirmText: 'Delete',
      confirmColor: 'error',
    }))
  ) {
    return
  }
  try {
    await deleteQuiz(resolvedCourseUuid.value, quiz.uuid)
    toast.success('Quiz deleted.')
    await loadAssignments()
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to delete quiz.'))
  }
}

function openCreateHandinDialog(moduleUuid: string): void {
  targetHandinModUuid.value = moduleUuid
  editingHandin.value = undefined
  handinDialogOpen.value = true
}

function openEditHandinDialog(asgn: Assignment): void {
  targetHandinModUuid.value = asgn.moduleUuid
  editingHandin.value = asgn
  handinDialogOpen.value = true
}

async function handleDeleteHandin(asgn: Assignment): Promise<void> {
  if (
    !(await confirm({
      title: 'Delete Assignment',
      message: `Delete assignment "${asgn.title}"?`,
      confirmText: 'Delete',
      confirmColor: 'error',
    }))
  ) {
    return
  }
  try {
    await deleteHandin(resolvedCourseUuid.value, asgn.moduleUuid, asgn.uuid)
    toast.success('Assignment deleted.')
    await loadAssignments()
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to delete assignment.'))
  }
}

function openStudentSubmit(asgn: Assignment): void {
  selectedHandinForSubmit.value = asgn
  studentSubmitDialogOpen.value = true
}

function openTeacherSubmissions(asgn: Assignment): void {
  selectedHandinForTeacher.value = asgn
  teacherSubmissionsDialogOpen.value = true
}

async function loadAllModulePages(): Promise<void> {
  const courseId = resolvedCourseUuid.value
  if (!courseId || modules.value.length === 0) {
    modulePages.value = {}
    lockedModules.value = {}
    return
  }
  await loadAssignments()
  const pagesMap: Record<string, FileUpload[]> = {}
  const lockedMap: Record<string, boolean> = {}
  let hasError = false

  await Promise.all(
    modules.value.map(async mod => {
      try {
        pagesMap[mod.uuid] = await listFiles(courseId, mod.uuid)
      } catch (error: unknown) {
        pagesMap[mod.uuid] = []
        const is403 = error instanceof ApiError && error.status === 403
        const errMsg = getApiErrorMessage(error, '').toLowerCase()
        if (is403 && (errMsg.includes('prerequisite') || Boolean(mod.dependsOn))) {
          lockedMap[mod.uuid] = true
        } else {
          hasError = true
        }
      }
    }),
  )

  if (resolvedCourseUuid.value !== courseId) {
    return
  }
  modulePages.value = pagesMap
  lockedModules.value = lockedMap
  if (hasError) {
    toast.error('Failed to load materials for one or more modules.')
  }
}

async function refreshModuleMaterials(moduleUuid: string): Promise<void> {
  try {
    modulePages.value[moduleUuid] = await listFiles(resolvedCourseUuid.value, moduleUuid)
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to refresh module materials.'))
  }
}

function openUploadDialog(modUuid: string): void {
  targetModuleUuid.value = modUuid
  uploadDialogOpen.value = true
}

function openEditMaterialDialog(modUuid: string, page: FileUpload): void {
  editMaterialTargetModUuid.value = modUuid
  editingMaterialPage.value = page
  editMaterialDialogOpen.value = true
}

async function handleDeleteMaterial(moduleUuid: string, pageUuid: string): Promise<void> {
  if (
    !(await confirm({
      title: 'Delete Material',
      message: 'Are you sure you want to delete this course material?',
      confirmText: 'Delete',
      confirmColor: 'error',
    }))
  ) {
    return
  }
  try {
    await deleteFile(resolvedCourseUuid.value, moduleUuid, pageUuid)
    toast.success('Course material deleted.')
    await refreshModuleMaterials(moduleUuid)
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to delete course material.'))
  }
}

function viewMaterial(moduleUuid: string, pageUuid: string): void {
  void router.push(`/courses/${resolvedCourseUuid.value}/modules/${moduleUuid}/files/${pageUuid}`)
}

async function fetchCourseDetails(): Promise<void> {
  const courseId = resolvedCourseUuid.value
  if (!courseId) {
    loadError.value = 'Missing course identifier.'
    loading.value = false
    return
  }

  const requestId = ++detailCounter
  loading.value = true
  loadError.value = undefined

  try {
    const [fetchedCourse, myCourses] = await Promise.all([
      getCourse(courseId),
      isAuthenticated.value ? getMyCourses() : Promise.resolve([]),
    ])
    if (requestId !== detailCounter) {
      return
    }

    course.value = fetchedCourse
    const mine = myCourses.find(mc => mc.course.uuid === courseId)
    userRole.value = mine?.role

    const requestedTab = parseTabQuery(route.query.tab)
    if (requestedTab && canAccessTab(requestedTab)) {
      activeTab.value = requestedTab
    } else {
      activeTab.value = isEnrolled.value ? 'modules' : 'overview'
    }

    if (isEnrolled.value) {
      modules.value = await listModules(courseId)
      if (requestId !== detailCounter) {
        return
      }
      await loadAllModulePages()
      if (requestId !== detailCounter) {
        return
      }

      if (isTeacher.value) {
        try {
          enrollments.value = await getCourseEnrollments(courseId)
        } catch (error: unknown) {
          toast.error(getApiErrorMessage(error, 'Failed to load course participants.'))
          enrollments.value = []
        }
      } else {
        enrollments.value = []
      }

      await loadGroups()
    } else {
      modules.value = []
      enrollments.value = []
      groups.value = []
      modulePages.value = {}
      lockedModules.value = {}
      assignments.value = []
    }
  } catch (error: unknown) {
    if (requestId === detailCounter) {
      course.value = undefined
      loadError.value = getApiErrorMessage(error, 'Failed to load course details.')
    }
  } finally {
    if (requestId === detailCounter) {
      loading.value = false
    }
  }
}

async function handleSelfEnroll(): Promise<void> {
  enrolling.value = true
  try {
    await selfEnroll(resolvedCourseUuid.value)
    toast.success('Successfully enrolled in the course!')
    await fetchCourseDetails()
    activeTab.value = 'modules'
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to enroll in course.'))
  } finally {
    enrolling.value = false
  }
}

function openCreateModuleDialog(): void {
  editingModule.value = undefined
  moduleDialogOpen.value = true
}

function openEditModuleDialog(mod: Module): void {
  editingModule.value = mod
  moduleDialogOpen.value = true
}

async function refreshModules(): Promise<void> {
  modules.value = await listModules(resolvedCourseUuid.value)
  await loadAllModulePages()
}

async function moveModule(index: number, direction: -1 | 1): Promise<void> {
  if (reorderingModules.value) {
    return
  }
  const targetIndex = index + direction
  if (targetIndex < 0 || targetIndex >= modules.value.length) {
    return
  }

  const previous = [...modules.value]
  const reordered = [...modules.value]
  const current = reordered[index]
  const target = reordered[targetIndex]
  if (!current || !target) {
    return
  }
  reordered[index] = target
  reordered[targetIndex] = current

  reorderingModules.value = true
  modules.value = reordered
  try {
    await reorderModules(
      resolvedCourseUuid.value,
      reordered.map(m => m.uuid),
    )
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to reorder modules.'))
    try {
      modules.value = await listModules(resolvedCourseUuid.value)
    } catch {
      modules.value = previous
    }
  } finally {
    reorderingModules.value = false
  }
}

async function handleDeleteModule(moduleUuid: string): Promise<void> {
  if (
    !(await confirm({
      title: 'Delete Module',
      message: 'Are you sure you want to delete this module?',
      confirmText: 'Delete',
      confirmColor: 'error',
    }))
  ) {
    return
  }
  try {
    await deleteModule(resolvedCourseUuid.value, moduleUuid)
    toast.success('Module deleted.')
    await refreshModules()
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to delete module.'))
  }
}

async function refreshEnrollmentsAndCourse(): Promise<void> {
  const [nextEnrollments, nextCourse] = await Promise.all([
    getCourseEnrollments(resolvedCourseUuid.value),
    getCourse(resolvedCourseUuid.value),
  ])
  enrollments.value = nextEnrollments
  course.value = nextCourse
}

async function handleRemoveEnrollment(enrollment: Enrollment): Promise<void> {
  if (!enrollment.userUuid || enrollment.role !== 'STUDENT') {
    return
  }
  if (
    !(await confirm({
      title: 'Remove Participant',
      message: `Are you sure you want to remove ${enrollment.userEmail} from this course?`,
      confirmText: 'Remove',
      confirmColor: 'error',
    }))
  ) {
    return
  }
  deletingEnrollmentUserUuid.value = enrollment.userUuid
  try {
    await removeEnrollment(resolvedCourseUuid.value, enrollment.userUuid)
    toast.success(`Removed ${enrollment.userEmail} from course.`)
    await refreshEnrollmentsAndCourse()
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to remove participant.'))
  } finally {
    deletingEnrollmentUserUuid.value = undefined
  }
}

async function handleUpdateCourse(payload: UpdateCoursePayload): Promise<void> {
  savingCourse.value = true
  try {
    course.value = await updateCourse(resolvedCourseUuid.value, payload)
    toast.success('Course details updated successfully.')
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to update course.'))
  } finally {
    savingCourse.value = false
  }
}

async function handleDeleteCourse(): Promise<void> {
  if (
    !(await confirm({
      title: 'Delete Course',
      message:
        'Are you ABSOLUTELY sure you want to delete this course? This action cannot be undone.',
      confirmText: 'Delete Course',
      confirmColor: 'error',
    }))
  ) {
    return
  }
  deletingCourse.value = true
  try {
    await deleteCourse(resolvedCourseUuid.value)
    toast.success('Course deleted.')
    emit('course-deleted')
    void router.push('/courses')
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to delete course.'))
  } finally {
    deletingCourse.value = false
  }
}

async function loadGroups(): Promise<void> {
  if (!resolvedCourseUuid.value) {
    return
  }
  loadingGroups.value = true
  try {
    groups.value = await listCourseGroups(resolvedCourseUuid.value)
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to load course groups.'))
  } finally {
    loadingGroups.value = false
  }
}

function openCreateGroupDialog(): void {
  editingGroup.value = undefined
  groupDialogOpen.value = true
}

function openEditGroupDialog(group: Group): void {
  editingGroup.value = group
  groupDialogOpen.value = true
}

async function handleDeleteGroup(group: Group): Promise<void> {
  if (
    !(await confirm({
      title: 'Delete Group',
      message: `Are you sure you want to delete "${group.name}"? Members will be unassigned.`,
      confirmText: 'Delete Group',
      confirmColor: 'error',
    }))
  ) {
    return
  }
  actionLoading.value[group.uuid] = true
  try {
    await deleteGroup(group.uuid)
    toast.success(`Group "${group.name}" deleted.`)
    await loadGroups()
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to delete group.'))
  } finally {
    delete actionLoading.value[group.uuid]
  }
}

async function handleJoinGroup(group: Group): Promise<void> {
  actionLoading.value[group.uuid] = true
  try {
    await joinGroup(group.uuid)
    toast.success(`Joined "${group.name}"!`)
    await loadGroups()
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to join group.'))
  } finally {
    delete actionLoading.value[group.uuid]
  }
}

async function handleLeaveGroup(group: Group): Promise<void> {
  if (
    !(await confirm({
      title: 'Leave Group',
      message: `Are you sure you want to leave "${group.name}"?`,
      confirmText: 'Leave Group',
      confirmColor: 'error',
    }))
  ) {
    return
  }
  actionLoading.value[group.uuid] = true
  try {
    await leaveGroup(group.uuid)
    toast.success(`Left "${group.name}".`)
    await loadGroups()
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to leave group.'))
  } finally {
    delete actionLoading.value[group.uuid]
  }
}

onMounted(() => {
  void fetchCourseDetails()
})

watch(resolvedCourseUuid, () => {
  void fetchCourseDetails()
})
</script>
