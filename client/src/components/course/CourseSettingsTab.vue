<template>
  <div>
    <v-card class="pa-6 bg-surface mb-6" elevation="1" rounded="lg">
      <h2 class="text-h5 font-weight-bold text-high-emphasis mb-4">Course Details</h2>

      <v-form ref="formRef" @submit.prevent="handleSubmit">
        <v-text-field
          v-model="form.title"
          class="mb-3"
          label="Course Title"
          required
          :rules="[v => !!v.trim() || 'Course title is required']"
          variant="outlined"
        />
        <v-textarea
          v-model="form.description"
          class="mb-3"
          label="Course Description"
          rows="3"
          variant="outlined"
        />
        <v-row dense>
          <v-col cols="6">
            <v-select
              v-model="form.term"
              :items="['AUTUMN', 'SPRING']"
              label="Term"
              variant="outlined"
            />
          </v-col>
          <v-col cols="6">
            <v-select
              v-model="form.year"
              :items="[2025, 2026, 2027, 2028]"
              label="Year"
              variant="outlined"
            />
          </v-col>
        </v-row>

        <GradingScaleFields v-model="gradingScale" class="mt-4 mb-3" />

        <v-btn
          class="mt-2"
          color="primary"
          :loading="saving"
          size="large"
          type="submit"
          variant="elevated"
        >
          Save Changes
        </v-btn>
      </v-form>
    </v-card>

    <!-- Danger Zone -->
    <v-card class="pa-6 bg-surface border-error" elevation="0" rounded="lg">
      <h3 class="text-h6 font-weight-bold text-error mb-2">Danger Zone</h3>
      <p class="text-body-2 text-medium-emphasis mb-4">
        Deleting this course will permanently remove all modules, enrollments, and student
        submissions.
      </p>
      <v-btn
        color="error"
        :loading="deleting"
        prepend-icon="mdi-delete"
        variant="outlined"
        @click="emit('delete-course')"
      >
        Delete Course
      </v-btn>
    </v-card>
  </div>
</template>

<script setup lang="ts">
import type { Course, GradingScale, UpdateCoursePayload } from '@/types/course'
import { ref, watch } from 'vue'
import GradingScaleFields from '@/components/GradingScaleFields.vue'
import { useToast } from '@/composables/useToast'
import { DEFAULT_GRADING_SCALE, isValidGradingScale } from '@/util/grading'

interface Props {
  course: Course
  saving: boolean
  deleting: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'save-course': [payload: UpdateCoursePayload]
  'delete-course': []
}>()

const toast = useToast()

const formRef = ref<{ validate: () => Promise<{ valid: boolean }> } | undefined>(undefined)
const form = ref<UpdateCoursePayload>({
  title: '',
  description: '',
  term: 'AUTUMN',
  year: 2026,
})
const gradingScale = ref<GradingScale>({ ...DEFAULT_GRADING_SCALE })

watch(
  () => props.course,
  nextCourse => {
    form.value = {
      title: nextCourse.title,
      description: nextCourse.description,
      term: nextCourse.term,
      year: nextCourse.year,
    }
    gradingScale.value = nextCourse.gradingScale
      ? { ...nextCourse.gradingScale }
      : { ...DEFAULT_GRADING_SCALE }
  },
  { immediate: true },
)

async function handleSubmit(): Promise<void> {
  const result = await formRef.value?.validate()
  if (result && !result.valid) {
    return
  }
  if (!isValidGradingScale(gradingScale.value)) {
    toast.error('Invalid grading scale: Must satisfy 100 >= A > B > C > D > E > 0.')
    return
  }
  emit('save-course', {
    ...form.value,
    title: form.value.title.trim(),
    gradingScale: { ...gradingScale.value },
  })
}
</script>
