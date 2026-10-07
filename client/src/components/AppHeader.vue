<template>
  <v-app-bar title="DAT250 Learning Platform" elevation="2" class="app-header">
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

<style lang="scss" scoped>
.app-header {
  background-color: rgb(var(--v-theme-primary)) !important;
  color: #ffffff !important;

  :deep(.v-app-bar__content) {
    color: #ffffff;
  }

  :deep(.v-btn) {
    color: #ffffff;
  }
}
</style>
