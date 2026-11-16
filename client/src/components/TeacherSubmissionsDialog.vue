<template>
  <v-dialog v-model="open" max-width="950" persistent>
    <v-card class="bg-surface pa-2" rounded="lg">
      <v-card-title
        class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis d-flex align-center justify-space-between"
      >
        <span>Student Submissions: {{ assignment?.title }}</span>
        <v-btn
          aria-label="Close submissions dialog"
          icon="mdi-close"
          size="small"
          variant="text"
          @click="close"
        />
      </v-card-title>

      <v-card-subtitle class="px-4 text-medium-emphasis d-flex align-center justify-space-between">
        <span>{{ submissions.length }} total submissions recorded</span>
        <div class="d-flex align-center ga-2">
          <v-btn
            v-if="submissions.length > 0"
            color="primary"
            density="compact"
            :loading="releasing"
            prepend-icon="mdi-bullhorn-outline"
            size="small"
            variant="tonal"
            @click="handleReleaseGrades"
          >
            Release All Grades
          </v-btn>
          <v-btn
            density="compact"
            prepend-icon="mdi-refresh"
            size="small"
            variant="text"
            @click="loadSubmissions"
          >
            Refresh
          </v-btn>
        </div>
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
        <v-card v-else border class="bg-surface" elevation="0" rounded="md">
          <v-table density="comfortable">
            <thead>
              <tr>
                <th class="text-left font-weight-bold">
                  {{ isGroupAssignment ? 'Group / Submitter' : 'Student Email' }}
                </th>
                <th class="text-center font-weight-bold">Attempt</th>
                <th class="text-left font-weight-bold">Submitted At</th>
                <th class="text-center font-weight-bold">Timing</th>
                <th class="text-center font-weight-bold">Grade</th>
                <th class="text-right font-weight-bold">Deliverable</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="sub in filteredSubmissions" :key="sub.uuid">
                <td class="font-weight-medium py-2">
                  <template v-if="sub.isGroup">
                    <div class="d-flex align-center ga-1">
                      <v-chip color="info" size="x-small" variant="tonal">
                        <v-icon icon="mdi-account-group" size="x-small" start />
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
                  <v-chip color="primary" size="x-small" variant="flat">
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
                <td class="text-center py-2">
                  <template v-if="sub.grade">
                    <div class="d-flex flex-column align-center ga-1">
                      <div class="d-flex align-center ga-1">
                        <v-chip color="primary" size="x-small" variant="flat">
                          {{ sub.grade.score }} / {{ assignment?.maxPoints || 100 }}
                        </v-chip>
                        <v-chip
                          :color="sub.grade.isVisibleToStudent ? 'success' : 'warning'"
                          size="x-small"
                          variant="tonal"
                        >
                          {{ sub.grade.isVisibleToStudent ? 'Released' : 'Hidden' }}
                        </v-chip>
                      </div>
                      <v-btn
                        color="primary"
                        density="compact"
                        prepend-icon="mdi-pencil"
                        size="x-small"
                        variant="text"
                        @click="openGradeDialog(sub)"
                      >
                        Edit Grade
                      </v-btn>
                    </div>
                  </template>
                  <template v-else>
                    <div class="d-flex flex-column align-center ga-1">
                      <v-chip color="default" size="x-small" variant="tonal"> Ungraded </v-chip>
                      <v-btn
                        color="primary"
                        density="compact"
                        prepend-icon="mdi-check-decagram"
                        size="x-small"
                        variant="tonal"
                        @click="openGradeDialog(sub)"
                      >
                        Grade
                      </v-btn>
                    </div>
                  </template>
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
                    v-else-if="isSafeHttpUrl(sub.payloadUrl)"
                    color="primary"
                    density="compact"
                    :href="sub.payloadUrl"
                    prepend-icon="mdi-open-in-new"
                    rel="noopener noreferrer"
                    size="small"
                    target="_blank"
                    variant="text"
                  >
                    Open Link
                  </v-btn>
                  <span v-else class="text-caption text-medium-emphasis">Invalid URL</span>
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

    <GradeDialog
      v-model="gradeDialogOpen"
      :assignment-title="assignment?.title"
      :attempt="selectedSubForGrading?.attempt"
      :content-uuid="assignment?.uuid || ''"
      :course-uuid="courseUuid"
      :group-name="selectedSubForGrading?.groupName"
      :initial-feedback="selectedSubForGrading?.grade?.feedback"
      :initial-score="selectedSubForGrading?.grade?.score"
      :initial-visible="selectedSubForGrading?.grade?.isVisibleToStudent ?? true"
      :max-points="assignment?.maxPoints || 100"
      :member-emails="selectedSubForGrading?.memberEmails"
      :student-email="selectedSubForGrading?.userEmail"
      :submission-uuid="selectedSubForGrading?.uuid || ''"
      @saved="onGradeSaved"
    />
  </v-dialog>
