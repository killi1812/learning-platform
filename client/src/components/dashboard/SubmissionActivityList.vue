<template>
  <div v-for="row in rows" :key="row.uuid" class="mb-3">
    <div class="d-flex justify-space-between text-body-2">
      <router-link class="text-high-emphasis font-weight-medium" :to="`/courses/${row.courseUuid}`">
        {{ row.title }}
      </router-link>
      <span class="text-medium-emphasis">
        {{ row.submitted }} / {{ row.enrolled }}
        <span v-if="row.late > 0">· {{ row.late }} late</span>
      </span>
    </div>
    <div class="text-caption text-medium-emphasis">
      {{ row.courseTitle }} · {{ formatDateTime(row.deadline) }}
    </div>
    <v-progress-linear
      class="mt-1"
      color="primary"
      height="8"
      :model-value="row.enrolled > 0 ? (row.submitted / row.enrolled) * 100 : 0"
      rounded
    />
  </div>
</template>

<script setup lang="ts">
import type { SubmissionActivityItem } from '@/types/dashboard'
import { formatDateTime } from '@/util/dateTime'

defineProps<{ rows: readonly SubmissionActivityItem[] }>()
</script>
