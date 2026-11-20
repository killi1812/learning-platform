<template>
  <div>
    <div class="d-flex align-baseline ga-2 mb-2">
      <span class="text-h4 font-weight-bold text-high-emphasis">{{ total }}</span>
      <span class="text-body-2 text-medium-emphasis">waiting for grading</span>
    </div>
    <v-list class="pa-0 bg-transparent" density="compact">
      <v-list-item
        v-for="item in visibleItems"
        :key="item.submissionUuid"
        class="px-0"
        to="/grades"
      >
        <v-list-item-title class="font-weight-medium">{{ item.assignmentTitle }}</v-list-item-title>
        <v-list-item-subtitle>
          {{ item.courseName }} ·
          {{ item.isGroup ? item.groupName : (item.userName ?? item.userEmail) }}
        </v-list-item-subtitle>
        <template #append>
          <v-chip v-if="item.isLate" color="warning" size="small" variant="tonal">Late</v-chip>
        </template>
      </v-list-item>
    </v-list>
  </div>
</template>

<script setup lang="ts">
import type { PendingSubmissionDto } from '@/types/grade'
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{ items: readonly PendingSubmissionDto[]; limit?: number }>(),
  { limit: 5 },
)

const total = computed(() => props.items.length)
const visibleItems = computed(() => props.items.slice(0, props.limit))
</script>
