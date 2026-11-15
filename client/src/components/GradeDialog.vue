<template>
  <v-dialog
    :model-value="modelValue"
    max-width="650"
    persistent
    @update:model-value="emit('update:modelValue', $event)"
  >
    <v-card class="bg-surface pa-2" rounded="lg">
      <v-card-title
        class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis d-flex align-center justify-space-between"
      >
        <span>Grade Submission</span>
        <v-btn icon="mdi-close" variant="text" size="small" :disabled="saving" @click="close" />
      </v-card-title>

      <v-card-subtitle class="px-4 text-medium-emphasis">
        <div>{{ assignmentTitle || 'Assignment' }} · Attempt #{{ attempt || 1 }}</div>
        <div v-if="studentEmail" class="text-caption">Student: {{ studentEmail }}</div>
      </v-card-subtitle>

      <v-card-text class="px-4 pt-4">
        <!-- Score Input -->
        <div class="d-flex align-center ga-3 mb-4">
          <v-text-field
            v-model.number="score"
            label="Score"
            type="number"
            min="0"
            :max="effectiveMaxPoints"
            step="0.5"
            variant="outlined"
            density="comfortable"
            :rules="[scoreRule]"
            style="max-width: 180px"
          />
          <div class="text-h6 text-medium-emphasis">/ {{ effectiveMaxPoints }} pts</div>
          <v-chip
            v-if="percent !== null"
            :color="percent >= 50 ? 'success' : 'error'"
            size="small"
            variant="tonal"
            class="ml-2"
          >
            {{ percent.toFixed(1) }}%
          </v-chip>
        </div>

        <!-- Feedback Tabs (Write / Preview) -->
        <div class="mb-2">
          <div class="d-flex align-center justify-space-between mb-1">
            <span class="text-subtitle-2 font-weight-bold text-high-emphasis">
              Evaluation Feedback
            </span>
            <v-tabs v-model="feedbackTab" density="compact" color="primary">
              <v-tab value="write">
                <v-icon start icon="mdi-pencil-outline" size="small" />
                Write
              </v-tab>
              <v-tab value="preview">
                <v-icon start icon="mdi-eye-outline" size="small" />
                Preview
              </v-tab>
            </v-tabs>
          </div>

          <div v-show="feedbackTab === 'write'">
            <v-textarea
              v-model="feedback"
              label="Feedback (Markdown supported)"
              placeholder="Provide constructive feedback, notes on edge cases, or rubric breakdown..."
              rows="5"
              variant="outlined"
              counter="10000"
              :maxlength="10000"
              auto-grow
            />
          </div>

          <div
            v-show="feedbackTab === 'preview'"
            class="pa-3 rounded border bg-surface-variant mb-4"
          >
            <MarkdownView
              :source="feedback"
              empty-text="Nothing to preview yet. Switch to Write tab to add feedback."
            />
          </div>
        </div>

        <!-- Student Visibility Switch -->
        <v-switch
          v-model="isVisibleToStudent"
          color="primary"
          label="Make grade & feedback visible to student immediately"
          messages="If disabled, student cannot see score until grades are released in bulk."
        />
      </v-card-text>

      <v-card-actions class="pa-4 bg-surface-variant">
        <v-spacer />
        <v-btn color="medium-emphasis" variant="text" :disabled="saving" @click="close">
          Cancel
        </v-btn>
        <v-btn
          color="primary"
          variant="elevated"
          :loading="saving"
          :disabled="!isValid"
          @click="handleSave"
        >
          Save Grade
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useToast } from '@/composables/useToast'
import { getApiErrorMessage } from '@/api/client'
import { upsertGrade } from '@/api/grade'
import MarkdownView from '@/components/MarkdownView.vue'
import type { GradeDto } from '@/types/grade'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    courseUuid: string
    contentUuid: string
    submissionUuid: string
    assignmentTitle?: string
    studentEmail?: string
    attempt?: number
    maxPoints?: number
    initialScore?: number
    initialFeedback?: string
    initialVisible?: boolean
  }>(),
  {
    assignmentTitle: '',
    studentEmail: '',
    attempt: 1,
    maxPoints: 100,
    initialScore: undefined,
    initialFeedback: '',
    initialVisible: true,
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', val: boolean): void
  (e: 'saved', grade: GradeDto): void
}>()

const toast = useToast()

const score = ref<number | undefined>(undefined)
const feedback = ref('')
const isVisibleToStudent = ref(true)
const feedbackTab = ref<'write' | 'preview'>('write')
const saving = ref(false)

const effectiveMaxPoints = computed(() =>
  props.maxPoints && props.maxPoints > 0 ? props.maxPoints : 100,
)

const percent = computed(() => {
  if (score.value === undefined || score.value === null || isNaN(score.value)) {
    return null
  }
  return (score.value / effectiveMaxPoints.value) * 100
})

const scoreRule = (val: unknown) => {
  if (val === undefined || val === null || val === '') {
    return 'Score is required'
  }
  const n = Number(val)
  if (isNaN(n)) return 'Must be a valid number'
  if (n < 0) return 'Score cannot be negative'
  if (n > effectiveMaxPoints.value) return `Score cannot exceed ${effectiveMaxPoints.value}`
  return true
}

const isValid = computed(() => {
  if (score.value === undefined || score.value === null || isNaN(score.value)) {
    return false
  }
  return score.value >= 0 && score.value <= effectiveMaxPoints.value
})

function close() {
  if (saving.value) return
  emit('update:modelValue', false)
}

watch(
  () => props.modelValue,
  isOpen => {
    if (isOpen) {
      feedbackTab.value = 'write'
      score.value = props.initialScore !== undefined ? props.initialScore : undefined
      feedback.value = props.initialFeedback || ''
      isVisibleToStudent.value = props.initialVisible ?? true
    }
  },
  { immediate: true },
)

async function handleSave() {
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
    emit('update:modelValue', false)
  } catch (err: unknown) {
    toast.error(getApiErrorMessage(err, 'Failed to save grade.'))
  } finally {
    saving.value = false
  }
}
</script>
