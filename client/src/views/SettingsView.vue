<template>
  <v-container class="py-6">
    <h1 class="text-h4 font-weight-bold mb-2">Account Settings</h1>
    <p class="text-medium-emphasis mb-6">Manage profile and active authentication sessions</p>

    <v-alert
      v-if="sessionError"
      class="mb-4"
      closable
      color="error"
      icon="mdi-alert-circle"
      variant="tonal"
      @click:close="sessionError = undefined"
    >
      {{ sessionError }}
    </v-alert>

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
            :model-value="currentUser?.sub || currentUser?.jti || currentUser?.userUuid || 'N/A'"
            readonly
            variant="outlined"
          />
        </v-col>
      </v-row>

      <v-divider class="my-4" />

      <div class="d-flex justify-end ga-3">
        <v-btn
          color="warning"
          :loading="isLoggingOutAll"
          prepend-icon="mdi-logout-variant"
          variant="outlined"
          @click="handleLogoutAll"
        >
          Sign Out of All Devices
        </v-btn>
        <v-btn
          color="error"
          :loading="isLoggingOut"
          prepend-icon="mdi-logout"
          variant="elevated"
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
            Active Devices &amp; Sessions
          </v-card-title>
          <v-card-subtitle class="px-0">
            Connected devices with active access or refresh tokens
          </v-card-subtitle>
        </div>
        <v-btn
          aria-label="Refresh active sessions"
          density="comfortable"
          icon="mdi-refresh"
          :loading="isLoadingSessions"
          variant="text"
          @click="fetchSessions"
        />
      </div>

      <v-list v-if="sessions.length > 0" class="bg-transparent pa-0" lines="two">
        <v-list-item
          v-for="session in sessions"
          :key="session.id"
          class="border rounded-lg mb-2 pa-3"
        >
          <template #prepend>
            <v-avatar class="mr-3" color="primary" rounded="lg" variant="tonal">
              <v-icon color="primary" :icon="getDeviceIcon(session.deviceName)" />
            </v-avatar>
          </template>

          <v-list-item-title class="font-weight-bold d-flex align-center ga-2">
            {{ session.deviceName }}
            <v-chip
              v-if="session.isCurrent"
              class="font-weight-medium"
              color="success"
              label
              size="x-small"
            >
              Current Device
            </v-chip>
          </v-list-item-title>

          <v-list-item-subtitle class="text-caption mt-1">
            Signed in: {{ formatDateTime(session.createdAt) }} &bull; Expires:
            {{ formatDateTime(session.expiresAt) }}
          </v-list-item-subtitle>

          <template #append>
            <v-btn
              v-if="!session.isCurrent"
              color="error"
              :loading="revokingId === session.id"
              prepend-icon="mdi-delete-outline"
              size="small"
              variant="text"
              @click="handleRevoke(session.id)"
            >
              Revoke
            </v-btn>
          </template>
        </v-list-item>
      </v-list>

      <div v-else-if="isLoadingSessions" class="text-center py-6 text-medium-emphasis">
        <v-progress-circular class="mr-2" color="primary" indeterminate size="24" />
        Loading active sessions...
      </div>

      <div v-else class="text-center py-6 text-medium-emphasis">
        No other active sessions found.
      </div>
    </v-card>
  </v-container>
</template>

<script setup lang="ts">
import type { SessionInfo } from '@/types/auth'
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { listUserSessions, revokeUserSession } from '@/api/auth'
import { getApiErrorMessage } from '@/api/client'
import { useAuth } from '@/composables/useAuth'
import { useConfirm } from '@/composables/useConfirm'
import { formatDateTime } from '@/util/dateTime'

const router = useRouter()
const { currentUser, logout, logoutAll } = useAuth()
const { confirm } = useConfirm()

const sessions = ref<SessionInfo[]>([])
const isLoadingSessions = ref(false)
const sessionError = ref<string | undefined>(undefined)
const revokingId = ref<string | undefined>(undefined)
const isLoggingOut = ref(false)
const isLoggingOutAll = ref(false)

async function fetchSessions(): Promise<void> {
  isLoadingSessions.value = true
  sessionError.value = undefined
  try {
    sessions.value = await listUserSessions()
  } catch (error: unknown) {
    sessionError.value = getApiErrorMessage(error, 'Failed to load active sessions')
  } finally {
    isLoadingSessions.value = false
  }
}

async function handleRevoke(sessionId: string): Promise<void> {
  revokingId.value = sessionId
  sessionError.value = undefined
  try {
    await revokeUserSession(sessionId)
    sessions.value = sessions.value.filter(s => s.id !== sessionId)
  } catch (error: unknown) {
    sessionError.value = getApiErrorMessage(error, 'Failed to revoke session')
  } finally {
    revokingId.value = undefined
  }
}

async function handleLogout(): Promise<void> {
  isLoggingOut.value = true
  try {
    await logout()
    void router.push('/login')
  } finally {
    isLoggingOut.value = false
  }
}

async function handleLogoutAll(): Promise<void> {
  const ok = await confirm({
    title: 'Sign out of all devices',
    message:
      'Are you sure you want to sign out of all active devices? You will need to sign in again everywhere.',
    confirmText: 'Sign Out All',
    confirmColor: 'warning',
  })
  if (!ok) {
    return
  }

  isLoggingOutAll.value = true
  sessionError.value = undefined
  try {
    await logoutAll()
    void router.push('/login')
  } catch (error: unknown) {
    sessionError.value = getApiErrorMessage(error, 'Failed to sign out of all devices')
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

onMounted(() => {
  void fetchSessions()
})
</script>
