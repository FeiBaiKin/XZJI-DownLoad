import { defineStore } from 'pinia'
import { ref } from 'vue'

type ThemeMode = 'dark' | 'light'

export const useThemeStore = defineStore('theme', () => {
  // 初始主题：localStorage 优先，默认深色（对齐原项目）
  const stored = (localStorage.getItem('theme') as ThemeMode | null) ?? 'dark'
  const isDark = ref(stored === 'dark')

  function persist() {
    localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
  }

  function toggle() {
    isDark.value = !isDark.value
    persist()
  }

  // 显式设置主题（首页主题下拉用）
  function setTheme(mode: ThemeMode) {
    isDark.value = mode === 'dark'
    persist()
  }

  return { isDark, toggle, setTheme }
})
