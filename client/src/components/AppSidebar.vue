<template>
  <v-navigation-drawer v-model="drawerOpen" temporary>
    <v-list>
      <v-list-item title="Dashboard" prepend-icon="mdi-home" />
      <v-list-item title="My Courses" prepend-icon="mdi-book-open" />
      <v-list-item title="Messages" prepend-icon="mdi-message" />
      <v-list-item title="Groups" prepend-icon="mdi-account-group" />
      <v-list-item title="Grades" prepend-icon="mdi-chart-bar" />
    </v-list>
    <v-divider />
    <v-list>
      <v-list-item title="Settings" prepend-icon="mdi-cog" />
      <v-list-item title="Logout" prepend-icon="mdi-logout" />
    </v-list>
  </v-navigation-drawer>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

interface Props {
  modelValue: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const drawerOpen = computed({
  get: () => props.modelValue,
  set: value => emit('update:modelValue', value),
})
</script>
