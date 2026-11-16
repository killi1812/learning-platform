<template>
  <v-dialog v-model="open" max-width="640" persistent>
    <v-card class="bg-surface pa-2" rounded="lg">
      <v-card-title
        class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis d-flex align-center justify-space-between"
      >
        <span>Submit: {{ assignment?.title }}</span>
        <v-btn
          aria-label="Close submission dialog"
          :disabled="submitting"
          icon="mdi-close"
          size="small"
          variant="text"
          @click="close"
        />
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
            class="mb-4"
            density="comfortable"
            icon="mdi-account-group"
            type="info"
            variant="tonal"
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
            class="mb-4"
            density="comfortable"
            icon="mdi-alert-circle-outline"
            type="warning"
            variant="tonal"
          >
            <div class="font-weight-medium">Group Required</div>
            <div class="text-caption mt-1">
              This is a group assignment. You must join a study group in this course before you can
              submit.
            </div>
            <div class="mt-2">
              <v-btn
                color="warning"
                size="small"
                :to="`/courses/${courseUuid}?tab=groups`"
                variant="elevated"
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
            <v-icon icon="mdi-file-upload-outline" start />
            File Upload
          </v-tab>
          <v-tab value="url">
            <v-icon icon="mdi-link-variant" start />
            Repository / URL
          </v-tab>
        </v-tabs>

        <v-tabs-window v-model="submitMode">
          <!-- FILE SUBMISSION -->
          <v-tabs-window-item value="file">
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
                color="primary"
                height="8"
                :model-value="uploadProgress"
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
              :disabled="!hasSelectedFile || groupBlocked"
              :loading="submitting"
              size="large"
              variant="elevated"
              @click="handleSubmitFile"
            >
              Upload & Submit File
            </v-btn>
          </v-tabs-window-item>

          <!-- URL SUBMISSION -->
          <v-tabs-window-item value="url">
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
              :disabled="!submitUrlInput.trim() || groupBlocked"
              :loading="submitting"
              size="large"
              variant="elevated"
              @click="handleSubmitUrl"
            >
              Submit Link
            </v-btn>
          </v-tabs-window-item>
        </v-tabs-window>

        <!-- SUBMISSION HISTORY -->
        <v-divider class="my-5" />
        <h3
          class="text-subtitle-1 font-weight-bold text-high-emphasis mb-2 d-flex align-center justify-space-between"
        >
          <span>Submission History</span>
          <v-btn
            aria-label="Refresh submission history"
            density="compact"
            icon="mdi-refresh"
            size="small"
            title="Refresh history"
            variant="text"
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
              <v-chip class="font-weight-bold mr-2" color="primary" size="x-small">
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
              <template v-else-if="isSafeHttpUrl(sub.payloadUrl)">
                <a
                  class="text-primary text-decoration-none"
                  :href="sub.payloadUrl"
                  rel="noopener noreferrer"
                  target="_blank"
                >
                  {{ sub.payloadUrl }}
                </a>
              </template>
              <template v-else>
                <span class="text-medium-emphasis">{{ sub.payloadUrl || 'Invalid URL' }}</span>
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
                <v-chip color="info" size="x-small" variant="tonal">
                  <v-icon icon="mdi-account-group" size="x-small" start />
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
                  <v-chip color="primary" size="x-small" variant="flat">
                    {{ sub.grade.score }} / {{ assignment?.maxPoints || 100 }} pts
                  </v-chip>
                </div>
                <MarkdownView
                  v-if="sub.grade.feedback"
                  class="text-caption mt-1"
                  :source="sub.grade.feedback"
                />
                <div v-else class="text-caption text-medium-emphasis font-italic">
                  No written feedback provided.
                </div>
              </div>
            </v-list-item-subtitle>

            <template #append>
              <v-btn
                v-if="sub.payloadType === 'FILE'"
                aria-label="Download submitted file"
                color="primary"
                icon="mdi-download"
                size="small"
                title="Download submitted file"
                variant="text"
                @click="handleDownload(sub)"
              />
            </template>
          </v-list-item>
        </v-list>
      </v-card-text>

      <v-card-actions class="pa-4 bg-surface-variant">
        <v-spacer />
        <v-btn color="medium-emphasis" :disabled="submitting" variant="text" @click="close">
          Close
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import type { Group } from '@/types/course'
import type { Assignment } from '@/types/quiz'
import type { Submission } from '@/types/submission'
import { isAxiosError } from 'axios'
import { computed, onUnmounted, ref, watch } from 'vue'
import { getApiErrorMessage } from '@/api/client'
import { listCourseGroups } from '@/api/course'
import {
  downloadSubmissionFile,
  listMySubmissions,
  submitFileTwoPhase,
  submitUrl,
} from '@/api/submission'
import MarkdownView from '@/components/MarkdownView.vue'
import { useAuth } from '@/composables/useAuth'
import { useToast } from '@/composables/useToast'
import { formatDateTime } from '@/util/dateTime'
import { formatBytes } from '@/util/file'
import { findUserGroup } from '@/util/group'
import { isSafeHttpUrl } from '@/util/url'

const open = defineModel<boolean>({ required: true })

const props = defineProps<{
  courseUuid: string
  assignment?: Assignment
}>()

