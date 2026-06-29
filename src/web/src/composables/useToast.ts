import { ref } from 'vue'

export type ToastKind = 'info' | 'success' | 'error'

export interface ToastMessage {
  id: number
  message: string
  kind: ToastKind
}

const toasts = ref<ToastMessage[]>([])
let nextToastId = 1

function removeToast(id: number) {
  toasts.value = toasts.value.filter((toast) => toast.id !== id)
}

function showToast(message: string, kind: ToastKind = 'info', timeoutMs = 4_000) {
  const id = nextToastId++
  toasts.value.push({ id, message, kind })
  window.setTimeout(() => removeToast(id), timeoutMs)
  return id
}

export function useToast() {
  return {
    toasts,
    showToast,
    removeToast,
  }
}
