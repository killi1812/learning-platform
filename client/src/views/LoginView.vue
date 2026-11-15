<template>
  <div class="auth-page d-flex align-center justify-center">
    <v-container max-width="480">
      <v-card class="login-card pa-4 pa-sm-6" elevation="8" rounded="xl">
        <!-- Brand Header -->
        <div class="text-center mb-6">
          <v-avatar class="mb-3 elevation-4" color="primary" size="64">
            <v-icon icon="mdi-school" size="36" />
          </v-avatar>
          <h1 class="text-h5 font-weight-bold text-high-emphasis">DAT250 Learning Platform</h1>
          <p class="text-body-2 text-medium-emphasis mt-1">
            Sign in to access your courses, groups, and grades
          </p>
        </div>

        <!-- Error Alert -->
        <v-alert
          v-if="authError"
          class="mb-4"
          closable
          color="error"
          icon="mdi-alert-circle-outline"
          rounded="lg"
          variant="tonal"
          @click:close="clearAuthError"
        >
          {{ authError }}
        </v-alert>

        <!-- Redirect Notice (if redirected from a protected route) -->
        <v-alert
          v-if="redirectReason"
          class="mb-4"
          color="info"
          icon="mdi-information-outline"
          rounded="lg"
          variant="tonal"
        >
          {{ redirectReason }}
        </v-alert>

        <!-- Login Form -->
        <v-form ref="formRef" @submit.prevent="handleSubmit">
          <v-text-field
            v-model="email"
            autocomplete="email"
            class="mb-2"
            density="comfortable"
            label="Email Address"
            name="email"
            prepend-inner-icon="mdi-email-outline"
            required
            :rules="[rules.required, rules.email]"
            type="email"
            variant="outlined"
          />

          <v-text-field
            v-model="password"
            :append-inner-icon="showPassword ? 'mdi-eye-off-outline' : 'mdi-eye-outline'"
            autocomplete="current-password"
            class="mb-5"
            density="comfortable"
            label="Password"
            name="password"
            prepend-inner-icon="mdi-lock-outline"
            required
            :rules="[rules.required]"
            :type="showPassword ? 'text' : 'password'"
            variant="outlined"
            @click:append-inner="showPassword = !showPassword"
          />

          <v-btn
            block
            class="text-capitalize font-weight-bold"
            color="primary"
            elevation="2"
            :loading="isSubmitting"
            rounded="lg"
            size="large"
            type="submit"
          >
            Sign In
          </v-btn>
        </v-form>

        <v-divider class="my-5" />

        <div class="text-center text-body-2 text-medium-emphasis">
          Don't have an account?
          <router-link
            class="text-primary font-weight-bold text-decoration-none ml-1"
            to="/register"
          >
            Sign Up
          </router-link>
        </div>
      </v-card>
    </v-container>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { getSafeRedirect } from '@/util/redirect'
import { EMAIL_PATTERN } from '@/util/validation'

const router = useRouter()
const route = useRoute()
const { login, isSubmitting, authError, clearAuthError } = useAuth()

const formRef = ref<{ validate: () => Promise<{ valid: boolean }> } | undefined>(undefined)
const email = ref('')
const password = ref('')
const showPassword = ref(false)

const rules = {
  required: (value: string) => !!value.trim() || 'This field is required',
  email: (value: string) =>
    EMAIL_PATTERN.test(value.trim()) || 'Please enter a valid email address',
}

const redirectReason = computed(() => {
  if (route.query.redirect) {
    return 'Please sign in to access the requested page.'
  }
  return undefined
})

onMounted(() => {
  clearAuthError()
})

onBeforeUnmount(() => {
  clearAuthError()
})

async function handleSubmit(): Promise<void> {
  const result = await formRef.value?.validate()
  if (result && !result.valid) {
    return
  }

  const success = await login({
    email: email.value.trim(),
    password: password.value,
  })

  if (success) {
    const target = getSafeRedirect(route.query.redirect)
    void router.push(target)
  }
}
</script>

<style scoped>
.login-card {
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  backdrop-filter: blur(8px);
}
</style>
