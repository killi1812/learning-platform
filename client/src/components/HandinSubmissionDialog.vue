<template>
  <v-dialog
    :model-value="modelValue"
    max-width="640"
    persistent
    @update:model-value="emit('update:modelValue', $event)"
  >
    <v-card class="bg-surface pa-2" rounded="lg">
      <v-card-title
        class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis d-flex align-center justify-space-between"
      >
        <span>Submit: {{ assignment?.title }}</span>
        <v-btn icon="mdi-close" variant="text" size="small" @click="close" />
      </v-card-title>

      <v-card-subtitle class="px-4 text-medium-emphasis">
        {{ deadlineInfo }}
      </v-card-subtitle>

      <v-card-text class="px-4 pt-3">
        <!-- Group Assignment Status Banner -->
        <template v-if="assignment?.allowsGroupSubmission">
          <div
            v-if="loadingGroups"
            class="py-2 d-flex align-center ga-2 text-caption text-medium-emphasis mb-3"
          >
            <v-progress-circular color="primary" indeterminate size="16" />
            <span>Checking group membership...</span>
          </div>

          <v-alert
            v-else-if="userGroup"
            type="info"
            variant="tonal"
            density="comfortable"
            class="mb-4"
            icon="mdi-account-group"
          >
            <div>
              Submitting on behalf of <strong>{{ userGroup.name }}</strong>
            </div>
            <div v-if="groupMemberSummary" class="text-caption mt-1">
              Group members: {{ groupMemberSummary }}
            </div>
          </v-alert>

          <v-alert
            v-else
            type="warning"
            variant="tonal"
            density="comfortable"
            class="mb-4"
            icon="mdi-alert-circle-outline"
          >
            <div class="font-weight-medium">Group Required</div>
            <div class="text-caption mt-1">
              This is a group assignment. You must join a study group in this course before you can
              submit.
            </div>
            <div class="mt-2">
              <v-btn
                size="small"
                color="warning"
                variant="elevated"
                :to="`/courses/${courseUuid}?tab=groups`"
                @click="close"
              >
                Go to Study Groups
              </v-btn>
            </div>
          </v-alert>
        </template>

        <!-- Mode Tabs -->
        <v-tabs v-model="submitMode" class="mb-4" color="primary" density="compact">
          <v-tab value="file">
            <v-icon start icon="mdi-file-upload-outline" />
            File Upload
          </v-tab>
          <v-tab value="url">
            <v-icon start icon="mdi-link-variant" />
            Repository / URL
          </v-tab>
        </v-tabs>

        <!-- FILE SUBMISSION -->
        <div v-if="submitMode === 'file'">
          <v-file-input
            v-model="selectedFile"
            accept=".pdf,.docx,.pptx,.txt,.md,.png,.jpg,.jpeg,.gif,.webp,.json,.zip,.tar.gz,.tgz"
            class="mb-2"
            clearable
            :disabled="submitting"
            label="Select deliverable file"
            placeholder="Choose document, code archive, or image (up to 200 MB)"
            prepend-icon="mdi-paperclip"
            show-size
            variant="outlined"
          />
          <div class="text-caption text-medium-emphasis mb-3">
            Supported formats: PDF, Word, PowerPoint, Text, Markdown, Images, JSON, ZIP, Tarball.
            Max 200 MB.
          </div>

          <!-- Upload Progress Bar -->
          <div v-if="submitting" class="mb-4">
            <div class="d-flex justify-space-between text-caption font-weight-bold mb-1">
              <span>{{ uploadStatusText }}</span>
              <span>{{ uploadProgress }}%</span>
            </div>
            <v-progress-linear
              :model-value="uploadProgress"
              color="primary"
              height="8"
              rounded
              striped
            />
            <div v-if="uploadBytesText" class="text-caption text-medium-emphasis mt-1 text-right">
              {{ uploadBytesText }}
            </div>
          </div>

          <v-btn
            block
            color="primary"
            :disabled="!selectedFile || (assignment?.allowsGroupSubmission && !userGroup)"
            :loading="submitting"
            size="large"
            variant="elevated"
            @click="handleSubmitFile"
          >
            Upload & Submit File
          </v-btn>
        </div>

        <!-- URL SUBMISSION -->
        <div v-else-if="submitMode === 'url'">
          <v-text-field
            v-model="submitUrlInput"
            class="mb-3"
            :disabled="submitting"
            label="Repository or Document URL"
            placeholder="https://git.app.uib.no/username/project.git"
            type="url"
            variant="outlined"
          />

          <v-btn
            block
            color="primary"
            :disabled="!submitUrlInput.trim() || (assignment?.allowsGroupSubmission && !userGroup)"
            :loading="submitting"
            size="large"
            variant="elevated"
            @click="handleSubmitUrl"
          >
            Submit Link
          </v-btn>
        </div>

        <!-- SUBMISSION HISTORY -->
        <v-divider class="my-5" />
        <h3
          class="text-subtitle-1 font-weight-bold text-high-emphasis mb-2 d-flex align-center justify-space-between"
        >
          <span>Submission History</span>
          <v-btn
            density="compact"
            icon="mdi-refresh"
            size="small"
            variant="text"
            title="Refresh history"
            @click="loadHistory"
          />
        </h3>

        <div v-if="loadingHistory" class="py-4 text-center">
          <v-progress-circular color="primary" indeterminate size="24" />
        </div>
        <div
          v-else-if="history.length === 0"
          class="text-caption text-medium-emphasis font-italic py-2"
        >
          No previous submissions recorded for this assignment.
        </div>
        <v-list v-else class="pa-0 bg-transparent" density="compact">
          <v-list-item
            v-for="sub in history"
            :key="sub.uuid"
            class="pa-2 mb-2 bg-surface-variant rounded-md"
          >
            <template #prepend>
              <v-chip size="x-small" color="primary" class="font-weight-bold mr-2">
                #{{ sub.attempt }}
              </v-chip>
            </template>

            <v-list-item-title class="text-body-2 font-weight-bold">
              <template v-if="sub.payloadType === 'FILE'">
                {{ sub.fileName || 'Uploaded File' }}
                <span
                  v-if="sub.fileSize"
                  class="text-caption font-weight-regular text-medium-emphasis"
                >
                  ({{ formatBytes(sub.fileSize) }})
                </span>
              </template>
              <template v-else>
                <a
                  :href="sub.payloadUrl"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-primary text-decoration-none"
                >
                  {{ sub.payloadUrl }}
                </a>
              </template>
            </v-list-item-title>

            <v-list-item-subtitle class="text-caption text-medium-emphasis">
              <div>
                Submitted {{ formatDateTime(sub.submittedAt) }}
                <v-chip
                  class="ml-1"
                  :color="sub.isLate ? 'warning' : 'success'"
                  size="x-small"
                  variant="tonal"
                >
                  {{ sub.isLate ? 'Late' : 'On Time' }}
                </v-chip>
              </div>
              <div v-if="sub.isGroup" class="d-flex align-center ga-1 mt-1">
                <v-chip size="x-small" color="info" variant="tonal">
                  <v-icon start icon="mdi-account-group" size="x-small" />
                  {{ sub.groupName || 'Group' }}
                </v-chip>
                <span v-if="sub.userEmail" class="text-caption text-medium-emphasis">
                  submitted by {{ sub.userEmail }}
                </span>
              </div>
              <!-- Evaluated Grade & Feedback -->
              <div v-if="sub.grade" class="mt-2 pa-2 rounded bg-surface border">
                <div class="d-flex align-center justify-space-between mb-1">
                  <span class="text-caption font-weight-bold text-high-emphasis">
                    Grade & Feedback
                  </span>
                  <v-chip size="x-small" color="primary" variant="flat">
                    {{ sub.grade.score }} / {{ assignment?.maxPoints || 100 }} pts
                  </v-chip>
                </div>
                <MarkdownView
                  v-if="sub.grade.feedback"
                  :source="sub.grade.feedback"
                  class="text-caption mt-1"
                />
                <div v-else class="text-caption text-medium-emphasis font-italic">
                  No written feedback provided.
                </div>
              </div>
            </v-list-item-subtitle>

            <template #append>
              <v-btn
                v-if="sub.payloadType === 'FILE'"
                icon="mdi-download"
                size="small"
                variant="text"
                color="primary"
                title="Download submitted file"
                @click="handleDownload(sub)"
              />
            </template>
          </v-list-item>
        </v-list>
      </v-card-text>

      <v-card-actions class="pa-4 bg-surface-variant">
        <v-spacer />
        <v-btn color="medium-emphasis" variant="text" :disabled="submitting" @click="close">
          Close
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useToast } from '@/composables/useToast'
import { useAuth } from '@/composables/useAuth'
import { getApiErrorMessage } from '@/api/client'
import { listCourseGroups } from '@/api/course'
import {
  submitFileTwoPhase,
  submitUrl,
  listMySubmissions,
  downloadSubmissionFile,
} from '@/api/submission'
import MarkdownView from '@/components/MarkdownView.vue'
import type { Assignment } from '@/types/quiz'
import type { Submission } from '@/types/submission'
import type { Group } from '@/types/course'