const emit = defineEmits<{
  submitted: []
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

let historyController: AbortController | undefined
let groupsController: AbortController | undefined
let uploadController: AbortController | undefined

const resolvedSelectedFile = computed(() =>
  Array.isArray(selectedFile.value) ? selectedFile.value[0] : selectedFile.value,
)
const hasSelectedFile = computed(() => Boolean(resolvedSelectedFile.value))

const userGroup = computed(() => findUserGroup(courseGroups.value, currentUser.value))
const groupBlocked = computed(() =>
  Boolean(props.assignment?.allowsGroupSubmission && !userGroup.value),
)

const groupMemberSummary = computed(() => {
  if (!userGroup.value?.members?.length) return ''
  return userGroup.value.members.map(m => m.username || m.email).join(', ')
})

const deadlineInfo = computed(() => {
  if (!props.assignment?.deadline) return 'No deadline set for this assignment.'
  const formatted = formatDateTime(props.assignment.deadline) || props.assignment.deadline
  const late = props.assignment.acceptsLateSubmissions
    ? ' (Late submissions accepted)'
    : ' (Late submissions not accepted)'
  return `Due: ${formatted}${late}`
})

function abortInFlight(): void {
  historyController?.abort()
  historyController = undefined
  groupsController?.abort()
  groupsController = undefined
  uploadController?.abort()
  uploadController = undefined
}

function close(): void {
  if (submitting.value) return
  abortInFlight()
  open.value = false
}

async function loadGroups(): Promise<void> {
  groupsController?.abort()
  if (!props.courseUuid || !props.assignment?.allowsGroupSubmission) {
    courseGroups.value = []
    return
  }
  const controller = new AbortController()
  groupsController = controller
  loadingGroups.value = true
  try {
    const groups = await listCourseGroups(props.courseUuid)
    if (!controller.signal.aborted) {
      courseGroups.value = groups
    }
  } catch (error: unknown) {
    if (!controller.signal.aborted) {
      courseGroups.value = []
      toast.error(getApiErrorMessage(error, 'Failed to check group membership.'))
    }
  } finally {
    if (groupsController === controller) {
      loadingGroups.value = false
      groupsController = undefined
    }
  }
}

async function loadHistory(): Promise<void> {
  historyController?.abort()
  if (!props.courseUuid || !props.assignment?.uuid) {
    history.value = []
    return
  }
  const controller = new AbortController()
  historyController = controller
  loadingHistory.value = true
  try {
    const items = await listMySubmissions(
      props.courseUuid,
      props.assignment.uuid,
      controller.signal,
    )
    if (!controller.signal.aborted) {
      history.value = items
    }
  } catch (error: unknown) {
    if (controller.signal.aborted) return
    history.value = []
    toast.error(getApiErrorMessage(error, 'Failed to load submission history.'))
  } finally {
    if (historyController === controller) {
      loadingHistory.value = false
      historyController = undefined
    }
  }
}

watch([open, () => props.courseUuid, () => props.assignment?.uuid], ([isOpen]) => {
  if (isOpen) {
    selectedFile.value = undefined
    submitUrlInput.value = ''
    uploadProgress.value = 0
    uploadStatusText.value = ''
    uploadBytesText.value = ''
    history.value = []
    courseGroups.value = []
    void loadHistory()
    void loadGroups()
  } else {
    abortInFlight()
  }
})

onUnmounted(() => {
  abortInFlight()
})

async function handleSubmitFile(): Promise<void> {
  if (!props.assignment?.uuid || !props.courseUuid) return
  if (groupBlocked.value) {
    toast.error('You must join a study group in this course before submitting.')
    return
  }

  const rawFile = resolvedSelectedFile.value
  if (!rawFile) {
    toast.error('Please select a file to submit.')
    return
  }

  if (rawFile.size > 200 * 1024 * 1024) {
    toast.error('File size exceeds 200 MB limit.')
    return
  }

  uploadController?.abort()
  const controller = new AbortController()
  uploadController = controller

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
      controller.signal,
    )
    toast.success('Assignment submitted successfully! Processing in background.')
    selectedFile.value = undefined
    await loadHistory()
    emit('submitted')
  } catch (error: unknown) {
    const canceled =
      controller.signal.aborted || (isAxiosError(error) && error.code === 'ERR_CANCELED')
    if (!canceled) {
      toast.error(getApiErrorMessage(error, 'Failed to submit file.'))
    }
  } finally {
    if (uploadController === controller) {
      uploadController = undefined
    }
    submitting.value = false
  }
}

async function handleSubmitUrl(): Promise<void> {
  if (!props.assignment?.uuid || !props.courseUuid) return
  if (groupBlocked.value) {
    toast.error('You must join a study group in this course before submitting.')
    return
  }

  const trimmed = submitUrlInput.value.trim()
  if (!isSafeHttpUrl(trimmed)) {
    toast.error('Please enter a valid http:// or https:// URL.')
    return
  }

  submitting.value = true
  try {
    await submitUrl(props.courseUuid, props.assignment.uuid, trimmed)
    toast.success('Assignment URL submitted successfully!')
    submitUrlInput.value = ''
    await loadHistory()
    emit('submitted')
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to submit URL.'))
  } finally {
    submitting.value = false
  }
}

async function handleDownload(sub: Submission): Promise<void> {
  if (!props.courseUuid || !props.assignment?.uuid) return
  try {
    await downloadSubmissionFile(props.courseUuid, props.assignment.uuid, sub.uuid, sub.fileName)
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to download file.'))
  }
}
</script>
