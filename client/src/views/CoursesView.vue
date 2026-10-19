<template>
  <v-container class="py-6" max-width="1400">
    <div class="d-flex flex-wrap align-center justify-space-between mb-6 ga-4">
      <h1 class="text-h4 font-weight-bold text-high-emphasis my-0">My Courses</h1>
      <v-btn
        color="primary"
        prepend-icon="mdi-book-search-outline"
        size="large"
        variant="elevated"
        to="/catalog"
      >
        Explore Course Catalog
      </v-btn>
    </div>

    <div v-if="loading" class="d-flex justify-center py-12">
      <v-progress-circular color="primary" indeterminate size="64" />
    </div>

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
        to="/catalog"
      >
        Explore Courses
      </v-btn>
    </v-card>

    <div v-else>
      <!-- Teaching Section (only shown if teaching at least one course) -->
      <div v-if="teachingCourses.length > 0" class="mb-8">
        <h2 class="text-h6 font-weight-bold text-high-emphasis mb-4">Teaching:</h2>
        <v-row>
          <v-col v-for="item in teachingCourses" :key="item.course.uuid" cols="12" md="6" lg="4">
            <CourseCard
              :course="item.course"
              :role="item.role"
              :to="`/courses/${item.course.uuid}`"
            />
          </v-col>
        </v-row>
      </div>

      <!-- Enrolled Section (only shown if enrolled as student in at least one course) -->
      <div v-if="enrolledCourses.length > 0">
        <h2 class="text-h6 font-weight-bold text-high-emphasis mb-4">Enrolled:</h2>
        <v-row>
          <v-col v-for="item in enrolledCourses" :key="item.course.uuid" cols="12" md="6" lg="4">
            <CourseCard
              :course="item.course"
              :role="item.role"
              :to="`/courses/${item.course.uuid}`"
            />
          </v-col>
        </v-row>
      </div>
    </div>
  </v-container>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import CourseCard from '@/components/CourseCard.vue'
import { useToast } from '@/composables/useToast'
import { getMyCourses, type MyCourse } from '@/api/course'

const toast = useToast()
const myCourses = ref<MyCourse[]>([])
const loading = ref(false)

const teachingCourses = computed(() => myCourses.value.filter(item => item.role === 'TEACHER'))
const enrolledCourses = computed(() => myCourses.value.filter(item => item.role === 'STUDENT'))

async function fetchMyCourses() {
  loading.value = true
  try {
    myCourses.value = await getMyCourses()
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to load enrolled courses.')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchMyCourses()
})
</script>
