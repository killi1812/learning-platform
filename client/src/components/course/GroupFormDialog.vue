<template>
  <v-dialog v-model="open" max-width="500" persistent>
    <v-card class="bg-surface pa-2" rounded="lg">
      <v-card-title class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis">
        {{ group ? 'Edit Group' : 'Create Group' }}
      </v-card-title>
      <v-card-subtitle class="px-4 text-medium-emphasis">
        {{
          group
            ? 'Update the group name and maximum member capacity.'
            : 'Add a new group for students to collaborate and submit assignments.'
        }}
      </v-card-subtitle>

      <v-form ref="formRef" @submit.prevent="handleSave">
        <v-card-text class="px-4 pt-4">
          <v-text-field
            v-model="name"
            class="mb-3"
            label="Group Name"
            :maxlength="100"
            placeholder="e.g. Group 1 or Team Alpha"
            required
            :rules="[
              v => !!v.trim() || 'Group name is required',
              v => v.trim().length >= 2 || 'Must be at least 2 characters',
              v => v.trim().length <= 100 || 'Must be at most 100 characters',
            ]"
            variant="outlined"
          />

          <v-text-field
            v-model.number="maxMembers"
            class="mb-3"
            hint="Maximum number of students permitted in this group (at least 1)"
            label="Maximum Members"
            max="100"
            min="1"
            persistent-hint
            required
            :rules="[v => (v >= 1 && v <= 100) || 'Must be between 1 and 100']"
            type="number"
            variant="outlined"
          />
        </v-card-text>

        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="medium-emphasis" variant="text" @click="open = false"> Cancel </v-btn>
          <v-btn color="primary" :loading="saving" type="submit" variant="elevated">
            {{ group ? 'Save Changes' : 'Create Group' }}
          </v-btn>
        </v-card-actions>
      </v-form>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import type { Group } from '@/types/course'
import { ref, watch } from 'vue'
import { getApiErrorMessage } from '@/api/client'
import { createGroup, updateGroup } from '@/api/course'
import { useToast } from '@/composables/useToast'

interface Props {
  courseUuid: string
  group?: Group
  defaultName: string
}

const props = defineProps<Props>()
const open = defineModel<boolean>({ required: true })

const emit = defineEmits<{
  saved: []
}>()

const toast = useToast()

const formRef = ref<{ validate: () => Promise<{ valid: boolean }> } | undefined>(undefined)
const name = ref('')
const maxMembers = ref(4)
const saving = ref(false)

watch(
  [open, () => props.group, () => props.defaultName],
  ([isOpen, nextGroup, nextDefaultName]) => {
    if (!isOpen) {
      return
    }
    if (nextGroup) {
      name.value = nextGroup.name
      maxMembers.value = nextGroup.maxMembers
    } else {
      name.value = nextDefaultName
      maxMembers.value = 4
    }
  },
  { immediate: true },
)

async function handleSave(): Promise<void> {
  const result = await formRef.value?.validate()
  if ((result && !result.valid) || !props.courseUuid) {
    return
  }

  saving.value = true
  try {
    if (props.group) {
      await updateGroup(props.group.uuid, {
        name: name.value.trim(),
        maxMembers: Number(maxMembers.value),
      })
      toast.success('Group updated successfully.')
    } else {
      await createGroup({
        courseUuid: props.courseUuid,
        name: name.value.trim(),
        maxMembers: Number(maxMembers.value),
      })
      toast.success('Group created successfully.')
    }
    open.value = false
    emit('saved')
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to save group.'))
  } finally {
    saving.value = false
  }
}
</script>
