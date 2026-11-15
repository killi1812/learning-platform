<template>
  <div>
    <div class="d-flex align-center justify-space-between mb-4">
      <h2 class="text-h5 font-weight-bold text-high-emphasis">Course Participants</h2>
      <v-btn
        color="primary"
        prepend-icon="mdi-account-plus"
        variant="elevated"
        @click="emit('add-participant')"
      >
        Add Participant
      </v-btn>
    </div>

    <v-card class="bg-surface" elevation="1" rounded="lg">
      <v-table>
        <thead>
          <tr>
            <th class="text-left font-weight-bold">User Email</th>
            <th class="text-left font-weight-bold">Course Role</th>
            <th class="text-left font-weight-bold">Enrolled At</th>
            <th class="text-right font-weight-bold">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="e in enrollments" :key="e.userEmail">
            <td class="font-weight-medium">{{ e.userEmail }}</td>
            <td>
              <v-chip
                :color="e.role === 'TEACHER' ? 'primary' : 'secondary'"
                size="small"
                variant="flat"
              >
                {{ e.role }}
              </v-chip>
            </td>
            <td class="text-medium-emphasis">
              {{ formatShortDate(e.enrolledAt) }}
            </td>
            <td class="text-right">
              <v-btn
                v-if="e.role === 'STUDENT'"
                aria-label="Remove participant"
                color="error"
                :disabled="!e.userUuid"
                icon="mdi-delete-outline"
                :loading="deletingUserUuid === e.userUuid"
                size="small"
                title="Remove participant"
                variant="text"
                @click="emit('remove-participant', e)"
              />
            </td>
          </tr>
        </tbody>
      </v-table>
    </v-card>
  </div>
</template>

<script setup lang="ts">
import type { Enrollment } from '@/types/course'
import { formatShortDate } from '@/util/dateTime'

interface Props {
  enrollments: readonly Enrollment[]
  deletingUserUuid?: string
}

defineProps<Props>()

const emit = defineEmits<{
  'add-participant': []
  'remove-participant': [enrollment: Enrollment]
}>()
</script>
