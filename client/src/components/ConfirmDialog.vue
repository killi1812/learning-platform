<template>
  <v-dialog max-width="460" :model-value="state.isOpen" @update:model-value="onModelUpdate">
    <v-card class="bg-surface pa-2" rounded="lg">
      <v-card-title class="text-h6 font-weight-bold pt-4 px-4 text-high-emphasis">
        {{ state.title || 'Are you sure?' }}
      </v-card-title>
      <v-card-text class="px-4 py-2 text-body-1 text-medium-emphasis">
        {{ state.message }}
      </v-card-text>
      <v-card-actions class="px-4 pb-3 pt-2 d-flex justify-end ga-2">
        <v-btn color="medium-emphasis" variant="text" @click="handleCancel">
          {{ state.cancelText || 'Cancel' }}
        </v-btn>
        <v-btn :color="state.confirmColor || 'primary'" variant="elevated" @click="handleConfirm">
          {{ state.confirmText || 'Confirm' }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script setup lang="ts">
import { useConfirm } from '@/composables/useConfirm'

const { state, handleConfirm, handleCancel } = useConfirm()

function onModelUpdate(val: boolean) {
  if (!val) {
    handleCancel()
  }
}
</script>
