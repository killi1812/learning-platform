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
      <h3
        class="text-h6 font-weight-bold text-high-emphasis my-0 mb-2"
        style="white-space: normal; line-height: 1.35"
      >
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
        <span class="text-caption text-medium-emphasis"> {{ course.term }} {{ course.year }} </span>

        <v-btn color="primary" size="small" variant="text">Open Course</v-btn>
      </slot>
    </v-card-actions>
  </v-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Course, CourseRole } from '@/types/course'

const props = defineProps<{
  course: Course
  role?: CourseRole | string
  to?: string
}>()

const emit = defineEmits<{
  (e: 'click'): void
}>()

const formattedTeachers = computed(() => {
  if (!props.course.teachers || props.course.teachers.length === 0) {
    return 'No professors assigned'
  }
  return props.course.teachers.join(', ')
})
</script>
