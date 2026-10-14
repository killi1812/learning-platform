<template>
  <div class="login-page d-flex align-center justify-center">
    <v-container max-width="480">
      <v-card class="login-card pa-4 pa-sm-6" elevation="8" rounded="xl">
        <!-- Brand Header -->
        <div class="text-center mb-6">
          <v-avatar color="primary" size="64" class="mb-3 elevation-4">
            <v-icon color="white" icon="mdi-school" size="36" />
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
          @click:close="authError = undefined"
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
            autocomplete="current-password"
            class="mb-3"
            density="comfortable"
            label="Password"
            name="password"
            prepend-inner-icon="mdi-lock-outline"
            required
            :rules="[rules.required]"
            :type="showPassword ? 'text' : 'password'"
            variant="outlined"
            :append-inner-icon="showPassword ? 'mdi-eye-off-outline' : 'mdi-eye-outline'"
            @click:append-inner="showPassword = !showPassword"
          />

          <div class="d-flex align-center justify-space-between mb-4">
            <v-checkbox
              v-model="rememberMe"
              color="primary"
              density="compact"
              hide-details
              label="Remember me"
            />
            <span class="text-caption text-primary cursor-pointer font-weight-medium">
              Need help?
            </span>
          </div>

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
            to="/register"
            class="text-primary font-weight-bold text-decoration-none ml-1"
          >
            Sign Up
          </router-link>
        </div>
      </v-card>
    </v-container>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { getSafeRedirect } from '@/util/redirect'

const router = useRouter()
const route = useRoute()
const { login, isSubmitting, authError } = useAuth()

const email = ref<string>()
const password = ref<string>()
const showPassword = ref(false)
const rememberMe = ref(true)

const rules = {
  required: (value: string) => !!value || 'This field is required',
  email: (value: string) => /.+@.+\..+/.test(value) || 'Please enter a valid email address',
}

const redirectReason = computed(() => {
  if (route.query.redirect) {
    return 'Please sign in to access the requested page.'
  }
  return undefined
})

async function handleSubmit() {
  if (!email.value || !password.value) {
    return
  }

  const success = await login({
    email: email.value.trim(),
    password: password.value,
  })

  if (success) {
    const target = getSafeRedirect(route.query.redirect)
    router.push(target)
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  background: radial-gradient(
    circle at 50% 30%,
    rgba(var(--v-theme-primary), 0.08),
    transparent 70%
  );
}

.login-card {
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  backdrop-filter: blur(8px);
}

.cursor-pointer {
  cursor: pointer;
}
</style>
