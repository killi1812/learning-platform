const pad = (value: number): string => String(value).padStart(2, '0')

export function parseValidDate(iso?: string): Date | undefined {
  if (!iso) {
    return undefined
  }
  const date = new Date(iso)
  return Number.isNaN(date.getTime()) ? undefined : date
}

/** Formats an ISO timestamp as day.month.year, e.g. 23.10.26. */
export function formatShortDay(iso: string): string {
  const date = parseValidDate(iso)
  if (!date) {
    return ''
  }
  return `${pad(date.getDate())}.${pad(date.getMonth() + 1)}.${pad(date.getFullYear() % 100)}`
}

/** Formats an ISO timestamp as 24-hour hh:mm, e.g. 14:05. */
export function formatShortTime(iso: string): string {
  const date = parseValidDate(iso)
  if (!date) {
    return ''
  }
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`
}

/** Formats an ISO timestamp as localized short date and time. */
export function formatDateTime(iso?: string, fallback = ''): string {
  const date = parseValidDate(iso)
  if (!date) {
    return fallback
  }
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/** Formats an ISO timestamp as localized date (year, short month, numeric day). */
export function formatShortDate(iso?: string, fallback = ''): string {
  const date = parseValidDate(iso)
  if (!date) {
    return fallback
  }
  return date.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

/** Formats an ISO timestamp as numeric day + short month, e.g. 15 Nov. */
export function formatMonthDay(iso?: string, fallback = ''): string {
  const date = parseValidDate(iso)
  if (!date) {
    return fallback
  }
  return date.toLocaleDateString(undefined, {
    day: 'numeric',
    month: 'short',
  })
}

/** Converts an ISO timestamp to `YYYY-MM-DDTHH:mm` in local timezone for `<input type="datetime-local">`. */
export function toLocalDateTimeInput(iso?: string): string {
  const date = parseValidDate(iso)
  if (!date) {
    return ''
  }
  const offsetMs = date.getTimezoneOffset() * 60_000
  return new Date(date.getTime() - offsetMs).toISOString().slice(0, 16)
}

/** Converts a local `YYYY-MM-DDTHH:mm` input value to an ISO UTC string. */
export function fromLocalDateTimeInput(value?: string): string | undefined {
  if (!value) {
    return undefined
  }
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? undefined : date.toISOString()
}
