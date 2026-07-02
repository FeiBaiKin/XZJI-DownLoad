<script setup lang="ts">
// 公开页面 /page/:slug：1:1 移植自原项目 App.jsx 的 PageView
// 样式来自全局 styles/content-page.css
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import DOMPurify from 'dompurify'
import { contentApi } from '@/api/content'

const route = useRoute()
const page = ref<any>(null)
const loading = ref(true)

async function load(slug: string) {
  loading.value = true
  try {
    page.value = await contentApi.public.page(slug)
  } catch {
    page.value = null
  }
  loading.value = false
}

onMounted(() => { load(route.params.slug as string) })
watch(() => route.params.slug, (slug) => { if (slug) load(slug as string) })
</script>

<template>
  <div v-if="loading" class="content-page">
    <div class="loading-state">加载中...</div>
  </div>
  <div v-else-if="page" class="content-page">
    <div class="content-page-wrap">
      <router-link to="/" class="auth-back">
        <span class="auth-back-icon">&#8592;</span>
        <span>返回首页</span>
      </router-link>
      <h1 class="content-page-title">{{ page.title }}</h1>
      <div class="page-content" v-html="DOMPurify.sanitize(page.content)" />
    </div>
  </div>
  <div v-else class="content-page">
    <div class="content-page-wrap" style="text-align: center;">
      <h1 class="content-page-title">页面不存在</h1>
      <p style="color: var(--text-secondary); margin: 0 0 1.5rem;">该页面未发布或不存在</p>
      <router-link to="/" class="auth-link">返回首页</router-link>
    </div>
  </div>
</template>
