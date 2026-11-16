<template>
  <div class="auth-page d-flex align-center justify-center">
    <v-container max-width="540">
      <v-card class="pa-6 pa-sm-8 text-center" elevation="8" rounded="xl">
        <v-avatar class="mb-4" color="warning" size="80" variant="tonal">
          <v-icon color="warning" icon="mdi-shield-lock-outline" size="48" />
        </v-avatar>

        <div class="mb-2">
          <v-chip class="font-weight-bold" color="warning" label size="small">
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
          class="mb-6 pa-3 bg-surface-variant text-left"
          rounded="lg"
          variant="flat"
        >
          <div class="text-caption text-medium-emphasis">Attempted URL:</div>
          <code class="text-caption font-weight-bold text-primary">{{ attemptedPath }}</code>
        </v-card>

        <div class="d-flex justify-center">
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
import { useRoute, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { getSafeRedirect } from '@/util/redirect'

const router = useRouter()
const route = useRoute()
const { isAuthenticated, currentUser } = useAuth()

const attemptedPath = computed(() => {
  const redirect = getSafeRedirect(route.query.redirect, '')
  return redirect || undefined
})

function goToLogin(): void {
  if (attemptedPath.value) {
    void router.push({ path: '/login', query: { redirect: attemptedPath.value } })
  } else {
    void router.push('/login')
  }
}

function goToDashboard(): void {
  void router.push('/dashboard')
}
</script>
