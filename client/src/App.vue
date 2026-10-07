<template>
  <v-app>
    <AppHeader @toggle-drawer="drawer = !drawer" />
    <AppSidebar v-model="drawer" />

    <v-main class="bg-grey-lighten-4">
      <DashboardView />
    </v-main>
  </v-app>
</template>

<script lang="ts" setup>
import { ref } from 'vue'
import AppHeader from '@/components/AppHeader.vue'
import AppSidebar from '@/components/AppSidebar.vue'
import DashboardView from '@/components/DashboardView.vue'

const drawer = ref(false)
</script>
