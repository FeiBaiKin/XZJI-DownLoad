<script setup lang="ts">
// 用户中心布局：桌面端左侧边栏 + 移动端顶部横向导航
// UI 重构为 Naive UI + Tailwind；导航项与路由保持不变
import { computed, type Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Crown, Gauge, Heart, KeyRound, LogOut, User } from '@lucide/vue'
import AppIcon from '../components/AppIcon.vue'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

interface NavItem {
  to: string
  label: string
  icon: Component
}
const navItems: NavItem[] = [
  { to: '/user', label: '个人中心', icon: Gauge },
  { to: '/user/profile', label: '个人资料', icon: User },
  { to: '/user/favorites', label: '我的收藏', icon: Heart },
  { to: '/user/membership', label: '会员中心', icon: Crown },
  { to: '/user/security', label: '安全设置', icon: KeyRound },
]

const displayName = computed(() => auth.user?.nickname || auth.user?.username || '用户')
const roleLabel = computed(() => (auth.user?.role === 'admin' ? '管理员' : '普通用户'))

function isActive(path: string) {
  return route.path === path
}

function handleLogout() {
  auth.logout()
  router.push('/login')
}
</script>

<template>
  <div class="min-h-dvh lg:flex">
    <!-- 桌面侧边栏 -->
    <aside class="sticky top-0 hidden h-dvh w-60 shrink-0 flex-col border-r border-zinc-200 bg-white lg:flex dark:border-zinc-800 dark:bg-zinc-950">
      <div class="px-4 pt-5 pb-3">
        <router-link
          to="/"
          class="inline-flex items-center gap-1.5 text-sm text-zinc-500 transition-colors hover:text-zinc-900 focus-visible:underline focus-visible:outline-none dark:text-zinc-400 dark:hover:text-zinc-100"
        >
          <AppIcon :icon="ArrowLeft" scale="sm" />
          返回首页
        </router-link>
      </div>

      <nav class="flex-1 space-y-1 px-3 py-2">
        <router-link
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="flex items-center gap-2.5 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:focus-visible:ring-emerald-400/60"
          :class="isActive(item.to)
            ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-400/10 dark:text-emerald-400'
            : 'text-zinc-600 hover:bg-zinc-100 hover:text-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-900 dark:hover:text-zinc-100'"
        >
          <AppIcon :icon="item.icon" scale="sm" />
          {{ item.label }}
        </router-link>
      </nav>

      <div class="border-t border-zinc-200 p-4 dark:border-zinc-800">
        <div class="flex items-center justify-between gap-2">
          <div class="min-w-0">
            <p class="truncate text-sm font-medium text-zinc-900 dark:text-zinc-100">{{ displayName }}</p>
            <p class="text-xs text-zinc-500 dark:text-zinc-400">{{ roleLabel }}</p>
          </div>
          <button
            type="button"
            class="flex size-8 shrink-0 items-center justify-center rounded-lg text-zinc-400 transition-colors hover:bg-red-50 hover:text-red-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:hover:bg-red-500/10 dark:focus-visible:ring-emerald-400/60"
            title="退出登录"
            aria-label="退出登录"
            @click="handleLogout"
          >
            <AppIcon :icon="LogOut" scale="sm" />
          </button>
        </div>
      </div>
    </aside>

    <!-- 移动端顶栏 -->
    <div class="sticky top-0 z-20 border-b border-zinc-200 bg-white/85 backdrop-blur-md lg:hidden dark:border-zinc-800 dark:bg-zinc-950/85">
      <div class="flex h-12 items-center justify-between px-4">
        <router-link to="/" class="inline-flex items-center gap-1 text-sm text-zinc-500 focus-visible:underline focus-visible:outline-none dark:text-zinc-400">
          <AppIcon :icon="ArrowLeft" scale="sm" />
          首页
        </router-link>
        <span class="truncate text-sm font-medium text-zinc-900 dark:text-zinc-100">{{ displayName }}</span>
        <button
          type="button"
          class="flex size-8 items-center justify-center rounded-lg text-zinc-400 transition-colors hover:text-red-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:focus-visible:ring-emerald-400/60"
          title="退出登录"
          aria-label="退出登录"
          @click="handleLogout"
        >
          <AppIcon :icon="LogOut" scale="sm" />
        </button>
      </div>
      <nav class="scrollbar-none flex gap-1 overflow-x-auto px-3 pb-2">
        <router-link
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="shrink-0 rounded-full px-3.5 py-1.5 text-[13px] font-medium whitespace-nowrap transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:focus-visible:ring-emerald-400/60"
          :class="isActive(item.to)
            ? 'bg-emerald-600 text-white dark:bg-emerald-400 dark:text-zinc-950'
            : 'text-zinc-500 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:bg-zinc-900'"
        >
          {{ item.label }}
        </router-link>
      </nav>
    </div>

    <!-- 主内容区 -->
    <main class="min-w-0 flex-1 px-4 py-8 sm:px-8 lg:px-10 lg:py-10">
      <div class="mx-auto max-w-4xl">
        <RouterView v-slot="{ Component, route: r }">
          <Transition name="page-shift" mode="out-in">
            <component :is="Component" :key="r.path" />
          </Transition>
        </RouterView>
      </div>
    </main>
  </div>
</template>
