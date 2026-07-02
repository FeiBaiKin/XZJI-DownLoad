<script setup lang="ts">
import { watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useThemeStore } from '@/stores/theme'
import { useSiteSettingsStore } from '@/stores/siteSettings'
import ToastContainer from '@/components/ToastContainer.vue'

const theme = useThemeStore()
const site = useSiteSettingsStore()
const route = useRoute()

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
})
</script>

<template>
  <RouterView />
  <ToastContainer />
</template>
