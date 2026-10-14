<template>
  <div class="unauthorized-page d-flex align-center justify-center">
    <v-container max-width="540">
      <v-card class="pa-6 pa-sm-8 text-center" elevation="8" rounded="xl">
        <v-avatar color="warning-lighten-4" size="80" class="mb-4">
          <v-icon color="warning" icon="mdi-shield-lock-outline" size="48" />
        </v-avatar>

        <div class="mb-2">
          <v-chip color="warning" label size="small" class="font-weight-bold">
            401 / 403 ACCESS DENIED
          </v-chip>
        </div>

        <h1 class="text-h4 font-weight-bold text-high-emphasis mb-2">Unauthorized Page</h1>

        <p class="text-body-1 text-medium-emphasis mb-4">
          <template v-if="isAuthenticated">
            You are signed in as
            <strong class="text-high-emphasis">{{ currentUser?.username || 'user' }}</strong
            >, but your account does not have sufficient permissions to view this resource.
          </template>
          <template v-else>
            You do not have permission to view this resource. This page is protected and requires an
            authenticated account.
          </template>
        </p>

        <v-card
          v-if="attemptedPath"
          class="mb-6 pa-3 bg-grey-lighten-4 text-left"
          variant="flat"
          rounded="lg"
        >
          <div class="text-caption text-medium-emphasis">Attempted URL:</div>
          <code class="text-caption font-weight-bold text-primary">{{ attemptedPath }}</code>
        </v-card>

        <div class="d-flex justify-center">
          <!-- When user is logged in: offer redirect to dashboard -->
          <v-btn
            v-if="isAuthenticated"
            class="text-capitalize font-weight-bold px-6"
            color="primary"
            elevation="2"
            prepend-icon="mdi-home"
            rounded="lg"
            size="large"
            @click="goToDashboard"
          >
            Return to Dashboard
          </v-btn>

          <!-- When user is NOT logged in: only offer login -->
          <v-btn
            v-else
            class="text-capitalize font-weight-bold px-6"
            color="primary"
            elevation="2"
            prepend-icon="mdi-login"
            rounded="lg"
            size="large"
            @click="goToLogin"
          >
            Sign In Now
          </v-btn>
        </div>
      </v-card>
    </v-container>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuth } from '@/composables/useAuth'

const router = useRouter()
const route = useRoute()
const { isAuthenticated, currentUser } = useAuth()

const attemptedPath = computed(() => {
  const redirect = route.query.redirect
  return typeof redirect === 'string' ? redirect : undefined
})

function goToLogin() {
  if (attemptedPath.value) {
    router.push({ path: '/login', query: { redirect: attemptedPath.value } })
  } else {
    router.push('/login')
  }
}

function goToDashboard() {
  router.push('/dashboard')
}
</script>

<style scoped>
.unauthorized-page {
  min-height: 100vh;
  background: radial-gradient(
    circle at 50% 30%,
    rgba(var(--v-theme-warning), 0.08),
    transparent 70%
  );
}
</style>
