import { readonly, ref } from 'vue'

export interface ConfirmOptions {
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  confirmColor?: string
}

interface ConfirmState extends ConfirmOptions {
  isOpen: boolean
}

const state = ref<ConfirmState>({
  isOpen: false,
  message: '',
})

let pendingResolve: ((value: boolean) => void) | undefined

function settle(result: boolean): void {
  const resolve = pendingResolve
  pendingResolve = undefined
  state.value = {
    ...state.value,
    isOpen: false,
  }
  resolve?.(result)
}

export function useConfirm() {
  function confirm(options: ConfirmOptions | string): Promise<boolean> {
    const opts: ConfirmOptions = typeof options === 'string' ? { message: options } : options
    if (pendingResolve) {
      pendingResolve(false)
      pendingResolve = undefined
    }

    return new Promise(resolve => {
      pendingResolve = resolve
      state.value = {
        ...opts,
        isOpen: true,
      }
    })
  }

  function handleConfirm(): void {
    settle(true)
  }

  function handleCancel(): void {
    settle(false)
  }

  return {
    state: readonly(state),
    confirm,
    handleConfirm,
    handleCancel,
  }
}
