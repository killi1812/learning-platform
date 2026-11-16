<template>
  <v-dialog v-model="open" max-width="650" persistent>
    <v-card class="bg-surface pa-2" rounded="lg">
      <v-card-title
        class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis d-flex align-center justify-space-between"
      >
        <span>Grade Submission</span>
        <v-btn
          aria-label="Close grade dialog"
          :disabled="saving"
          icon="mdi-close"
          size="small"
          variant="text"
          @click="close"
        />
      </v-card-title>

      <v-card-subtitle class="px-4 text-medium-emphasis">
        <div>{{ assignmentTitle || 'Assignment' }} · Attempt #{{ attempt || 1 }}</div>
        <div v-if="groupName" class="d-flex flex-wrap align-center ga-1 mt-1">
          <v-chip color="info" size="x-small" variant="tonal">
            <v-icon icon="mdi-account-group" size="x-small" start />
            {{ groupName }}
          </v-chip>
          <span v-if="studentEmail" class="text-caption">Submitted by {{ studentEmail }}</span>
          <span v-if="memberEmails?.length" class="text-caption">
            · Members: {{ memberEmails.join(', ') }}
          </span>
        </div>
        <div v-else-if="studentEmail" class="text-caption">Student: {{ studentEmail }}</div>
      </v-card-subtitle>

      <v-form ref="formRef" @submit.prevent="handleSave">
        <v-card-text class="px-4 pt-4">
          <div class="d-flex align-center ga-3 mb-4">
            <v-text-field
              class="grade-dialog__score-field"
              density="comfortable"
              label="Score"
              :max="effectiveMaxPoints"
              min="0"
              :model-value="score"
              :rules="[scoreRule]"
              step="0.5"
              type="number"
              variant="outlined"
              @update:model-value="onScoreInput"
            />
            <div class="text-h6 text-medium-emphasis">/ {{ effectiveMaxPoints }} pts</div>
            <v-chip
              v-if="percent !== undefined"
              class="ml-2"
              :color="percent >= 50 ? 'success' : 'error'"
              size="small"
              variant="tonal"
            >
              {{ percent.toFixed(1) }}%
            </v-chip>
          </div>

          <div class="mb-2">
            <div class="d-flex align-center justify-space-between mb-1">
              <span class="text-subtitle-2 font-weight-bold text-high-emphasis">
                Evaluation Feedback
              </span>
              <v-tabs v-model="feedbackTab" color="primary" density="compact">
                <v-tab value="write">
                  <v-icon icon="mdi-pencil-outline" size="small" start />
                  Write
                </v-tab>
                <v-tab value="preview">
                  <v-icon icon="mdi-eye-outline" size="small" start />
                  Preview
                </v-tab>
              </v-tabs>
            </div>

            <v-tabs-window v-model="feedbackTab">
              <v-tabs-window-item value="write">
                <v-textarea
                  v-model="feedback"
                  auto-grow
                  counter="10000"
                  label="Feedback (Markdown supported)"
                  :maxlength="10000"
                  placeholder="Provide constructive feedback, notes on edge cases, or rubric breakdown..."
                  rows="5"
                  variant="outlined"
                />
              </v-tabs-window-item>

              <v-tabs-window-item value="preview">
                <div class="pa-3 rounded border bg-surface-variant mb-4">
                  <MarkdownView
                    empty-text="Nothing to preview yet. Switch to Write tab to add feedback."
                    :source="feedback"
                  />
                </div>
              </v-tabs-window-item>
            </v-tabs-window>
          </div>

          <v-switch
            v-model="isVisibleToStudent"
            color="primary"
            :label="
              groupName
                ? 'Make grade & feedback visible to group members immediately'
                : 'Make grade & feedback visible to student immediately'
            "
            :messages="
              groupName
                ? 'If disabled, group members cannot see score until grades are released in bulk.'
                : 'If disabled, student cannot see score until grades are released in bulk.'
            "
          />
        </v-card-text>

        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="medium-emphasis" :disabled="saving" variant="text" @click="close">
            Cancel
          </v-btn>
          <v-btn
            color="primary"
            :disabled="!isValid"
            :loading="saving"
            type="submit"
            variant="elevated"
          >
            Save Grade
          </v-btn>
        </v-card-actions>
      </v-form>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import type { GradeDto } from '@/types/grade'
