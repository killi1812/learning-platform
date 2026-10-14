<template>
  <v-container class="py-6">
    <h1 class="text-h4 font-weight-bold mb-2">{{ title }}</h1>
    <p class="text-medium-emphasis mb-6">Protected section for {{ title.toLowerCase() }}</p>

    <v-card class="pa-8 text-center" elevation="1" rounded="lg">
      <v-avatar color="primary-lighten-4" size="64" class="mb-4">
        <v-icon color="primary" :icon="icon" size="36" />
      </v-avatar>
      <div class="text-h6 font-weight-bold mb-1">{{ title }} Content</div>
      <p class="text-body-2 text-medium-emphasis">
        This area is securely protected by DAT250 authentication guards.
      </p>
    </v-card>
  </v-container>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    title?: string
    icon?: string
  }>(),
  {
    title: 'Page',
    icon: 'mdi-folder-outline',
  },
)
</script>
