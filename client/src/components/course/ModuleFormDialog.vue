<template>
  <v-dialog v-model="open" max-width="500" persistent>
    <v-card class="bg-surface pa-2" rounded="lg">
      <v-card-title class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis">
        {{ module ? 'Edit Module' : 'Create Module' }}
      </v-card-title>

      <v-form ref="formRef" @submit.prevent="handleSave">
        <v-card-text class="px-4 pt-4">
          <v-text-field
            v-model="title"
            class="mb-3"
            label="Module Title"
            placeholder="e.g. Week 1: Architecture & Patterns"
            required
            :rules="[v => !!v.trim() || 'Module title is required']"
            variant="outlined"
          />

          <v-select
            v-model="dependsOnUuid"
            clearable
            item-title="title"
            item-value="uuid"
            :items="availablePrerequisites"
            label="Prerequisite Module (Optional)"
            placeholder="None"
            variant="outlined"
          />
        </v-card-text>

        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="medium-emphasis" variant="text" @click="open = false"> Cancel </v-btn>
          <v-btn color="primary" :loading="saving" type="submit" variant="elevated">
            {{ module ? 'Save' : 'Create' }}
          </v-btn>
        </v-card-actions>
      </v-form>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import type { Module } from '@/types/course'
import { computed, ref, watch } from 'vue'
import { getApiErrorMessage } from '@/api/client'
import { createModule, updateModule } from '@/api/course'
import { useToast } from '@/composables/useToast'

interface Props {
  courseUuid: string
  modules: readonly Module[]
  module?: Module
}

const props = defineProps<Props>()
const open = defineModel<boolean>({ required: true })

const emit = defineEmits<{
  saved: []
}>()

const toast = useToast()

const formRef = ref<{ validate: () => Promise<{ valid: boolean }> } | undefined>(undefined)
const title = ref('')
const position = ref<number | undefined>(undefined)
const dependsOnUuid = ref<string | undefined>(undefined)
const saving = ref(false)

watch(
  [open, () => props.module],
  ([isOpen, nextModule]) => {
    if (!isOpen) {
      return
    }
    if (nextModule) {
      title.value = nextModule.title
      position.value = nextModule.position
      dependsOnUuid.value = nextModule.dependsOnUuid
    } else {
      title.value = ''
      position.value = undefined
      dependsOnUuid.value = undefined
    }
  },
  { immediate: true },
)

const availablePrerequisites = computed(() => {
  const disallowed = new Set<string>()
  if (props.module) {
    disallowed.add(props.module.uuid)
    let added = true
    while (added) {
      added = false
      for (const m of props.modules) {
        if (m.dependsOnUuid && disallowed.has(m.dependsOnUuid) && !disallowed.has(m.uuid)) {
          disallowed.add(m.uuid)
          added = true
        }
      }
    }
  }
  return props.modules.filter(m => !disallowed.has(m.uuid))
})

async function handleSave(): Promise<void> {
  const result = await formRef.value?.validate()
  if ((result && !result.valid) || !props.courseUuid) {
    return
  }

  saving.value = true
  try {
    if (props.module) {
      await updateModule(props.courseUuid, props.module.uuid, {
        title: title.value.trim(),
        position: position.value,
        dependsOnUuid: dependsOnUuid.value,
      })
      toast.success('Module updated successfully.')
    } else {
      await createModule(props.courseUuid, {
        title: title.value.trim(),
        dependsOnUuid: dependsOnUuid.value,
      })
      toast.success('Module created successfully.')
    }
    open.value = false
    emit('saved')
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to save module.'))
  } finally {
    saving.value = false
  }
}
</script>
