<template>
  <v-container class="py-6" max-width="900">
    <div class="d-flex align-center justify-space-between mb-4">
      <v-btn color="medium-emphasis" prepend-icon="mdi-arrow-left" variant="text" @click="goBack">
        Back to Quiz
      </v-btn>
      <h1 class="text-h6 font-weight-bold my-0">Quiz results</h1>
      <span />
    </div>

    <div v-if="loading" class="d-flex justify-center py-12">
      <v-progress-circular color="primary" indeterminate size="64" />
    </div>
    <v-alert v-else-if="error" color="error" icon="mdi-alert-circle-outline" variant="tonal">
      {{ error }}
    </v-alert>

    <v-card v-else class="bg-surface" elevation="1" rounded="lg">
      <v-table>
        <thead>
          <tr>
            <th>Student</th>
            <th>Email</th>
            <th class="text-right">Attempts</th>
            <th class="text-right">Best score</th>
            <th class="text-center">Result</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.email">
            <td>{{ row.username }}</td>
            <td>{{ row.email }}</td>
            <td class="text-right">{{ row.attempts }}</td>
            <td class="text-right">
              {{ row.percent === undefined ? '—' : `${Math.round(row.percent)}%` }}
            </td>
            <td class="text-center">
              <template v-if="row.passed === undefined">Not taken</template>
              <v-icon
                v-else
                :aria-label="row.passed ? 'Passed' : 'Failed'"
                :color="row.passed ? 'success' : 'error'"
                :icon="row.passed ? 'mdi-check-circle' : 'mdi-close-circle'"
                size="small"
              />
            </td>
          </tr>
          <tr v-if="rows.length === 0">
            <td class="text-center text-medium-emphasis py-6" colspan="5">
              No students enrolled yet.
            </td>
          </tr>
        </tbody>
      </v-table>
    </v-card>
  </v-container>
</template>

<script setup lang="ts">
import type { StudentResultRow } from '@/types/quiz'
import { onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { getApiErrorMessage } from '@/api/client'
import { getQuizResults } from '@/api/quiz'

const props = defineProps<{
  courseUuid: string
  moduleUuid: string
  quizUuid: string
}>()

const router = useRouter()
const loading = ref(true)
const error = ref('')
const rows = ref<StudentResultRow[]>([])
let loadCounter = 0

function goBack(): void {
  void router.push(
    `/courses/${props.courseUuid}/modules/${props.moduleUuid}/quizzes/${props.quizUuid}/edit`,
  )
}

async function loadResults(): Promise<void> {
  const requestId = ++loadCounter
  loading.value = true
  error.value = ''
  try {
    const resultRows = await getQuizResults(props.courseUuid, props.quizUuid)
    if (requestId === loadCounter) {
      rows.value = resultRows
    }
  } catch (error_: unknown) {
    if (requestId === loadCounter) {
      error.value = getApiErrorMessage(error_, 'Failed to load results.')
    }
  } finally {
    if (requestId === loadCounter) {
      loading.value = false
    }
  }
}

onMounted(() => {
  void loadResults()
})

watch(
  () => [props.courseUuid, props.quizUuid],
  () => {
    void loadResults()
  },
)
</script>
