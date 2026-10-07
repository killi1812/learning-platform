<template>
  <v-app>
    <AppHeader />
    <AppSidebar />

    <v-main class="bg-background">
      <DashboardView />
    </v-main>
  </v-app>
</template>

<script lang="ts" setup>
import AppHeader from '@/components/AppHeader.vue'
import AppSidebar from '@/components/AppSidebar.vue'
import DashboardView from '@/components/DashboardView.vue'
</script>
