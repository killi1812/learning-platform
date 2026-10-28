import { API_BASE_URL, authApi, executeRefreshToken, getApiErrorMessage } from './client'
import {
  isAppNotification,
  type AppNotification,
  type MarkAllSeenResult,
  type SendNotificationPayload,
  type UnreadCount,
} from '@/types/notification'

async function wrapRequest<T>(promise: Promise<{ data: T }>): Promise<T> {
  try {
    const response = await promise
    return response.data
  } catch (err: unknown) {
    throw new Error(getApiErrorMessage(err))
  }
}

export async function listNotifications(): Promise<AppNotification[]> {
  return wrapRequest(authApi.get<AppNotification[]>('/notification'))
}

export async function getUnreadCount(): Promise<number> {
  const result = await wrapRequest(authApi.get<UnreadCount>('/notification/unread-count'))
  return result.count
}

export async function sendNotification(
  payload: SendNotificationPayload,
): Promise<AppNotification[]> {
  return wrapRequest(authApi.post<AppNotification[]>('/notification/send', payload))
}

export async function markNotificationSeen(uuid: string): Promise<AppNotification> {
  return wrapRequest(authApi.post<AppNotification>(`/notification/${uuid}/seen`))
}

export async function markAllNotificationsSeen(): Promise<MarkAllSeenResult> {
  return wrapRequest(authApi.post<MarkAllSeenResult>('/notification/seen-all'))
}

export async function deleteNotification(uuid: string): Promise<void> {
  await wrapRequest(authApi.delete(`/notification/${uuid}`))
}

/** Thrown by the stream opener so callers can tell 401 (refresh and retry) from other failures. */
export class NotificationStreamError extends Error {
  constructor(
    message: string,
    readonly status: number,
  ) {
    super(message)
  }
}

/**
 * Opens the SSE stream with fetch, because EventSource cannot send the Authorization header.
 * Resolves when the server ends the stream (token expiry or server shutdown).
 * Rejects with NotificationStreamError on a non-2xx response or a network failure.
 */
export async function openNotificationStream(
  signal: AbortSignal,
  onNotification: (notification: AppNotification) => void,
): Promise<void> {
  const token = localStorage.getItem('accessToken')
  const response = await fetch(`${API_BASE_URL}/notification/stream`, {
    headers: {
      Accept: 'text/event-stream',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    credentials: 'include',
    signal,
  })

  if (!response.ok || !response.body) {
    throw new NotificationStreamError('Notification stream rejected', response.status)
  }

  const reader = response.body.pipeThrough(new TextDecoderStream()).getReader()
  let buffer = ''
  let eventName = ''
  let dataLines: string[] = []

  const dispatch = () => {
    if (eventName === 'notification' && dataLines.length > 0) {
      try {
        const parsed: unknown = JSON.parse(dataLines.join('\n'))
        if (isAppNotification(parsed)) {
          onNotification(parsed)
        }
      } catch {
        // Malformed payload: skip this event, keep the stream open.
      }
    }
    eventName = ''
    dataLines = []
  }

  while (true) {
    const { value, done } = await reader.read()
    if (done) break

    buffer += value
    const lines = buffer.split('\n')
    buffer = lines.pop() ?? ''

    for (const raw of lines) {
      const line = raw.endsWith('\r') ? raw.slice(0, -1) : raw
      if (line === '') {
        dispatch()
      } else if (line.startsWith(':')) {
        // Comment line (heartbeat). Ignore.
      } else if (line.startsWith('event:')) {
        eventName = line.slice('event:'.length).trim()
      } else if (line.startsWith('data:')) {
        dataLines.push(line.slice('data:'.length).replace(/^ /, ''))
      }
    }
  }
}

/** Refreshes the access token through the shared deduplicated refresh flow. */
export async function refreshAccessToken(): Promise<boolean> {
  try {
    await executeRefreshToken()
    return true
  } catch {
    return false
  }
}