import type { VForm } from 'vuetify/components'
import { computed, ref, watch } from 'vue'
import { getApiErrorMessage } from '@/api/client'
import { upsertGrade } from '@/api/grade'
import MarkdownView from '@/components/MarkdownView.vue'
import { useToast } from '@/composables/useToast'
import { DEFAULT_HANDIN_MAX_POINTS } from '@/util/grading'

const open = defineModel<boolean>({ required: true })

const props = withDefaults(
  defineProps<{
    courseUuid: string
    contentUuid: string
    submissionUuid: string
    assignmentTitle?: string
    studentEmail?: string
    groupName?: string
    memberEmails?: readonly string[]
    attempt?: number
    maxPoints?: number
    initialScore?: number
    initialFeedback?: string
    initialVisible?: boolean
  }>(),
  {
    assignmentTitle: '',
    studentEmail: '',
    groupName: '',
    memberEmails: undefined,
    attempt: 1,
    maxPoints: DEFAULT_HANDIN_MAX_POINTS,
    initialScore: undefined,
    initialFeedback: '',
    initialVisible: true,
  },
)

const emit = defineEmits<{
  saved: [grade: GradeDto]
}>()

const toast = useToast()

const formRef = ref<InstanceType<typeof VForm> | undefined>(undefined)
const score = ref<number | undefined>(undefined)
const feedback = ref('')
const isVisibleToStudent = ref(true)
const feedbackTab = ref<'write' | 'preview'>('write')
const saving = ref(false)

const effectiveMaxPoints = computed(() =>
  props.maxPoints && props.maxPoints > 0 ? props.maxPoints : DEFAULT_HANDIN_MAX_POINTS,
)

const percent = computed(() => {
  if (typeof score.value !== 'number' || Number.isNaN(score.value)) {
    return undefined
  }
  return (score.value / effectiveMaxPoints.value) * 100
})

function onScoreInput(raw: string): void {
  const trimmed = raw.trim()
  if (trimmed === '') {
    score.value = undefined
    return
  }
  const parsed = Number(trimmed)
  score.value = Number.isNaN(parsed) ? undefined : parsed
}

function scoreRule(val: unknown): string | boolean {
  if (typeof val !== 'number' && typeof val !== 'string') {
    return 'Score is required'
  }
  if (typeof val === 'string' && val.trim() === '') {
    return 'Score is required'
  }
  const n = Number(val)
  if (Number.isNaN(n)) return 'Must be a valid number'
  if (n < 0) return 'Score cannot be negative'
  if (n > effectiveMaxPoints.value) return `Score cannot exceed ${effectiveMaxPoints.value}`
  return true
}

const isValid = computed(() => {
  return (
    typeof score.value === 'number' &&
    Number.isFinite(score.value) &&
    score.value >= 0 &&
    score.value <= effectiveMaxPoints.value
  )
})

function close(): void {
  if (saving.value) return
  open.value = false
}

watch(
  open,
  isOpen => {
    if (isOpen) {
      feedbackTab.value = 'write'
      score.value = props.initialScore
      feedback.value = props.initialFeedback || ''
      isVisibleToStudent.value = props.initialVisible ?? true
      formRef.value?.resetValidation()
    }
  },
  { immediate: true },
)

async function handleSave(): Promise<void> {
  if (formRef.value) {
    const result = await formRef.value.validate()
    if (!result.valid) return
  }
  if (!isValid.value || score.value === undefined) return
  if (!props.courseUuid || !props.contentUuid || !props.submissionUuid) return

  saving.value = true
  try {
    const updated = await upsertGrade(props.courseUuid, props.contentUuid, props.submissionUuid, {
      score: score.value,
      feedback: feedback.value.trim(),
      isVisibleToStudent: isVisibleToStudent.value,
    })
    toast.success('Grade recorded successfully!')
    emit('saved', updated)
    open.value = false
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to save grade.'))
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.grade-dialog__score-field {
  max-width: 180px;
}
</style>
