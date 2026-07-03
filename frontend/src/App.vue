<script setup lang="ts">
import { watch, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useThemeStore } from '@/stores/theme'
import { useSiteSettingsStore } from '@/stores/siteSettings'
import { useLicenseStore } from '@/stores/license'
import ToastContainer from '@/components/ToastContainer.vue'
import LicenseBlocked from '@/components/LicenseBlocked.vue'

const theme = useThemeStore()
const site = useSiteSettingsStore()
const license = useLicenseStore()
const route = useRoute()

// 授权停服遮罩：前台所有页面显示「授权已过期」；管理后台/登录/安装/找回密码放行，
// 便于管理员登录后台重新激活（接入文档「停止对外服务」约定）
const LICENSE_EXEMPT_PREFIXES = ['/admin', '/login', '/install', '/forgot-password']
const licenseBlocked = computed(() =>
  license.checked && !license.valid && !LICENSE_EXEMPT_PREFIXES.some((p) => route.path === p || route.path.startsWith(p + '/'))
)

// 同步 data-theme 到 document
watch(() => theme.isDark, (v) => {
  document.documentElement.setAttribute('data-theme', v ? 'dark' : 'light')
}, { immediate: true })

// 同步网站名称到 document.title
watch(() => site.siteName, (name) => {
  document.title = name || 'XZJI'
}, { immediate: true })

// 路由切换时可叠加页面标题（通过 route.meta.title）
watch(() => route.meta?.title, (pageTitle) => {
  if (pageTitle) {
    document.title = `${pageTitle} - ${site.siteName || 'XZJI'}`
  } else {
    document.title = site.siteName || 'XZJI'
  }
})

onMounted(() => {
  site.fetch()
  license.fetch()
})
</script>

<template>
  <LicenseBlocked v-if="licenseBlocked" />
  <RouterView v-else />
  <ToastContainer />
</template>
