<template>
  <div class="auth-page d-flex align-center justify-center">
    <v-container max-width="500">
      <v-card class="register-card pa-4 pa-sm-6" elevation="8" rounded="xl">
        <!-- Brand Header -->
        <div class="text-center mb-6">
          <v-avatar class="mb-3 elevation-4" color="primary" size="64">
            <v-icon icon="mdi-account-plus" size="36" />
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
          @click:close="clearAuthError"
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
            :rules="[rules.required, rules.minUsername, rules.validUsername]"
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
            :append-inner-icon="showPassword ? 'mdi-eye-off-outline' : 'mdi-eye-outline'"
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
          <router-link class="text-primary font-weight-bold text-decoration-none ml-1" to="/login">
            Sign In
          </router-link>
        </div>
      </v-card>
    </v-container>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { getSafeRedirect } from '@/util/redirect'
import { EMAIL_PATTERN } from '@/util/validation'

const router = useRouter()
const route = useRoute()
const { register, isSubmitting, authError, clearAuthError } = useAuth()

const formRef = ref<{ validate: () => Promise<{ valid: boolean }> } | undefined>(undefined)
const username = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const showPassword = ref(false)

const rules = {
  required: (v: string) => !!v.trim() || 'This field is required',
  minUsername: (v: string) =>
    (v.length >= 3 && v.length <= 30) || 'Username must be between 3 and 30 characters',
  validUsername: (v: string) =>
    /^[a-z0-9]+$/i.test(v) || 'Username must be alphanumeric (no spaces or symbols)',
  email: (v: string) => EMAIL_PATTERN.test(v.trim()) || 'Please enter a valid email address',
  minPassword: (v: string) => v.length >= 8 || 'Password must be at least 8 characters',
  matchPassword: (v: string) => v === password.value || 'Passwords do not match',
}

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

  const success = await register({
    username: username.value.trim(),
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
.register-card {
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  backdrop-filter: blur(8px);
}
</style>
