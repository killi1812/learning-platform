<template>
  <v-container class="py-6" max-width="1200">
    <!-- Back Navigation & Actions -->
    <div class="d-flex align-center justify-space-between mb-4">
      <v-btn color="medium-emphasis" prepend-icon="mdi-arrow-left" variant="text" @click="goBack">
        Back to Courses
      </v-btn>

      <v-chip
        v-if="userRole"
        :color="userRole === 'TEACHER' ? 'primary' : 'secondary'"
        size="default"
        variant="flat"
      >
        {{ userRole }}
      </v-chip>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="d-flex justify-center py-12">
      <v-progress-circular color="primary" indeterminate size="64" />
    </div>

    <div v-else-if="course">
      <!-- Course Hero Card -->
      <v-card class="mb-6 pa-6 bg-surface" elevation="2" rounded="lg">
        <div class="d-flex flex-wrap align-center justify-space-between ga-2 mb-2">
          <v-chip color="secondary" size="small" variant="tonal">
            {{ course.term }} {{ course.year }}
          </v-chip>
        </div>

        <h1 class="text-h4 font-weight-bold text-high-emphasis mb-2">
          {{ course.title }}
        </h1>

        <div
          v-if="course.teachers && course.teachers.length > 0"
          class="d-flex align-center ga-2 mb-3 text-body-2 text-medium-emphasis"
        >
          <v-icon icon="mdi-account-tie-outline" size="small" />
          <span>
            {{ course.teachers.length === 1 ? 'Professor' : 'Professors' }}:
            <strong class="text-high-emphasis">{{ course.teachers.join(', ') }}</strong>
          </span>
        </div>

        <p class="text-body-1 text-medium-emphasis mb-0">
          {{ course.description || 'No course description provided.' }}
        </p>
      </v-card>

      <!-- Tabs Navigation -->
      <v-tabs v-model="activeTab" class="mb-6" color="primary">
        <v-tab value="modules">
          <v-icon start icon="mdi-view-module" />
          Course Modules ({{ modules.length }})
        </v-tab>
        <v-tab v-if="isTeacher" value="participants">
          <v-icon start icon="mdi-account-group" />
          Participants ({{ enrollments.length }})
        </v-tab>
        <v-tab v-if="isTeacher" value="settings">
          <v-icon start icon="mdi-cog" />
          Course Settings
        </v-tab>
      </v-tabs>

      <!-- TAB 1: MODULES -->
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
            :key="mod.id"
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
                <v-menu>
                  <template #activator="{ props: menuProps }">
                    <v-btn
                      v-bind="menuProps"
                      color="primary"
                      density="comfortable"
                      prepend-icon="mdi-plus"
                      size="small"
                      variant="tonal"
                    >
                      Add Content
                    </v-btn>
                  </template>
                  <v-list density="compact">
                    <v-list-item
                      v-for="option in contentTypeOptions"
                      :key="option.type"
                      :prepend-icon="option.icon"
                      :title="option.label"
                      @click="handleAddContentClick(mod, option.type)"
                    />
                  </v-list>
                </v-menu>

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
                  @click="handleDeleteModule(mod.id)"
                />
              </div>
            </div>

            <!-- Module Content Items Slot & List -->
            <slot :is-teacher="isTeacher" :module="mod" name="module-content">
              <v-list
                v-if="mod.contents && mod.contents.length > 0"
                class="mt-3 bg-surface-variant rounded pa-1"
                density="comfortable"
              >
                <v-list-item
                  v-for="item in mod.contents"
                  :key="item.id"
                  class="rounded"
                  link
                  @click="handleSelectContent(mod, item)"
                >
                  <template #prepend>
                    <v-icon
                      color="primary"
                      :icon="getContentTypeMeta(item.type).icon"
                      size="small"
                    />
                  </template>

                  <v-list-item-title class="text-body-2 font-weight-medium text-high-emphasis">
                    {{ item.title }}
                  </v-list-item-title>

                  <template #append>
                    <v-chip color="secondary" size="x-small" variant="tonal">
                      {{ getContentTypeMeta(item.type).label }}
                    </v-chip>
                  </template>
                </v-list-item>
              </v-list>

              <v-sheet
                v-else
                class="mt-3 pa-3 rounded text-caption bg-surface-variant d-flex align-center justify-space-between ga-2"
              >
                <div class="d-flex align-center ga-2">
                  <v-icon color="medium-emphasis" icon="mdi-layers-outline" size="small" />
                  <span class="text-medium-emphasis">No content items in this module yet.</span>
                </div>
              </v-sheet>
            </slot>
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
              <tr v-for="e in enrollments" :key="e.id">
                <td class="font-weight-medium">{{ e.userEmail || `User #${e.userId}` }}</td>
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
            v-model="moduleForm.dependsOnId"
            clearable
            :items="availablePrerequisites"
            item-title="title"
            item-value="id"
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
  </v-container>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { useToast } from '@/composables/useToast'
import {
  courseApi as api,
  type Course,
  type Module,
  type ContentType,
  type ModuleContentItem,
  type Enrollment,
  type UpdateCoursePayload,
} from '@/api/course'

const props = defineProps<{
  courseUuid?: string
}>()

