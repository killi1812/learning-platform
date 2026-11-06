<template>
  <v-card
    class="pa-3 bg-surface-variant d-flex align-center justify-space-between"
    elevation="0"
    rounded="md"
  >
    <div class="d-flex align-center ga-3 overflow-hidden flex-grow-1">
      <v-avatar size="32" color="secondary" variant="tonal">
        <v-icon icon="mdi-file-upload-outline" size="small" />
      </v-avatar>
      <div class="d-flex flex-column overflow-hidden">
        <span class="text-body-2 font-weight-bold text-high-emphasis text-truncate">
          {{ assignment.title }}
        </span>
        <span class="text-caption text-medium-emphasis text-truncate">
          <template v-if="showModule">{{ assignment.moduleTitle }} • </template>
          {{ deadlineText }}
          <template v-if="assignment.attemptsUsed > 0">
            • {{ assignment.attemptsUsed }}
            {{ assignment.attemptsUsed === 1 ? 'submission' : 'submissions' }}
          </template>
        </span>
      </div>
    </div>

    <div class="d-flex align-center ga-1 ml-2">
      <v-chip v-if="statusChip" :color="statusChip.color" size="small" variant="tonal">
        {{ statusChip.text }}
      </v-chip>

      <template v-if="isTeacher">
        <v-btn
          color="primary"
          icon="mdi-account-group-outline"
          size="small"
          title="Submissions"
          variant="text"
          @click="emit('submissions')"
        />
        <v-btn
          color="primary"
          icon="mdi-pencil-outline"
          size="small"
          title="Edit assignment"
          variant="text"
          @click="emit('edit')"
        />
        <v-btn
          color="error"
          icon="mdi-delete-outline"
          size="small"
          title="Delete assignment"
          variant="text"
          @click="emit('delete')"
        />
      </template>

      <template v-else>
        <v-btn
          :disabled="isClosed"
          color="primary"
          size="small"
          variant="tonal"
          @click="emit('submit')"
        >
          {{ submitLabel }}
        </v-btn>
      </template>
    </div>
  </v-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Assignment } from '@/types/quiz'

const props = defineProps<{
  assignment: Assignment
  isTeacher: boolean
  showModule?: boolean
}>()

const emit = defineEmits<{
  (e: 'edit'): void
  (e: 'submissions'): void
  (e: 'delete'): void
  (e: 'submit'): void
}>()

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

const isPastDeadline = computed(() => {
  if (!props.assignment.deadline) return false
  return Date.now() > new Date(props.assignment.deadline).getTime()
})

const isClosed = computed(() => {
  return isPastDeadline.value && !props.assignment.acceptsLateSubmissions
})

const deadlineText = computed(() => {
  if (!props.assignment.deadline) return 'No deadline'
  return `Due ${formatDateTime(props.assignment.deadline)}`
})

const statusChip = computed(() => {
  if (isClosed.value) {
    return { text: 'Closed', color: 'error' }
  }
  if (isPastDeadline.value) {
    return { text: 'Late Allowed', color: 'warning' }
  }
  if (props.assignment.attemptsUsed > 0) {
    return { text: 'Submitted', color: 'success' }
  }
  return undefined
})

const submitLabel = computed(() => {
  if (isClosed.value) return 'Closed'
  return props.assignment.attemptsUsed > 0 ? 'Resubmit' : 'Submit'
})
</script>
