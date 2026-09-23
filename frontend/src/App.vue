<template>
  <v-app>
    <v-app-bar title="DAT250 Learning Platform">
      <template #append>
        <v-btn icon="mdi-theme-light-dark" @click="$vuetify.theme.cycle()" />
      </template>
    </v-app-bar>

    <v-main class="bg-grey-lighten-4">
      <AuthCard />
    </v-main>
  </v-app>
</template>

<script lang="ts" setup>
import AuthCard from '@/components/AuthCard.vue'
</script>
