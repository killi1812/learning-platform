const pad = (value: number): string => String(value).padStart(2, '0')

/** Formats an ISO timestamp as day.month.year, e.g. 23.10.26. */
export function formatShortDay(iso: string): string {
  const date = new Date(iso)
  return `${pad(date.getDate())}.${pad(date.getMonth() + 1)}.${pad(date.getFullYear() % 100)}`
}

/** Formats an ISO timestamp as 24-hour hh:mm, e.g. 14:05. */
export function formatShortTime(iso: string): string {
  const date = new Date(iso)
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`
}
