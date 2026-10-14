<template>
  <v-app-bar title="DAT250 Learning Platform" color="primary" elevation="2" class="app-header">
    <template #append>
      <v-btn icon="mdi-theme-light-dark" class="mr-2" @click="$vuetify.theme.cycle()" />

      <template v-if="isAuthenticated">
        <v-chip class="mr-2 font-weight-medium" color="white" variant="tonal">
          <v-icon icon="mdi-account" start />
          {{ currentUser?.username || 'User' }}
        </v-chip>
      </template>

      <v-btn v-else prepend-icon="mdi-login" variant="tonal" color="white" to="/login">
        Sign In
      </v-btn>
    </template>
  </v-app-bar>
</template>

<script lang="ts" setup>
import { useAuth } from '@/composables/useAuth'

const { isAuthenticated, currentUser } = useAuth()
</script>
