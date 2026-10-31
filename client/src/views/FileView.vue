<template>
  <v-container class="py-6" max-width="1200">
    <!-- Header Navigation -->
    <div class="d-flex align-center justify-space-between mb-4">
      <v-btn color="medium-emphasis" prepend-icon="mdi-arrow-left" variant="text" @click="goBack">
        Back to Course
      </v-btn>

      <h1
        v-if="page"
        class="text-h6 font-weight-bold text-high-emphasis my-0 text-truncate text-center flex-grow-1 mx-4"
      >
        {{ page.title }}
      </h1>

      <div class="d-flex align-center ga-2">
        <v-btn
          v-if="previewBlobUrl || page?.sourceUrl"
          color="primary"
          prepend-icon="mdi-open-in-new"
          variant="tonal"
          @click="openExternal"
        >
          {{ isFile ? 'Open File / Download' : 'Open Link' }}
        </v-btn>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="d-flex flex-column align-center justify-center py-12">
      <v-progress-circular color="primary" indeterminate size="64" />
      <span class="mt-4 text-body-1 text-medium-emphasis">Loading course material...</span>
    </div>

    <!-- Error Alert -->
    <v-alert
      v-else-if="error"
      class="mb-6"
      color="error"
      icon="mdi-alert-circle-outline"
      variant="tonal"
    >
      {{ error }}
    </v-alert>

    <div v-else-if="page">
      <!-- Embedded Preview Section -->
      <v-card class="bg-surface overflow-hidden" elevation="2" rounded="lg">
        <!-- 1. PDF Preview -->
        <div v-if="isPdf && previewBlobUrl && !previewFailed" class="preview-container">
          <iframe class="embedded-frame" :src="previewBlobUrl" title="PDF Document Preview" />
        </div>

        <!-- 2. Word (.docx) Document Preview -->
        <div
          v-else-if="isDocx && (previewData || previewBlobUrl) && !previewFailed"
          class="preview-container"
        >
          <vue-office-docx
            v-if="previewData"
            class="office-frame"
            :src="previewData"
            @error="onDocxError"
          />
        </div>

        <!-- 3. PPTX Presentation Preview -->
        <div
          v-else-if="isPptx && (previewData || previewBlobUrl) && !previewFailed"
          class="preview-container"
        >
          <vue-office-pptx
            v-if="previewData"
            class="office-frame"
            :src="previewData"
            @error="onPptxError"
          />
        </div>

        <!-- 4. Image Preview -->
        <div
          v-else-if="isImage && previewBlobUrl && !previewFailed"
          class="d-flex justify-center align-center pa-6 bg-surface-variant preview-container"
        >
          <img :alt="page.title" class="img-preview rounded-lg" :src="previewBlobUrl" />
        </div>

        <!-- 5. Markdown / Plain Text Preview -->
        <div v-else-if="isText && !previewFailed" class="pa-6">
          <v-card class="pa-4 bg-surface-variant font-monospace" elevation="0" rounded="lg">
            <pre class="text-body-2 text-high-emphasis text-preview">{{ textContent }}</pre>
          </v-card>
        </div>

        <!-- 6. External URL / Iframe Preview -->
        <div v-else-if="page.sourceUrl && !previewFailed" class="preview-container">
          <iframe
            class="embedded-frame"
            sandbox="allow-scripts allow-forms allow-popups"
            :src="page.sourceUrl"
            title="External Resource"
          />
        </div>

        <!-- 7. Fallback File Card -->
        <div v-else class="pa-12 text-center bg-surface-variant">
          <v-icon class="mb-3 text-medium-emphasis" :icon="formatIcon" size="64" />
          <h3 class="text-h6 font-weight-bold text-high-emphasis mb-2">
            {{ page.fileName || page.title }}
          </h3>
          <p class="text-body-2 text-medium-emphasis mb-4">
            This file type cannot be previewed directly in the browser. Click below to download or
            view it.
          </p>
          <v-btn
            v-if="previewBlobUrl"
            color="primary"
            prepend-icon="mdi-download"
            variant="elevated"
            @click="openExternal"
          >
            Download File
          </v-btn>
        </div>
      </v-card>
    </div>
  </v-container>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, onErrorCaptured } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from '@/composables/useToast'
import VueOfficePptx from '@vue-office/pptx'
import VueOfficeDocx from '@vue-office/docx'
import { getFile, getFilePreviewBlob } from '@/api/content'
import type { FileUpload } from '@/types/content'

const props = defineProps<{
  courseUuid: string
  moduleUuid: string
  fileUuid: string
}>()

const router = useRouter()
const toast = useToast()

let isMounted = false

const loading = ref(true)
const error = ref<string | undefined>(undefined)
const page = ref<FileUpload | undefined>(undefined)
const previewBlobUrl = ref<string | undefined>(undefined)
const previewData = ref<Blob | ArrayBuffer | undefined>(undefined)
const previewFailed = ref(false)
const textContent = ref<string>('')

const fileNameLower = computed(() => page.value?.fileName?.toLowerCase() || '')
const isFile = computed(() => !!page.value?.fileName)

