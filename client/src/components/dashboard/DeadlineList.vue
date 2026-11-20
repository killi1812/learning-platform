<template>
  <v-list class="pa-0 bg-transparent" density="compact">
    <v-list-item
      v-for="item in visibleItems"
      :key="item.uuid"
      class="px-0"
      :to="`/courses/${item.courseUuid}`"
    >
      <template #prepend>
        <v-icon
          class="mr-3 text-medium-emphasis"
          :icon="item.type === 'QUIZ' ? 'mdi-help-circle-outline' : 'mdi-file-upload-outline'"
        />
      </template>
      <v-list-item-title class="font-weight-medium">{{ item.title }}</v-list-item-title>
      <v-list-item-subtitle
        >{{ item.courseTitle }} · {{ formatDateTime(item.deadline) }}</v-list-item-subtitle
      >
      <template #append>
        <v-chip v-if="submittedUuids?.has(item.uuid)" color="success" size="small" variant="tonal">
          Submitted
        </v-chip>
        <v-chip
          v-else
          :color="URGENCY_COLOR[deadlineUrgency(item.deadline, now)]"
          size="small"
          variant="tonal"
        >
          {{ formatTimeLeft(item.deadline, now) }}
        </v-chip>
      </template>
    </v-list-item>
  </v-list>
</template>

<script setup lang="ts">
import type { DeadlineItem, DeadlineUrgency } from '@/types/dashboard'
import { computed } from 'vue'
import { deadlineUrgency, formatTimeLeft } from '@/util/dashboard'
import { formatDateTime } from '@/util/dateTime'

const URGENCY_COLOR: Record<DeadlineUrgency, string> = {
  overdue: 'error',
  soon: 'warning',
  later: 'info',
}

const props = withDefaults(
  defineProps<{
    items: readonly DeadlineItem[]
    now: Date
    limit?: number
    submittedUuids?: ReadonlySet<string>
  }>(),
  { limit: 6, submittedUuids: undefined },
)

const visibleItems = computed(() => props.items.slice(0, props.limit))
</script>
