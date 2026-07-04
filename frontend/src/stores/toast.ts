import { defineStore } from 'pinia'
import { computed } from 'vue'
import { createDiscreteApi, darkTheme, type ConfigProviderProps } from 'naive-ui'
import { useThemeStore } from '@/stores/theme'
import { lightThemeOverrides, darkThemeOverrides } from '@/theme'

// 全局消息通知：对外保持 success/error/info API（历史沿用「toast」命名），
// 内部改由 Naive UI 离散 API 渲染，随主题联动，无需挂载容器组件。
export const useToastStore = defineStore('toast', () => {
  const theme = useThemeStore()

  const configProviderProps = computed<ConfigProviderProps>(() => ({
    theme: theme.isDark ? darkTheme : null,
    themeOverrides: theme.isDark ? darkThemeOverrides : lightThemeOverrides,
  }))

  const { message } = createDiscreteApi(['message'], { configProviderProps })

  function success(msg: string) {
    message.success(msg)
  }
  function error(msg: string) {
    message.error(msg, { duration: 4000 })
  }
  function info(msg: string) {
    message.info(msg)
  }
  function addToast(msg: string, type: 'info' | 'success' | 'error' = 'info') {
    message[type](msg)
  }

  return { success, error, info, addToast }
})
