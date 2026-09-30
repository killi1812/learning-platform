<template>
  <v-container class="py-8" max-width="600">
    <!-- Backend Connection Status -->
    <v-card class="mb-6" elevation="2" rounded="lg">
      <v-card-item>
        <template #prepend>
          <v-icon
            :color="backendOnline ? 'success' : checkingStatus ? 'warning' : 'error'"
            :icon="
              backendOnline
                ? 'mdi-server-network'
                : checkingStatus
                  ? 'mdi-loading mdi-spin'
                  : 'mdi-server-network-off'
            "
            size="large"
          />
        </template>
        <v-card-title class="text-h6">Backend Connection Status</v-card-title>
        <v-card-subtitle>
          {{
            checkingStatus
              ? 'Checking backend connection...'
              : backendOnline
                ? 'Connected to Spring Boot API'
                : 'Unable to connect to backend'
          }}
        </v-card-subtitle>

        <template #append>
          <v-chip
            :color="backendOnline ? 'success' : checkingStatus ? 'warning' : 'error'"
            label
            size="small"
          >
            {{ backendOnline ? 'ONLINE' : checkingStatus ? 'CHECKING' : 'OFFLINE' }}
          </v-chip>
          <v-btn
            class="ml-2"
            density="compact"
            icon="mdi-refresh"
            :loading="checkingStatus"
            variant="text"
            @click="checkBackendStatus"
          />
        </template>
      </v-card-item>
    </v-card>

    <!-- Simple Authentication Test -->
    <v-card elevation="3" rounded="lg">
      <v-card-title class="text-h5 pt-6 px-6 font-weight-bold"> Backend Auth Test </v-card-title>
      <v-card-subtitle class="px-6">
        Test communication between Vue/Vuetify and Spring Boot.
      </v-card-subtitle>

      <v-card-text class="pt-4 px-6">
        <!-- Auth Response Alert -->
        <v-alert
          v-if="authResult"
          class="mb-4"
          closable
          :color="authResult.success ? 'success' : 'error'"
          :icon="authResult.success ? 'mdi-check-circle' : 'mdi-alert-circle'"
          :text="authResult.message"
          variant="tonal"
          @click:close="authResult = null"
        />

        <v-form @submit.prevent="handleLogin">
          <v-text-field
            v-model="username"
            autocomplete="username"
            label="Username"
            name="username"
            prepend-inner-icon="mdi-account"
            required
            variant="outlined"
          />

          <v-text-field
            v-model="password"
            autocomplete="current-password"
            label="Password"
            name="password"
            prepend-inner-icon="mdi-lock"
            required
            type="password"
            variant="outlined"
          />

          <div class="text-caption text-grey mb-4">
            Default credentials: <v-kbd>admin</v-kbd> / <v-kbd>password</v-kbd> or
            <v-kbd>user</v-kbd> / <v-kbd>password</v-kbd>
          </div>

          <v-btn block color="primary" :loading="loading" size="large" type="submit">
            Test Authentication
          </v-btn>
        </v-form>

        <v-divider v-if="authenticatedUser" class="my-4" />

        <div v-if="authenticatedUser" class="d-flex align-center justify-space-between">
          <div>
            <span class="text-caption text-grey">Authenticated as:</span>
            <div class="text-subtitle-1 font-weight-bold text-success">
              {{ authenticatedUser }}
            </div>
          </div>
          <v-btn color="grey" size="small" variant="outlined" @click="logout"> Reset </v-btn>
        </div>
      </v-card-text>
    </v-card>
  </v-container>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

const username = ref('admin')
const password = ref('password')
const loading = ref(false)
const checkingStatus = ref(false)
const backendOnline = ref(false)
const authenticatedUser = ref<string | null>(null)
const authResult = ref<{ success: boolean; message: string } | null>(null)

async function checkBackendStatus() {
  checkingStatus.value = true
  try {
    const res = await fetch('/api/auth/status')
    if (res.ok) {
      backendOnline.value = true
    } else {
      backendOnline.value = false
    }
  } catch (err) {
    backendOnline.value = false
  } finally {
    checkingStatus.value = false
  }
}

async function handleLogin() {
  loading.value = true
  authResult.value = null

  try {
    const res = await fetch('/api/auth/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        username: username.value,
        password: password.value,
      }),
    })

    const data = await res.json()

    if (res.ok && data.authenticated) {
      authenticatedUser.value = data.username
      authResult.value = {
        success: true,
        message: data.message || 'Authentication successful! Backend connected.',
      }
      backendOnline.value = true
    } else {
      authenticatedUser.value = null
      authResult.value = {
        success: false,
        message: data.message || 'Authentication failed: Invalid credentials.',
      }
    }
  } catch (err) {
    authenticatedUser.value = null
    authResult.value = {
      success: false,
      message: 'Failed to reach backend at /api/auth/login. Is backend running?',
    }
    backendOnline.value = false
  } finally {
    loading.value = false
  }
}

function logout() {
  authenticatedUser.value = null
  authResult.value = null
}

onMounted(() => {
  checkBackendStatus()
})
</script>
