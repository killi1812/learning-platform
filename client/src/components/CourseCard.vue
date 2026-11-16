<template>
  <v-card
    class="bg-surface d-flex flex-column h-100"
    elevation="2"
    hover
    rounded="lg"
    :to="to"
    @click="emit('click')"
  >
    <div class="pa-4 pb-2">
      <h3 class="course-card__title text-h6 font-weight-bold text-high-emphasis my-0 mb-2">
        {{ course.title }}
      </h3>

      <v-chip
        v-if="role"
        :color="role === 'TEACHER' ? 'primary' : 'secondary'"
        size="small"
        variant="flat"
      >
        {{ role }}
      </v-chip>
    </div>

    <v-card-text class="flex-grow-1 text-body-2 text-medium-emphasis pt-2">
      {{ formattedTeachers }}
    </v-card-text>

    <v-divider />

    <v-card-actions class="pa-4 bg-surface-variant d-flex justify-space-between">
      <slot name="actions">
        <span class="text-caption text-on-surface-variant">
          {{ course.term }} {{ course.year }}
        </span>
        <span class="text-button text-primary font-weight-medium">Open Course</span>
      </slot>
    </v-card-actions>
  </v-card>
</template>

<script setup lang="ts">
import type { Course, CourseRole } from '@/types/course'
import { computed } from 'vue'

const props = defineProps<{
  course: Course
  role?: CourseRole | string
  to?: string
}>()

const emit = defineEmits<{
  click: []
}>()

const formattedTeachers = computed(() => {
  if (!props.course.teachers || props.course.teachers.length === 0) {
    return 'No professors assigned'
  }
  return props.course.teachers.join(', ')
})
</script>

<style scoped>
.course-card__title {
  white-space: normal;
  line-height: 1.35;
}
</style>
