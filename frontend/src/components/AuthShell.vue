<script setup lang="ts">
// 认证页共享外壳：左侧品牌面板（lg 以上显示）+ 右侧表单区
// 登录 / 注册 / 找回密码 三页复用
import { computed } from 'vue'
import { NIcon } from 'naive-ui'
import { ArrowLeft } from '@vicons/tabler'
import { useSiteSettingsStore } from '@/stores/siteSettings'

withDefaults(defineProps<{
  title: string
  subtitle?: string
  backTo?: string
  backLabel?: string
}>(), {
  backTo: '/',
  backLabel: '返回首页',
})

const site = useSiteSettingsStore()
const logoLetter = computed(() => (site.siteName?.[0] || 'X').toUpperCase())
</script>

<template>
  <div class="flex min-h-dvh">
    <!-- 品牌面板：始终深色，形成左右对比 -->
    <aside class="relative hidden w-[42%] flex-col justify-between overflow-hidden bg-zinc-950 p-10 lg:flex">
      <div aria-hidden="true" class="pointer-events-none absolute -top-40 -left-40 size-[480px] rounded-full bg-emerald-500/15 blur-3xl"></div>
      <div aria-hidden="true" class="pointer-events-none absolute -right-32 -bottom-32 size-[380px] rounded-full bg-emerald-400/8 blur-3xl"></div>

      <router-link to="/" class="relative flex items-center gap-2.5">
        <img v-if="site.siteLogoType === 'image' && site.siteLogoUrl" :src="site.siteLogoUrl" :alt="site.siteName" class="h-8 w-auto" />
        <template v-else>
          <span class="grid size-8 place-items-center rounded-lg bg-emerald-400 text-sm font-bold text-zinc-950">{{ logoLetter }}</span>
          <span class="text-[15px] font-semibold tracking-tight text-zinc-100">{{ site.siteName }}</span>
        </template>
      </router-link>

      <div class="relative">
        <h2 class="text-3xl leading-snug font-bold tracking-tight text-zinc-50">
          关键词直达资源，<br />会员权益一步到位。
        </h2>
        <p v-if="site.siteDescription" class="mt-4 max-w-sm text-sm leading-relaxed text-zinc-400">
          {{ site.siteDescription }}
        </p>
      </div>

      <p class="relative text-xs text-zinc-600">{{ site.footerText || site.siteName }}</p>
    </aside>

    <!-- 表单区 -->
    <main class="flex flex-1 flex-col px-6 py-8 sm:px-10">
      <div class="flex items-center justify-between">
        <router-link
          :to="backTo"
          class="inline-flex items-center gap-1.5 text-sm text-zinc-500 transition-colors hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
        >
          <NIcon :size="16"><ArrowLeft /></NIcon>
          {{ backLabel }}
        </router-link>
        <!-- 移动端顶部小 Logo -->
        <router-link to="/" class="lg:hidden">
          <span class="grid size-8 place-items-center rounded-lg bg-emerald-600 text-sm font-bold text-white dark:bg-emerald-400 dark:text-zinc-950">
            {{ logoLetter }}
          </span>
        </router-link>
      </div>

      <div class="flex flex-1 items-center justify-center py-10">
        <div class="w-full max-w-sm">
          <h1 class="text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">{{ title }}</h1>
          <p v-if="subtitle" class="mt-1.5 text-sm text-zinc-500 dark:text-zinc-400">{{ subtitle }}</p>
          <div class="mt-8">
            <slot />
          </div>
          <div v-if="$slots.footer" class="mt-8 text-center text-sm text-zinc-500 dark:text-zinc-400">
            <slot name="footer" />
          </div>
        </div>
      </div>
    </main>
  </div>
</template>