const isPdf = computed(() => {
  return page.value?.contentType?.includes('pdf') || fileNameLower.value.endsWith('.pdf')
})

const isDocx = computed(() => {
  return page.value?.contentType?.includes('word') || fileNameLower.value.endsWith('.docx')
})

const isPptx = computed(() => {
  return (
    page.value?.contentType?.includes('presentation') ||
    page.value?.contentType?.includes('powerpoint') ||
    fileNameLower.value.endsWith('.pptx')
  )
})

const isImage = computed(() => {
  return (
    page.value?.contentType?.startsWith('image/') ||
    ['.png', '.jpg', '.jpeg', '.gif', '.webp'].some(ext => fileNameLower.value.endsWith(ext))
  )
})

const isText = computed(() => {
  return (
    page.value?.contentType?.includes('text') ||
    fileNameLower.value.endsWith('.txt') ||
    fileNameLower.value.endsWith('.md') ||
    fileNameLower.value.endsWith('.json') ||
    fileNameLower.value.endsWith('.go') ||
    fileNameLower.value.endsWith('.ts')
  )
})

const formatIcon = computed(() => {
  if (isPdf.value) return 'mdi-file-pdf-box'
  if (isDocx.value) return 'mdi-file-word-box'
  if (isPptx.value) return 'mdi-file-powerpoint-box'
  if (isImage.value) return 'mdi-file-image'
  if (fileNameLower.value.endsWith('.md')) return 'mdi-language-markdown'
  if (isText.value) return 'mdi-file-document-outline'
  if (page.value?.sourceUrl) return 'mdi-link-variant'
  return 'mdi-file-outline'
})

function goBack() {
  router.push(`/courses/${props.courseUuid}`)
}

function openExternal() {
  if (previewBlobUrl.value) {
    const a = document.createElement('a')
    a.href = previewBlobUrl.value
    a.download = page.value?.fileName || 'download'
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
  } else if (page.value?.sourceUrl) {
    const url = page.value.sourceUrl
    if (url.startsWith('https://') || url.startsWith('http://')) {
      window.open(url, '_blank', 'noopener,noreferrer')
    }
  }
}

function onDocxError(err: unknown) {
  console.warn('DOCX preview failed, switching to fallback:', err)
  previewFailed.value = true
}

function onPptxError(err: unknown) {
  console.warn('PPTX preview failed, switching to fallback:', err)
  previewFailed.value = true
}

onErrorCaptured(err => {
  console.warn('Preview render error caught by boundary:', err)
  previewFailed.value = true
  return false
})

async function loadPage() {
  loading.value = true
  error.value = undefined
  previewFailed.value = false

  try {
    const fetchedPage = await getFile(props.courseUuid, props.moduleUuid, props.fileUuid)
    if (!isMounted) return
    page.value = fetchedPage

    if (fetchedPage.fileName) {
      const blob = await getFilePreviewBlob(props.courseUuid, props.moduleUuid, props.fileUuid)
      if (!isMounted) return
      previewBlobUrl.value = URL.createObjectURL(blob)

      if (isPptx.value || isDocx.value) {
        previewData.value = await blob.arrayBuffer()
      } else {
        previewData.value = blob
      }

      if (isText.value) {
        textContent.value = await blob.text()
      }
    }
  } catch (err: unknown) {
    if (!isMounted) return
    const errorMsg = err instanceof Error ? err.message : 'Failed to load file material'
    error.value = errorMsg
    if (
      errorMsg.includes('Forbidden') ||
      errorMsg.includes('insufficient permissions') ||
      errorMsg.includes('403') ||
      errorMsg.includes('prerequisite')
    ) {
      toast.error('You must be enrolled and satisfy module prerequisites to access this material.')
      router.push(`/courses/${props.courseUuid}`)
    }
  } finally {
    if (isMounted) {
      loading.value = false
    }
  }
}

onMounted(() => {
  isMounted = true
  loadPage()
})

onUnmounted(() => {
  isMounted = false
  if (previewBlobUrl.value) {
    URL.revokeObjectURL(previewBlobUrl.value)
    previewBlobUrl.value = undefined
  }
})
</script>

<style scoped>
.preview-container {
  width: 100%;
  height: 80vh;
  min-height: 600px;
}

.embedded-frame {
  width: 100%;
  height: 100%;
  border: none;
}

.office-frame {
  width: 100%;
  height: 100%;
  overflow: auto;
}

:deep(.vue-office-docx) {
  height: 100%;
  overflow-y: auto;
}

:deep(.vue-office-docx .docx-wrapper > section.docx) {
  margin-bottom: 5px;
}

@media screen and (max-width: 800px) {
  :deep(.vue-office-docx .docx-wrapper) {
    padding: 10px;
  }
  :deep(.vue-office-docx .docx-wrapper > section.docx) {
    padding: 10px !important;
    width: 100% !important;
  }
}

.img-preview {
  max-width: 100%;
  max-height: 75vh;
  object-fit: contain;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.text-preview {
  white-space: pre-wrap;
  word-break: break-word;
  margin: 0;
  max-height: 70vh;
  overflow-y: auto;
}
</style>
