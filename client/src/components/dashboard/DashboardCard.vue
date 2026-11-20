<template>
  <v-card class="bg-surface h-100" elevation="2" rounded="lg">
    <div class="d-flex align-center ga-2 pa-4 pb-2">
      <v-icon v-if="icon" color="primary" :icon="icon" size="20" />
      <h2 class="text-subtitle-1 font-weight-bold text-high-emphasis">{{ title }}</h2>
    </div>

    <div v-if="loading" class="d-flex justify-center pa-6">
      <v-progress-circular color="primary" indeterminate size="28" />
    </div>
    <v-alert v-else-if="error" class="mx-4 mb-4" color="error" density="compact" variant="tonal">
      {{ error }}
    </v-alert>
    <p v-else-if="empty" class="text-body-2 text-medium-emphasis px-4 pb-4">{{ empty }}</p>
    <div v-else class="px-4 pb-4">
      <slot />
    </div>
  </v-card>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  icon?: string
  loading?: boolean
  error?: string
  /** Message shown instead of the slot when there is nothing to list. */
  empty?: string
}>()
</script>
