import { ref } from 'vue'

export type ToastType = 'success' | 'error' | 'warning' | 'info'

export interface ToastItem {
  text: string
  color?: string
  prependIcon?: string
  timeout?: number
  closable?: boolean
}

const queue = ref<ToastItem[]>([])

const ICON_MAP: Record<ToastType, string> = {
  success: 'mdi-check-circle-outline',
  error: 'mdi-alert-circle-outline',
  warning: 'mdi-alert-outline',
  info: 'mdi-information-outline',
}

export function useToast() {
  function show(message: string, type: ToastType = 'info', timeout = 4000) {
    queue.value.push({
      text: message,
      color: type,
      prependIcon: ICON_MAP[type],
      timeout,
      closable: true,
    })
  }

  function success(message: string, timeout?: number) {
    show(message, 'success', timeout)
  }

  function error(message: string, timeout?: number) {
    show(message, 'error', timeout)
  }

  function warning(message: string, timeout?: number) {
    show(message, 'warning', timeout)
  }

  function info(message: string, timeout?: number) {
    show(message, 'info', timeout)
  }

  function clear() {
    queue.value = []
  }

  return {
    queue,
    show,
    success,
    error,
    warning,
    info,
    clear,
  }
}
