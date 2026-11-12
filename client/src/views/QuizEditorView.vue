<template>
  <v-container class="py-6" max-width="900">
    <div class="d-flex align-center justify-space-between mb-4">
      <v-btn color="medium-emphasis" prepend-icon="mdi-arrow-left" variant="text" @click="goBack">
        Back to Course
      </v-btn>
      <div v-if="quiz" class="d-flex align-center ga-2">
        <v-chip :color="quiz.published ? 'secondary' : undefined" size="small" variant="tonal">
          {{ quiz.published ? 'Published' : 'Draft' }}
        </v-chip>
        <v-btn
          v-if="quiz.published"
          prepend-icon="mdi-account-check-outline"
          variant="text"
          @click="openResults"
        >
          Results
        </v-btn>
        <v-btn :loading="saving" variant="tonal" color="primary" @click="save(false)">Save</v-btn>
        <v-btn
          v-if="!quiz.published"
          :loading="saving"
          color="primary"
          variant="elevated"
          @click="save(true)"
        >
          Publish
        </v-btn>
      </div>
    </div>

    <div v-if="loading" class="d-flex justify-center py-12">
      <v-progress-circular color="primary" indeterminate size="64" />
    </div>
    <v-alert v-else-if="error" color="error" icon="mdi-alert-circle-outline" variant="tonal">
      {{ error }}
    </v-alert>

    <div v-else-if="quiz">
      <v-alert
        v-if="quiz.published"
        class="mb-4"
        color="warning"
        icon="mdi-alert-outline"
        variant="tonal"
      >
        This quiz is published. Saving any change deletes every student attempt, and students must
        take the quiz again.
      </v-alert>

      <!-- Settings -->
      <v-card class="pa-5 mb-6 bg-surface" elevation="1" rounded="lg">
        <h2 class="text-h6 font-weight-bold mb-4">Quiz settings</h2>
        <v-text-field v-model="form.title" label="Title" variant="outlined" />
        <div class="d-flex flex-wrap ga-4">
          <v-text-field
            v-model="form.visibleFrom"
            class="flex-grow-1"
            label="Available from"
            type="datetime-local"
            variant="outlined"
          />
          <v-text-field
            v-model="form.visibleUntil"
            class="flex-grow-1"
            label="Available until"
            type="datetime-local"
            variant="outlined"
          />
        </div>
        <div class="d-flex flex-wrap ga-4">
          <v-text-field
            v-model.number="form.passPercent"
            class="flex-grow-1"
            label="Minimum percent to pass"
            max="100"
            min="0"
            suffix="%"
            type="number"
            variant="outlined"
          />
          <v-text-field
            v-model.number="form.maxAttempts"
            class="flex-grow-1"
            label="Attempts per student"
            min="1"
            type="number"
            variant="outlined"
          />
        </div>
      </v-card>

      <!-- Questions -->
      <div class="d-flex align-center justify-space-between mb-3">
        <h2 class="text-h6 font-weight-bold">Questions ({{ questions.length }})</h2>
        <v-menu>
          <template #activator="{ props: menuProps }">
            <v-btn
              v-bind="menuProps"
              color="primary"
              prepend-icon="mdi-plus"
              append-icon="mdi-chevron-down"
              variant="tonal"
            >
              Add question
            </v-btn>
          </template>
          <v-list density="compact" elevation="2" rounded="md">
            <v-list-item
              prepend-icon="mdi-checkbox-multiple-marked-outline"
              title="Multiple choice"
              @click="addQuestion('MULTIPLE_CHOICE')"
            />
            <v-list-item
              prepend-icon="mdi-toggle-switch-outline"
              title="True / false"
              @click="addQuestion('TRUE_FALSE')"
            />
          </v-list>
        </v-menu>
      </div>

      <v-card
        v-if="questions.length === 0"
        class="pa-6 text-center text-medium-emphasis bg-surface"
        elevation="0"
        rounded="lg"
      >
        No questions yet. Add the first one.
      </v-card>

      <v-card
        v-for="(q, index) in questions"
        :key="q.key"
        class="pa-4 mb-4 bg-surface"
        elevation="1"
        rounded="lg"
      >
        <div class="d-flex align-center justify-space-between mb-2">
          <div class="d-flex align-center ga-2">
            <span class="text-subtitle-1 font-weight-bold">Question {{ index + 1 }}</span>
            <v-chip size="x-small" variant="tonal">
              {{ q.type === 'TRUE_FALSE' ? 'True / false' : 'Multiple choice' }}
            </v-chip>
          </div>
          <div class="d-flex ga-1">
            <v-btn
              :disabled="index === 0"
              density="comfortable"
              icon="mdi-arrow-up"
              title="Move up"
              variant="text"
              @click="moveQuestion(index, -1)"
            />
            <v-btn
              :disabled="index === questions.length - 1"
              density="comfortable"
              icon="mdi-arrow-down"
              title="Move down"
              variant="text"
              @click="moveQuestion(index, 1)"
            />
            <v-btn
              color="error"
              density="comfortable"
              icon="mdi-delete-outline"
              title="Delete question"
              variant="text"
              @click="removeQuestion(index)"
            />
          </div>
        </div>

        <v-textarea
          v-model="q.prompt"
          auto-grow
          label="Question text"
          rows="2"
          variant="outlined"
        />

        <!-- True / false -->
        <v-radio-group
          v-if="q.type === 'TRUE_FALSE'"
          :model-value="q.correct[0]"
          hide-details
          inline
          @update:model-value="
            (value: number | null) => (q.correct = value === null ? [] : [value])
          "
        >
          <span class="text-caption text-medium-emphasis mr-4 align-self-center">
            Correct answer:
          </span>
          <v-radio label="True" :value="0" />
          <v-radio label="False" :value="1" />
        </v-radio-group>

        <!-- Multiple choice -->
        <div v-else>
          <div class="text-caption text-medium-emphasis mb-2">
            Tick every option that is correct. A student scores one point for each option marked the
            same as your key.
          </div>
          <div v-for="(_, optIndex) in q.options" :key="optIndex" class="d-flex align-center ga-2">
            <v-checkbox-btn
              :model-value="q.correct.includes(optIndex)"
              @update:model-value="toggleCorrect(q, optIndex)"
            />
            <v-text-field
              v-model="q.options[optIndex]"
              density="compact"
              hide-details
              :label="`Option ${optIndex + 1}`"
              variant="outlined"
              class="mb-2"
            />
            <v-btn
              :disabled="q.options.length <= 2"
              color="error"
              density="comfortable"
              icon="mdi-close"
              title="Remove option"
              variant="text"
              @click="removeOption(q, optIndex)"
            />
          </div>
          <v-btn
            :disabled="q.options.length >= 8"
            prepend-icon="mdi-plus"
            size="small"
            variant="text"
            @click="q.options.push('')"
          >
            Add option
          </v-btn>
        </div>
      </v-card>
    </div>
  </v-container>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { getQuizForEdit, publishQuiz, replaceQuestions, updateQuiz } from '@/api/quiz'
