<template>
  <v-card
    class="pa-3 bg-surface-variant d-flex align-center justify-space-between"
    elevation="0"
    rounded="md"
  >
    <div class="d-flex align-center ga-3 overflow-hidden flex-grow-1">
      <v-avatar color="primary" size="32" variant="tonal">
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
          aria-label="Quiz results"
          color="primary"
          icon="mdi-account-check-outline"
          size="small"
          title="Results"
          variant="text"
          @click="emit('results')"
        />
        <v-btn
          aria-label="Edit quiz"
          color="primary"
          icon="mdi-pencil-outline"
          size="small"
          title="Edit quiz"
          variant="text"
          @click="emit('edit')"
        />
        <v-btn
          aria-label="Delete quiz"
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
          class="d-inline-flex align-center ga-1 text-body-2 font-weight-bold text-high-emphasis mr-2"
        >
          <v-icon
            :aria-label="assignment.myResult.passed ? 'Passed' : 'Not passed'"
            :color="assignment.myResult.passed ? 'success' : 'error'"
            :icon="assignment.myResult.passed ? 'mdi-check-circle' : 'mdi-close-circle'"
            size="small"
          />
          <span>{{ Math.round(assignment.myResult.percent) }}%</span>
        </span>
        <v-btn
          color="primary"
          :disabled="!canStart"
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
import type { Assignment } from '@/types/quiz'
import { computed } from 'vue'
import { formatMonthDay, parseValidDate } from '@/util/dateTime'

const props = defineProps<{
  assignment: Assignment
  isTeacher: boolean
  showModule?: boolean
}>()

const emit = defineEmits<{
  edit: []
  results: []
  delete: []
  take: []
}>()

const windowText = computed(() => {
  const { visibleFrom, visibleUntil } = props.assignment
  const from = formatMonthDay(visibleFrom)
  const until = formatMonthDay(visibleUntil)
  if (from && until) return `${from} – ${until}`
  if (from) return `from ${from}`
  if (until) return `until ${until}`
  return 'always open'
})

const isOpen = computed(() => {
  const now = Date.now()
  const from = parseValidDate(props.assignment.visibleFrom)
  const until = parseValidDate(props.assignment.visibleUntil)
  if (from && now < from.getTime()) return false
  if (until && now > until.getTime()) return false
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
