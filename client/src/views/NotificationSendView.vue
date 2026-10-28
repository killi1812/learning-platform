<template>
  <v-container class="py-6" max-width="720">
    <h1 class="text-h4 font-weight-bold mb-6">Send notification</h1>

    <v-card class="pa-6" elevation="2" rounded="lg">
      <v-form @submit.prevent="onSend">
        <v-textarea
          v-model="recipientsInput"
          label="Recipient UUIDs"
          hint="Separate with spaces, commas or new lines"
          persistent-hint
          rows="3"
          variant="outlined"
          class="mb-2"
        />
        <v-text-field v-model="title" label="Title" variant="outlined" />
        <v-textarea v-model="body" label="Body" rows="3" variant="outlined" />
        <v-text-field
          v-model="url"
          label="Link (optional)"
          hint="Internal path, e.g. /courses/<course-uuid>"
          persistent-hint
          variant="outlined"
        />

        <div class="d-flex justify-end">
          <v-btn type="submit" color="primary" prepend-icon="mdi-send" :loading="sending">
            Send
          </v-btn>
        </div>
      </v-form>
    </v-card>
  </v-container>
</template>

<script lang="ts" setup>
import { ref } from 'vue'
import { sendNotification } from '@/api/notification'
import { useToast } from '@/composables/useToast'

const UUID_PATTERN = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

const recipientsInput = ref('')
const title = ref('Test notification')
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

async function onSend(): Promise<void> {
  const userUuids = parseUuids(recipientsInput.value)
  if (userUuids.length === 0) {
    toast.error('Enter at least one recipient UUID')
    return
  }

  const invalid = userUuids.find(id => !UUID_PATTERN.test(id))
  if (invalid) {
    toast.error(`Invalid UUID: ${invalid}`)
    return
  }

  sending.value = true
  try {
    const sent = await sendNotification({
      userUuids,
      title: title.value,
      body: body.value,
      url: url.value.trim() || undefined,
    })
    toast.success(`Sent ${sent.length} notification(s)`)
    body.value = ''
  } catch (err: unknown) {
    toast.error(err instanceof Error ? err.message : 'Send failed')
  } finally {
    sending.value = false
  }
}
</script>
