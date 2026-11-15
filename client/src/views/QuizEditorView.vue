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
        <v-btn color="primary" :loading="saving" variant="tonal" @click="save(false)">Save</v-btn>
        <v-btn
          v-if="!quiz.published"
          color="primary"
          :loading="saving"
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
              append-icon="mdi-chevron-down"
              color="primary"
              prepend-icon="mdi-plus"
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
              aria-label="Move question up"
              density="comfortable"
              :disabled="index === 0"
              icon="mdi-arrow-up"
              title="Move up"
              variant="text"
              @click="moveQuestion(index, -1)"
            />
            <v-btn
              aria-label="Move question down"
              density="comfortable"
              :disabled="index === questions.length - 1"
              icon="mdi-arrow-down"
              title="Move down"
              variant="text"
              @click="moveQuestion(index, 1)"
            />
            <v-btn
              aria-label="Delete question"
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
          hide-details
          inline
          :model-value="q.correct[0]"
          @update:model-value="
            (value: number | null) =>
              (q.correct = value === null || value === undefined ? [] : [value])
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
              class="mb-2"
              density="compact"
              hide-details
              :label="`Option ${optIndex + 1}`"
              variant="outlined"
            />
            <v-btn
              aria-label="Remove option"
              color="error"
              density="comfortable"
              :disabled="q.options.length <= 2"
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
import type { EditQuiz, QuestionType } from '@/types/quiz'
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { onBeforeRouteLeave, useRouter } from 'vue-router'
import { getApiErrorMessage } from '@/api/client'
import { getQuizForEdit, publishQuiz, replaceQuestions, updateQuiz } from '@/api/quiz'
import { useConfirm } from '@/composables/useConfirm'
import { useToast } from '@/composables/useToast'
import { fromLocalDateTimeInput, toLocalDateTimeInput } from '@/util/dateTime'

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
const savedSnapshot = ref('')
let nextKey = 0
let loadCounter = 0

const form = reactive({
  title: '',
  visibleFrom: '',
  visibleUntil: '',
  passPercent: 50,
  maxAttempts: 1,
})

function serializeDraft(): string {
  return JSON.stringify({
    form: { ...form },
    questions: questions.value.map(q => ({
      type: q.type,
      prompt: q.prompt,
      options: q.options,
      correct: q.correct,
    })),
  })
}

const isDirty = computed(() => Boolean(quiz.value) && serializeDraft() !== savedSnapshot.value)

function applyQuiz(loaded: EditQuiz): void {
  quiz.value = loaded
  form.title = loaded.title
  form.visibleFrom = toLocalDateTimeInput(loaded.visibleFrom)
  form.visibleUntil = toLocalDateTimeInput(loaded.visibleUntil)
  form.passPercent = loaded.passPercent
  form.maxAttempts = loaded.maxAttempts
  questions.value = loaded.questions.map(q => ({
    key: nextKey++,
    type: q.type,
    prompt: q.prompt,
    options: [...q.options],
    correct: [...q.correct],
  }))
  savedSnapshot.value = serializeDraft()
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

function addQuestion(type: QuestionType): void {
  questions.value.push(newQuestion(type))
}

function removeQuestion(index: number): void {
  questions.value.splice(index, 1)
}

function moveQuestion(index: number, delta: number): void {
  const target = index + delta
  if (target < 0 || target >= questions.value.length) {
    return
  }
  const [moved] = questions.value.splice(index, 1)
  if (moved) {
    questions.value.splice(target, 0, moved)
  }
}

function toggleCorrect(q: DraftQuestion, optIndex: number): void {
  if (q.correct.includes(optIndex)) {
    q.correct = q.correct.filter(i => i !== optIndex)
    return
  }
  const next = [...q.correct, optIndex]
  next.sort((a, b) => a - b)
  q.correct = next
}

function removeOption(q: DraftQuestion, optIndex: number): void {
  q.options.splice(optIndex, 1)
  q.correct = q.correct.filter(i => i !== optIndex).map(i => (i > optIndex ? i - 1 : i))
}

function validateQuestion(q: DraftQuestion, index: number): string | undefined {
  const label = `Question ${index + 1}`
  if (!q.prompt.trim()) {
    return `${label} has no text.`
  }
  if (q.type === 'MULTIPLE_CHOICE') {
    if (q.options.some(o => !o.trim())) {
      return `${label} has an empty option.`
    }
    if (q.correct.length === 0) {
      return `${label} needs at least one correct option.`
    }
  }
  if (q.type === 'TRUE_FALSE' && q.correct.length !== 1) {
    return `${label} needs a correct answer.`
  }
  return undefined
}

/** Returns a message for the first problem found, or undefined when the quiz is valid. */
function validate(): string | undefined {
  if (form.title.trim().length < 2) {
    return 'Title must be at least 2 characters.'
  }
  if (form.passPercent < 0 || form.passPercent > 100) {
    return 'Pass percent must be 0 to 100.'
  }
  if (!Number.isInteger(form.maxAttempts) || form.maxAttempts < 1) {
    return 'Attempts must be 1 or more.'
  }
  if (form.visibleFrom && form.visibleUntil && form.visibleUntil < form.visibleFrom) {
    return 'The end date is before the start date.'
  }
  return questions.value
    .map((q, index) => validateQuestion(q, index))
    .find((problem): problem is string => problem !== undefined)
}

async function save(publish: boolean): Promise<void> {
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
  ) {
    return
  }

  saving.value = true
  try {
    await updateQuiz(props.courseUuid, props.quizUuid, {
      title: form.title.trim(),
      visibleFrom: fromLocalDateTimeInput(form.visibleFrom),
      visibleUntil: fromLocalDateTimeInput(form.visibleUntil),
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
    if (publish) {
      saved = await publishQuiz(props.courseUuid, props.quizUuid)
    }
    applyQuiz(saved)
    toast.success(publish ? 'Quiz published.' : 'Quiz saved.')
  } catch (error_: unknown) {
    toast.error(getApiErrorMessage(error_, 'Failed to save quiz.'))
  } finally {
    saving.value = false
  }
}

function goBack(): void {
  void router.push(`/courses/${props.courseUuid}`)
}

function openResults(): void {
  void router.push(
    `/courses/${props.courseUuid}/modules/${props.moduleUuid}/quizzes/${props.quizUuid}/results`,
  )
}

async function loadQuiz(): Promise<void> {
  const requestId = ++loadCounter
  loading.value = true
  error.value = ''
  try {
    const loaded = await getQuizForEdit(props.courseUuid, props.quizUuid)
    if (requestId === loadCounter) {
      applyQuiz(loaded)
    }
  } catch (error_: unknown) {
    if (requestId === loadCounter) {
      error.value = getApiErrorMessage(error_, 'Failed to load quiz.')
    }
  } finally {
    if (requestId === loadCounter) {
      loading.value = false
    }
  }
}

onBeforeRouteLeave(async () => {
  if (!isDirty.value) {
    return true
  }
  return await confirm({
    title: 'Unsaved Changes',
    message: 'You have unsaved changes to this quiz. Leave without saving?',
    confirmText: 'Leave',
    confirmColor: 'warning',
  })
})

onMounted(() => {
  void loadQuiz()
})

watch(
  () => [props.courseUuid, props.moduleUuid, props.quizUuid],
  () => {
    void loadQuiz()
  },
)
</script>
