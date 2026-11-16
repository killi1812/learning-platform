// Named AppNotification to avoid shadowing the DOM Notification type.
export interface AppNotification {
  uuid: string
  title: string
  body: string
  /** Internal path such as /courses/<uuid>. Empty string when the notification has no link. */
  url: string
  createdAt: string
  /** Nullable in API JSON payload when unread. */
  seenAt: string | null
}

export interface SendNotificationPayload {
  userUuids: string[]
  title: string
  body: string
  url?: string
}
