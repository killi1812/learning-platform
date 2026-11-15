<template>
  <v-container class="py-6" max-width="800">
    <div class="d-flex align-center justify-space-between mb-4">
      <v-btn color="medium-emphasis" prepend-icon="mdi-arrow-left" variant="text" @click="goBack">
        Back to Course
      </v-btn>
      <span v-if="attempt && !result && questionCount > 0" class="text-body-2 text-medium-emphasis">
        {{ answeredCount }} of {{ questionCount }} answered
      </span>
    </div>

    <div v-if="loading" class="d-flex justify-center py-12">
      <v-progress-circular color="primary" indeterminate size="64" />
    </div>
    <v-alert v-else-if="error" color="error" icon="mdi-alert-circle-outline" variant="tonal">
      {{ error }}
    </v-alert>

    <!-- Result: percent and pass/fail only -->
    <v-card v-else-if="result" class="pa-8 text-center bg-surface" elevation="2" rounded="lg">
      <v-icon
        :color="result.passed ? 'success' : 'error'"
        :icon="result.passed ? 'mdi-check-circle' : 'mdi-close-circle'"
        size="64"
      />
      <h2 class="text-h4 font-weight-bold mt-3">{{ Math.round(result.percent) }}%</h2>
      <p class="text-body-1 text-medium-emphasis mt-2 mb-0">
        {{ result.passed ? 'Passed' : 'Not passed' }} · attempt {{ result.attemptsUsed }} of
        {{ result.maxAttempts }}
      </p>
      <p class="text-caption text-medium-emphasis mt-1">Your best attempt counts.</p>
      <v-btn class="mt-6" color="primary" variant="elevated" @click="goBack">Done</v-btn>
    </v-card>

    <!-- Empty quiz fallback -->
    <v-card
      v-else-if="attempt && questionCount === 0"
      class="pa-8 text-center bg-surface"
      elevation="2"
      rounded="lg"
    >
      <v-icon class="mb-3 text-medium-emphasis" icon="mdi-help-circle-outline" size="48" />
      <h2 class="text-h6 font-weight-bold">No questions available</h2>
      <p class="text-body-2 text-medium-emphasis mt-2 mb-6">
        This quiz does not contain any questions yet.
      </p>
      <v-btn color="primary" variant="elevated" @click="goBack">Back to Course</v-btn>
    </v-card>

    <!-- One question at a time -->
    <v-card v-else-if="attempt && current" class="pa-6 bg-surface" elevation="2" rounded="lg">
      <div class="text-caption text-medium-emphasis mb-1">
        Question {{ index + 1 }} of {{ questionCount }}
      </div>
      <h2 class="quiz-prompt text-h6 font-weight-bold mb-4">{{ current.prompt }}</h2>

      <v-radio-group
        v-if="current.type === 'TRUE_FALSE'"
        :model-value="selected[0]"
        @update:model-value="
          (value: number | null) =>
            void choose(value === null || value === undefined ? [] : [value])
        "
      >
        <v-radio label="True" :value="0" />
        <v-radio label="False" :value="1" />
      </v-radio-group>

      <div v-else>
        <v-checkbox
          v-for="(option, optIndex) in current.options"
          :key="optIndex"
          density="comfortable"
          hide-details
          :label="option"
          :model-value="selected.includes(optIndex)"
          @update:model-value="toggle(optIndex)"
        />
      </div>

      <div class="d-flex justify-space-between mt-6">
        <v-btn :disabled="index === 0" variant="text" @click="index--">Previous</v-btn>
        <v-btn v-if="index < questionCount - 1" color="primary" variant="tonal" @click="index++">
          Next
        </v-btn>
        <v-btn v-else color="primary" :loading="submitting" variant="elevated" @click="submit">
          Submit
        </v-btn>
      </div>
      <p v-if="index === questionCount - 1 && unansweredCount > 0" class="text-caption mt-2 mb-0">
        {{ unansweredCount }} unanswered
        {{ unansweredCount === 1 ? 'question counts' : 'questions count' }}
        as wrong.
      </p>
    </v-card>
  </v-container>
</template>

<script setup lang="ts">
import type { Attempt, QuizResult } from '@/types/quiz'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { getApiErrorMessage } from '@/api/client'
import { saveAnswer, startAttempt, submitAttempt } from '@/api/quiz'
import { useToast } from '@/composables/useToast'

const props = defineProps<{
  courseUuid: string
  moduleUuid: string
  quizUuid: string
}>()

