<template>
  <div>
    <div v-if="myFinalGrades.length === 0" class="py-8 text-center text-medium-emphasis">
      <v-icon class="mb-2 text-disabled" icon="mdi-certificate-outline" size="48" />
      <p class="text-body-1 font-weight-medium mb-1">No courses yet</p>
      <p class="text-caption">Your course grades will appear here once you are enrolled.</p>
    </div>

    <div v-else class="d-flex flex-column ga-3">
      <v-card
        v-for="fg in myFinalGrades"
        :key="fg.courseUuid"
        class="pa-4 bg-surface-variant"
        elevation="0"
        rounded="lg"
        :to="{ path: `/courses/${fg.courseUuid}`, query: { tab: 'grades' } }"
      >
        <div class="d-flex align-center justify-space-between flex-wrap ga-3">
          <div>
            <div class="text-caption text-medium-emphasis text-uppercase font-weight-bold">
              {{ fg.courseCode }}
            </div>
            <div class="text-h6 font-weight-bold text-high-emphasis">{{ fg.courseName }}</div>
          </div>

          <div v-if="fg.finished" class="d-flex align-center ga-4">
            <div class="text-right">
              <div class="text-h5 font-weight-bold text-high-emphasis">
                {{ formatPoints(fg.totalScore) }} / {{ formatPoints(fg.totalMaxScore) }} pts
              </div>
              <div class="text-caption text-medium-emphasis">
                {{ formatPoints(fg.percent) }}% · Finished
              </div>
            </div>
            <v-avatar
              class="text-h5 font-weight-bold elevation-1"
              :color="getGradeBadgeColor(fg.finalGrade)"
              size="48"
            >
              {{ fg.finalGrade }}
            </v-avatar>
          </div>

          <div v-else class="d-flex align-center ga-4">
            <div class="text-right">
              <div
                v-if="fg.progressMaxScore > 0"
                class="text-h5 font-weight-bold text-high-emphasis"
              >
                {{ formatPoints(fg.progressScore) }} / {{ formatPoints(fg.progressMaxScore) }} pts
              </div>
              <div v-else class="text-body-2 text-medium-emphasis">Nothing graded yet</div>
            </div>
            <v-chip color="info" size="small" variant="tonal">In progress</v-chip>
          </div>
        </div>
      </v-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CourseFinalGradeDto } from '@/types/grade'
import { getGradeBadgeColor } from '@/util/grading'

interface Props {
  myFinalGrades: readonly CourseFinalGradeDto[]
}

defineProps<Props>()

const numberFormat = new Intl.NumberFormat(undefined, { maximumFractionDigits: 2 })

function formatPoints(value: number): string {
  return numberFormat.format(value)
}
</script>
