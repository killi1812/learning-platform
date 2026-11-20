<template>
  <div v-for="course in courses" :key="course.courseUuid" class="mb-3">
    <div class="d-flex justify-space-between text-body-2">
      <router-link
        class="text-high-emphasis font-weight-medium"
        :to="`/courses/${course.courseUuid}`"
      >
        {{ course.courseName }}
      </router-link>
      <span class="text-medium-emphasis">
        {{ course.totalScore }} / {{ course.totalMaxScore }} pts
      </span>
    </div>
    <v-progress-linear
      class="mt-1"
      color="primary"
      height="8"
      :model-value="course.percent"
      rounded
    />
  </div>
</template>

<script setup lang="ts">
import type { CourseFinalGradeDto } from '@/types/grade'

defineProps<{ courses: readonly CourseFinalGradeDto[] }>()
</script>
