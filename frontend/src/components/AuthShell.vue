<script setup lang="ts">
// 认证页共享外壳：居中卡片布局（登录 / 注册 / 找回密码）
// 取消全宽分栏与左侧品牌图；沿用站点 zinc + emerald 语言
import { computed } from 'vue'
import { ArrowLeft } from '@lucide/vue'
import AppIcon from './AppIcon.vue'
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
  <div class="auth-shell relative flex min-h-dvh flex-col">
    <!-- 背景：中性底 + 极轻 emerald 光晕（非紫蓝 mesh） -->
    <div aria-hidden="true" class="pointer-events-none absolute inset-0 overflow-hidden">
      <div class="absolute inset-0 bg-zinc-50 dark:bg-zinc-950" />
      <div
        class="absolute top-[-20%] left-1/2 size-[min(720px,90vw)] -translate-x-1/2 rounded-full bg-emerald-400/12 blur-3xl dark:bg-emerald-400/8"
      />
      <div
        class="absolute right-[-10%] bottom-[-15%] size-[min(420px,70vw)] rounded-full bg-zinc-300/40 blur-3xl dark:bg-zinc-800/50"
      />
    </div>

    <!-- 顶栏：返回 + 站点标识 -->
    <header class="relative z-10 flex items-center justify-between gap-4 px-5 pt-6 sm:px-8 sm:pt-8">
      <router-link
        :to="backTo"
        class="inline-flex items-center gap-1.5 rounded-lg px-1 py-1 text-sm text-zinc-500 transition-colors hover:text-zinc-900 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-500 active:scale-[0.98] dark:text-zinc-400 dark:hover:text-zinc-100"
      >
        <AppIcon :icon="ArrowLeft" scale="sm" />
        {{ backLabel }}
      </router-link>

      <router-link
        to="/"
        class="inline-flex items-center gap-2 rounded-lg focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-500"
      >
        <img
          v-if="site.siteLogoType === 'image' && site.siteLogoUrl"
          :src="site.siteLogoUrl"
          :alt="site.siteName"
          class="h-7 w-auto"
        />
        <template v-else>
          <span
            class="grid size-7 place-items-center rounded-lg bg-emerald-600 text-xs font-bold text-white dark:bg-emerald-400 dark:text-zinc-950"
          >
            {{ logoLetter }}
          </span>
          <span class="hidden text-sm font-semibold tracking-tight text-zinc-800 sm:inline dark:text-zinc-100">
            {{ site.siteName }}
          </span>
        </template>
      </router-link>
    </header>

    <!-- 主区：居中卡片 -->
    <main class="relative z-10 flex flex-1 items-center justify-center px-4 py-10 sm:px-6 sm:py-14">
      <div class="auth-card w-full max-w-[400px]">
        <div
          class="rounded-2xl border border-zinc-200/90 bg-white/90 p-7 shadow-[0_1px_2px_rgb(24_24_27/0.04),0_12px_40px_-12px_rgb(24_24_27/0.12)] backdrop-blur-sm sm:p-8 dark:border-zinc-800 dark:bg-zinc-900/90 dark:shadow-[0_1px_2px_rgb(0_0_0/0.3),0_16px_48px_-16px_rgb(0_0_0/0.55)]"
        >
          <div class="mb-7">
            <h1 class="text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">
              {{ title }}
            </h1>
            <p v-if="subtitle" class="mt-1.5 text-sm leading-relaxed text-zinc-500 dark:text-zinc-400">
              {{ subtitle }}
            </p>
          </div>

          <slot />

          <div
            v-if="$slots.footer"
            class="mt-7 border-t border-zinc-100 pt-6 text-center text-sm text-zinc-500 dark:border-zinc-800 dark:text-zinc-400"
          >
            <slot name="footer" />
          </div>
        </div>

        <p
          v-if="site.footerText"
          class="mt-6 text-center text-[11px] leading-relaxed text-zinc-400 dark:text-zinc-600"
        >
          {{ site.footerText }}
        </p>
      </div>
    </main>
  </div>
</template>

<style scoped>
/* 轻量入场：仅 opacity + transform；尊重 reduced-motion */
.auth-card {
  animation: auth-card-in 0.45s cubic-bezier(0.16, 1, 0.3, 1) both;
}

@keyframes auth-card-in {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (prefers-reduced-motion: reduce) {
  .auth-card {
    animation: none;
  }
}
</style>
