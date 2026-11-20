<template>
  <div class="d-flex flex-wrap ga-2">
    <v-btn
      v-for="action in actions"
      :key="action.to"
      color="primary"
      :prepend-icon="action.icon"
      :to="action.to"
      variant="tonal"
    >
      {{ action.label }}
    </v-btn>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ isTeacher: boolean }>()

const actions = computed(() => [
  { label: 'Browse Catalog', to: '/catalog', icon: 'mdi-compass-outline' },
  { label: 'Assignments', to: '/assignments', icon: 'mdi-clipboard-text-outline' },
  { label: 'Grades', to: '/grades', icon: 'mdi-school-outline' },
  ...(props.isTeacher
    ? [{ label: 'Send Notification', to: '/notification/send', icon: 'mdi-bell-plus-outline' }]
    : []),
])
</script>