const emit = defineEmits<{
  (e: 'back'): void
  (e: 'course-deleted'): void
  (e: 'add-content', payload: { module: Module; type: ContentType }): void
  (e: 'select-content', payload: { module: Module; content: ModuleContentItem }): void
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

const contentTypeOptions: Array<{ type: ContentType; label: string; icon: string }> = [
  { type: 'STATIC_PAGE', label: 'Static Page', icon: 'mdi-file-document-outline' },
  { type: 'QUIZ', label: 'Quiz', icon: 'mdi-help-circle-outline' },
  { type: 'ASSIGNMENT', label: 'Assignment', icon: 'mdi-clipboard-text-outline' },
  { type: 'CODING_EXERCISE', label: 'Coding Exercise', icon: 'mdi-code-tags' },
]

function getContentTypeMeta(type: ContentType) {
  return (
    contentTypeOptions.find(opt => opt.type === type) || {
      type,
      label: type,
      icon: 'mdi-file-outline',
    }
  )
}

function handleAddContentClick(mod: Module, type: ContentType) {
  emit('add-content', { module: mod, type })
}

function handleSelectContent(mod: Module, item: ModuleContentItem) {
  emit('select-content', { module: mod, content: item })
}

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
const editingModuleId = ref<number | undefined>(undefined)
const moduleForm = ref<{ title: string; dependsOnId?: number }>({
  title: '',
  dependsOnId: undefined,
})
const savingModule = ref(false)

// Participant dialog
const addEnrollmentDialogOpen = ref(false)
const newEnrollmentEmail = ref('')
const newEnrollmentRole = ref<'STUDENT' | 'TEACHER'>('STUDENT')
const addingEnrollment = ref(false)

const availablePrerequisites = computed(() => {
  return modules.value.filter(m => {
    if (isEditingModule.value && m.id === editingModuleId.value) {
      return false
    }
    return true
  })
})

async function fetchCourseDetails() {
  if (!resolvedCourseUuid.value) return
  loading.value = true
  try {
    course.value = await api.getCourse(resolvedCourseUuid.value)
    editCourseData.value = {
      title: course.value.title,
      description: course.value.description,
      term: course.value.term,
      year: course.value.year,
    }

    // Determine current user's role in this course
    const myCourses = await api.getMyCourses().catch(() => [])
    const mine = myCourses.find(mc => mc.course.uuid === resolvedCourseUuid.value)
    userRole.value = mine ? mine.role : undefined

    // Load modules
    modules.value = await api.listModules(resolvedCourseUuid.value)

    // Load enrollments if teacher
    if (isTeacher.value) {
      enrollments.value = await api.getCourseEnrollments(resolvedCourseUuid.value).catch(() => [])
    }
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to load course details.')
  } finally {
    loading.value = false
  }
}

function openCreateModuleDialog() {
  isEditingModule.value = false
  editingModuleId.value = undefined
  moduleForm.value = { title: '', dependsOnId: undefined }
  moduleDialogOpen.value = true
}

function openEditModuleDialog(mod: Module) {
  isEditingModule.value = true
  editingModuleId.value = mod.id
  moduleForm.value = { title: mod.title, dependsOnId: mod.dependsOnId }
  moduleDialogOpen.value = true
}

async function handleSaveModule() {
  if (!moduleForm.value.title.trim()) return
  savingModule.value = true
  try {
    if (isEditingModule.value && editingModuleId.value) {
      await api.updateModule(resolvedCourseUuid.value, editingModuleId.value, {
        title: moduleForm.value.title,
        dependsOnId: moduleForm.value.dependsOnId,
      })
      toast.success('Module updated successfully.')
    } else {
      await api.createModule(resolvedCourseUuid.value, {
        title: moduleForm.value.title,
        dependsOnId: moduleForm.value.dependsOnId,
      })
      toast.success('Module created successfully.')
    }
    moduleDialogOpen.value = false
    modules.value = await api.listModules(resolvedCourseUuid.value)
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
    await api.reorderModules(
      resolvedCourseUuid.value,
      reordered.map(m => m.id),
    )
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to reorder modules.')
    modules.value = await api.listModules(resolvedCourseUuid.value)
  }
}

async function handleDeleteModule(moduleId: number) {
  if (!confirm('Are you sure you want to delete this module?')) return
  try {
    await api.deleteModule(resolvedCourseUuid.value, moduleId)
    toast.success('Module deleted.')
    modules.value = await api.listModules(resolvedCourseUuid.value)
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to delete module.')
  }
}

async function handleAddEnrollment() {
  if (!newEnrollmentEmail.value.trim()) return
  addingEnrollment.value = true
  try {
    await api.addEnrollment(resolvedCourseUuid.value, {
      email: newEnrollmentEmail.value,
      role: newEnrollmentRole.value,
    })
    addEnrollmentDialogOpen.value = false
    newEnrollmentEmail.value = ''
    toast.success('Participant enrolled successfully.')
    enrollments.value = await api.getCourseEnrollments(resolvedCourseUuid.value)
    course.value = await api.getCourse(resolvedCourseUuid.value)
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to add participant.')
  } finally {
    addingEnrollment.value = false
  }
}

async function handleUpdateCourse() {
  savingCourse.value = true
  try {
    const updated = await api.updateCourse(resolvedCourseUuid.value, editCourseData.value)
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
    await api.deleteCourse(resolvedCourseUuid.value)
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
