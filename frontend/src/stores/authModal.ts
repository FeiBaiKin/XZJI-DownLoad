import { defineStore } from 'pinia'
import { ref } from 'vue'

export type AuthModalTab = 'login' | 'register' | 'forgot'

const DISMISS_KEY = 'xzji_auth_prompt_dismissed'

/** 前台登录/注册弹窗状态（首页 3s 引导 + 各入口手动打开） */
export const useAuthModalStore = defineStore('authModal', () => {
  const open = ref(false)
  const tab = ref<AuthModalTab>('login')

  function show(next: AuthModalTab = 'login') {
    tab.value = next
    open.value = true
  }

  function hide() {
    open.value = false
  }

  function dismissPrompt() {
    try {
      sessionStorage.setItem(DISMISS_KEY, '1')
    } catch { /* ignore */ }
    hide()
  }

  function isPromptDismissed(): boolean {
    try {
      return sessionStorage.getItem(DISMISS_KEY) === '1'
    } catch {
      return false
    }
  }

  return { open, tab, show, hide, dismissPrompt, isPromptDismissed }
})
