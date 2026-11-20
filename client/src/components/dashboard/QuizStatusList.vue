<template>
  <v-list class="pa-0 bg-transparent" density="compact">
    <v-list-item
      v-for="quiz in quizzes"
      :key="quiz.uuid"
      class="px-0"
      :to="`/courses/${quiz.courseUuid}`"
    >
      <v-list-item-title class="font-weight-medium">{{ quiz.title }}</v-list-item-title>
      <v-list-item-subtitle>{{ quiz.courseTitle }}</v-list-item-subtitle>
      <template #append>
        <v-chip :color="quiz.quizFinished ? 'success' : 'info'" size="small" variant="tonal">
          {{ quiz.quizFinished ? 'Done' : `${quiz.attemptsUsed} / ${quiz.maxAttempts} attempts` }}
        </v-chip>
      </template>
    </v-list-item>
  </v-list>
</template>

<script setup lang="ts">
import type { DeadlineItem } from '@/types/dashboard'

defineProps<{ quizzes: readonly DeadlineItem[] }>()
</script>
