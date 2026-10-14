<template>
  <v-navigation-drawer permanent width="220">
    <v-list density="compact" nav>
      <v-list-item to="/dashboard" title="Dashboard" prepend-icon="mdi-home" />
      <v-list-item to="/courses" title="My Courses" prepend-icon="mdi-book-open" />
      <v-list-item to="/messages" title="Messages" prepend-icon="mdi-message" />
      <v-list-item to="/groups" title="Groups" prepend-icon="mdi-account-group" />
      <v-list-item to="/grades" title="Grades" prepend-icon="mdi-chart-bar" />
      <v-list-item
        v-if="hasRole('admin') || hasRole('superadmin')"
        to="/admin"
        title="Admin"
        prepend-icon="mdi-shield-crown"
      />
    </v-list>
    <v-divider />
    <v-list density="compact" nav>
      <v-list-item to="/settings" title="Settings" prepend-icon="mdi-cog" />
      <v-list-item
        title="Logout"
        prepend-icon="mdi-logout"
        class="text-error"
        @click="handleLogout"
      />
    </v-list>
  </v-navigation-drawer>
</template>

<script lang="ts" setup>
import { useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'

const router = useRouter()
const { logout, hasRole } = useAuth()

async function handleLogout() {
  await logout()
  router.push('/login')
}
</script>
