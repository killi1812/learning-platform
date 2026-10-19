<template>
  <v-app>
    <AppHeader v-if="showNavigation" />
    <AppSidebar v-if="showNavigation" />

    <v-main class="bg-background">
      <router-view />
    </v-main>

    <v-snackbar-queue v-model="toastQueue" closable location="bottom end" :total-visible="3" />
  </v-app>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AppHeader from '@/components/AppHeader.vue'
import AppSidebar from '@/components/AppSidebar.vue'
import { useToast } from '@/composables/useToast'

const route = useRoute()
const { queue: toastQueue } = useToast()
const showNavigation = computed(() => !route.meta.hideNavigation)
</script>
