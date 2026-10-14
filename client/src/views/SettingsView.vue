<template>
  <v-container class="py-6">
    <h1 class="text-h4 font-weight-bold mb-2">Account Settings</h1>
    <p class="text-medium-emphasis mb-6">Manage profile and authentication session</p>

    <v-card class="pa-6 mb-6" elevation="2" rounded="lg">
      <v-card-title class="text-h6 font-weight-bold px-0">User Profile</v-card-title>
      <v-card-subtitle class="px-0 mb-4">Current session details from JWT</v-card-subtitle>

      <v-row>
        <v-col cols="12" sm="6">
          <v-text-field
            label="Username"
            :model-value="currentUser?.username || 'N/A'"
            readonly
            variant="outlined"
          />
        </v-col>
        <v-col cols="12" sm="6">
          <v-text-field
            label="Role"
            :model-value="currentUser?.role || 'user'"
            readonly
            variant="outlined"
          />
        </v-col>
        <v-col cols="12" sm="6">
          <v-text-field
            label="Email"
            :model-value="currentUser?.email || 'N/A'"
            readonly
            variant="outlined"
          />
        </v-col>
        <v-col cols="12" sm="6">
          <v-text-field
            label="User ID"
            :model-value="currentUser?.sub || currentUser?.uuid || 'N/A'"
            readonly
            variant="outlined"
          />
        </v-col>
      </v-row>

      <v-divider class="my-4" />

      <div class="d-flex justify-end">
        <v-btn color="error" prepend-icon="mdi-logout" variant="outlined" @click="handleLogout">
          Sign Out
        </v-btn>
      </div>
    </v-card>
  </v-container>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'

const router = useRouter()
const { currentUser, logout } = useAuth()

async function handleLogout() {
  await logout()
  router.push('/login')
}
</script>
