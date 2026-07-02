import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { authApi, type UserBrief } from '@/api/auth'

// user 持久化到 localStorage，首页等公开页面可同步读取角色信息
function loadUser(): UserBrief | null {
  try {
    const raw = localStorage.getItem('user')
    return raw ? (JSON.parse(raw) as UserBrief) : null
  } catch {
    return null
  }
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<UserBrief | null>(loadUser())
  const initialized = ref(false)

  // isLoggedIn 基于 user 是否存在（不再依赖 localStorage token）
  const isLoggedIn = computed(() => !!user.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  // user 任意变更同步写回 localStorage
  watch(
    user,
    (u) => {
      if (u) localStorage.setItem('user', JSON.stringify(u))
      else localStorage.removeItem('user')
    },
    { deep: true },
  )

  function setAuth(_t: string, u: UserBrief) {
    user.value = u
    // JWT 已通过 HttpOnly Cookie 携带，不再存 localStorage
  }

  function clearAuth() {
    user.value = null
  }

  async function fetchProfile() {
    try {
      const u = await authApi.profile()
      user.value = u
    } catch {
      clearAuth()
    } finally {
      initialized.value = true
    }
  }

  async function login(username: string, password: string) {
    const res = await authApi.login(username, password)
    setAuth(res.token, res.user)
  }

  async function register(data: { username: string; password: string; email?: string; code?: string }) {
    const res = await authApi.register(data)
    setAuth(res.token, res.user)
  }

  async function logout() {
    try { await authApi.logout() } catch {}
    clearAuth()
  }

  return { user, isLoggedIn, isAdmin, initialized, setAuth, clearAuth, fetchProfile, login, register, logout }
})
