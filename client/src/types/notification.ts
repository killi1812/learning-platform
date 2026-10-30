// Named AppNotification to avoid shadowing the DOM Notification type.
export interface AppNotification {
  uuid: string
  title: string
  body: string
  /** Internal path such as /courses/<uuid>. Empty string when the notification has no link. */
  url: string
  createdAt: string
  seenAt: string | null
}

export interface UnreadCount {
  count: number
}

export interface SendNotificationPayload {
  userUuids: string[]
  title: string
  body: string
  url?: string
}

export interface MarkAllSeenResult {
  updated: number
}

export function isAppNotification(value: unknown): value is AppNotification {
  if (typeof value !== 'object' || value === null) {
    return false
  }
  const v = value as Record<string, unknown>
  return (
    typeof v.uuid === 'string' &&
    typeof v.title === 'string' &&
    typeof v.body === 'string' &&
    typeof v.url === 'string' &&
    typeof v.createdAt === 'string' &&
    (v.seenAt === null || typeof v.seenAt === 'string')
  )
}