import { useToast } from '@/composables/useToast'
import { useConfirm } from '@/composables/useConfirm'
import type { EditQuiz, QuestionType } from '@/types/quiz'

interface DraftQuestion {
  key: number
  type: QuestionType
  prompt: string
  options: string[]
  correct: number[]
}

const props = defineProps<{
  courseUuid: string
  moduleUuid: string
  quizUuid: string
}>()

const router = useRouter()
const toast = useToast()
const { confirm } = useConfirm()

const loading = ref(true)
const saving = ref(false)
const error = ref('')
const quiz = ref<EditQuiz | undefined>(undefined)
const questions = ref<DraftQuestion[]>([])
let nextKey = 0

const form = reactive({
  title: '',
  visibleFrom: '',
  visibleUntil: '',
  passPercent: 50,
  maxAttempts: 1,
})

function toLocalInput(iso?: string): string {
  if (!iso) return ''
  const date = new Date(iso)
  const offsetMs = date.getTimezoneOffset() * 60_000
  return new Date(date.getTime() - offsetMs).toISOString().slice(0, 16)
}

function fromLocalInput(value: string): string | undefined {
  return value ? new Date(value).toISOString() : undefined
}

function applyQuiz(loaded: EditQuiz) {
  quiz.value = loaded
  form.title = loaded.title
  form.visibleFrom = toLocalInput(loaded.visibleFrom)
  form.visibleUntil = toLocalInput(loaded.visibleUntil)
  form.passPercent = loaded.passPercent
  form.maxAttempts = loaded.maxAttempts
  questions.value = loaded.questions.map(q => ({
    key: nextKey++,
    type: q.type,
    prompt: q.prompt,
    options: [...q.options],
    correct: [...q.correct],
  }))
}

