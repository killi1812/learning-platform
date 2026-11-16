<template>
  <v-container class="py-6" max-width="720">
    <h1 class="text-h4 font-weight-bold mb-6">Send notification</h1>

    <v-card class="pa-6" elevation="2" rounded="lg">
      <v-form ref="formRef" @submit.prevent="onSend">
        <v-textarea
          v-model="recipientsInput"
          class="mb-2"
          hint="Separate with spaces, commas or new lines"
          label="Recipient UUIDs"
          persistent-hint
          rows="3"
          :rules="[rules.recipients]"
          variant="outlined"
        />
        <v-text-field
          v-model="title"
          class="mb-2"
          label="Title"
          :rules="[rules.required]"
          variant="outlined"
        />
        <v-textarea
          v-model="body"
          class="mb-2"
          label="Body"
          rows="3"
          :rules="[rules.required]"
          variant="outlined"
        />
        <v-text-field
          v-model="url"
          class="mb-4"
          hint="Internal path, e.g. /courses/<course-uuid>"
          label="Link (optional)"
          persistent-hint
          :rules="[rules.internalUrl]"
          variant="outlined"
        />

        <div class="d-flex justify-end">
          <v-btn color="primary" :loading="sending" prepend-icon="mdi-send" type="submit">
            Send
          </v-btn>
        </div>
      </v-form>
    </v-card>
  </v-container>
</template>

<script lang="ts" setup>
import { ref } from 'vue'
import { getApiErrorMessage } from '@/api/client'
import { sendNotification } from '@/api/notification'
import { useToast } from '@/composables/useToast'
import { isInternalPath } from '@/util/url'

const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

const formRef = ref<{ validate: () => Promise<{ valid: boolean }> } | undefined>(undefined)
const recipientsInput = ref('')
const title = ref('')
const body = ref('')
const url = ref('')
const sending = ref(false)
const toast = useToast()

function parseUuids(raw: string): string[] {
  return raw
    .split(/[\s,]+/)
    .map(part => part.trim())
    .filter(part => part.length > 0)
}

const rules = {
  required: (v: string) => !!v.trim() || 'This field is required',
  recipients: (v: string) => {
    const ids = parseUuids(v)
    if (ids.length === 0) {
      return 'Enter at least one recipient UUID'
    }
    const invalid = ids.find(id => !UUID_PATTERN.test(id))
    return invalid ? `Invalid UUID: ${invalid}` : true
  },
  internalUrl: (v: string) => {
    const trimmed = v.trim()
    if (!trimmed) {
      return true
    }
    return isInternalPath(trimmed) || 'Must be an internal app path starting with /'
  },
}

async function onSend(): Promise<void> {
  const result = await formRef.value?.validate()
  if (result && !result.valid) {
    return
  }

  const userUuids = parseUuids(recipientsInput.value)

  sending.value = true
  try {
    const sent = await sendNotification({
      userUuids,
      title: title.value.trim(),
      body: body.value.trim(),
      url: url.value.trim() || undefined,
    })
    toast.success(`Sent ${sent.length} notification(s)`)
    body.value = ''
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Send failed'))
  } finally {
    sending.value = false
  }
}
</script>
