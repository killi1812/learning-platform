import { ref } from 'vue'

export type ToastType = 'success' | 'error' | 'warning' | 'info'

export interface ToastItem {
  text: string
  color?: string
  prependIcon?: string
  timeout?: number
  closable?: boolean
}

const MAX_TOAST_QUEUE = 50
const queue = ref<ToastItem[]>([])

const ICON_MAP: Record<ToastType, string> = {
  success: 'mdi-check-circle-outline',
  error: 'mdi-alert-circle-outline',
  warning: 'mdi-alert-outline',
  info: 'mdi-information-outline',
}

function show(message: string, type: ToastType = 'info', timeout = 4000): void {
  if (queue.value.length >= MAX_TOAST_QUEUE) {
    queue.value.shift()
  }
  queue.value.push({
    text: message,
    color: type,
    prependIcon: ICON_MAP[type],
    timeout,
    closable: true,
  })
}

function success(message: string, timeout?: number): void {
  show(message, 'success', timeout)
}

function error(message: string, timeout?: number): void {
  show(message, 'error', timeout)
}

function warning(message: string, timeout?: number): void {
  show(message, 'warning', timeout)
}

function info(message: string, timeout?: number): void {
  show(message, 'info', timeout)
}

export function useToastQueue() {
  return { queue }
}

export function useToast() {
  return {
    show,
    success,
    error,
    warning,
    info,
  }
}
