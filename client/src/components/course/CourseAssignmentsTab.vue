<template>
  <div>
    <h2 class="text-h5 font-weight-bold text-high-emphasis mb-4">Assignments</h2>
    <div v-if="assignments.length > 0" class="d-flex flex-column ga-2">
      <template v-for="asgn in assignments" :key="asgn.uuid">
        <HandinAssignmentItem
          v-if="asgn.type === 'HANDIN'"
          :assignment="asgn"
          :is-teacher="isTeacher"
          show-module
          @delete="emit('delete-handin', asgn)"
          @edit="emit('edit-handin', asgn)"
          @submissions="emit('submissions-handin', asgn)"
          @submit="emit('submit-handin', asgn)"
        />
        <QuizListItem
          v-else
          :assignment="asgn"
          :is-teacher="isTeacher"
          show-module
          @delete="emit('delete-quiz', asgn)"
          @edit="emit('edit-quiz', asgn)"
          @results="emit('results-quiz', asgn)"
          @take="emit('take-quiz', asgn)"
        />
      </template>
    </div>
    <v-card v-else class="pa-6 bg-surface mb-6 text-center" elevation="1" rounded="lg">
      <v-icon class="mb-3 text-medium-emphasis" icon="mdi-clipboard-text-outline" size="48" />
      <p class="text-body-1 text-medium-emphasis mb-0">
        Assignments will be available here once published.
      </p>
    </v-card>
  </div>
</template>

<script setup lang="ts">
import type { Assignment } from '@/types/quiz'
import HandinAssignmentItem from '@/components/HandinAssignmentItem.vue'
import QuizListItem from '@/components/QuizListItem.vue'

interface Props {
  assignments: readonly Assignment[]
  isTeacher: boolean
}

defineProps<Props>()

const emit = defineEmits<{
  'edit-handin': [asgn: Assignment]
  'submissions-handin': [asgn: Assignment]
  'delete-handin': [asgn: Assignment]
  'submit-handin': [asgn: Assignment]
  'edit-quiz': [quiz: Assignment]
  'results-quiz': [quiz: Assignment]
  'delete-quiz': [quiz: Assignment]
  'take-quiz': [quiz: Assignment]
}>()
</script>
