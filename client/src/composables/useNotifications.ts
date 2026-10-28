import { ref } from 'vue'
import {
  deleteNotification,
  getUnreadCount,
  listNotifications,
  markAllNotificationsSeen,
  markNotificationSeen,
  NotificationStreamError,
  openNotificationStream,
  refreshAccessToken,
} from '@/api/notification'
import type { AppNotification } from '@/types/notification'

const RECONNECT_MIN_MS = 1000
const RECONNECT_MAX_MS = 30000

// Module-level state: one inbox per app, shared by every consumer.
const items = ref<AppNotification[]>([])
const unreadCount = ref(0)
let activeController: AbortController | null = null

function sleep(ms: number, signal: AbortSignal): Promise<void> {
  return new Promise(resolve => {
    const onDone = () => {
      clearTimeout(timer)
      signal.removeEventListener('abort', onDone)
      resolve()
    }
    const timer = setTimeout(onDone, ms)
    signal.addEventListener('abort', onDone)
  })
}

async function reload(): Promise<void> {
  const [list, count] = await Promise.all([listNotifications(), getUnreadCount()])
  items.value = list
  unreadCount.value = count
}

async function syncUnreadCount(): Promise<void> {
  unreadCount.value = await getUnreadCount()
}

function applyIncoming(notification: AppNotification): void {
  if (items.value.some(item => item.uuid === notification.uuid)) {
    return
  }
  items.value = [notification, ...items.value]
  if (!notification.seenAt) {
    unreadCount.value += 1
  }
}

async function streamLoop(signal: AbortSignal): Promise<void> {
  let delay = RECONNECT_MIN_MS

  while (!signal.aborted) {
    try {
      await openNotificationStream(signal, applyIncoming)
      // Clean close: server ends the stream when the access token expires. Reconnect soon.
      delay = RECONNECT_MIN_MS
    } catch (err: unknown) {
      if (signal.aborted) return
      if (err instanceof NotificationStreamError && err.status === 401) {
        // Token expired while disconnected. Refresh once; stop if the session is gone.
        if (!(await refreshAccessToken())) return
        delay = 0
      } else {
        delay = Math.min(delay * 2, RECONNECT_MAX_MS)
      }
    }

    await sleep(delay, signal)
    if (signal.aborted) return

    // Events sent while disconnected are not replayed. Refetch the inbox to recover them.
    await reload().catch(() => undefined)
  }
}

/** Starts the inbox load and the live stream. Safe to call more than once. */
export function startNotificationSync(): void {
  if (activeController) return

  activeController = new AbortController()
  const { signal } = activeController

  reload().catch(() => undefined)
  streamLoop(signal).catch(() => undefined)
}

/** Stops the live stream and clears local state. Call on unmount and logout. */
export function stopNotificationSync(): void {
  activeController?.abort()
  activeController = null
  items.value = []
  unreadCount.value = 0
}

export function useNotifications() {
  async function markSeen(notification: AppNotification): Promise<void> {
    if (notification.seenAt) return

    const updated = await markNotificationSeen(notification.uuid)
    items.value = items.value.map(item => (item.uuid === updated.uuid ? updated : item))
    await syncUnreadCount()
  }

  async function markAllSeen(): Promise<void> {
    await markAllNotificationsSeen()
    await reload()
  }

  async function remove(notification: AppNotification): Promise<void> {
    await deleteNotification(notification.uuid)
    items.value = items.value.filter(item => item.uuid !== notification.uuid)
    await syncUnreadCount()
  }

  return {
    items,
    unreadCount,
    markSeen,
    markAllSeen,
    remove,
  }
}
