<template>
  <v-dialog v-model="open" max-width="540" persistent>
    <v-card class="bg-surface pa-2" rounded="lg">
      <v-card-title class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis">
        Edit Material Details
      </v-card-title>
      <v-card-subtitle class="px-4 text-medium-emphasis">
        Update the title and description for this course material.
      </v-card-subtitle>

      <v-form ref="formRef" @submit.prevent="handleSave">
        <v-card-text class="px-4 pt-4">
          <v-text-field
            v-model="title"
            class="mb-3"
            label="Title"
            required
            :rules="[v => !!v.trim() || 'Title is required']"
            variant="outlined"
          />

          <v-textarea
            v-model="description"
            class="mb-3"
            label="Description / Notes (Optional)"
            rows="3"
            variant="outlined"
          />
        </v-card-text>

        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="medium-emphasis" variant="text" @click="open = false"> Cancel </v-btn>
          <v-btn color="primary" :loading="saving" type="submit" variant="elevated">
            Save Changes
          </v-btn>
        </v-card-actions>
      </v-form>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import type { FileUpload } from '@/types/content'
import { ref, watch } from 'vue'
import { getApiErrorMessage } from '@/api/client'
import { updateFile } from '@/api/content'
import { useToast } from '@/composables/useToast'

interface Props {
  courseUuid: string
  moduleUuid: string
  page?: FileUpload
}

const props = defineProps<Props>()
const open = defineModel<boolean>({ required: true })

const emit = defineEmits<{
  saved: [moduleUuid: string]
}>()

const toast = useToast()

const formRef = ref<{ validate: () => Promise<{ valid: boolean }> } | undefined>(undefined)
const title = ref('')
const description = ref('')
const saving = ref(false)

watch(
  [open, () => props.page],
  ([isOpen, nextPage]) => {
    if (!isOpen || !nextPage) {
      return
    }
    title.value = nextPage.title
    description.value = nextPage.description ?? ''
  },
  { immediate: true },
)

async function handleSave(): Promise<void> {
  if (!props.page || !props.courseUuid || !props.moduleUuid) {
    return
  }
  const result = await formRef.value?.validate()
  if (result && !result.valid) {
    return
  }

  saving.value = true
  try {
    await updateFile(props.courseUuid, props.moduleUuid, props.page.uuid, {
      title: title.value.trim(),
      description: description.value.trim() || undefined,
    })
    toast.success('Course material updated successfully.')
    open.value = false
    emit('saved', props.moduleUuid)
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to update course material.'))
  } finally {
    saving.value = false
  }
}
</script>
