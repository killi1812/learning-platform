<template>
  <v-dialog v-model="open" max-width="500" persistent>
    <v-card class="bg-surface pa-2" rounded="lg">
      <v-card-title class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis">
        Add Participant
      </v-card-title>
      <v-card-subtitle class="px-4 text-medium-emphasis">
        Enroll an existing user into this course by their registered email.
      </v-card-subtitle>

      <v-form ref="formRef" @submit.prevent="handleAdd">
        <v-card-text class="px-4 pt-4">
          <v-text-field
            v-model="email"
            class="mb-3"
            label="User Email"
            placeholder="student@hvl.no"
            required
            :rules="[rules.required, rules.email]"
            type="email"
            variant="outlined"
          />

          <v-select
            v-model="role"
            :items="['STUDENT', 'TEACHER']"
            label="Role"
            variant="outlined"
          />
        </v-card-text>

        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="medium-emphasis" variant="text" @click="open = false"> Cancel </v-btn>
          <v-btn color="primary" :loading="saving" type="submit" variant="elevated">
            Add Participant
          </v-btn>
        </v-card-actions>
      </v-form>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import type { CourseRole } from '@/types/course'
import { ref, watch } from 'vue'
import { getApiErrorMessage } from '@/api/client'
import { addEnrollment } from '@/api/course'
import { useToast } from '@/composables/useToast'
import { EMAIL_PATTERN } from '@/util/validation'

interface Props {
  courseUuid: string
}

const props = defineProps<Props>()
const open = defineModel<boolean>({ required: true })

const emit = defineEmits<{
  saved: []
}>()

const toast = useToast()

const formRef = ref<{ validate: () => Promise<{ valid: boolean }> } | undefined>(undefined)
const email = ref('')
const role = ref<CourseRole>('STUDENT')
const saving = ref(false)

const rules = {
  required: (v: string) => !!v.trim() || 'Email address is required',
  email: (v: string) => EMAIL_PATTERN.test(v.trim()) || 'Please enter a valid email address',
}

watch(open, isOpen => {
  if (isOpen) {
    email.value = ''
    role.value = 'STUDENT'
  }
})

async function handleAdd(): Promise<void> {
  const result = await formRef.value?.validate()
  if ((result && !result.valid) || !props.courseUuid) {
    return
  }

  saving.value = true
  try {
    await addEnrollment(props.courseUuid, {
      email: email.value.trim(),
      role: role.value,
    })
    open.value = false
    toast.success('Participant enrolled successfully.')
    emit('saved')
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to add participant.'))
  } finally {
    saving.value = false
  }
}
</script>
