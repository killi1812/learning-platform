<template>
  <v-container class="py-6" max-width="1400">
    <!-- Welcome Header -->
    <v-card class="mb-8 pa-6 bg-surface" elevation="2" rounded="lg">
      <div class="d-flex flex-wrap align-center justify-space-between ga-4">
        <div>
          <h1 class="text-h4 font-weight-bold text-high-emphasis mb-2">My Learning Dashboard</h1>
          <p class="text-subtitle-1 text-medium-emphasis mb-0">
            Overview of your active courses, modules, and roles.
          </p>
        </div>

        <v-btn
          color="primary"
          prepend-icon="mdi-compass-outline"
          size="large"
          variant="elevated"
          @click="openCatalog"
        >
          Browse Course Catalog
        </v-btn>
      </div>
    </v-card>

    <!-- Courses Section -->
    <div class="courses-section">
      <div class="d-flex align-center justify-space-between mb-4">
        <h2 class="text-h5 font-weight-bold text-high-emphasis">My courses</h2>
        <v-chip color="secondary" size="small" variant="tonal">
          {{ myCourses.length }} Active
        </v-chip>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="d-flex justify-center py-12">
        <v-progress-circular color="primary" indeterminate size="64" />
      </div>

      <!-- Empty State -->
      <v-card
        v-else-if="myCourses.length === 0"
        class="pa-12 text-center bg-surface-variant"
        elevation="0"
        rounded="lg"
      >
        <v-icon class="mb-4 text-medium-emphasis" icon="mdi-school-outline" size="64" />
        <h3 class="text-h6 font-weight-medium text-high-emphasis">
          You are not enrolled in any courses
        </h3>
        <p class="text-body-2 text-medium-emphasis mb-6">
          Browse our course catalog to freely register for courses this semester.
        </p>
        <v-btn
          color="primary"
          prepend-icon="mdi-book-search-outline"
          variant="elevated"
          @click="openCatalog"
        >
          Explore Courses
        </v-btn>
      </v-card>

      <!-- Courses Grid -->
      <div v-else>
        <!-- Teaching Section (only shown if teaching at least one course) -->
        <div v-if="teachingCourses.length > 0" class="mb-8">
          <h3 class="text-h6 font-weight-bold text-high-emphasis mb-4">Teaching:</h3>
          <v-row>
            <v-col v-for="item in teachingCourses" :key="item.course.uuid" cols="12" md="6" lg="4">
              <CourseCard
                :course="item.course"
                :role="item.role"
                @click="selectCourse(item.course.uuid)"
              />
            </v-col>
          </v-row>
        </div>

        <!-- Enrolled Section (only shown if enrolled as student in at least one course) -->
        <div v-if="enrolledCourses.length > 0">
          <h3 class="text-h6 font-weight-bold text-high-emphasis mb-4">Enrolled:</h3>
          <v-row>
            <v-col v-for="item in enrolledCourses" :key="item.course.uuid" cols="12" md="6" lg="4">
              <CourseCard
                :course="item.course"
                :role="item.role"
                @click="selectCourse(item.course.uuid)"
              />
            </v-col>
          </v-row>
        </div>
      </div>
    </div>
  </v-container>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import CourseCard from '@/components/CourseCard.vue'
import { useToast } from '@/composables/useToast'
import { getMyCourses, type MyCourse } from '@/api/course'

const emit = defineEmits<{
  (e: 'select-course', courseUuid: string): void
  (e: 'open-catalog'): void
}>()

const router = useRouter()
const toast = useToast()
const myCourses = ref<MyCourse[]>([])
const loading = ref(false)

const teachingCourses = computed(() => myCourses.value.filter(item => item.role === 'TEACHER'))
const enrolledCourses = computed(() => myCourses.value.filter(item => item.role === 'STUDENT'))

function openCatalog() {
  emit('open-catalog')
  router.push('/catalog')
}

function selectCourse(courseUuid: string) {
  emit('select-course', courseUuid)
  router.push(`/courses/${courseUuid}`)
}

async function fetchMyCourses() {
  loading.value = true
  try {
    myCourses.value = await getMyCourses()
  } catch (err: unknown) {
    myCourses.value = []
    toast.error(err instanceof Error ? err.message : 'Failed to load enrolled courses.')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchMyCourses()
})
</script>
