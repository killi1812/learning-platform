<template>
  <v-navigation-drawer aria-label="Main navigation" permanent width="220">
    <v-list density="compact" nav>
      <v-list-item prepend-icon="mdi-home" title="Dashboard" to="/dashboard" />
      <v-list-item prepend-icon="mdi-book-open" title="My Courses" to="/courses" />
      <v-list-item prepend-icon="mdi-book-open-page-variant" title="Course Catalog" to="/catalog" />
      <v-list-item
        prepend-icon="mdi-clipboard-text-outline"
        title="Assignments"
        to="/assignments"
      />
      <v-list-item prepend-icon="mdi-account-group" title="Groups" to="/groups" />
      <v-list-item prepend-icon="mdi-chart-bar" title="Grades" to="/grades" />
      <v-list-item
        v-if="isAdminOrSuperadmin"
        prepend-icon="mdi-shield-crown"
        title="Admin"
        to="/admin"
      />
    </v-list>
    <v-divider />
    <v-list density="compact" nav>
      <v-list-item prepend-icon="mdi-cog" title="Settings" to="/settings" />
      <v-list-item
        class="text-error"
        link
        prepend-icon="mdi-logout"
        title="Logout"
        @click="handleLogout"
      />
    </v-list>
  </v-navigation-drawer>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'

const router = useRouter()
const { logout, isAdminOrSuperadmin } = useAuth()

async function handleLogout() {
  await logout()
  await router.push('/login')
}
</script>
