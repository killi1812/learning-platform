<template>
  <v-app-bar title="DAT250 Learning Platform" elevation="0">
    <template #prepend>
      <v-app-bar-nav-icon @click="emits('toggle-drawer')" />
    </template>
    <template #append>
      <v-btn icon="mdi-theme-light-dark" @click="$vuetify.theme.cycle()" />
    </template>
  </v-app-bar>
</template>

<script lang="ts" setup>
const emits = defineEmits<{
  'toggle-drawer': []
}>()
</script>
