<script setup lang="ts">
// 公开页面 /page/:slug：渲染后台「页面管理」发布的富文本内容
// 业务逻辑保持不变；UI 重构为 Naive UI + Tailwind
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { NIcon, NSkeleton, NButton } from 'naive-ui'
import { PhArrowLeft } from '@phosphor-icons/vue'
import DOMPurify from 'dompurify'
import { contentApi, type Page } from '@/api/content'

const route = useRoute()
const page = ref<Page | null>(null)
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
  <div class="min-h-dvh">
    <div class="mx-auto max-w-3xl px-4 py-10 sm:px-6 sm:py-14">
      <router-link
        to="/"
        class="inline-flex items-center gap-1.5 text-sm text-zinc-500 transition-colors hover:text-zinc-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 focus-visible:rounded dark:text-zinc-400 dark:hover:text-zinc-100 dark:focus-visible:ring-emerald-400/60"
      >
        <NIcon :size="16"><PhArrowLeft /></NIcon>
        返回首页
      </router-link>

      <!-- 加载骨架 -->
      <div v-if="loading" class="mt-8 space-y-4">
        <NSkeleton text width="40%" :sharp="false" size="large" />
        <NSkeleton text :repeat="5" :sharp="false" />
      </div>

      <!-- 页面内容 -->
      <article v-else-if="page" class="mt-8">
        <h1 class="text-3xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">{{ page.title }}</h1>
        <div class="rich-content mt-6 text-[15px]" v-html="DOMPurify.sanitize(page.content)" />
      </article>

      <!-- 页面不存在 -->
      <div v-else class="mt-20 text-center">
        <p class="font-mono text-5xl font-bold text-zinc-200 dark:text-zinc-800">404</p>
        <h1 class="mt-4 text-xl font-semibold text-zinc-900 dark:text-zinc-100">页面不存在</h1>
        <p class="mt-2 text-sm text-zinc-500 dark:text-zinc-400">该页面未发布或不存在</p>
        <NButton type="primary" secondary class="mt-6" @click="$router.push('/')">返回首页</NButton>
      </div>
    </div>
  </div>
</template>
