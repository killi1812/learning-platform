<template>
  <div>
    <!-- Enrollment Protection Alert -->
    <v-alert
      v-if="!isEnrolled"
      class="mb-6"
      color="info"
      icon="mdi-shield-lock-outline"
      variant="tonal"
    >
      <div class="d-flex flex-wrap align-center justify-space-between ga-3">
        <div>
          <div class="text-subtitle-1 font-weight-bold">You are not enrolled in this course</div>
          <div class="text-body-2 text-medium-emphasis">
            Enroll as a participant to access modules, materials, assignments, and groups.
          </div>
        </div>
        <v-btn
          color="primary"
          :loading="enrolling"
          prepend-icon="mdi-account-plus"
          variant="elevated"
          @click="emit('enroll')"
        >
          Enroll in Course
        </v-btn>
      </div>
    </v-alert>

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
      <p class="text-body-1 text-medium-emphasis mb-0 course-description">
        {{ course.description || 'No course description provided.' }}
      </p>
    </v-card>
  </div>
</template>

<script setup lang="ts">
import type { Course, CourseRole } from '@/types/course'

interface Props {
  course: Course
  userRole?: CourseRole
  isEnrolled: boolean
  enrolling: boolean
}

defineProps<Props>()

const emit = defineEmits<{
  enroll: []
}>()
</script>

<style scoped>
.course-description {
  white-space: pre-line;
}
</style>
