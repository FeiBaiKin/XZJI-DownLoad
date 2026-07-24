import { defineStore } from 'pinia'
import type { MessageApi } from 'naive-ui'

type ToastType = 'info' | 'success' | 'error'

interface PendingToast {
  message: string
  type: ToastType
}

let messageApi: MessageApi | null = null
const pendingToasts: PendingToast[] = []

function showToast(toast: PendingToast) {
  if (!messageApi) {
    // API 还未完成桥接时先排队，避免应用启动阶段丢失错误提示。
    pendingToasts.push(toast)
    return
  }

  if (toast.type === 'error') {
    messageApi.error(toast.message, { duration: 4000 })
  } else if (toast.type === 'success') {
    messageApi.success(toast.message)
  } else {
    messageApi.info(toast.message)
  }
}

/** 由 NMessageProvider 子树中的 ToastBridge 调用。 */
export function registerToastApi(api: MessageApi) {
  messageApi = api
  while (pendingToasts.length > 0) {
    const toast = pendingToasts.shift()
    if (toast) showToast(toast)
  }
}

/** ToastBridge 卸载时清理引用，避免热更新后继续使用失效的 provider。 */
export function unregisterToastApi(api: MessageApi) {
  if (messageApi === api) messageApi = null
}

// 全局消息通知：对外保持 success/error/info API（历史沿用「toast」命名）。
// 不再在 Pinia store setup 阶段调用 createDiscreteApi，避免 Naive UI 内部
// useMessage 在根应用挂载过程中找不到对应的 NMessageProvider。
export const useToastStore = defineStore('toast', () => {
  function success(msg: string) {
    showToast({ message: msg, type: 'success' })
  }

  function error(msg: string) {
    showToast({ message: msg, type: 'error' })
  }

  function info(msg: string) {
    showToast({ message: msg, type: 'info' })
  }

  function addToast(msg: string, type: ToastType = 'info') {
    showToast({ message: msg, type })
  }

  return { success, error, info, addToast }
})
