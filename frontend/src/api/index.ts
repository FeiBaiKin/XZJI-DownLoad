import axios from 'axios'

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
      import('@/stores/license')
        .then(({ useLicenseStore }) => useLicenseStore().markBlocked(err.response?.data?.error))
        .catch(() => {})
    }
    if (err.response?.status === 401) {
      try {
        const { useAuthStore } = require('@/stores/auth')
        useAuthStore().clearAuth()
      } catch {}
      const path = window.location.pathname
      const onAuthPage =
        path.startsWith('/login') || path.startsWith('/install') ||
        path.startsWith('/register') || path.startsWith('/admin/login')
      if (!onAuthPage) {
        // 后台路径的 401 回后台登录页，其余回前台登录页
        window.location.href = path.startsWith('/admin') ? '/admin/login' : '/login'
      }
    }
    return Promise.reject(err)
  }
)

export default api
