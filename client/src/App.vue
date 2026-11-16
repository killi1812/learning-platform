<template>
  <v-app>
    <a class="skip-link" href="#main-content">Skip to main content</a>
    <AppHeader v-if="showNavigation" />
    <AppSidebar v-if="showNavigation" />

    <v-main id="main-content" class="bg-background" tabindex="-1">
      <router-view />
    </v-main>

    <ConfirmDialog />
    <v-snackbar-queue v-model="toastQueue" closable location="bottom end" :total-visible="3" />
  </v-app>
</template>

<script setup lang="ts">
import { computed, onErrorCaptured } from 'vue'
import { useRoute } from 'vue-router'
import AppHeader from '@/components/AppHeader.vue'
import AppSidebar from '@/components/AppSidebar.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import { useToast, useToastQueue } from '@/composables/useToast'

const route = useRoute()
const toast = useToast()
const { queue: toastQueue } = useToastQueue()
const showNavigation = computed(() => !route.meta.hideNavigation)

onErrorCaptured(err => {
  console.error('Unhandled view error:', err)
  toast.error(err instanceof Error ? err.message : 'An unexpected error occurred.')
  return false
})
</script>
