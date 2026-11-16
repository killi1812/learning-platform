<template>
  <div v-if="loading" class="d-flex justify-center py-12">
    <v-progress-circular color="primary" indeterminate size="64" />
  </div>

  <v-alert v-else-if="error" color="error" icon="mdi-alert-circle-outline" variant="tonal">
    <div class="d-flex flex-wrap align-center justify-space-between ga-3">
      <span>{{ error }}</span>
      <v-btn color="error" size="small" variant="outlined" @click="emit('retry')"> Retry </v-btn>
    </div>
  </v-alert>

  <v-card
    v-else-if="courses.length === 0"
    class="pa-12 text-center bg-surface-variant"
    elevation="0"
    rounded="lg"
  >
    <v-icon class="mb-4 text-medium-emphasis" icon="mdi-school-outline" size="64" />
    <h2 class="text-h6 font-weight-medium text-high-emphasis">
      You are not enrolled in any courses
    </h2>
    <p class="text-body-2 text-medium-emphasis mb-6">
      Browse our course catalog to freely register for courses this semester.
    </p>
    <v-btn color="primary" prepend-icon="mdi-book-search-outline" to="/catalog" variant="elevated">
      Explore Courses
    </v-btn>
  </v-card>

  <div v-else>
    <div v-if="teachingCourses.length > 0" class="mb-8">
      <h2 class="text-h6 font-weight-bold text-high-emphasis mb-4">Teaching:</h2>
      <v-row>
        <v-col v-for="item in teachingCourses" :key="item.course.uuid" cols="12" lg="4" md="6">
          <CourseCard
            :course="item.course"
            :role="item.role"
            :to="`/courses/${item.course.uuid}`"
          />
        </v-col>
      </v-row>
    </div>

    <div v-if="enrolledCourses.length > 0">
      <h2 class="text-h6 font-weight-bold text-high-emphasis mb-4">Enrolled:</h2>
      <v-row>
        <v-col v-for="item in enrolledCourses" :key="item.course.uuid" cols="12" lg="4" md="6">
          <CourseCard
            :course="item.course"
            :role="item.role"
            :to="`/courses/${item.course.uuid}`"
          />
        </v-col>
      </v-row>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { MyCourse } from '@/types/course'
import { computed } from 'vue'
import CourseCard from '@/components/CourseCard.vue'

const props = defineProps<{
  courses: readonly MyCourse[]
  loading: boolean
  error?: string
}>()

const emit = defineEmits<{
  retry: []
}>()

const teachingCourses = computed(() => props.courses.filter(item => item.role === 'TEACHER'))
const enrolledCourses = computed(() => props.courses.filter(item => item.role === 'STUDENT'))
</script>
