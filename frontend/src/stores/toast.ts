import { defineStore } from 'pinia'
import { ref } from 'vue'

// 移植自原项目 ToastContext.jsx：Brutalist toast 通知
export interface ToastItem {
  id: number
  message: string
  type: 'info' | 'success' | 'error'
}

let toastId = 0

export const useToastStore = defineStore('toast', () => {
  const toasts = ref<ToastItem[]>([])

  function addToast(message: string, type: ToastItem['type'] = 'info', duration = 3500) {
    const id = ++toastId
    toasts.value.push({ id, message, type })
    if (duration > 0) {
      setTimeout(() => remove(id), duration)
    }
    return id
  }

  function remove(id: number) {
    toasts.value = toasts.value.filter((t) => t.id !== id)
  }

  function success(msg: string) {
    return addToast(msg, 'success', 2500)
  }
  function error(msg: string) {
    return addToast(msg, 'error')
  }
  function info(msg: string) {
    return addToast(msg, 'info')
  }

  return { toasts, addToast, remove, success, error, info }
})
