import type { AppNotification } from '@/types/notification'
import { readonly, ref } from 'vue'
import { executeRefreshToken } from '@/api/client'
import {
  deleteNotification,
  getUnreadCount,
  listNotifications,
  markAllNotificationsSeen,
  markNotificationSeen,
  NotificationStreamError,
  openNotificationStream,
} from '@/api/notification'

const RECONNECT_MIN_MS = 1000
const RECONNECT_MAX_MS = 30_000

// Module-level state: one inbox per app, shared by every consumer.
const items = ref<AppNotification[]>([])
const unreadCount = ref(0)
const isLoading = ref(false)
const syncError = ref<string | undefined>(undefined)

let activeController: AbortController | undefined
let activeConsumers = 0
let reloadSeq = 0

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

async function reload(signal?: AbortSignal): Promise<void> {
  const seq = ++reloadSeq
  isLoading.value = true
  try {
    const [list, count] = await Promise.all([listNotifications(signal), getUnreadCount(signal)])
    if (signal?.aborted || seq !== reloadSeq) {
      return
    }
    items.value = list
    unreadCount.value = count
    syncError.value = undefined
  } catch (error: unknown) {
    if (signal?.aborted) {
      return
    }
    syncError.value = error instanceof Error ? error.message : 'Failed to load notifications.'
    throw error
  } finally {
    if (seq === reloadSeq) {
      isLoading.value = false
    }
  }
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

  try {
    while (!signal.aborted) {
      try {
        await openNotificationStream(signal, applyIncoming)
        // Clean close: server ends the stream when the access token expires. Reconnect soon.
        delay = RECONNECT_MIN_MS
      } catch (error: unknown) {
        if (signal.aborted) {
          return
        }
        if (error instanceof NotificationStreamError && error.status === 401) {
          try {
            await executeRefreshToken()
            delay = 0
          } catch {
            return
          }
        } else {
          delay = Math.min(delay * 2, RECONNECT_MAX_MS)
        }
      }

      await sleep(delay, signal)
      if (signal.aborted) {
        return
      }

      // Events sent while disconnected are not replayed. Refetch the inbox to recover them.
      await reload(signal).catch(() => undefined)
    }
  } finally {
    if (activeController?.signal === signal) {
      activeController = undefined
    }
  }
}

/** Starts the inbox load and the live stream. Ref-counted across mounted consumers. */
export function startNotificationSync(): void {
  activeConsumers += 1
  if (activeController) {
    return
  }

  activeController = new AbortController()
  const { signal } = activeController

  void reload(signal).catch(() => undefined)
  void streamLoop(signal).catch(() => undefined)
}

/** Stops live sync when the last consumer unmounts, clearing local inbox state. */
export function stopNotificationSync(): void {
  activeConsumers = Math.max(0, activeConsumers - 1)
  if (activeConsumers > 0) {
    return
  }

  resetNotificationSync()
}

/** Immediately aborts the notification stream and clears local state (used on logout). */
export function resetNotificationSync(): void {
  activeConsumers = 0
  activeController?.abort()
  activeController = undefined
  reloadSeq += 1
  items.value = []
  unreadCount.value = 0
  isLoading.value = false
  syncError.value = undefined
}

export function useNotifications() {
  async function markSeen(notification: AppNotification): Promise<void> {
    if (notification.seenAt) {
      return
    }

    const updated = await markNotificationSeen(notification.uuid)
    items.value = items.value.map(item => (item.uuid === updated.uuid ? updated : item))
    await syncUnreadCount()
  }

  async function markAllSeen(): Promise<void> {
    await markAllNotificationsSeen()
    await reload(activeController?.signal)
  }

  async function remove(notification: AppNotification): Promise<void> {
    await deleteNotification(notification.uuid)
    items.value = items.value.filter(item => item.uuid !== notification.uuid)
    await syncUnreadCount()
  }

  return {
    items: readonly(items),
    unreadCount: readonly(unreadCount),
    isLoading: readonly(isLoading),
    syncError: readonly(syncError),
    markSeen,
    markAllSeen,
    remove,
  }
}
