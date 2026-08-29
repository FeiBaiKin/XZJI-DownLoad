<script setup lang="ts">
import { watch, onMounted, onUnmounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { NConfigProvider, NGlobalStyle, NMessageProvider, darkTheme, zhCN, dateZhCN } from 'naive-ui'
import { useThemeStore } from '@/stores/theme'
import { useSiteSettingsStore } from '@/stores/siteSettings'
import { useLicenseStore } from '@/stores/license'
import { lightThemeOverrides, darkThemeOverrides } from '@/theme'
import LicenseBlocked from '@/components/LicenseBlocked.vue'
import { useSEO } from '@/composables/useSEO'
import ToastBridge from '@/components/ToastBridge.vue'

const theme = useThemeStore()
const site = useSiteSettingsStore()
const license = useLicenseStore()
const route = useRoute()
useSEO()

const naiveTheme = computed(() => (theme.isDark ? darkTheme : null))
const overrides = computed(() => (theme.isDark ? darkThemeOverrides : lightThemeOverrides))

// 授权停服遮罩：前台所有页面显示「授权已过期」；管理后台/登录/安装/找回密码放行，
// 便于管理员登录后台重新激活（接入文档「停止对外服务」约定）
const LICENSE_EXEMPT_PREFIXES = ['/admin', '/login', '/install', '/forgot-password']
const licenseBlocked = computed(() =>
  license.checked && !license.valid && !LICENSE_EXEMPT_PREFIXES.some((p) => route.path === p || route.path.startsWith(p + '/'))
)

// 同步 data-theme 到 document（Tailwind dark: 变体与 wangEditor 暗色变量依赖此属性）
watch(() => theme.isDark, (v) => {
  document.documentElement.setAttribute('data-theme', v ? 'dark' : 'light')
}, { immediate: true })

// 同步动效档位到 <html data-motion>（后台「网站设置 → 界面动效」控制）：
// subtle / standard / expressive 三档；开关关闭时置 off，全站过渡与动画归零
const MOTION_PRESETS = ['subtle', 'standard', 'expressive']
watch(() => [site.motionEnabled, site.motionPreset] as const, ([enabled, preset]) => {
  document.documentElement.dataset.motion = enabled && MOTION_PRESETS.includes(preset) ? preset : 'off'
}, { immediate: true })

// 顶层 RouterView 仅在布局（路由一级记录）变化时过渡，布局内页面切换交给各布局的 page-shift
const layoutKey = computed(() => route.matched[0]?.path || route.path)

// 全局点击涟漪：pointerdown 时向命中的按钮注入一次性光晕（色随按钮文字色 currentColor，
// inset:0 + border-radius:inherit 限制在按钮内）。随动效开关与系统减少动态效果启停。
function spawnRipple(e: PointerEvent) {
  if (document.documentElement.dataset.motion === 'off') return
  const target = (e.target as Element | null)?.closest?.('button, [role="button"]') as HTMLElement | null
  if (!target || target.hasAttribute('disabled') || target.getAttribute('aria-disabled') === 'true') return
  const rect = target.getBoundingClientRect()
  if (!rect.width || !rect.height) return
  const ripple = document.createElement('span')
  ripple.className = 'motion-ripple'
  ripple.style.setProperty('--ripple-x', `${e.clientX - rect.left}px`)
  ripple.style.setProperty('--ripple-y', `${e.clientY - rect.top}px`)
  if (getComputedStyle(target).position === 'static') target.classList.add('motion-ripple-host')
  const remove = () => {
    ripple.remove()
    target.classList.remove('motion-ripple-host')
  }
  ripple.addEventListener('animationend', remove, { once: true })
  // 兜底：动画被系统/扩展禁用等极端情况下 animationend 不触发，超时强清理
  window.setTimeout(remove, 800)
  target.appendChild(ripple)
}

onMounted(() => {
  site.fetch()
  license.fetch()
  if (window.matchMedia('(prefers-reduced-motion: no-preference)').matches) {
    document.addEventListener('pointerdown', spawnRipple, { passive: true })
  }
})

onUnmounted(() => {
  document.removeEventListener('pointerdown', spawnRipple)
})
</script>

<template>
  <NConfigProvider :theme="naiveTheme" :theme-overrides="overrides" :locale="zhCN" :date-locale="dateZhCN">
    <NGlobalStyle />
    <!--
      Message API 必须处于 NMessageProvider 的子树中。
      ToastBridge 会把这里的 useMessage API 注册给 Pinia，避免在 store
      setup 阶段通过 createDiscreteApi 创建嵌套 Vue 应用。
    -->
    <NMessageProvider>
      <ToastBridge />
      <LicenseBlocked v-if="licenseBlocked" />
      <RouterView v-else v-slot="{ Component }">
        <Transition name="app-fade" mode="out-in" appear>
          <component :is="Component" :key="layoutKey" />
        </Transition>
      </RouterView>
    </NMessageProvider>
  </NConfigProvider>
</template>
