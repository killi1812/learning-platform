<template>
  <v-container class="py-6" max-width="1400">
    <!-- Top Header & Actions -->
    <div class="d-flex flex-wrap align-center justify-space-between mb-6 ga-4">
      <div>
        <h1 class="text-h4 font-weight-bold text-high-emphasis">Course Catalog</h1>
      </div>

      <v-btn
        v-if="isAuthenticated"
        color="primary"
        prepend-icon="mdi-plus"
        size="large"
        variant="elevated"
        @click="createDialogOpen = true"
      >
        Create Course
      </v-btn>
    </div>

    <!-- Filters Bar -->
    <v-card class="mb-6 pa-4 bg-surface" elevation="1" rounded="lg">
      <v-row align="center" dense>
        <v-col cols="12" md="6">
          <v-text-field
            v-model="searchQuery"
            clearable
            density="comfortable"
            hide-details
            label="Search courses by title or description..."
            prepend-inner-icon="mdi-magnify"
            variant="outlined"
          />
        </v-col>

        <v-col cols="6" md="3">
          <v-select
            v-model="selectedTerm"
            clearable
            density="comfortable"
            hide-details
            :items="['AUTUMN', 'SPRING']"
            label="Filter by Term"
            variant="outlined"
            @update:model-value="fetchCourses"
          />
        </v-col>

        <v-col cols="6" md="3">
          <v-select
            v-model="selectedYear"
            clearable
            density="comfortable"
            hide-details
            :items="[2025, 2026, 2027]"
            label="Filter by Year"
            variant="outlined"
            @update:model-value="fetchCourses"
          />
        </v-col>
      </v-row>
    </v-card>

    <!-- Loading State -->
    <div v-if="loading" class="d-flex justify-center py-12">
      <v-progress-circular color="primary" indeterminate size="64" />
    </div>

    <!-- Empty State -->
    <v-card
      v-else-if="filteredCourses.length === 0"
      class="pa-12 text-center bg-surface-variant"
      elevation="0"
      rounded="lg"
    >
      <v-icon class="mb-4 text-medium-emphasis" icon="mdi-book-search-outline" size="64" />
      <h3 class="text-h6 font-weight-medium text-high-emphasis">No courses found</h3>
      <p class="text-body-2 text-medium-emphasis">
        {{
          searchQuery || selectedTerm || selectedYear
            ? 'Try adjusting your filters or search terms.'
            : 'No courses are currently available in the catalog.'
        }}
      </p>
    </v-card>

    <!-- Course Grid -->
    <v-row v-else>
      <v-col v-for="course in filteredCourses" :key="course.uuid" cols="12" md="6" lg="4">
        <CourseCard
          :course="course"
          :role="enrolledMap[course.uuid]"
          @click="selectCourse(course.uuid)"
        >
          <template #actions>
            <span class="text-caption text-medium-emphasis">
              {{ course.term }} {{ course.year }}
            </span>

            <div class="d-flex align-center ga-2">
              <v-btn
                color="primary"
                size="small"
                variant="text"
                @click.stop="selectCourse(course.uuid)"
              >
                View Modules
              </v-btn>

              <v-btn
                v-if="!enrolledMap[course.uuid]"
                color="secondary"
                :disabled="!isAuthenticated"
                :loading="enrollingUuid === course.uuid"
                prepend-icon="mdi-account-plus"
                size="small"
                variant="elevated"
                @click.stop="handleEnroll(course.uuid)"
              >
                {{ isAuthenticated ? 'Enroll' : 'Login to Enroll' }}
              </v-btn>
            </div>
          </template>
        </CourseCard>
      </v-col>
    </v-row>

    <!-- Create Course Dialog -->
    <v-dialog v-model="createDialogOpen" max-width="600" persistent>
      <v-card class="bg-surface pa-2" rounded="lg">
        <v-card-title class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis">
          Create New Course
        </v-card-title>
        <v-card-subtitle class="px-4 text-medium-emphasis">
          You will automatically be enrolled as the course Teacher.
        </v-card-subtitle>

        <v-card-text class="px-4 pt-4">
          <v-form ref="createForm" @submit.prevent="handleCreateCourse">
            <v-text-field
              v-model="newCourse.title"
              class="mb-3"
              label="Course Title"
              placeholder="e.g. DAT250 Advanced Software Technologies"
              required
              :rules="[v => !!v || 'Course title is required']"
              variant="outlined"
            />

            <v-textarea
              v-model="newCourse.description"
              class="mb-3"
              label="Description"
              placeholder="Overview of syllabus, prerequisites, and learning outcomes..."
              rows="3"
              variant="outlined"
            />

            <v-row dense>
              <v-col cols="6">
                <v-select
                  v-model="newCourse.term"
                  :items="['AUTUMN', 'SPRING']"
                  label="Term"
                  required
                  variant="outlined"
                />
              </v-col>
              <v-col cols="6">
                <v-select
                  v-model="newCourse.year"
                  :items="[2025, 2026, 2027, 2028]"
                  label="Year"
                  required
                  variant="outlined"
                />
              </v-col>
            </v-row>

            <!-- Grading Scale (A-F Minimum %) -->
            <div class="mt-4 mb-2">
              <div class="d-flex align-center justify-space-between mb-2">
                <div>
                  <span class="text-subtitle-2 font-weight-bold text-high-emphasis">
                    Grading Scale (Minimum % for A–E)
                  </span>
                  <div class="text-caption text-medium-emphasis">
                    Thresholds for final subject grades. F is anything below E.
                  </div>
                </div>
                <v-btn
                  size="x-small"
                  variant="text"
                  prepend-icon="mdi-refresh"
                  @click="resetNewCourseGradingScale"
                >
                  Reset
                </v-btn>
              </div>

              <v-row dense>
                <v-col cols="6" sm="4">
                  <v-text-field
                    v-model.number="newCourseGradingScale.scaleA"
                    label="Grade A (≥ %)"
                    type="number"
                    min="1"
                    max="100"
                    density="compact"
                    variant="outlined"
                  />
                </v-col>
                <v-col cols="6" sm="4">
                  <v-text-field
                    v-model.number="newCourseGradingScale.scaleB"
                    label="Grade B (≥ %)"
                    type="number"
                    min="1"
                    max="100"
                    density="compact"
                    variant="outlined"
                  />
                </v-col>
                <v-col cols="6" sm="4">
                  <v-text-field
                    v-model.number="newCourseGradingScale.scaleC"
                    label="Grade C (≥ %)"
                    type="number"
                    min="1"
                    max="100"
                    density="compact"
                    variant="outlined"
                  />
                </v-col>
                <v-col cols="6" sm="4">
                  <v-text-field
                    v-model.number="newCourseGradingScale.scaleD"
                    label="Grade D (≥ %)"
                    type="number"
                    min="1"
                    max="100"
                    density="compact"
                    variant="outlined"
                  />
                </v-col>
                <v-col cols="6" sm="4">
                  <v-text-field
                    v-model.number="newCourseGradingScale.scaleE"
                    label="Grade E (≥ %)"
                    type="number"
                    min="1"
                    max="100"
                    density="compact"
                    variant="outlined"
                  />
                </v-col>
                <v-col cols="6" sm="4" class="d-flex align-center">
                  <span class="text-caption text-medium-emphasis">
                    Grade F: &lt; {{ newCourseGradingScale.scaleE }}%
                  </span>
                </v-col>
              </v-row>
            </div>
          </v-form>
        </v-card-text>

        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="medium-emphasis" variant="text" @click="createDialogOpen = false">
            Cancel
          </v-btn>
          <v-btn color="primary" :loading="creating" variant="elevated" @click="handleCreateCourse">
            Create Course
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import CourseCard from '@/components/CourseCard.vue'
import { useAuth } from '@/composables/useAuth'
import { useToast } from '@/composables/useToast'
import { listCourses, getMyCourses, selfEnroll, createCourse } from '@/api/course'
import type { Course, CourseRole, CreateCoursePayload, GradingScale } from '@/types/course'