const props = defineProps<{
  modelValue: boolean
  courseUuid: string
  assignment?: Assignment
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void
  (e: 'submitted'): void
}>()

const toast = useToast()
const { currentUser } = useAuth()

const submitMode = ref<'file' | 'url'>('file')
const selectedFile = ref<File | File[] | undefined>(undefined)
const submitUrlInput = ref('')
const submitting = ref(false)
const uploadProgress = ref(0)
const uploadStatusText = ref('')
const uploadBytesText = ref('')

const history = ref<Submission[]>([])
const loadingHistory = ref(false)
const courseGroups = ref<Group[]>([])
const loadingGroups = ref(false)

const userGroup = computed<Group | null>(() => {
  if (!currentUser.value || !courseGroups.value.length) return null
  const uid = currentUser.value.userUuid
  const email = currentUser.value.email?.toLowerCase()
  return (
    courseGroups.value.find(g =>
      g.members?.some(m => (uid && m.uuid === uid) || (email && m.email?.toLowerCase() === email)),
    ) || null
  )
})

const groupMemberSummary = computed(() => {
  if (!userGroup.value || !userGroup.value.members?.length) return ''
  return userGroup.value.members.map(m => m.username || m.email).join(', ')
})

const deadlineInfo = computed(() => {
  if (!props.assignment?.deadline) return 'No deadline set for this assignment.'
  const d = new Date(props.assignment.deadline).toLocaleString()
  const late = props.assignment.acceptsLateSubmissions
    ? ' (Late submissions accepted)'
    : ' (Late submissions not accepted)'
  return `Due: ${d}${late}`
})

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString()
}

