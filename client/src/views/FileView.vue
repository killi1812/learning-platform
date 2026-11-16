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
          v-if="previewBlobUrl || safeSourceUrl"
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
        <div v-if="canRenderPdf && previewBlobUrl && !previewFailed" class="preview-container">
          <iframe
            class="embedded-frame"
            sandbox="allow-same-origin"
            :src="previewBlobUrl"
            title="PDF Document Preview"
          />
        </div>

        <!-- 2. Word (.docx) Document Preview -->
        <div v-else-if="isDocx && previewData && !previewFailed" class="preview-container">
          <vue-office-docx class="office-frame" :src="previewData" @error="onDocxError" />
        </div>

        <!-- 3. PPTX Presentation Preview -->
        <div v-else-if="isPptx && previewData && !previewFailed" class="preview-container">
          <vue-office-pptx class="office-frame" :src="previewData" @error="onPptxError" />
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
        <div v-else-if="safeSourceUrl && !previewFailed" class="preview-container">
          <iframe
            class="embedded-frame"
            sandbox="allow-scripts allow-forms allow-popups"
            :src="safeSourceUrl"
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
            {{
              hasUnsafeSourceUrl
                ? 'This external link uses an unsupported URL scheme and cannot be opened.'
                : 'This file type cannot be previewed directly in the browser. Click below to download or view it.'
            }}
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
import type { FileUpload } from '@/types/content'
import VueOfficeDocx from '@vue-office/docx'
import VueOfficePptx from '@vue-office/pptx'
import { computed, onBeforeUnmount, onErrorCaptured, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ApiError, getApiErrorMessage } from '@/api/client'
import { getFile, getFilePreviewBlob } from '@/api/content'
import { useToast } from '@/composables/useToast'
import {
  getFileFormatIcon,
  isImageMaterial,
  isPdfMaterial,
  isPresentationMaterial,
  isTextMaterial,
  isWordMaterial,
} from '@/util/file'
import { isSafeHttpUrl } from '@/util/url'

const props = defineProps<{
  courseUuid: string
  moduleUuid: string
  fileUuid: string
}>()

const router = useRouter()
const toast = useToast()

const loading = ref(true)
const error = ref<string | undefined>(undefined)
const page = ref<FileUpload | undefined>(undefined)
const previewBlobUrl = ref<string | undefined>(undefined)
const previewData = ref<Blob | ArrayBuffer | undefined>(undefined)
const previewMimeType = ref('')
const previewFailed = ref(false)
const textContent = ref('')

let isUnmounted = false
let activeController: AbortController | undefined

const fileNameLower = computed(() => page.value?.fileName?.toLowerCase() ?? '')
const isFile = computed(() => Boolean(page.value?.fileName))
const isPdf = computed(() => isPdfMaterial(page.value))
const isDocx = computed(() => isWordMaterial(page.value) && !fileNameLower.value.endsWith('.doc'))
const isPptx = computed(
  () => isPresentationMaterial(page.value) && !fileNameLower.value.endsWith('.ppt'),
)
const isImage = computed(() => isImageMaterial(page.value))
const isText = computed(() => isTextMaterial(page.value))
const formatIcon = computed(() => getFileFormatIcon(page.value))

const canRenderPdf = computed(() => {
  if (!isPdf.value) {
    return false
  }
  if (previewMimeType.value.startsWith('application/pdf')) {
    return true
  }
  return (
    previewMimeType.value === '' &&
    Boolean(page.value?.contentType?.toLowerCase().startsWith('application/pdf'))
  )
})

const safeSourceUrl = computed(() => {
  const source = page.value?.sourceUrl
  return isSafeHttpUrl(source) ? source : undefined
})

const hasUnsafeSourceUrl = computed(() => Boolean(page.value?.sourceUrl) && !safeSourceUrl.value)

function clearPreviewState(): void {
  if (previewBlobUrl.value) {
    URL.revokeObjectURL(previewBlobUrl.value)
    previewBlobUrl.value = undefined
  }
  previewData.value = undefined
  previewMimeType.value = ''
  textContent.value = ''
}

function goBack(): void {
  void router.push(`/courses/${props.courseUuid}`)
}

function openExternal(): void {
  if (previewBlobUrl.value) {
    const anchor = document.createElement('a')
    anchor.href = previewBlobUrl.value
    anchor.download = page.value?.fileName || 'download'
    document.body.append(anchor)
    anchor.click()
    anchor.remove()
  } else if (safeSourceUrl.value) {
    window.open(safeSourceUrl.value, '_blank', 'noopener,noreferrer')
  }
}

function onDocxError(err: unknown): void {
  console.warn('DOCX preview failed, switching to fallback:', err)
  previewFailed.value = true
}

function onPptxError(err: unknown): void {
  console.warn('PPTX preview failed, switching to fallback:', err)
  previewFailed.value = true
}

onErrorCaptured(err => {
  console.warn('Preview render error caught by boundary:', err)
  previewFailed.value = true
  return false
})

async function loadPage(): Promise<void> {
  activeController?.abort()
  const controller = new AbortController()
  activeController = controller

  clearPreviewState()
  loading.value = true
  error.value = undefined
  previewFailed.value = false

  try {
    const fetchedPage = await getFile(
      props.courseUuid,
      props.moduleUuid,
      props.fileUuid,
      controller.signal,
    )
    if (isUnmounted || controller.signal.aborted) {
      return
    }
    page.value = fetchedPage

    if (fetchedPage.fileName) {
      const blob = await getFilePreviewBlob(
        props.courseUuid,
        props.moduleUuid,
        props.fileUuid,
        controller.signal,
      )
      if (isUnmounted || controller.signal.aborted) {
        return
      }

      previewMimeType.value = blob.type.toLowerCase()
      previewBlobUrl.value = URL.createObjectURL(blob)

      previewData.value = isPptx.value || isDocx.value ? await blob.arrayBuffer() : blob

      if (isText.value && !controller.signal.aborted) {
        textContent.value = await blob.text()
      }
    }
  } catch (error_: unknown) {
    if (isUnmounted || controller.signal.aborted) {
      return
    }
    clearPreviewState()
    const errorMsg = getApiErrorMessage(error_, 'Failed to load file material')
    error.value = errorMsg
    if (error_ instanceof ApiError && error_.status === 403) {
      toast.error('You must be enrolled and satisfy module prerequisites to access this material.')
      void router.push(`/courses/${props.courseUuid}`)
    }
  } finally {
    if (!isUnmounted && activeController === controller) {
      loading.value = false
    }
  }
}

onMounted(() => {
  void loadPage()
})

watch(
  () => [props.courseUuid, props.moduleUuid, props.fileUuid],
  () => {
    void loadPage()
  },
)

onBeforeUnmount(() => {
  isUnmounted = true
  activeController?.abort()
  clearPreviewState()
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