</template>

<script setup lang="ts">
import type { GradeDto } from '@/types/grade'
import type { Assignment } from '@/types/quiz'
import type { Submission } from '@/types/submission'
import { computed, onUnmounted, ref, watch } from 'vue'
import { getApiErrorMessage } from '@/api/client'
import { releaseAssignmentGrades } from '@/api/grade'
import { downloadSubmissionFile, listAssignmentSubmissions } from '@/api/submission'
import GradeDialog from '@/components/GradeDialog.vue'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@/composables/useToast'
import { formatDateTime } from '@/util/dateTime'
import { isSafeHttpUrl } from '@/util/url'

const open = defineModel<boolean>({ required: true })

const props = defineProps<{
  courseUuid: string
  assignment?: Assignment
}>()

const toast = useToast()
const { confirm } = useConfirm()

const submissions = ref<Submission[]>([])
const loading = ref(false)
const releasing = ref(false)
const searchQuery = ref('')

const gradeDialogOpen = ref(false)
const selectedSubForGrading = ref<Submission | undefined>(undefined)

let loadController: AbortController | undefined

const isGroupAssignment = computed(() => {
  return Boolean(props.assignment?.allowsGroupSubmission || submissions.value.some(s => s.isGroup))
})

function close(): void {
  loadController?.abort()
  loadController = undefined
  open.value = false
}

const filteredSubmissions = computed(() => {
  const trimmed = searchQuery.value.trim()
  if (!trimmed) return submissions.value
  const query = trimmed.toLowerCase()
  return submissions.value.filter(s => {
    if (s.userEmail?.toLowerCase().includes(query)) return true
    if (s.groupName?.toLowerCase().includes(query)) return true
    if (s.memberEmails?.some(m => m.toLowerCase().includes(query))) return true
    return false
  })
})

async function loadSubmissions(): Promise<void> {
  loadController?.abort()
  if (!props.courseUuid || !props.assignment?.uuid) {
    submissions.value = []
    return
  }
  const controller = new AbortController()
  loadController = controller
  loading.value = true
  try {
    const data = await listAssignmentSubmissions(
      props.courseUuid,
      props.assignment.uuid,
      controller.signal,
    )
    if (!controller.signal.aborted) {
      submissions.value = data
    }
  } catch (error: unknown) {
    if (!controller.signal.aborted) {
      toast.error(getApiErrorMessage(error, 'Failed to load student submissions.'))
    }
  } finally {
    if (loadController === controller) {
      loading.value = false
      loadController = undefined
    }
  }
}

function openGradeDialog(sub: Submission): void {
  selectedSubForGrading.value = sub
  gradeDialogOpen.value = true
}

function onGradeSaved(savedGrade: GradeDto): void {
  if (!selectedSubForGrading.value) return
  selectedSubForGrading.value.grade = {
    score: savedGrade.score,
    feedback: savedGrade.feedback,
    isVisibleToStudent: savedGrade.isVisibleToStudent,
    gradedAt: savedGrade.gradedAt,
    releasedAt: savedGrade.releasedAt,
  }
}

async function handleReleaseGrades(): Promise<void> {
  if (!props.courseUuid || !props.assignment?.uuid) return
  const confirmed = await confirm({
    title: 'Release Grades',
    message:
      'Are you sure you want to release all hidden grades for this assignment? Students will receive notifications and will be able to see their score and feedback.',
    confirmText: 'Release Grades',
    confirmColor: 'primary',
  })
  if (!confirmed) return

  releasing.value = true
  try {
    const res = await releaseAssignmentGrades(props.courseUuid, props.assignment.uuid)
    toast.success(`${res.released} grade(s) released successfully!`)
    await loadSubmissions()
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to release assignment grades.'))
  } finally {
    releasing.value = false
  }
}

watch([open, () => props.courseUuid, () => props.assignment?.uuid], ([isOpen]) => {
  if (isOpen) {
    searchQuery.value = ''
    submissions.value = []
    selectedSubForGrading.value = undefined
    void loadSubmissions()
  } else {
    loadController?.abort()
    loadController = undefined
  }
})

onUnmounted(() => {
  loadController?.abort()
})

async function handleDownload(sub: Submission): Promise<void> {
  if (!props.courseUuid || !props.assignment?.uuid) return
  try {
    await downloadSubmissionFile(props.courseUuid, props.assignment.uuid, sub.uuid, sub.fileName)
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to download file.'))
  }
}
</script>