const emit = defineEmits<{
  (e: 'select-course', courseUuid: string): void
}>()

const router = useRouter()
const { isAuthenticated } = useAuth()
const toast = useToast()

const courses = ref<Course[]>([])
const enrolledMap = ref<Record<string, CourseRole>>({})
const loading = ref(false)
const enrollingUuid = ref<string | undefined>(undefined)

const searchQuery = ref('')
const selectedTerm = ref<string>('')
const selectedYear = ref<number | undefined>(undefined)

const createDialogOpen = ref(false)
const creating = ref(false)
const newCourse = ref<CreateCoursePayload>({
  title: '',
  description: '',
  term: 'AUTUMN',
  year: 2026,
})

const DEFAULT_GRADING_SCALE: GradingScale = {
  scaleA: 90,
  scaleB: 80,
  scaleC: 60,
  scaleD: 50,
  scaleE: 40,
}

const newCourseGradingScale = ref<GradingScale>({ ...DEFAULT_GRADING_SCALE })

function resetNewCourseGradingScale() {
  newCourseGradingScale.value = { ...DEFAULT_GRADING_SCALE }
}

const filteredCourses = computed(() => {
  return courses.value.filter(c => {
    if (
      searchQuery.value &&
      !c.title.toLowerCase().includes(searchQuery.value.toLowerCase()) &&
      !c.description?.toLowerCase().includes(searchQuery.value.toLowerCase())
    ) {
      return false
    }
    return true
  })
})