const router = useRouter()
const toast = useToast()

const loading = ref(true)
const submitting = ref(false)
const error = ref('')
const attempt = ref<Attempt | undefined>(undefined)
const result = ref<QuizResult | undefined>(undefined)
const index = ref(0)

let isUnmounted = false
let loadCounter = 0
const inFlightSaves = new Map<string, Promise<void>>()
const latestSaveSeq = new Map<string, number>()

const questionCount = computed(() => attempt.value?.questions.length ?? 0)
const current = computed(() => attempt.value?.questions[index.value])
const selected = computed(() =>
  current.value ? (attempt.value?.answers[current.value.uuid] ?? []) : [],
)
const answeredCount = computed(() => Object.keys(attempt.value?.answers ?? {}).length)
const unansweredCount = computed(() => Math.max(0, questionCount.value - answeredCount.value))

/** Serializes saves per question so rapid checkbox toggles never race out of order. */
async function choose(values: number[]): Promise<void> {
  const question = current.value
  const activeAttempt = attempt.value
  if (!activeAttempt || !question) {
    return
  }

  const questionUuid = question.uuid
  const courseUuid = props.courseUuid
  const quizUuid = props.quizUuid
  const nextSeq = (latestSaveSeq.get(questionUuid) ?? 0) + 1
  latestSaveSeq.set(questionUuid, nextSeq)

  const previousValue = activeAttempt.answers[questionUuid]
  if (values.length > 0) {
    activeAttempt.answers[questionUuid] = values
  } else {
    delete activeAttempt.answers[questionUuid]
  }

  const priorSave = inFlightSaves.get(questionUuid) ?? Promise.resolve()
  const queuedSave = priorSave
    .catch(() => undefined)
    .then(async () => {
      await saveAnswer(courseUuid, quizUuid, questionUuid, values)
    })
    .catch((error_: unknown) => {
      if (
        !isUnmounted &&
        latestSaveSeq.get(questionUuid) === nextSeq &&
        attempt.value === activeAttempt
      ) {
        if (previousValue && previousValue.length > 0) {
          activeAttempt.answers[questionUuid] = previousValue
        } else {
          delete activeAttempt.answers[questionUuid]
        }
        toast.error(getApiErrorMessage(error_, 'Failed to save answer.'))
      }
      throw error_
    })

  inFlightSaves.set(questionUuid, queuedSave)
  try {
    await queuedSave
  } finally {
    if (inFlightSaves.get(questionUuid) === queuedSave) {
      inFlightSaves.delete(questionUuid)
    }
  }
}

function toggle(optIndex: number): void {
  if (selected.value.includes(optIndex)) {
    void choose(selected.value.filter(i => i !== optIndex))
    return
  }
  const next = [...selected.value, optIndex]
  next.sort((a, b) => a - b)
  void choose(next)
}

async function submit(): Promise<void> {
  submitting.value = true
  try {
    const pending = [...inFlightSaves.values()]
    if (pending.length > 0) {
      const settled = await Promise.allSettled(pending)
      if (settled.some(item => item.status === 'rejected')) {
        return
      }
    }
    result.value = await submitAttempt(props.courseUuid, props.quizUuid)
  } catch (error_: unknown) {
    toast.error(getApiErrorMessage(error_, 'Failed to submit quiz.'))
  } finally {
    submitting.value = false
  }
}

function goBack(): void {
  void router.push(`/courses/${props.courseUuid}`)
}

async function loadAttempt(): Promise<void> {
  const requestId = ++loadCounter
  loading.value = true
  error.value = ''
  attempt.value = undefined
  result.value = undefined
  index.value = 0
  inFlightSaves.clear()
  latestSaveSeq.clear()

  try {
    const started = await startAttempt(props.courseUuid, props.quizUuid)
    if (!isUnmounted && requestId === loadCounter) {
      attempt.value = started
    }
  } catch (error_: unknown) {
    if (!isUnmounted && requestId === loadCounter) {
      error.value = getApiErrorMessage(error_, 'Failed to start quiz.')
    }
  } finally {
    if (!isUnmounted && requestId === loadCounter) {
      loading.value = false
    }
  }
}

onMounted(() => {
  void loadAttempt()
})

watch(
  () => [props.courseUuid, props.quizUuid],
  () => {
    void loadAttempt()
  },
)

onBeforeUnmount(() => {
  isUnmounted = true
})
</script>

<style scoped>
.quiz-prompt {
  white-space: pre-wrap;
}
</style>
