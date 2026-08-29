import { defineStore } from 'pinia'
import { nextTick, ref } from 'vue'

type ThemeMode = 'dark' | 'light'

// View Transitions API 最小类型（Firefox/Safari 旧版无此 API，走瞬切降级）
type ViewTransitionDocument = Document & {
  startViewTransition?: (callback: () => void | Promise<void>) => unknown
}

export const useThemeStore = defineStore('theme', () => {
  // 初始主题：localStorage 优先，默认深色（对齐原项目）
  const stored = (localStorage.getItem('theme') as ThemeMode | null) ?? 'dark'
  const isDark = ref(stored === 'dark')

  function persist() {
    localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
  }

  function apply(mode: ThemeMode) {
    isDark.value = mode === 'dark'
    persist()
  }

  // 主题切换过渡：View Transitions 从触发点圆形扩散揭示新主题。
  // 浏览器不支持 / 后台关闭动效（data-motion='off'）/ 系统减少动态效果时直接切换。
  // 时长与缓动全部由 CSS（::view-transition-new(root) 的 vt-reveal 动画）管理，
  // 这里只负责写入扩散起点坐标——不要在 JS 里解析带单位的时长
  // （构建器会把 620ms 压缩成 .62s，parseFloat 会得到 0.62 毫秒，实测踩坑）。
  function withReveal(mutate: () => void, point?: { x: number; y: number }) {
    const doc = document as ViewTransitionDocument
    const motion = document.documentElement.dataset.motion
    const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    if (!doc.startViewTransition || motion === 'off' || reduced) {
      mutate()
      return
    }
    const style = document.documentElement.style
    style.setProperty('--vt-x', `${point?.x ?? window.innerWidth / 2}px`)
    style.setProperty('--vt-y', `${point?.y ?? 0}px`)
    doc.startViewTransition(() => {
      mutate()
      return nextTick()
    })
  }

  function toggle(event?: MouseEvent) {
    // 只有真实指针点击才取坐标（键盘触发无意义）
    const point =
      event && (event.clientX || event.clientY) ? { x: event.clientX, y: event.clientY } : undefined
    withReveal(() => apply(isDark.value ? 'light' : 'dark'), point)
  }

  // 显式设置主题（弹窗等场景用）
  function setTheme(mode: ThemeMode, event?: MouseEvent) {
    const point =
      event && (event.clientX || event.clientY) ? { x: event.clientX, y: event.clientY } : undefined
    withReveal(() => apply(mode), point)
  }

  return { isDark, toggle, setTheme }
})
