<script setup lang="ts">
import { watch, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { NConfigProvider, NGlobalStyle, darkTheme, zhCN, dateZhCN } from 'naive-ui'
import { useThemeStore } from '@/stores/theme'
import { useSiteSettingsStore } from '@/stores/siteSettings'
import { useLicenseStore } from '@/stores/license'
import { lightThemeOverrides, darkThemeOverrides } from '@/theme'
import LicenseBlocked from '@/components/LicenseBlocked.vue'
import { useSEO } from '@/composables/useSEO'

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

onMounted(() => {
  site.fetch()
  license.fetch()
})
</script>

<template>
  <NConfigProvider :theme="naiveTheme" :theme-overrides="overrides" :locale="zhCN" :date-locale="dateZhCN">
    <NGlobalStyle />
    <LicenseBlocked v-if="licenseBlocked" />
    <RouterView v-else />
  </NConfigProvider>
</template>
