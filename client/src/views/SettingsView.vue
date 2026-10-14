<template>
  <v-container class="py-6">
    <h1 class="text-h4 font-weight-bold mb-2">Account Settings</h1>
    <p class="text-medium-emphasis mb-6">Manage profile and active authentication sessions</p>

    <!-- User Profile Card -->
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

      <div class="d-flex justify-end ga-3">
        <v-btn
          color="warning"
          prepend-icon="mdi-logout-variant"
          variant="outlined"
          :loading="isLoggingOutAll"
          @click="handleLogoutAll"
        >
          Sign Out of All Devices
        </v-btn>
        <v-btn
          color="error"
          prepend-icon="mdi-logout"
          variant="elevated"
          :loading="isLoggingOut"
          @click="handleLogout"
        >
          Sign Out This Device
        </v-btn>
      </div>
    </v-card>

    <!-- Active Sessions Card -->
    <v-card class="pa-6" elevation="2" rounded="lg">
      <div class="d-flex align-center justify-space-between mb-4">
        <div>
          <v-card-title class="text-h6 font-weight-bold px-0 pb-1">
            Active Devices & Sessions
          </v-card-title>
          <v-card-subtitle class="px-0">
            Connected devices with active access or refresh tokens
          </v-card-subtitle>
        </div>
        <v-btn
          icon="mdi-refresh"
          variant="text"
          density="comfortable"
          :loading="isLoadingSessions"
          @click="fetchSessions"
        />
      </div>

      <v-alert
        v-if="sessionError"
        class="mb-4"
        color="error"
        icon="mdi-alert-circle"
        variant="tonal"
        closable
        @click:close="sessionError = undefined"
      >
        {{ sessionError }}
      </v-alert>

      <v-list v-if="sessions.length > 0" lines="two" class="bg-transparent pa-0">
        <v-list-item
          v-for="session in sessions"
          :key="session.id"
          class="border rounded-lg mb-2 pa-3"
        >
          <template #prepend>
            <v-avatar color="primary-lighten-5" rounded="lg" class="mr-3">
              <v-icon color="primary" :icon="getDeviceIcon(session.deviceName)" />
            </v-avatar>
          </template>

          <v-list-item-title class="font-weight-bold d-flex align-center ga-2">
            {{ session.deviceName }}
            <v-chip
              v-if="session.isCurrent"
              color="success"
              size="x-small"
              label
              class="font-weight-medium"
            >
              Current Device
            </v-chip>
          </v-list-item-title>

          <v-list-item-subtitle class="text-caption mt-1">
            Signed in: {{ formatDate(session.createdAt) }} &bull; Expires:
            {{ formatDate(session.expiresAt) }}
          </v-list-item-subtitle>

          <template #append>
            <v-btn
              v-if="!session.isCurrent"
              color="error"
              variant="text"
              size="small"
              prepend-icon="mdi-delete-outline"
              :loading="revokingId === session.id"
              @click="handleRevoke(session.id)"
            >
              Revoke
            </v-btn>
          </template>
        </v-list-item>
      </v-list>

      <div v-else-if="isLoadingSessions" class="text-center py-6 text-medium-emphasis">
        <v-progress-circular indeterminate color="primary" size="24" class="mr-2" />
        Loading active sessions...
      </div>

      <div v-else class="text-center py-6 text-medium-emphasis">
        No other active sessions found.
      </div>
    </v-card>
  </v-container>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { listUserSessions, revokeUserSession, type SessionInfo } from '@/api/auth'

const router = useRouter()
const { currentUser, logout, logoutAll } = useAuth()

const sessions = ref<SessionInfo[]>([])
const isLoadingSessions = ref(false)
const sessionError = ref<string | undefined>(undefined)
const revokingId = ref<string | undefined>(undefined)
const isLoggingOut = ref(false)
const isLoggingOutAll = ref(false)

async function fetchSessions() {
  isLoadingSessions.value = true
  sessionError.value = undefined
  try {
    sessions.value = await listUserSessions()
  } catch (err: unknown) {
    const errorMsg =
      (err as { response?: { data?: { error?: string } } }).response?.data?.error ||
      'Failed to load active sessions'
    sessionError.value = errorMsg
  } finally {
    isLoadingSessions.value = false
  }
}

async function handleRevoke(sessionId: string) {
  revokingId.value = sessionId
  sessionError.value = undefined
  try {
    await revokeUserSession(sessionId)
    sessions.value = sessions.value.filter(s => s.id !== sessionId)
  } catch (err: unknown) {
    const errorMsg =
      (err as { response?: { data?: { error?: string } } }).response?.data?.error ||
      'Failed to revoke session'
    sessionError.value = errorMsg
  } finally {
    revokingId.value = undefined
  }
}

async function handleLogout() {
  isLoggingOut.value = true
  try {
    await logout()
    router.push('/login')
  } finally {
    isLoggingOut.value = false
  }
}

async function handleLogoutAll() {
  isLoggingOutAll.value = true
  try {
    await logoutAll()
    router.push('/login')
  } finally {
    isLoggingOutAll.value = false
  }
}

function getDeviceIcon(deviceName: string): string {
  const lower = deviceName.toLowerCase()
  if (lower.includes('ios') || lower.includes('android')) {
    return 'mdi-cellphone'
  }
  return 'mdi-laptop'
}

function formatDate(dateStr: string): string {
  try {
    const date = new Date(dateStr)
    return date.toLocaleString(undefined, {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  } catch {
    return dateStr
  }
}

onMounted(() => {
  fetchSessions()
})
</script>
