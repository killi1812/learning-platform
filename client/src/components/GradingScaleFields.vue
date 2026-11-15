<template>
  <div>
    <div class="d-flex align-center justify-space-between mb-2">
      <div>
        <span class="text-subtitle-2 font-weight-bold text-high-emphasis">
          Grading Scale (Minimum % for A–E)
        </span>
        <div class="text-caption text-medium-emphasis">
          Thresholds for final subject grades. F is anything below E.
        </div>
      </div>
      <v-btn prepend-icon="mdi-refresh" size="x-small" variant="text" @click="resetToDefaults">
        Reset
      </v-btn>
    </div>

    <v-row dense>
      <v-col cols="6" sm="4">
        <v-text-field
          v-model.number="scale.scaleA"
          density="compact"
          label="Grade A (≥ %)"
          max="100"
          min="1"
          type="number"
          variant="outlined"
        />
      </v-col>
      <v-col cols="6" sm="4">
        <v-text-field
          v-model.number="scale.scaleB"
          density="compact"
          label="Grade B (≥ %)"
          max="100"
          min="1"
          type="number"
          variant="outlined"
        />
      </v-col>
      <v-col cols="6" sm="4">
        <v-text-field
          v-model.number="scale.scaleC"
          density="compact"
          label="Grade C (≥ %)"
          max="100"
          min="1"
          type="number"
          variant="outlined"
        />
      </v-col>
      <v-col cols="6" sm="4">
        <v-text-field
          v-model.number="scale.scaleD"
          density="compact"
          label="Grade D (≥ %)"
          max="100"
          min="1"
          type="number"
          variant="outlined"
        />
      </v-col>
      <v-col cols="6" sm="4">
        <v-text-field
          v-model.number="scale.scaleE"
          density="compact"
          label="Grade E (≥ %)"
          max="100"
          min="1"
          type="number"
          variant="outlined"
        />
      </v-col>
      <v-col class="d-flex align-center" cols="6" sm="4">
        <span class="text-caption text-medium-emphasis"> Grade F: &lt; {{ scale.scaleE }}% </span>
      </v-col>
    </v-row>
  </div>
</template>

<script setup lang="ts">
import type { GradingScale } from '@/types/course'
import { DEFAULT_GRADING_SCALE } from '@/util/grading'

const scale = defineModel<GradingScale>({ required: true })

function resetToDefaults(): void {
  scale.value = { ...DEFAULT_GRADING_SCALE }
}
</script>