function formatBytes(bytes?: number): string {
  if (!bytes) return ''
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function close() {
  if (submitting.value) return
  emit('update:modelValue', false)
}

async function loadGroups() {
  if (!props.courseUuid || !props.assignment?.allowsGroupSubmission) {
    courseGroups.value = []
    return
  }
  loadingGroups.value = true
  try {
    courseGroups.value = await listCourseGroups(props.courseUuid)
  } catch {
    courseGroups.value = []
  } finally {
    loadingGroups.value = false
  }
}

async function loadHistory() {
  if (!props.courseUuid || !props.assignment?.uuid) return
  loadingHistory.value = true
  try {
    history.value = await listMySubmissions(props.courseUuid, props.assignment.uuid)
  } catch {
    // If not found or initial, keep empty
    history.value = []
  } finally {
    loadingHistory.value = false
  }
}

watch(
  () => props.modelValue,
  open => {
    if (open) {
      selectedFile.value = undefined
      submitUrlInput.value = ''
      uploadProgress.value = 0
      uploadStatusText.value = ''
      uploadBytesText.value = ''
      loadHistory()
      loadGroups()
    }
  },
)

async function handleSubmitFile() {
  if (!props.assignment?.uuid || !props.courseUuid) return
  if (props.assignment?.allowsGroupSubmission && !userGroup.value) {
    toast.error('You must join a study group in this course before submitting.')
    return
  }

  const rawFile = Array.isArray(selectedFile.value) ? selectedFile.value[0] : selectedFile.value
  if (!rawFile) {
    toast.error('Please select a file to submit.')
    return
  }

  if (rawFile.size > 200 * 1024 * 1024) {
    toast.error('File size exceeds 200 MB limit.')
    return
  }

  submitting.value = true
  uploadProgress.value = 0
  uploadStatusText.value = 'Reserving submission timestamp...'
  uploadBytesText.value = ''

  try {
    await submitFileTwoPhase(
      props.courseUuid,
      props.assignment.uuid,
      rawFile,
      (pct, loaded, total) => {
        uploadProgress.value = pct
        uploadStatusText.value = `Streaming deliverable... ${pct}%`
        uploadBytesText.value = `${formatBytes(loaded)} / ${formatBytes(total)}`
      },
    )
    toast.success('Assignment submitted successfully! Processing in background.')
    selectedFile.value = undefined
    await loadHistory()
    emit('submitted')
  } catch (err: unknown) {
    toast.error(getApiErrorMessage(err, 'Failed to submit file.'))
  } finally {
    submitting.value = false
  }
}

async function handleSubmitUrl() {
  if (!props.assignment?.uuid || !props.courseUuid) return
  if (props.assignment?.allowsGroupSubmission && !userGroup.value) {
    toast.error('You must join a study group in this course before submitting.')
    return
  }

  const trimmed = submitUrlInput.value.trim()
  if (!trimmed) {
    toast.error('Please enter a valid submission URL.')
    return
  }

  submitting.value = true
  try {
    await submitUrl(props.courseUuid, props.assignment.uuid, trimmed)
    toast.success('Assignment URL submitted successfully!')
    submitUrlInput.value = ''
    await loadHistory()
    emit('submitted')
  } catch (err: unknown) {
    toast.error(getApiErrorMessage(err, 'Failed to submit URL.'))
  } finally {
    submitting.value = false
  }
}

async function handleDownload(sub: Submission) {
  if (!props.courseUuid || !props.assignment?.uuid) return
  try {
    await downloadSubmissionFile(props.courseUuid, props.assignment.uuid, sub.uuid, sub.fileName)
  } catch (err: unknown) {
    toast.error(getApiErrorMessage(err, 'Failed to download file.'))
  }
}
</script>
