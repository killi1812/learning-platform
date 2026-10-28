<template>
  <v-container class="py-6" max-width="1400">
    <!-- Welcome Header -->
    <v-card class="mb-8 pa-6 bg-surface" elevation="2" rounded="lg">
      <div>
        <h1 class="text-h4 font-weight-bold text-high-emphasis mb-2">My Learning Dashboard</h1>
        <p class="text-subtitle-1 text-medium-emphasis mb-0">
          Overview of your active courses, modules, and roles.
        </p>
      </div>
    </v-card>

    <!-- Courses Section -->
    <div class="courses-section">
      <!-- Loading State -->
      <div v-if="loading" class="d-flex justify-center py-12">
        <v-progress-circular color="primary" indeterminate size="64" />
      </div>

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
import { getMyCourses } from '@/api/course'
import type { MyCourse } from '@/types/course'

const emit = defineEmits<{
  (e: 'select-course', courseUuid: string): void
}>()

const router = useRouter()
const toast = useToast()
const myCourses = ref<MyCourse[]>([])
const loading = ref(false)

const teachingCourses = computed(() => myCourses.value.filter(item => item.role === 'TEACHER'))
const enrolledCourses = computed(() => myCourses.value.filter(item => item.role === 'STUDENT'))

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
