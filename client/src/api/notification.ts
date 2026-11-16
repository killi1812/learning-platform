import type { AppNotification, SendNotificationPayload } from '@/types/notification'
import { API_BASE_URL, authApi, getStoredAccessToken, request } from './client'

interface UnreadCount {
  count: number
}

interface MarkAllSeenResult {
  updated: number
}

function isAppNotification(value: unknown): value is AppNotification {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    return false
  }
  return (
    'uuid' in value &&
    typeof value.uuid === 'string' &&
    'title' in value &&
    typeof value.title === 'string' &&
    'body' in value &&
    typeof value.body === 'string' &&
    'url' in value &&
    typeof value.url === 'string' &&
    'createdAt' in value &&
    typeof value.createdAt === 'string' &&
    'seenAt' in value &&
    (value.seenAt === null || typeof value.seenAt === 'string')
  )
}

export function listNotifications(signal?: AbortSignal): Promise<AppNotification[]> {
  return request(authApi.get<AppNotification[]>('/notification', { signal }))
}

export async function getUnreadCount(signal?: AbortSignal): Promise<number> {
  const result = await request(authApi.get<UnreadCount>('/notification/unread-count', { signal }))
  return result.count
}

export function sendNotification(payload: SendNotificationPayload): Promise<AppNotification[]> {
  return request(authApi.post<AppNotification[]>('/notification/send', payload))
}

export function markNotificationSeen(uuid: string): Promise<AppNotification> {
  return request(authApi.post<AppNotification>(`/notification/${encodeURIComponent(uuid)}/seen`))
}

export function markAllNotificationsSeen(): Promise<MarkAllSeenResult> {
  return request(authApi.post<MarkAllSeenResult>('/notification/seen-all'))
}

export async function deleteNotification(uuid: string): Promise<void> {
  await request(authApi.delete(`/notification/${encodeURIComponent(uuid)}`))
}

/** Thrown by the stream opener so callers can tell 401 (refresh and retry) from other HTTP failures. */
export class NotificationStreamError extends Error {
  constructor(
    message: string,
    readonly status: number,
  ) {
    super(message)
    this.name = 'NotificationStreamError'
  }
}

/**
 * Opens the SSE stream with fetch, because EventSource cannot send the Authorization header.
 * Resolves when the server ends the stream (token expiry or server shutdown).
 * Rejects with NotificationStreamError on a non-2xx HTTP response, or a network/abort error on failure.
 */
export async function openNotificationStream(
  signal: AbortSignal,
  onNotification: (notification: AppNotification) => void,
): Promise<void> {
  const token = getStoredAccessToken()
  const response = await fetch(`${API_BASE_URL}/notification/stream`, {
    headers: {
      Accept: 'text/event-stream',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    credentials: 'include',
    signal,
  })

  if (!response.ok || !response.body) {
    void response.body?.cancel().catch(() => undefined)
    throw new NotificationStreamError('Notification stream rejected', response.status)
  }

  const reader = response.body.pipeThrough(new TextDecoderStream()).getReader()
  let buffer = ''
  let eventName = ''
  let dataLines: string[] = []

  const dispatch = () => {
    if ((eventName === 'notification' || eventName === '') && dataLines.length > 0) {
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

  const processLine = (raw: string) => {
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

  try {
    while (true) {
      const { value, done } = await reader.read()
      if (done) {
        if (buffer.length > 0) {
          processLine(buffer)
        }
        dispatch()
        break
      }

      buffer += value
      const lines = buffer.split('\n')
      buffer = lines.pop() ?? ''
      for (const line of lines) {
        processLine(line)
      }
    }
  } finally {
    await reader.cancel().catch(() => undefined)
    reader.releaseLock()
  }
}
