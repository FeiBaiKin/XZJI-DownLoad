/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

// 扩展 axios 请求配置（不覆盖整个 axios 模块）
import type { AxiosRequestConfig } from 'axios'
declare module 'axios' {
  interface AxiosRequestConfig {
    /** 为 true 时 401 不整页跳转（会话探测等） */
    skipAuthRedirect?: boolean
  }
}
