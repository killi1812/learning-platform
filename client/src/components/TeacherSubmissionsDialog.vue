<template>
  <v-dialog
    :model-value="modelValue"
    max-width="850"
    persistent
    @update:model-value="emit('update:modelValue', $event)"
  >
    <v-card class="bg-surface pa-2" rounded="lg">
      <v-card-title
        class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis d-flex align-center justify-space-between"
      >
        <span>Student Submissions: {{ assignment?.title }}</span>
        <v-btn icon="mdi-close" variant="text" size="small" @click="close" />
      </v-card-title>

      <v-card-subtitle class="px-4 text-medium-emphasis d-flex align-center justify-space-between">
        <span>{{ submissions.length }} total submissions recorded</span>
        <v-btn
          density="compact"
          prepend-icon="mdi-refresh"
          variant="text"
          size="small"
          @click="loadSubmissions"
        >
          Refresh
        </v-btn>
      </v-card-subtitle>

      <v-card-text class="px-4 pt-3">
        <!-- Search filter -->
        <v-text-field
          v-model="searchQuery"
          class="mb-3"
          clearable
          density="compact"
          :label="
            isGroupAssignment
              ? 'Search by student, group name, or member email'
              : 'Search by student email'
          "
          prepend-inner-icon="mdi-magnify"
          variant="outlined"
        />

        <div v-if="loading" class="py-8 text-center">
          <v-progress-circular color="primary" indeterminate size="32" />
        </div>
        <div
          v-else-if="filteredSubmissions.length === 0"
          class="py-6 text-center text-medium-emphasis"
        >
          {{
            searchQuery
              ? 'No submissions match your search.'
              : 'No submissions received for this assignment yet.'
          }}
        </div>
        <v-card v-else class="bg-surface" elevation="0" rounded="md" border>
          <v-table density="comfortable">
            <thead>
              <tr>
                <th class="text-left font-weight-bold">
                  {{ isGroupAssignment ? 'Group / Submitter' : 'Student Email' }}
                </th>
                <th class="text-center font-weight-bold">Attempt</th>
                <th class="text-left font-weight-bold">Submitted At</th>
                <th class="text-center font-weight-bold">Timing</th>
                <th class="text-right font-weight-bold">Deliverable</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="sub in filteredSubmissions" :key="sub.uuid">
                <td class="font-weight-medium py-2">
                  <template v-if="sub.isGroup">
                    <div class="d-flex align-center ga-1">
                      <v-chip size="x-small" color="info" variant="tonal">
                        <v-icon start icon="mdi-account-group" size="x-small" />
                        {{ sub.groupName || 'Group' }}
                      </v-chip>
                      <span class="text-caption text-medium-emphasis">
                        by {{ sub.userEmail }}
                      </span>
                    </div>
                    <div
                      v-if="sub.memberEmails?.length"
                      class="text-caption text-medium-emphasis mt-1"
                    >
                      Members: {{ sub.memberEmails.join(', ') }}
                    </div>
                  </template>
                  <template v-else>
                    {{ sub.userEmail }}
                  </template>
                </td>
                <td class="text-center">
                  <v-chip size="x-small" color="primary" variant="flat">
                    #{{ sub.attempt }}
                  </v-chip>
                </td>
                <td class="text-medium-emphasis">
                  {{ formatDateTime(sub.submittedAt) }}
                </td>
                <td class="text-center">
                  <v-chip
                    :color="sub.isLate ? 'warning' : 'success'"
                    size="x-small"
                    variant="tonal"
                  >
                    {{ sub.isLate ? 'Late' : 'On Time' }}
                  </v-chip>
                </td>
                <td class="text-right">
                  <v-btn
                    v-if="sub.payloadType === 'FILE'"
                    color="primary"
                    density="compact"
                    prepend-icon="mdi-download"
                    size="small"
                    variant="tonal"
                    @click="handleDownload(sub)"
                  >
                    {{ sub.fileName || 'Download' }}
                  </v-btn>
                  <v-btn
                    v-else
                    :href="sub.payloadUrl"
                    color="primary"
                    density="compact"
                    prepend-icon="mdi-open-in-new"
                    size="small"
                    target="_blank"
                    variant="text"
                  >
                    Open Link
                  </v-btn>
                </td>
              </tr>
            </tbody>
          </v-table>
        </v-card>
      </v-card-text>

      <v-card-actions class="pa-4 bg-surface-variant">
        <v-spacer />
        <v-btn color="medium-emphasis" variant="text" @click="close"> Close </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useToast } from '@/composables/useToast'
import { getApiErrorMessage } from '@/api/client'
import { listAssignmentSubmissions, downloadSubmissionFile } from '@/api/submission'
import type { Assignment } from '@/types/quiz'
import type { Submission } from '@/types/submission'

const props = defineProps<{
  modelValue: boolean
  courseUuid: string
  assignment?: Assignment
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void
}>()

const toast = useToast()
const submissions = ref<Submission[]>([])
const loading = ref(false)
const searchQuery = ref('')

const isGroupAssignment = computed(() => {
  return Boolean(props.assignment?.allowsGroupSubmission || submissions.value.some(s => s.isGroup))
})

function close() {
  emit('update:modelValue', false)
}

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString()
}

const filteredSubmissions = computed(() => {
  if (!searchQuery.value.trim()) return submissions.value
  const query = searchQuery.value.toLowerCase()
  return submissions.value.filter(s => {
    if (s.userEmail?.toLowerCase().includes(query)) return true
    if (s.groupName?.toLowerCase().includes(query)) return true
    if (s.memberEmails?.some(m => m.toLowerCase().includes(query))) return true
    return false
  })
})

async function loadSubmissions() {
  if (!props.courseUuid || !props.assignment?.uuid) return
  loading.value = true
  try {
    submissions.value = await listAssignmentSubmissions(props.courseUuid, props.assignment.uuid)
  } catch (err: unknown) {
    toast.error(getApiErrorMessage(err, 'Failed to load student submissions.'))
  } finally {
    loading.value = false
  }
}

watch(
  () => props.modelValue,
  open => {
    if (open) {
      searchQuery.value = ''
      loadSubmissions()
    }
  },
)

async function handleDownload(sub: Submission) {
  if (!props.courseUuid || !props.assignment?.uuid) return
  try {
    await downloadSubmissionFile(props.courseUuid, props.assignment.uuid, sub.uuid, sub.fileName)
  } catch (err: unknown) {
    toast.error(getApiErrorMessage(err, 'Failed to download file.'))
  }
}
</script>
