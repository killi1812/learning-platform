import { ref } from 'vue'

export interface ConfirmOptions {
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  confirmColor?: string
}

interface ConfirmState extends ConfirmOptions {
  isOpen: boolean
  resolve?: (value: boolean) => void
}

const state = ref<ConfirmState>({
  isOpen: false,
  message: '',
})

export function useConfirm() {
  function confirm(options: ConfirmOptions | string): Promise<boolean> {
    const opts: ConfirmOptions = typeof options === 'string' ? { message: options } : options

    return new Promise(resolve => {
      state.value = {
        ...opts,
        isOpen: true,
        resolve,
      }
    })
  }

  function handleConfirm() {
    state.value.isOpen = false
    state.value.resolve?.(true)
  }

  function handleCancel() {
    state.value.isOpen = false
    state.value.resolve?.(false)
  }

  return {
    state,
    confirm,
    handleConfirm,
    handleCancel,
  }
}
