<template>
  <v-container class="pt-2 pb-6" max-width="1200">
    <!-- Loading State -->
    <div v-if="loading" class="d-flex justify-center py-12">
      <v-progress-circular color="primary" indeterminate size="64" />
    </div>

    <div v-else-if="course">
      <!-- Course Title -->
      <h1 class="text-h4 font-weight-bold text-high-emphasis mt-0 mb-4">
        {{ course.title }}
      </h1>

      <!-- Tabs Navigation -->
      <v-tabs v-model="activeTab" class="mb-6" color="primary">
        <v-tab value="overview">
          <v-icon start icon="mdi-information-outline" />
          Overview
        </v-tab>
        <v-tab value="modules">
          <v-icon start icon="mdi-view-module" />
          Course Modules ({{ modules.length }})
        </v-tab>
        <v-tab v-if="isTeacher" value="participants">
          <v-icon start icon="mdi-account-group" />
          Participants ({{ enrollments.length }})
        </v-tab>
        <v-tab value="assignments">
          <v-icon start icon="mdi-clipboard-text-outline" />
          Assignments
        </v-tab>
        <v-tab value="groups">
          <v-icon start icon="mdi-account-multiple" />
          Groups
        </v-tab>
        <v-tab v-if="isTeacher" value="settings">
          <v-icon start icon="mdi-cog" />
          Course Settings
        </v-tab>
      </v-tabs>

      <!-- TAB: OVERVIEW -->
      <div v-if="activeTab === 'overview'">
        <v-card class="pa-6 bg-surface mb-6" elevation="1" rounded="lg">
          <div class="d-flex flex-wrap align-center ga-3 mb-4">
            <v-chip
              v-if="userRole"
              :color="userRole === 'TEACHER' ? 'primary' : 'secondary'"
              size="default"
              variant="flat"
            >
              {{ userRole }}
            </v-chip>
            <v-chip color="secondary" size="default" variant="tonal">
              {{ course.term }} {{ course.year }}
            </v-chip>
          </div>

          <div
            v-if="course.teachers && course.teachers.length > 0"
            class="d-flex align-center ga-2 mb-4 text-body-1 text-medium-emphasis"
          >
            <v-icon icon="mdi-account-tie-outline" size="small" />
            <span>
              {{ course.teachers.length === 1 ? 'Professor' : 'Professors' }}:
              <strong class="text-high-emphasis">{{ course.teachers.join(', ') }}</strong>
            </span>
          </div>

          <v-divider class="mb-4" />

          <h2 class="text-h6 font-weight-bold text-high-emphasis mb-2">Description</h2>
          <p class="text-body-1 text-medium-emphasis mb-0" style="white-space: pre-line">
            {{ course.description || 'No course description provided.' }}
          </p>
        </v-card>
      </div>

      <!-- TAB: MODULES -->
      <div v-if="activeTab === 'modules'">
        <div class="d-flex align-center justify-space-between mb-4">
          <h2 class="text-h5 font-weight-bold text-high-emphasis my-0">Course Modules</h2>
          <v-btn
            v-if="isTeacher"
            color="primary"
            prepend-icon="mdi-plus"
            variant="elevated"
            @click="openCreateModuleDialog"
          >
            Add Module
          </v-btn>
        </div>

        <!-- Empty State -->
        <v-card
          v-if="modules.length === 0"
          class="pa-8 text-center bg-surface-variant"
          elevation="0"
          rounded="lg"
        >
          <v-icon class="mb-3 text-medium-emphasis" icon="mdi-folder-open-outline" size="48" />
          <h3 class="text-h6 font-weight-medium text-high-emphasis">No modules created yet</h3>
          <p class="text-body-2 text-medium-emphasis">
            {{
              isTeacher
                ? 'Click "Add Module" to begin designing your course structure.'
                : 'The teacher has not published any modules for this course yet.'
            }}
          </p>
        </v-card>

        <!-- Module List -->
        <div v-else class="d-flex flex-column ga-3">
          <v-card
            v-for="(mod, index) in modules"
            :key="mod.uuid"
            class="pa-4 bg-surface"
            elevation="1"
            rounded="lg"
          >
            <div class="d-flex flex-wrap align-center justify-space-between ga-2">
              <div class="d-flex align-center ga-3">
                <v-avatar color="primary" size="32" variant="tonal">
                  <span class="text-caption font-weight-bold">{{ index + 1 }}</span>
                </v-avatar>

                <div class="d-flex flex-column justify-center">
                  <h3 class="text-subtitle-1 font-weight-bold text-high-emphasis my-0">
                    {{ mod.title }}
                  </h3>
                  <div v-if="mod.dependsOn" class="d-flex align-center ga-1 mt-1">
                    <v-icon color="secondary" icon="mdi-link-variant" size="small" />
                    <span class="text-caption text-secondary">
                      Prerequisite: {{ mod.dependsOn.title }}
                    </span>
                  </div>
                </div>
              </div>

              <!-- Teacher Reorder and Manage Controls -->
              <div v-if="isTeacher" class="d-flex align-center ga-1">
                <v-btn
                  density="comfortable"
                  :disabled="index === 0"
                  icon="mdi-arrow-up"
                  title="Move Up"
                  variant="text"
                  @click="moveModule(index, -1)"
                />
                <v-btn
                  density="comfortable"
                  :disabled="index === modules.length - 1"
                  icon="mdi-arrow-down"
                  title="Move Down"
                  variant="text"
                  @click="moveModule(index, 1)"
                />
                <v-btn
                  density="comfortable"
                  icon="mdi-pencil-outline"
                  title="Edit Module"
                  variant="text"
                  @click="openEditModuleDialog(mod)"
                />
                <v-btn
                  color="error"
                  density="comfortable"
                  icon="mdi-delete-outline"
                  title="Delete Module"
                  variant="text"
                  @click="handleDeleteModule(mod.uuid)"
                />
              </div>
            </div>

            <!-- Course Materials Section -->
            <v-divider class="my-3" />
            <div class="d-flex align-center justify-space-between mb-2">
              <span class="text-caption font-weight-bold text-medium-emphasis">
                Materials ({{ (modulePages[mod.uuid] || []).length }})
              </span>
              <v-menu v-if="isTeacher">
                <template #activator="{ props: menuProps }">
                  <v-btn
                    v-bind="menuProps"
                    size="small"
                    color="primary"
                    variant="text"
                    prepend-icon="mdi-plus"
                    append-icon="mdi-chevron-down"
                  >
                    Add
                  </v-btn>
                </template>
                <v-list density="compact" elevation="2" rounded="md">
                  <v-list-item
                    prepend-icon="mdi-file-document-outline"
                    title="Material"
                    @click="openUploadDialog(mod.uuid)"
                  />
                  <v-list-item
                    prepend-icon="mdi-help-circle-outline"
                    title="Quiz"
                    @click="handleComingSoon('Quiz')"
                  />
                  <v-list-item
                    prepend-icon="mdi-dots-horizontal"
                    title="etc"
                    @click="handleComingSoon('etc')"
                  />
                </v-list>
              </v-menu>
            </div>

            <!-- Material Items -->
            <div v-if="(modulePages[mod.uuid] || []).length > 0" class="d-flex flex-column ga-2">
              <v-card
                v-for="page in modulePages[mod.uuid]"
                :key="page.uuid"
                class="pa-3 bg-surface-variant d-flex align-center justify-space-between"
                elevation="0"
                rounded="md"
              >
                <div
                  class="d-flex align-center ga-3 overflow-hidden cursor-pointer flex-grow-1"
                  @click="viewMaterial(mod.uuid, page.uuid)"
                >
                  <v-avatar size="32" :color="getPageFormatColor(page)" variant="tonal">
                    <v-icon :icon="getPageFormatIcon(page)" size="small" />
                  </v-avatar>
                  <div class="d-flex flex-column overflow-hidden">
                    <span class="text-body-2 font-weight-bold text-high-emphasis text-truncate">
                      {{ page.title }}
                    </span>
                    <span class="text-caption text-medium-emphasis text-truncate">
                      <template v-if="page.fileSize">{{ formatBytes(page.fileSize) }} • </template>
                      <template v-if="page.createdAt">{{ formatDate(page.createdAt) }}</template>
                    </span>
                  </div>
                </div>

                <div class="d-flex align-center ga-1 ml-2">
                  <v-btn
                    size="small"
                    color="primary"
                    variant="tonal"
                    prepend-icon="mdi-eye-outline"
                    @click="viewMaterial(mod.uuid, page.uuid)"
                  >
                    Preview
                  </v-btn>
                  <v-btn
                    v-if="isTeacher"
                    size="small"
                    color="error"
                    icon="mdi-delete-outline"
                    variant="text"
                    title="Delete Material"
                    @click="handleDeleteMaterial(mod.uuid, page.uuid)"
                  />
                </div>
              </v-card>
            </div>
            <div v-else class="text-caption text-medium-emphasis py-1 font-italic">
              No materials uploaded for this module yet.
            </div>
          </v-card>
        </div>
      </div>

      <!-- TAB 2: PARTICIPANTS (Teachers only) -->
      <div v-if="activeTab === 'participants' && isTeacher">
        <div class="d-flex align-center justify-space-between mb-4">
          <h2 class="text-h5 font-weight-bold text-high-emphasis">Course Participants</h2>
          <v-btn
            color="primary"
            prepend-icon="mdi-account-plus"
            variant="elevated"
            @click="addEnrollmentDialogOpen = true"
          >
            Add Participant
          </v-btn>
        </div>

        <v-card class="bg-surface" elevation="1" rounded="lg">
          <v-table>
            <thead>
              <tr>
                <th class="text-left font-weight-bold">User Email</th>
                <th class="text-left font-weight-bold">Course Role</th>
                <th class="text-left font-weight-bold">Enrolled At</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="e in enrollments" :key="e.userEmail">
                <td class="font-weight-medium">{{ e.userEmail }}</td>
                <td>
                  <v-chip
                    :color="e.role === 'TEACHER' ? 'primary' : 'secondary'"
                    size="small"
                    variant="flat"
                  >
                    {{ e.role }}
                  </v-chip>
                </td>
                <td class="text-medium-emphasis">
                  {{ new Date(e.enrolledAt).toLocaleDateString() }}
                </td>
              </tr>
            </tbody>
          </v-table>
        </v-card>
      </div>

      <!-- TAB: ASSIGNMENTS -->
      <div v-if="activeTab === 'assignments'">
        <v-card class="pa-6 bg-surface mb-6 text-center" elevation="1" rounded="lg">
          <v-icon class="mb-3 text-medium-emphasis" icon="mdi-clipboard-text-outline" size="48" />
          <h2 class="text-h5 font-weight-bold text-high-emphasis mb-2">Assignments</h2>
          <p class="text-body-1 text-medium-emphasis mb-0">
            TODO: Set when assignments are implemented
          </p>
        </v-card>
      </div>

      <!-- TAB: GROUPS -->
      <div v-if="activeTab === 'groups'">
        <v-card class="pa-6 bg-surface mb-6 text-center" elevation="1" rounded="lg">
          <v-icon class="mb-3 text-medium-emphasis" icon="mdi-account-multiple" size="48" />
          <h2 class="text-h5 font-weight-bold text-high-emphasis mb-2">Groups</h2>
          <p class="text-body-1 text-medium-emphasis mb-0">TODO: Set when groups are implemented</p>
        </v-card>
      </div>

      <!-- TAB 3: SETTINGS (Teachers only) -->
      <div v-if="activeTab === 'settings' && isTeacher">
        <v-card class="pa-6 bg-surface mb-6" elevation="1" rounded="lg">
          <h2 class="text-h5 font-weight-bold text-high-emphasis mb-4">Course Details</h2>

          <v-form @submit.prevent="handleUpdateCourse">
            <v-text-field
              v-model="editCourseData.title"
              class="mb-3"
              label="Course Title"
              required
              variant="outlined"
            />
            <v-textarea
              v-model="editCourseData.description"
              class="mb-3"
              label="Course Description"
              rows="3"
              variant="outlined"
            />
            <v-row dense>
              <v-col cols="6">
                <v-select
                  v-model="editCourseData.term"
                  :items="['AUTUMN', 'SPRING']"
                  label="Term"
                  variant="outlined"
                />
              </v-col>
              <v-col cols="6">
                <v-select
                  v-model="editCourseData.year"
                  :items="[2025, 2026, 2027, 2028]"
                  label="Year"
                  variant="outlined"
                />
              </v-col>
            </v-row>

            <v-btn
              class="mt-2"
              color="primary"
              :loading="savingCourse"
              size="large"
              type="submit"
              variant="elevated"
            >
              Save Changes
            </v-btn>
          </v-form>
        </v-card>

        <!-- Danger Zone -->
        <v-card class="pa-6 bg-surface border-error" elevation="0" rounded="lg">
          <h3 class="text-h6 font-weight-bold text-error mb-2">Danger Zone</h3>
          <p class="text-body-2 text-medium-emphasis mb-4">
            Deleting this course will permanently remove all modules, enrollments, and student
            submissions.
          </p>
          <v-btn
            color="error"
            :loading="deletingCourse"
            prepend-icon="mdi-delete"
            variant="outlined"
            @click="handleDeleteCourse"
          >
            Delete Course
          </v-btn>
        </v-card>
      </div>
    </div>

    <!-- Create/Edit Module Dialog -->
    <v-dialog v-model="moduleDialogOpen" max-width="500" persistent>
      <v-card class="bg-surface pa-2" rounded="lg">
        <v-card-title class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis">
          {{ isEditingModule ? 'Edit Module' : 'Create Module' }}
        </v-card-title>

        <v-card-text class="px-4 pt-4">
          <v-text-field
            v-model="moduleForm.title"
            class="mb-3"
            label="Module Title"
            placeholder="e.g. Week 1: Architecture & Patterns"
            required
            variant="outlined"
          />

          <v-select
            v-model="moduleForm.dependsOnUuid"
            clearable
            :items="availablePrerequisites"
            item-title="title"
            item-value="uuid"
            label="Prerequisite Module (Optional)"
            placeholder="None"
            variant="outlined"
          />
        </v-card-text>

        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="medium-emphasis" variant="text" @click="moduleDialogOpen = false">
            Cancel
          </v-btn>
          <v-btn
            color="primary"
            :loading="savingModule"
            variant="elevated"
            @click="handleSaveModule"
          >
            {{ isEditingModule ? 'Save' : 'Create' }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Add Participant Dialog -->
    <v-dialog v-model="addEnrollmentDialogOpen" max-width="500" persistent>
      <v-card class="bg-surface pa-2" rounded="lg">
        <v-card-title class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis">
          Add Participant
        </v-card-title>
        <v-card-subtitle class="px-4 text-medium-emphasis">
          Enroll an existing user into this course by their registered email.
        </v-card-subtitle>

        <v-card-text class="px-4 pt-4">
          <v-text-field
            v-model="newEnrollmentEmail"
            class="mb-3"
            label="User Email"
            placeholder="student@hvl.no"
            required
            type="email"
            variant="outlined"
          />

          <v-select
            v-model="newEnrollmentRole"
            :items="['STUDENT', 'TEACHER']"
            label="Role"
            variant="outlined"
          />
        </v-card-text>

        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="medium-emphasis" variant="text" @click="addEnrollmentDialogOpen = false">
            Cancel
          </v-btn>
          <v-btn
            color="primary"
            :loading="addingEnrollment"
            variant="elevated"
            @click="handleAddEnrollment"
          >
            Add Participant
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- Upload Material Dialog -->
    <v-dialog v-model="uploadDialogOpen" max-width="540" persistent>
      <v-card class="bg-surface pa-2" rounded="lg">
        <v-card-title class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis">
          Upload Course Material
        </v-card-title>
        <v-card-subtitle class="px-4 text-medium-emphasis">
          Upload a file (.pdf, .pptx, .md, .txt) or enter an external URL for embedded view.
        </v-card-subtitle>

        <v-card-text class="px-4 pt-4">
          <v-file-input
            v-model="uploadFile"
            class="mb-3"
            clearable
            label="Select File"
            placeholder="Choose .pdf, .docx, .pptx, image, .md, .txt"
            prepend-icon="mdi-paperclip"
            show-size
            variant="outlined"
          />

          <v-text-field
            v-model="uploadSourceUrl"
            class="mb-3"
            label="External / Iframe URL (Optional if file uploaded)"
            placeholder="https://example.com/embed"
            variant="outlined"
          />

          <v-textarea
            v-model="uploadDescription"
            label="Description / Notes (Optional)"
            placeholder="Brief description of the material..."
            rows="2"
            variant="outlined"
          />
        </v-card-text>

        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="medium-emphasis" variant="text" @click="uploadDialogOpen = false">
            Cancel
          </v-btn>
          <v-btn
            color="primary"
            :disabled="!uploadFile && !uploadSourceUrl.trim()"
            :loading="uploading"
            variant="elevated"
            @click="handleUploadMaterial"
          >
            Upload
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { useToast } from '@/composables/useToast'
import {
  getCourse,
  getMyCourses,
  listModules,
  getCourseEnrollments,
  updateModule,
  createModule,
  reorderModules,
  deleteModule,
  addEnrollment,
  updateCourse,
  deleteCourse,
} from '@/api/course'
import { listStaticPages, uploadStaticPage, deleteStaticPage } from '@/api/content'
import type { Course, Module, Enrollment, UpdateCoursePayload } from '@/types/course'
import type { StaticPage } from '@/types/content'

const props = defineProps<{
  courseUuid?: string
}>()

const emit = defineEmits<{
  (e: 'back'): void
  (e: 'course-deleted'): void
}>()

const route = useRoute()
const router = useRouter()
const { hasRole } = useAuth()
const toast = useToast()

const resolvedCourseUuid = computed(() => props.courseUuid || String(route.params.uuid || ''))

const activeTab = ref('modules')
const loading = ref(false)
const course = ref<Course | undefined>(undefined)
const modules = ref<Module[]>([])
const enrollments = ref<Enrollment[]>([])
const userRole = ref<'STUDENT' | 'TEACHER' | undefined>(undefined)

const isTeacher = computed(
  () => userRole.value === 'TEACHER' || hasRole('admin') || hasRole('superadmin'),
)

function goBack() {
  emit('back')
  router.push('/courses')
}

// Settings edit form
const editCourseData = ref<UpdateCoursePayload>({
  title: '',
  description: '',
  term: 'AUTUMN',
  year: 2026,
})
const savingCourse = ref(false)
const deletingCourse = ref(false)

// Module dialog form
const moduleDialogOpen = ref(false)
const isEditingModule = ref(false)
const editingModuleUuid = ref<string | undefined>(undefined)
const moduleForm = ref<{ title: string; dependsOnUuid?: string }>({
  title: '',
  dependsOnUuid: undefined,
})
const savingModule = ref(false)

// Participant dialog
const addEnrollmentDialogOpen = ref(false)
const newEnrollmentEmail = ref('')
const newEnrollmentRole = ref<'STUDENT' | 'TEACHER'>('STUDENT')
const addingEnrollment = ref(false)

// Course materials state
const modulePages = ref<Record<string, StaticPage[]>>({})
const uploadDialogOpen = ref(false)
const targetModuleUuid = ref<string>('')
const uploadFile = ref<File | File[] | null>(null)
const uploadDescription = ref('')
const uploadSourceUrl = ref('')
const uploading = ref(false)

const availablePrerequisites = computed(() => {
  return modules.value.filter(m => {
    if (isEditingModule.value && m.uuid === editingModuleUuid.value) {
      return false
    }
    return true
  })
})

async function loadAllModulePages() {
  if (!resolvedCourseUuid.value) return
  const pagesMap: Record<string, StaticPage[]> = {}
  await Promise.all(
    modules.value.map(async mod => {
      try {
        pagesMap[mod.uuid] = await listStaticPages(resolvedCourseUuid.value, mod.uuid)
      } catch {
        pagesMap[mod.uuid] = []
      }
    }),
  )
  modulePages.value = pagesMap
}

function openUploadDialog(modUuid: string) {
  targetModuleUuid.value = modUuid
  uploadFile.value = null
  uploadDescription.value = ''
  uploadSourceUrl.value = ''
  uploadDialogOpen.value = true
}

async function handleUploadMaterial() {
  if (!targetModuleUuid.value || !resolvedCourseUuid.value) return
  const rawFile = Array.isArray(uploadFile.value) ? uploadFile.value[0] : uploadFile.value

  if (!rawFile && !uploadSourceUrl.value.trim()) {
    toast.error('Please select a file or provide a URL.')
    return
  }

  uploading.value = true
  try {
    await uploadStaticPage(resolvedCourseUuid.value, targetModuleUuid.value, {
      file: rawFile || undefined,
      description: uploadDescription.value.trim() || undefined,
      sourceUrl: uploadSourceUrl.value.trim() || undefined,
    })
    toast.success('Course material uploaded successfully.')
    uploadDialogOpen.value = false
    modulePages.value[targetModuleUuid.value] = await listStaticPages(
      resolvedCourseUuid.value,
      targetModuleUuid.value,
    )
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to upload course material.')
  } finally {
    uploading.value = false
  }
}

async function handleDeleteMaterial(moduleUuid: string, pageUuid: string) {
  if (!confirm('Are you sure you want to delete this course material?')) return
  try {
    await deleteStaticPage(resolvedCourseUuid.value, moduleUuid, pageUuid)
    toast.success('Course material deleted.')
    modulePages.value[moduleUuid] = await listStaticPages(resolvedCourseUuid.value, moduleUuid)
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to delete course material.')
  }
}

function viewMaterial(moduleUuid: string, pageUuid: string) {
  router.push(`/courses/${resolvedCourseUuid.value}/modules/${moduleUuid}/pages/${pageUuid}`)
}

function handleComingSoon(item: string) {
  toast.info(`${item} feature coming soon`)
}

function getPageFormatIcon(page: StaticPage): string {
  const name = page.fileName?.toLowerCase() || ''
  if (page.contentType?.includes('pdf') || name.endsWith('.pdf')) return 'mdi-file-pdf-box'
  if (page.contentType?.includes('word') || name.endsWith('.docx') || name.endsWith('.doc')) {
    return 'mdi-file-word-box'
  }
  if (
    page.contentType?.includes('presentation') ||
    page.contentType?.includes('powerpoint') ||
    name.endsWith('.pptx') ||
    name.endsWith('.ppt')
  ) {
    return 'mdi-file-powerpoint-box'
  }
  if (
    page.contentType?.startsWith('image/') ||
    ['.png', '.jpg', '.jpeg', '.gif', '.svg', '.webp'].some(ext => name.endsWith(ext))
  ) {
    return 'mdi-file-image'
  }
  if (name.endsWith('.md')) return 'mdi-language-markdown'
  if (page.contentType?.includes('text') || name.endsWith('.txt'))
    return 'mdi-file-document-outline'
  if (page.sourceUrl) return 'mdi-link-variant'
  return 'mdi-file-outline'
}

function getPageFormatColor(page: StaticPage): string {
  const name = page.fileName?.toLowerCase() || ''
  if (page.contentType?.includes('pdf') || name.endsWith('.pdf')) return 'error'
  if (page.contentType?.includes('word') || name.endsWith('.docx') || name.endsWith('.doc')) {
    return 'primary'
  }
  if (
    page.contentType?.includes('presentation') ||
    page.contentType?.includes('powerpoint') ||
    name.endsWith('.pptx') ||
    name.endsWith('.ppt')
  ) {
    return 'warning'
  }
  if (
    page.contentType?.startsWith('image/') ||
    ['.png', '.jpg', '.jpeg', '.gif', '.svg', '.webp'].some(ext => name.endsWith(ext))
  ) {
    return 'info'
  }
  if (name.endsWith('.md') || page.contentType?.includes('text') || name.endsWith('.txt'))
    return 'primary'
  if (page.sourceUrl) return 'info'
  return 'secondary'
}

function formatBytes(bytes?: number): string {
  if (!bytes) return ''
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function formatDate(isoDate?: string): string {
  if (!isoDate) return ''
  return new Date(isoDate).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

async function fetchCourseDetails() {
  if (!resolvedCourseUuid.value) return
  loading.value = true
  try {
    course.value = await getCourse(resolvedCourseUuid.value)
    editCourseData.value = {
      title: course.value.title,
      description: course.value.description,
      term: course.value.term,
      year: course.value.year,
    }

    // Determine current user's role in this course
    const myCourses = await getMyCourses().catch(() => [])
    const mine = myCourses.find(mc => mc.course.uuid === resolvedCourseUuid.value)
    userRole.value = mine ? mine.role : undefined

    // Load modules
    modules.value = await listModules(resolvedCourseUuid.value)
    await loadAllModulePages()

    // Load enrollments if teacher
    if (isTeacher.value) {
      enrollments.value = await getCourseEnrollments(resolvedCourseUuid.value).catch(() => [])
    }
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to load course details.')
  } finally {
    loading.value = false
  }
}

function openCreateModuleDialog() {
  isEditingModule.value = false
  editingModuleUuid.value = undefined
  moduleForm.value = { title: '', dependsOnUuid: undefined }
  moduleDialogOpen.value = true
}

function openEditModuleDialog(mod: Module) {
  isEditingModule.value = true
  editingModuleUuid.value = mod.uuid
  moduleForm.value = { title: mod.title, dependsOnUuid: mod.dependsOnUuid }
  moduleDialogOpen.value = true
}

async function handleSaveModule() {
  if (!moduleForm.value.title.trim()) return
  savingModule.value = true
  try {
    if (isEditingModule.value && editingModuleUuid.value) {
      await updateModule(resolvedCourseUuid.value, editingModuleUuid.value, {
        title: moduleForm.value.title,
        dependsOnUuid: moduleForm.value.dependsOnUuid,
      })
      toast.success('Module updated successfully.')
    } else {
      await createModule(resolvedCourseUuid.value, {
        title: moduleForm.value.title,
        dependsOnUuid: moduleForm.value.dependsOnUuid,
      })
      toast.success('Module created successfully.')
    }
    moduleDialogOpen.value = false
    modules.value = await listModules(resolvedCourseUuid.value)
    await loadAllModulePages()
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to save module.')
  } finally {
    savingModule.value = false
  }
}

async function moveModule(index: number, direction: -1 | 1) {
  const targetIndex = index + direction
  if (targetIndex < 0 || targetIndex >= modules.value.length) return

  const reordered = [...modules.value]
  const temp = reordered[index]
  reordered[index] = reordered[targetIndex]
  reordered[targetIndex] = temp

  modules.value = reordered
  try {
    await reorderModules(
      resolvedCourseUuid.value,
      reordered.map(m => m.uuid),
    )
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to reorder modules.')
    modules.value = await listModules(resolvedCourseUuid.value)
  }
}

async function handleDeleteModule(moduleUuid: string) {
  if (!confirm('Are you sure you want to delete this module?')) return
  try {
    await deleteModule(resolvedCourseUuid.value, moduleUuid)
    toast.success('Module deleted.')
    modules.value = await listModules(resolvedCourseUuid.value)
    await loadAllModulePages()
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to delete module.')
  }
}

async function handleAddEnrollment() {
  if (!newEnrollmentEmail.value.trim()) return
  addingEnrollment.value = true
  try {
    await addEnrollment(resolvedCourseUuid.value, {
      email: newEnrollmentEmail.value,
      role: newEnrollmentRole.value,
    })
    addEnrollmentDialogOpen.value = false
    newEnrollmentEmail.value = ''
    toast.success('Participant enrolled successfully.')
    enrollments.value = await getCourseEnrollments(resolvedCourseUuid.value)
    course.value = await getCourse(resolvedCourseUuid.value)
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to add participant.')
  } finally {
    addingEnrollment.value = false
  }
}

async function handleUpdateCourse() {
  savingCourse.value = true
  try {
    const updated = await updateCourse(resolvedCourseUuid.value, editCourseData.value)
    course.value = updated
    toast.success('Course details updated successfully.')
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to update course.')
  } finally {
    savingCourse.value = false
  }
}

async function handleDeleteCourse() {
  if (
    !confirm(
      'Are you ABSOLUTELY sure you want to delete this course? This action cannot be undone.',
    )
  )
    return
  deletingCourse.value = true
  try {
    await deleteCourse(resolvedCourseUuid.value)
    toast.success('Course deleted.')
    emit('course-deleted')
    router.push('/courses')
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to delete course.')
  } finally {
    deletingCourse.value = false
  }
}

onMounted(() => {
  fetchCourseDetails()
})
</script>
