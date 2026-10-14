<template>
  <div class="register-page d-flex align-center justify-center">
    <v-container max-width="500">
      <v-card class="register-card pa-4 pa-sm-6" elevation="8" rounded="xl">
        <!-- Brand Header -->
        <div class="text-center mb-6">
          <v-avatar color="primary" size="64" class="mb-3 elevation-4">
            <v-icon color="white" icon="mdi-account-plus" size="36" />
          </v-avatar>
          <h1 class="text-h5 font-weight-bold text-high-emphasis">Create an Account</h1>
          <p class="text-body-2 text-medium-emphasis mt-1">
            Join the DAT250 Learning Platform to start learning
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

        <!-- Registration Form -->
        <v-form ref="formRef" @submit.prevent="handleSubmit">
          <v-text-field
            v-model="username"
            autocomplete="username"
            class="mb-2"
            density="comfortable"
            label="Username"
            name="username"
            prepend-inner-icon="mdi-account-outline"
            required
            :rules="[rules.required, rules.minUsername]"
            variant="outlined"
          />

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
            autocomplete="new-password"
            class="mb-2"
            density="comfortable"
            label="Password"
            name="password"
            prepend-inner-icon="mdi-lock-outline"
            required
            :rules="[rules.required, rules.minPassword]"
            :type="showPassword ? 'text' : 'password'"
            variant="outlined"
            :append-inner-icon="showPassword ? 'mdi-eye-off-outline' : 'mdi-eye-outline'"
            @click:append-inner="showPassword = !showPassword"
          />

          <v-text-field
            v-model="confirmPassword"
            autocomplete="new-password"
            class="mb-4"
            density="comfortable"
            label="Confirm Password"
            name="confirmPassword"
            prepend-inner-icon="mdi-lock-check-outline"
            required
            :rules="[rules.required, rules.matchPassword]"
            :type="showPassword ? 'text' : 'password'"
            variant="outlined"
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
            Create Account
          </v-btn>
        </v-form>

        <v-divider class="my-5" />

        <div class="text-center text-body-2 text-medium-emphasis">
          Already have an account?
          <router-link to="/login" class="text-primary font-weight-bold text-decoration-none ml-1">
            Sign In
          </router-link>
        </div>
      </v-card>
    </v-container>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuth } from '@/composables/useAuth'

const router = useRouter()
const route = useRoute()
const { register, isSubmitting, authError } = useAuth()

const username = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const showPassword = ref(false)

const rules = {
  required: (v: string) => !!v || 'This field is required',
  minUsername: (v: string) => (v && v.length >= 2) || 'Username must be at least 2 characters',
  email: (v: string) => /.+@.+\..+/.test(v) || 'Please enter a valid email address',
  minPassword: (v: string) => (v && v.length >= 6) || 'Password must be at least 6 characters',
  matchPassword: (v: string) => v === password.value || 'Passwords do not match',
}

async function handleSubmit() {
  if (
    !username.value ||
    !email.value ||
    !password.value ||
    password.value !== confirmPassword.value
  ) {
    return
  }

  const success = await register({
    username: username.value,
    email: email.value,
    password: password.value,
  })

  if (success) {
    const target = typeof route.query.redirect === 'string' ? route.query.redirect : '/dashboard'
    router.push(target)
  }
}
</script>

<style scoped>
.register-page {
  min-height: 100vh;
  background: radial-gradient(
    circle at 50% 30%,
    rgba(var(--v-theme-primary), 0.08),
    transparent 70%
  );
}

.register-card {
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  backdrop-filter: blur(8px);
}
</style>
