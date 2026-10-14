<template>
  <v-app>
    <AppHeader v-if="showNavigation" />
    <AppSidebar v-if="showNavigation" />

    <v-main class="bg-background">
      <router-view />
    </v-main>
  </v-app>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AppHeader from '@/components/AppHeader.vue'
import AppSidebar from '@/components/AppSidebar.vue'

const route = useRoute()
const showNavigation = computed(() => !route.meta.hideNavigation)
</script>
