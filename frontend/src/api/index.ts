import axios from 'axios'
import { useLicenseStore } from '@/stores/license'

const api = axios.create({
  baseURL: '/api',
  timeout: 15000,
  withCredentials: true, // 自动发送 HttpOnly Cookie
})

// 从 cookie 读取 CSRF token，用于 X-CSRF-Token 请求头
function getCSRFToken(): string {
  const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]*)/)
  return match ? match[1] : ''
}

/** 前台公开路径：401 只清会话，不整页跳登录（改用弹窗引导） */
function isPublicFrontendPath(path: string): boolean {
  if (path === '/' || path === '') return true
  if (path.startsWith('/categories')) return true
  if (path.startsWith('/page/')) return true
  if (path.startsWith('/login') || path.startsWith('/register') || path.startsWith('/forgot-password')) return true
  if (path.startsWith('/install')) return true
  return false
}

api.interceptors.request.use((config) => {
  // 对非安全方法附加 CSRF token
  const unsafeMethods = ['post', 'put', 'patch', 'delete']
  if (config.method && unsafeMethods.includes(config.method)) {
    const csrfToken = getCSRFToken()
    if (csrfToken) {
      config.headers['X-CSRF-Token'] = csrfToken
    }
  }
  return config
})

api.interceptors.response.use(
  (res) => res,
  (err) => {
    // 授权停服：业务接口被拦截时点亮全局停服遮罩
    if (err.response?.status === 403 && err.response?.data?.code === 'license_blocked') {
      useLicenseStore().markBlocked(err.response?.data?.error)
    }
    if (err.response?.status === 401) {
      void import('@/stores/auth')
        .then(({ useAuthStore }) => useAuthStore().clearAuth())
        .catch(() => {})

      // 会话探测等请求显式跳过跳转
      if (err.config?.skipAuthRedirect) {
        return Promise.reject(err)
      }

      const path = window.location.pathname
      // 管理后台：仍整页去后台登录
      if (path.startsWith('/admin') && !path.startsWith('/admin/login')) {
        window.location.href = '/admin/login'
        return Promise.reject(err)
      }
      // 用户中心：需登录页（路由守卫也会兜底）
      if (path.startsWith('/user')) {
        window.location.href = '/login?redirect=' + encodeURIComponent(path)
        return Promise.reject(err)
      }
      // 首页 / 分类 / 自定义页等公开前台：不跳转，保持当前页（弹窗登录）
      if (isPublicFrontendPath(path)) {
        return Promise.reject(err)
      }
    }
    return Promise.reject(err)
  }
)

export default api
