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
              <template v-else>{{ row.passed ? '✅' : '❌' }}</template>
            </td>
          </tr>
          <tr v-if="rows.length === 0">
            <td colspan="5" class="text-center text-medium-emphasis py-6">
              No students enrolled yet.
            </td>
          </tr>
        </tbody>
      </v-table>
    </v-card>
  </v-container>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { getQuizResults } from '@/api/quiz'
import type { StudentResultRow } from '@/types/quiz'

const props = defineProps<{
  courseUuid: string
  moduleUuid: string
  quizUuid: string
}>()

const router = useRouter()
const loading = ref(true)
const error = ref('')
const rows = ref<StudentResultRow[]>([])

function goBack() {
  router.push(
    `/courses/${props.courseUuid}/modules/${props.moduleUuid}/quizzes/${props.quizUuid}/edit`,
  )
}

onMounted(async () => {
  try {
    rows.value = await getQuizResults(props.courseUuid, props.quizUuid)
  } catch (err: unknown) {
    error.value = err instanceof Error ? err.message : 'Failed to load results.'
  } finally {
    loading.value = false
  }
})
</script>
