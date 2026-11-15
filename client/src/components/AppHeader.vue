<template>
  <v-app-bar class="app-header" color="primary" elevation="2" title="DAT250 Learning Platform">
    <template #append>
      <v-btn
        aria-label="Toggle color theme"
        class="mr-2"
        icon="mdi-theme-light-dark"
        @click="theme.cycle()"
      />

      <template v-if="isAuthenticated">
        <NotificationBell />

        <v-chip class="mr-2 font-weight-medium" variant="tonal">
          <v-icon icon="mdi-account" start />
          {{ currentUser?.username || 'User' }}
        </v-chip>
      </template>

      <v-btn v-else prepend-icon="mdi-login" to="/login" variant="tonal">Sign In</v-btn>
    </template>
  </v-app-bar>
</template>

<script setup lang="ts">
import { useTheme } from 'vuetify'
import NotificationBell from '@/components/NotificationBell.vue'
import { useAuth } from '@/composables/useAuth'

const theme = useTheme()
const { isAuthenticated, currentUser } = useAuth()
</script>