function selectCourse(courseUuid: string) {
  emit('select-course', courseUuid)
  router.push(`/courses/${courseUuid}`)
}

async function fetchCourses() {
  loading.value = true
  try {
    const list = await listCourses(selectedTerm.value || undefined, selectedYear.value || undefined)
    courses.value = list
    await fetchMyEnrollments()
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to load course catalog.')
  } finally {
    loading.value = false
  }
}

async function fetchMyEnrollments() {
  if (!isAuthenticated.value) {
    enrolledMap.value = {}
    return
  }
  try {
    const myCourses = await getMyCourses()
    const map: Record<string, CourseRole> = {}
    myCourses.forEach(mc => {
      map[mc.course.uuid] = mc.role
    })
    enrolledMap.value = map
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to load enrolled courses.')
  }
}

async function handleEnroll(courseUuid: string) {
  enrollingUuid.value = courseUuid
  try {
    const res = await selfEnroll(courseUuid)
    enrolledMap.value[courseUuid] = res.role
    toast.success('Successfully enrolled as STUDENT in the course!')
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Enrollment failed.')
  } finally {
    enrollingUuid.value = undefined
  }
}

async function handleCreateCourse() {
  if (!newCourse.value.title.trim()) return

  const s = newCourseGradingScale.value
  if (
    s.scaleA > 100 ||
    s.scaleA <= s.scaleB ||
    s.scaleB <= s.scaleC ||
    s.scaleC <= s.scaleD ||
    s.scaleD <= s.scaleE ||
    s.scaleE <= 0
  ) {
    toast.error('Invalid grading scale: Must satisfy 100 ≥ A > B > C > D > E > 0.')
    return
  }

  creating.value = true
  try {
    const payload: CreateCoursePayload = {
      ...newCourse.value,
      gradingScale: { ...s },
    }
    const created = await createCourse(payload)
    createDialogOpen.value = false
    newCourse.value = {
      title: '',
      description: '',
      term: 'AUTUMN',
      year: 2026,
    }
    resetNewCourseGradingScale()
    toast.success(`Course "${created.title}" successfully created! You are enrolled as TEACHER.`)
    await fetchCourses()
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to create course.')
  } finally {
    creating.value = false
  }
}

onMounted(() => {
  fetchCourses()
})
</script>
