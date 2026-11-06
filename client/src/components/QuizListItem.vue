<template>
  <v-card
    class="pa-3 bg-surface-variant d-flex align-center justify-space-between"
    elevation="0"
    rounded="md"
  >
    <div class="d-flex align-center ga-3 overflow-hidden flex-grow-1">
      <v-avatar size="32" color="primary" variant="tonal">
        <v-icon icon="mdi-help-circle-outline" size="small" />
      </v-avatar>
      <div class="d-flex flex-column overflow-hidden">
        <span class="text-body-2 font-weight-bold text-high-emphasis text-truncate">
          {{ assignment.title }}
        </span>
        <span class="text-caption text-medium-emphasis text-truncate">
          <template v-if="showModule">{{ assignment.moduleTitle }} • </template>
          {{ windowText }} • pass at {{ assignment.passPercent }}%
        </span>
      </div>
    </div>

    <div class="d-flex align-center ga-1 ml-2">
      <v-chip v-if="isTeacher && !assignment.published" size="small" variant="tonal">Draft</v-chip>

      <template v-if="isTeacher">
        <v-btn
          v-if="assignment.published"
          color="primary"
          icon="mdi-account-check-outline"
          size="small"
          title="Results"
          variant="text"
          @click="emit('results')"
        />
        <v-btn
          color="primary"
          icon="mdi-pencil-outline"
          size="small"
          title="Edit quiz"
          variant="text"
          @click="emit('edit')"
        />
        <v-btn
          color="error"
          icon="mdi-delete-outline"
          size="small"
          title="Delete quiz"
          variant="text"
          @click="emit('delete')"
        />
      </template>

      <template v-else>
        <span
          v-if="assignment.myResult"
          class="text-body-2 font-weight-bold text-high-emphasis mr-2"
        >
          {{ assignment.myResult.passed ? '✅' : '❌' }}
          {{ Math.round(assignment.myResult.percent) }}%
        </span>
        <v-btn
          :disabled="!canStart"
          color="primary"
          size="small"
          variant="tonal"
          @click="emit('take')"
        >
          {{ startLabel }}
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
  (e: 'results'): void
  (e: 'delete'): void
  (e: 'take'): void
}>()

function formatDay(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, { day: 'numeric', month: 'short' })
}

const windowText = computed(() => {
  const { visibleFrom, visibleUntil } = props.assignment
  if (visibleFrom && visibleUntil) return `${formatDay(visibleFrom)} – ${formatDay(visibleUntil)}`
  if (visibleFrom) return `from ${formatDay(visibleFrom)}`
  if (visibleUntil) return `until ${formatDay(visibleUntil)}`
  return 'always open'
})

const isOpen = computed(() => {
  const now = Date.now()
  const { visibleFrom, visibleUntil } = props.assignment
  if (visibleFrom && now < new Date(visibleFrom).getTime()) return false
  if (visibleUntil && now > new Date(visibleUntil).getTime()) return false
  return true
})

const attemptsLeft = computed(() => props.assignment.attemptsUsed < props.assignment.maxAttempts)

const canStart = computed(() => isOpen.value && attemptsLeft.value)

const startLabel = computed(() => {
  if (!isOpen.value) return 'Closed'
  if (!attemptsLeft.value) return 'No attempts left'
  return props.assignment.attemptsUsed > 0 ? 'Retake' : 'Start'
})
</script>
