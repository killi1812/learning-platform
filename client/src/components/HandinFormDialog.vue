<template>
  <v-dialog v-model="open" max-width="500" persistent>
    <v-card class="bg-surface pa-2" rounded="lg">
      <v-card-title class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis">
        {{ isEditing ? 'Edit Assignment' : 'Create Hand-in Assignment' }}
      </v-card-title>
      <v-card-subtitle class="px-4 text-medium-emphasis">
        {{
          isEditing
            ? 'Update the title, deadline, and late submission rules.'
            : 'Configure the submission window, points, and group rules.'
        }}
      </v-card-subtitle>

      <v-form ref="formRef" @submit.prevent="handleSave">
        <v-card-text class="px-4 pt-4">
          <v-text-field
            v-model="title"
            class="mb-3"
            label="Assignment Title"
            placeholder="e.g. Lab 1: Storage & Queues"
            required
            :rules="[v => Boolean(v && v.trim()) || 'Title is required']"
            variant="outlined"
          />

          <v-text-field
            v-model="deadline"
            class="mb-3"
            label="Deadline (Optional)"
            type="datetime-local"
            variant="outlined"
          />

          <v-text-field
            v-model.number="maxPoints"
            class="mb-3"
            label="Max Points"
            max="999"
            messages="Total points possible for this assignment (1 - 999, default 100)"
            min="1"
            placeholder="100"
            :rules="[
              v =>
                v === undefined ||
                v === '' ||
                (Number(v) >= 1 && Number(v) <= 999) ||
                'Must be 1 - 999',
            ]"
            type="number"
            variant="outlined"
          />

          <v-switch
            v-model="acceptsLateSubmissions"
            color="primary"
            label="Accept late submissions"
          />

          <v-switch
            v-model="allowsGroupSubmission"
            color="primary"
            label="Group assignment"
            messages="One member submits on behalf of the whole group"
          />
        </v-card-text>

        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="medium-emphasis" :disabled="saving" variant="text" @click="open = false">
            Cancel
          </v-btn>
          <v-btn
            color="primary"
            :disabled="!title.trim()"
            :loading="saving"
            type="submit"
            variant="elevated"
          >
            {{ isEditing ? 'Save Changes' : 'Create' }}
          </v-btn>
        </v-card-actions>
      </v-form>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import type { Assignment } from '@/types/quiz'
import type { VForm } from 'vuetify/components'
import { computed, ref, watch } from 'vue'
import { getApiErrorMessage } from '@/api/client'
import { createHandin, updateHandin } from '@/api/submission'
import { useToast } from '@/composables/useToast'
import { fromLocalDateTimeInput, toLocalDateTimeInput } from '@/util/dateTime'

const open = defineModel<boolean>({ required: true })

const props = defineProps<{
  courseUuid: string
  moduleUuid: string
  assignment?: Assignment
}>()

const emit = defineEmits<{
  saved: []
}>()

const toast = useToast()

const formRef = ref<InstanceType<typeof VForm> | undefined>(undefined)
const title = ref('')
const deadline = ref('')
const maxPoints = ref<number | undefined>(100)
const acceptsLateSubmissions = ref(false)
const allowsGroupSubmission = ref(false)
const saving = ref(false)

const isEditing = computed(() => Boolean(props.assignment?.uuid))

watch(open, isOpen => {
  if (!isOpen) return
  if (props.assignment) {
    title.value = props.assignment.title
    deadline.value = toLocalDateTimeInput(props.assignment.deadline)
    maxPoints.value = props.assignment.maxPoints || 100
    acceptsLateSubmissions.value = props.assignment.acceptsLateSubmissions || false
    allowsGroupSubmission.value = props.assignment.allowsGroupSubmission || false
  } else {
    title.value = ''
    deadline.value = ''
    maxPoints.value = 100
    acceptsLateSubmissions.value = false
    allowsGroupSubmission.value = false
  }
  formRef.value?.resetValidation()
})

async function handleSave(): Promise<void> {
  if (!props.courseUuid || !props.moduleUuid) return
  if (formRef.value) {
    const result = await formRef.value.validate()
    if (!result.valid) return
  }
  if (!title.value.trim()) return

  saving.value = true
  try {
    const resolvedMaxPoints = maxPoints.value && maxPoints.value > 0 ? maxPoints.value : 100
    const payload = {
      title: title.value.trim(),
      deadline: fromLocalDateTimeInput(deadline.value),
      acceptsLateSubmissions: acceptsLateSubmissions.value,
      allowsGroupSubmission: allowsGroupSubmission.value,
      maxPoints: resolvedMaxPoints,
    }

    if (props.assignment?.uuid) {
      await updateHandin(props.courseUuid, props.moduleUuid, props.assignment.uuid, payload)
      toast.success('Assignment updated successfully.')
    } else {
      await createHandin(props.courseUuid, props.moduleUuid, payload)
      toast.success('Assignment created successfully.')
    }
    open.value = false
    emit('saved')
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to save assignment.'))
  } finally {
    saving.value = false
  }
}
</script>
