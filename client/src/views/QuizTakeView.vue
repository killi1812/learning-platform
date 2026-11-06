<template>
  <v-container class="py-6" max-width="800">
    <div class="d-flex align-center justify-space-between mb-4">
      <v-btn color="medium-emphasis" prepend-icon="mdi-arrow-left" variant="text" @click="goBack">
        Back to Course
      </v-btn>
      <span v-if="attempt && !result" class="text-body-2 text-medium-emphasis">
        {{ answeredCount }} of {{ attempt.questions.length }} answered
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
      <div class="result-emoji">{{ result.passed ? '✅' : '❌' }}</div>
      <h2 class="text-h4 font-weight-bold mt-2">{{ Math.round(result.percent) }}%</h2>
      <p class="text-body-1 text-medium-emphasis mt-2 mb-0">
        {{ result.passed ? 'Passed' : 'Not passed' }} · attempt {{ result.attemptsUsed }} of
        {{ result.maxAttempts }}
      </p>
      <p class="text-caption text-medium-emphasis mt-1">Your best attempt counts.</p>
      <v-btn class="mt-6" color="primary" variant="elevated" @click="goBack">Done</v-btn>
    </v-card>

    <!-- One question at a time -->
    <v-card v-else-if="current" class="pa-6 bg-surface" elevation="2" rounded="lg">
      <div class="text-caption text-medium-emphasis mb-1">
        Question {{ index + 1 }} of {{ attempt!.questions.length }}
      </div>
      <h2 class="quiz-prompt text-h6 font-weight-bold mb-4">{{ current.prompt }}</h2>

      <v-radio-group
        v-if="current.type === 'TRUE_FALSE'"
        :model-value="selected[0]"
        @update:model-value="(value: number | null) => choose(value === null ? [] : [value])"
      >
        <v-radio label="True" :value="0" />
        <v-radio label="False" :value="1" />
      </v-radio-group>

      <div v-else>
        <v-checkbox
          v-for="(option, optIndex) in current.options"
          :key="optIndex"
          :label="option"
          :model-value="selected.includes(optIndex)"
          density="comfortable"
          hide-details
          @update:model-value="toggle(optIndex)"
        />
      </div>

      <div class="d-flex justify-space-between mt-6">
        <v-btn :disabled="index === 0" variant="text" @click="index--">Previous</v-btn>
        <v-btn
          v-if="index < attempt!.questions.length - 1"
          color="primary"
          variant="tonal"
          @click="index++"
        >
          Next
        </v-btn>
        <v-btn v-else :loading="submitting" color="primary" variant="elevated" @click="submit">
          Submit
        </v-btn>
      </div>
      <p
        v-if="index === attempt!.questions.length - 1 && unansweredCount > 0"
        class="text-caption mt-2 mb-0"
      >
        {{ unansweredCount }} unanswered
        {{ unansweredCount === 1 ? 'question counts' : 'questions count' }}
        as wrong.
      </p>
    </v-card>
  </v-container>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { saveAnswer, startAttempt, submitAttempt } from '@/api/quiz'
import { useToast } from '@/composables/useToast'
import type { Attempt, QuizResult } from '@/types/quiz'

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

const current = computed(() => attempt.value?.questions[index.value])
const selected = computed(() =>
  current.value ? (attempt.value?.answers[current.value.uuid] ?? []) : [],
)
const answeredCount = computed(() => Object.keys(attempt.value?.answers ?? {}).length)
const unansweredCount = computed(() =>
  attempt.value ? attempt.value.questions.length - answeredCount.value : 0,
)

/** Saves the choice on the server first, so a refresh keeps it. Local state follows on success. */
async function choose(values: number[]) {
  const question = current.value
  if (!attempt.value || !question) return
  try {
    await saveAnswer(props.courseUuid, props.quizUuid, question.uuid, values)
    attempt.value.answers[question.uuid] = values
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to save answer.')
  }
}

function toggle(optIndex: number) {
  const next = selected.value.includes(optIndex)
    ? selected.value.filter(i => i !== optIndex)
    : [...selected.value, optIndex].sort((a, b) => a - b)
  choose(next)
}

async function submit() {
  submitting.value = true
  try {
    result.value = await submitAttempt(props.courseUuid, props.quizUuid)
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Failed to submit quiz.')
  } finally {
    submitting.value = false
  }
}

function goBack() {
  router.push(`/courses/${props.courseUuid}`)
}

onMounted(async () => {
  try {
    attempt.value = await startAttempt(props.courseUuid, props.quizUuid)
  } catch (err: unknown) {
    error.value = err instanceof Error ? err.message : 'Failed to start quiz.'
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.quiz-prompt {
  white-space: pre-wrap;
}

.result-emoji {
  font-size: 4rem;
  line-height: 1;
}
</style>
