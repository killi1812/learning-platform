<template>
  <v-dialog v-model="open" max-width="540" persistent>
    <v-card class="bg-surface pa-2" rounded="lg">
      <v-card-title class="text-h5 font-weight-bold pt-4 px-4 text-high-emphasis">
        Bulk Create Groups
      </v-card-title>
      <v-card-subtitle class="px-4 text-medium-emphasis">
        Instantly generate multiple numbered groups with minimal clicks.
      </v-card-subtitle>

      <v-form ref="formRef" @submit.prevent="handleBulkCreate">
        <v-card-text class="px-4 pt-4">
          <v-text-field
            v-model="prefix"
            class="mb-3"
            hint="Prefix for group names (e.g. 'Group' produces 'Group 1', 'Group 2', ...)"
            label="Group Prefix"
            :maxlength="50"
            persistent-hint
            placeholder="Group"
            :rules="[v => !v || v.length <= 50 || 'Must be at most 50 characters']"
            variant="outlined"
          />

          <div class="mb-3">
            <label class="text-caption font-weight-bold text-medium-emphasis mb-1 d-block">
              Quick Count Preset:
            </label>
            <div class="d-flex ga-2">
              <v-btn
                v-for="cnt in [5, 10, 15, 20]"
                :key="cnt"
                :color="count === cnt ? 'primary' : 'secondary'"
                size="small"
                :variant="count === cnt ? 'elevated' : 'tonal'"
                @click="count = cnt"
              >
                {{ cnt }} Groups
              </v-btn>
            </div>
          </div>

          <v-row class="mb-2" dense>
            <v-col cols="6">
              <v-text-field
                v-model.number="count"
                label="Number of Groups"
                max="50"
                min="1"
                required
                :rules="[v => (v >= 1 && v <= 50) || 'Must be between 1 and 50']"
                type="number"
                variant="outlined"
              />
            </v-col>
            <v-col cols="6">
              <v-text-field
                v-model.number="maxMembers"
                label="Capacity per Group"
                max="100"
                min="1"
                required
                :rules="[v => (v >= 1 && v <= 100) || 'Must be between 1 and 100']"
                type="number"
                variant="outlined"
              />
            </v-col>
          </v-row>

          <v-alert
            class="text-caption"
            color="info"
            density="compact"
            icon="mdi-information-outline"
            variant="tonal"
          >
            <div>
              <strong>Preview:</strong>
              {{ prefix || 'Group' }} 1, {{ prefix || 'Group' }} 2
              <template v-if="count > 3"> ... {{ prefix || 'Group' }} {{ count }} </template>
              <template v-else-if="count === 3"> , {{ prefix || 'Group' }} 3 </template>
              ({{ maxMembers }} students each)
            </div>
            <div class="mt-1 text-medium-emphasis">
              Numbers that already exist will be automatically skipped.
            </div>
          </v-alert>
        </v-card-text>

        <v-card-actions class="pa-4 bg-surface-variant">
          <v-spacer />
          <v-btn color="medium-emphasis" variant="text" @click="open = false"> Cancel </v-btn>
          <v-btn
            color="primary"
            :loading="saving"
            prepend-icon="mdi-plus-box-multiple"
            type="submit"
            variant="elevated"
          >
            Create {{ count }} Groups
          </v-btn>
        </v-card-actions>
      </v-form>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { getApiErrorMessage } from '@/api/client'
import { bulkCreateGroups } from '@/api/course'
import { useToast } from '@/composables/useToast'

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
const prefix = ref('Group')
const count = ref(10)
const maxMembers = ref(4)
const saving = ref(false)

watch(open, isOpen => {
  if (isOpen) {
    prefix.value = 'Group'
    count.value = 10
    maxMembers.value = 4
  }
})

async function handleBulkCreate(): Promise<void> {
  const result = await formRef.value?.validate()
  if ((result && !result.valid) || !props.courseUuid) {
    return
  }

  saving.value = true
  try {
    const created = await bulkCreateGroups({
      courseUuid: props.courseUuid,
      prefix: prefix.value.trim() || 'Group',
      count: Number(count.value),
      maxMembers: Number(maxMembers.value),
    })
    toast.success(`Successfully created ${created.length} groups!`)
    open.value = false
    emit('saved')
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Failed to bulk-create groups.'))
  } finally {
    saving.value = false
  }
}
</script>
