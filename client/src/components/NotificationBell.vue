<template>
  <div class="notification-bell mr-2">
    <v-menu location="bottom end" max-width="95vw" width="380">
      <template #activator="{ props: menuProps }">
        <v-btn v-bind="menuProps" aria-label="Notifications" icon>
          <v-icon icon="mdi-bell" />
        </v-btn>
      </template>

      <v-card>
        <v-card-title class="d-flex align-center justify-space-between py-2">
          <v-btn
            class="px-1 text-subtitle-1 font-weight-medium text-none"
            variant="text"
            @click="onOpenSend"
          >
            Notifications
          </v-btn>
          <v-btn
            :disabled="unreadCount === 0"
            size="small"
            variant="text"
            @click.stop="onMarkAllSeen"
          >
            Mark all read
          </v-btn>
        </v-card-title>
        <v-divider />

        <v-list v-if="items.length > 0" class="py-0" density="compact" max-height="420">
          <v-list-item
            v-for="item in items"
            :key="item.uuid"
            :active="!item.seenAt"
            class="notification-item"
            @click="onOpen(item)"
          >
            <v-list-item-title :class="['text-wrap', { 'font-weight-bold': !item.seenAt }]">
              {{ item.title }}
            </v-list-item-title>
            <v-list-item-subtitle class="text-wrap">{{ item.body }}</v-list-item-subtitle>
            <template #append>
              <div class="d-flex align-center ga-1">
                <div class="stamp text-caption text-medium-emphasis text-end">
                  <div class="stamp-date">{{ formatShortDay(item.createdAt) }}</div>
                  <div>{{ formatShortTime(item.createdAt) }}</div>
                </div>
                <v-btn
                  aria-label="Delete notification"
                  icon="mdi-delete-outline"
                  size="small"
                  variant="text"
                  @click.stop="onRemove(item)"
                />
              </div>
            </template>
          </v-list-item>
        </v-list>

        <v-card-text v-else class="text-medium-emphasis text-center">No notifications</v-card-text>
      </v-card>
    </v-menu>

    <!-- Outside the v-btn: v-btn clips overflow, which would cut off multi-digit counts. -->
    <span v-if="unreadCount > 0" class="notification-count">{{ unreadCount }}</span>
  </div>
</template>

<script setup lang="ts">
import type { AppNotification } from '@/types/notification'
import { onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { getApiErrorMessage } from '@/api/client'
import {
  startNotificationSync,
  stopNotificationSync,
  useNotifications,
} from '@/composables/useNotifications'
import { useToast } from '@/composables/useToast'
import { formatShortDay, formatShortTime } from '@/util/dateTime'
import { isInternalPath } from '@/util/url'

const { items, unreadCount, markSeen, markAllSeen, remove } = useNotifications()
const toast = useToast()
const router = useRouter()

// The bell is mounted only while authenticated (AppHeader v-if), so the stream follows the session.
onMounted(() => startNotificationSync())
onUnmounted(() => stopNotificationSync())

async function run(action: () => Promise<void>): Promise<void> {
  try {
    await action()
  } catch (error: unknown) {
    toast.error(getApiErrorMessage(error, 'Something went wrong'))
  }
}

function onOpen(item: AppNotification): Promise<void> {
  return run(async () => {
    await markSeen(item)
    if (item.url && isInternalPath(item.url)) {
      await router.push(item.url)
    }
  })
}

function onOpenSend(): Promise<void> {
  return run(async () => {
    await router.push('/notification/send')
  })
}

function onMarkAllSeen(): Promise<void> {
  return run(() => markAllSeen())
}

function onRemove(item: AppNotification): Promise<void> {
  return run(() => remove(item))
}
</script>

<style scoped>
/* Vuetify truncates titles and subtitles (nowrap, ellipsis, line clamp). Undo that so the full text shows. */
.notification-item :deep(.v-list-item__content) {
  min-width: 0;
}

.notification-item :deep(.v-list-item-title),
.notification-item :deep(.v-list-item-subtitle) {
  display: block;
  white-space: normal;
  overflow-wrap: anywhere;
  overflow: visible;
  text-overflow: clip;
  -webkit-line-clamp: unset;
}

.stamp {
  line-height: 1.2;
  white-space: nowrap;
}

/* Date is wider than the time in the same font, so it is scaled down to roughly match the time's width. */
.stamp-date {
  font-size: 0.65em;
}

.notification-bell {
  position: relative;
  display: inline-flex;
}

.notification-count {
  position: absolute;
  top: 2px;
  right: 2px;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 9px;
  background: rgb(var(--v-theme-error));
  color: rgb(var(--v-theme-on-error));
  font-size: 11px;
  font-weight: 600;
  line-height: 18px;
  text-align: center;
  pointer-events: none;
}
</style>