function newQuestion(type: QuestionType): DraftQuestion {
  return {
    key: nextKey++,
    type,
    prompt: '',
    options: type === 'TRUE_FALSE' ? ['True', 'False'] : ['', ''],
    correct: type === 'TRUE_FALSE' ? [0] : [],
  }
}

function addQuestion(type: QuestionType) {
  questions.value.push(newQuestion(type))
}

function removeQuestion(index: number) {
  questions.value.splice(index, 1)
}

function moveQuestion(index: number, delta: number) {
  const target = index + delta
  if (target < 0 || target >= questions.value.length) return
  const [moved] = questions.value.splice(index, 1)
  questions.value.splice(target, 0, moved!)
}

function toggleCorrect(q: DraftQuestion, optIndex: number) {
  q.correct = q.correct.includes(optIndex)
    ? q.correct.filter(i => i !== optIndex)
    : [...q.correct, optIndex].sort((a, b) => a - b)
}

function removeOption(q: DraftQuestion, optIndex: number) {
  q.options.splice(optIndex, 1)
  q.correct = q.correct.filter(i => i !== optIndex).map(i => (i > optIndex ? i - 1 : i))
}

/** Returns a message for the first problem found, or undefined when the quiz is valid. */
function validate(): string | undefined {
  if (form.title.trim().length < 2) return 'Title must be at least 2 characters.'
  if (form.passPercent < 0 || form.passPercent > 100) return 'Pass percent must be 0 to 100.'
  if (!Number.isInteger(form.maxAttempts) || form.maxAttempts < 1)
    return 'Attempts must be 1 or more.'
  if (form.visibleFrom && form.visibleUntil && form.visibleUntil < form.visibleFrom) {
    return 'The end date is before the start date.'
  }
  for (const [i, q] of questions.value.entries()) {
    const label = `Question ${i + 1}`
    if (!q.prompt.trim()) return `${label} has no text.`
    if (q.type === 'MULTIPLE_CHOICE') {
      if (q.options.some(o => !o.trim())) return `${label} has an empty option.`
      if (q.correct.length === 0) return `${label} needs at least one correct option.`
    }
    if (q.type === 'TRUE_FALSE' && q.correct.length !== 1) return `${label} needs a correct answer.`
  }
  return undefined
}

async function save(publish: boolean) {
  const problem = validate()
  if (problem) {
    toast.error(problem)
    return
  }
  if (publish && questions.value.length === 0) {
    toast.error('Add at least one question before publishing.')
    return
  }
  if (
    quiz.value?.published &&
    !(await confirm({
      title: 'Reset Attempts',
      message: 'Saving resets all student attempts. Continue?',
      confirmText: 'Continue',
      confirmColor: 'warning',
    }))
  )
    return

  saving.value = true
  try {
    await updateQuiz(props.courseUuid, props.quizUuid, {
      title: form.title.trim(),
      visibleFrom: fromLocalInput(form.visibleFrom),
      visibleUntil: fromLocalInput(form.visibleUntil),
      passPercent: form.passPercent,
      maxAttempts: form.maxAttempts,
    })
    let saved = await replaceQuestions(
      props.courseUuid,
      props.quizUuid,
      questions.value.map(q => ({
        type: q.type,
        prompt: q.prompt.trim(),
        options: q.type === 'TRUE_FALSE' ? [] : q.options.map(o => o.trim()),
        correct: q.correct,
      })),
    )
    if (publish) saved = await publishQuiz(props.courseUuid, props.quizUuid)
    applyQuiz(saved)
    toast.success(publish ? 'Quiz published.' : 'Quiz saved.')
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to save quiz.')
  } finally {
    saving.value = false
  }
}

function goBack() {
  router.push(`/courses/${props.courseUuid}`)
}

function openResults() {
  router.push(
    `/courses/${props.courseUuid}/modules/${props.moduleUuid}/quizzes/${props.quizUuid}/results`,
  )
}

onMounted(async () => {
  try {
    applyQuiz(await getQuizForEdit(props.courseUuid, props.quizUuid))
  } catch (err: unknown) {
    error.value = err instanceof Error ? err.message : 'Failed to load quiz.'
  } finally {
    loading.value = false
  }
})
</script>
