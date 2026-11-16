<template>
  <v-dialog v-model="open" max-width="540" persistent>
    <v-card class="bg-surface pa-2" rounded="lg">
      <v-card-title class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis">
        Upload Course Material
      </v-card-title>
      <v-card-subtitle class="px-4 text-medium-emphasis">
        Upload a file (.pdf, .pptx, .md, .txt) or enter an external URL for embedded view.
      </v-card-subtitle>

      <v-form ref="formRef" @submit.prevent="handleUpload">
        <v-card-text class="px-4 pt-4">
          <v-text-field
            v-model="title"
            class="mb-3"
            label="Title (Optional for files, required for URLs)"
            placeholder="e.g. Lecture 1 Slides"
            :rules="[rules.title]"
            variant="outlined"
          />

          <v-textarea
            v-model="description"
            class="mb-3"
            label="Description / Notes (Optional)"
            placeholder="Additional context or notes"
            rows="2"
            variant="outlined"
          />

          <v-file-input
            v-model="fileInput"
            class="mb-3"
            clearable
            label="Select File"
            placeholder="Choose .pdf, .docx, .pptx, image, .md, .txt"
            prepend-icon="mdi-paperclip"
            show-size
            variant="outlined"
          />

          <v-text-field
            v-model="sourceUrl"
            class="mb-3"
            label="External / Iframe URL (Optional if file uploaded)"
            placeholder="https://example.com/embed"
            :rules="[rules.sourceUrl]"
            variant="outlined"
          />
        </v-card-text>

        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="medium-emphasis" variant="text" @click="open = false"> Cancel </v-btn>
          <v-btn
            color="primary"
            :disabled="!canSubmit"
            :loading="uploading"
            type="submit"
            variant="elevated"
          >
            Upload
          </v-btn>
        </v-card-actions>
      </v-form>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { getApiErrorMessage } from '@/api/client'
import { uploadFile } from '@/api/content'
import { useToast } from '@/composables/useToast'
import { isSafeHttpUrl } from '@/util/url'

interface Props {
  courseUuid: string
  moduleUuid: string
}

const props = defineProps<Props>()
const open = defineModel<boolean>({ required: true })

const emit = defineEmits<{
  saved: [moduleUuid: string]
}>()

const toast = useToast()

const formRef = ref<{ validate: () => Promise<{ valid: boolean }> } | undefined>(undefined)
const fileInput = ref<File | File[] | undefined>(undefined)
const title = ref('')
const description = ref('')
const sourceUrl = ref('')
const uploading = ref(false)

const selectedFile = computed(() =>
  Array.isArray(fileInput.value) ? fileInput.value[0] : fileInput.value,
)

const canSubmit = computed(() => {
  const hasFile = Boolean(selectedFile.value)
  const hasUrl = Boolean(sourceUrl.value.trim())
  const hasTitle = Boolean(title.value.trim())
  return (hasFile || hasUrl) && (hasFile || hasTitle)
})

const rules = {
  title: (v: string) => {
    if (selectedFile.value || !sourceUrl.value.trim()) {
      return true
    }
    return !!v.trim() || 'Title is required when adding an external link'
  },
  sourceUrl: (v: string) => {
    const trimmed = v.trim()
    if (!trimmed) {
      return true
    }
    return isSafeHttpUrl(trimmed) || 'URL must start with https:// or http://'
  },
}

watch(open, isOpen => {
  if (isOpen) {
    fileInput.value = undefined
    title.value = ''
    description.value = ''
    sourceUrl.value = ''
  }
})

async function handleUpload(): Promise<void> {
  if (!props.moduleUuid || !props.courseUuid) {
    return
  }
  const result = await formRef.value?.validate()
  if (result && !result.valid) {
    return
  }

  const rawFile = selectedFile.value
  const trimmedUrl = sourceUrl.value.trim()
  if (!rawFile && !trimmedUrl) {
    toast.error('Please select a file or provide a URL.')
    return
  }
  if (!rawFile && !title.value.trim()) {
    toast.error('Title is required when adding an external link.')
    return
  }

  uploading.value = true
  try {
    await uploadFile(props.courseUuid, props.moduleUuid, {
      file: rawFile,
      title: title.value.trim() || undefined,
      description: description.value.trim() || undefined,
      sourceUrl: trimmedUrl || undefined,
    })
    toast.success('Course material uploaded successfully.')
    open.value = false
    emit('saved', props.moduleUuid)
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to upload course material.'))
  } finally {
    uploading.value = false
  }
}
</script>
