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
    if (err.response?.status === 401) {
      try {
        const { useAuthStore } = require('@/stores/auth')
        useAuthStore().clearAuth()
      } catch {}
      const path = window.location.pathname
      if (!path.startsWith('/login') && !path.startsWith('/install') && !path.startsWith('/register')) {
        window.location.href = '/login'
      }
    }
    return Promise.reject(err)
  }
)

export default api
