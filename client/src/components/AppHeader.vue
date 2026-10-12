<template>
  <v-app-bar title="DAT250 Learning Platform" color="primary" elevation="2" class="app-header">
    <template #append>
      <v-btn icon="mdi-theme-light-dark" @click="$vuetify.theme.cycle()" />
    </template>
  </v-app-bar>
</template>

<script lang="ts" setup></script>
