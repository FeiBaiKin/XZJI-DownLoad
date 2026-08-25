<script setup lang="ts">
// 公共布局：公开站点（首页 / 分类页）共享的 header + footer
// Header：Logo + 导航（首页/分类） + 会员入口 + 主题切换 + 用户菜单/登录注册
// Footer：自定义页面链接 + 站点名 + footerText + 备案号（平铺一行展示）
// 首页未登录：先展示内容，约 3 秒后弹窗引导登录/注册（可关闭，同会话不重复强弹）
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter, useRoute, RouterView } from 'vue-router'
import { NButton, NDropdown } from 'naive-ui'
import { BadgeCheck, MoonStar, ShieldCheck, Sun } from '@lucide/vue'
import AppIcon from '../components/AppIcon.vue'
import { useAuthStore } from '@/stores/auth'
import { useAuthModalStore } from '@/stores/authModal'
import { useThemeStore } from '@/stores/theme'
import { useSiteSettingsStore } from '@/stores/siteSettings'
import { contentApi, type Page } from '@/api/content'
import AuthModal from '@/components/AuthModal.vue'
import MembershipEntry from '@/components/MembershipEntry.vue'

const auth = useAuthStore()
const authModal = useAuthModalStore()
const theme = useThemeStore()
const router = useRouter()
const route = useRoute()
const settings = useSiteSettingsStore()

const AUTH_PROMPT_DELAY_MS = 3000
let authPromptTimer: ReturnType<typeof setTimeout> | null = null

function clearAuthPromptTimer() {
  if (authPromptTimer) {
    clearTimeout(authPromptTimer)
    authPromptTimer = null
  }
}

function scheduleHomeAuthPrompt() {
  clearAuthPromptTimer()
  // 仅首页 + 已完成鉴权初始化 + 游客 + 本会话未主动关闭过
  if (route.name !== 'home') return
  if (!auth.initialized || auth.isLoggedIn) return
  if (authModal.isPromptDismissed()) return
  if (authModal.open) return

  authPromptTimer = setTimeout(() => {
    authPromptTimer = null
    if (route.name !== 'home') return
    if (auth.isLoggedIn || authModal.isPromptDismissed()) return
    authModal.show('login')
  }, AUTH_PROMPT_DELAY_MS)
}

watch(
  () => [route.name, auth.initialized, auth.isLoggedIn] as const,
  () => {
    if (auth.isLoggedIn) {
      clearAuthPromptTimer()
      authModal.hide()
      return
    }
    scheduleHomeAuthPrompt()
  },
  { immediate: true },
)

onMounted(() => {
  settings.fetch()
  fetchPages()
})

onUnmounted(() => {
  clearAuthPromptTimer()
})

const footerPages = ref<Page[]>([])

async function fetchPages() {
  try { footerPages.value = await contentApi.public.pages() } catch { footerPages.value = [] }
}

const isLoggedIn = computed(() => auth.isLoggedIn)
const isAdmin = computed(() => auth.isAdmin)
const logoLetter = computed(() => (settings.siteName?.[0] || 'X').toUpperCase())

const navItems = [
  { label: '首页', to: '/' },
  { label: '分类', to: '/categories' },
]

// 用户下拉菜单
const userMenuOptions = computed(() => {
  const opts: { label: string; key: string }[] = [{ label: '用户中心', key: 'user' }]
  if (isAdmin.value) opts.push({ label: '管理后台', key: 'admin' })
  opts.push({ label: '退出登录', key: 'logout' })
  return opts
})
function handleUserMenuSelect(key: string) {
  if (key === 'user') router.push('/user')
  else if (key === 'admin') router.push('/admin')
  else if (key === 'logout') {
    auth.logout()
    router.push('/')
  }
}
</script>

<template>
  <div class="flex min-h-dvh flex-col bg-white text-zinc-900 dark:bg-zinc-950 dark:text-zinc-100">
    <!-- Header -->
    <header
      class="sticky top-0 z-30 border-b border-zinc-200/70 bg-white/80 backdrop-blur-md motion-safe:transition-colors dark:border-zinc-800/70 dark:bg-zinc-950/80"
    >
      <div class="mx-auto flex h-16 max-w-7xl items-center justify-between gap-4 px-4 sm:px-6">
        <div class="flex items-center gap-6">
          <router-link
            to="/"
            class="flex items-center gap-2.5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 focus-visible:rounded-lg dark:focus-visible:ring-emerald-400/60"
          >
            <img
              v-if="settings.siteLogoType === 'image' && settings.siteLogoUrl"
              :src="settings.siteLogoUrl"
              :alt="settings.siteName || 'Logo'"
              class="h-8 w-auto"
            />
            <template v-else>
              <span
                class="grid size-8 place-items-center rounded-lg bg-emerald-600 text-sm font-bold text-white dark:bg-emerald-400 dark:text-zinc-950"
              >{{ logoLetter }}</span>
              <span
                class="text-[15px] font-semibold tracking-tight text-zinc-900 dark:text-zinc-100"
              >{{ settings.siteName }}</span>
            </template>
          </router-link>

          <nav class="hidden items-center gap-1 sm:flex">
            <router-link
              v-for="item in navItems"
              :key="item.to"
              :to="item.to"
              class="rounded-full px-3 py-1.5 text-[13px] font-medium text-zinc-500 motion-safe:transition-colors hover:bg-zinc-100 hover:text-zinc-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:text-zinc-400 dark:hover:bg-zinc-800/70 dark:hover:text-zinc-100 dark:focus-visible:ring-emerald-400/60"
              active-class="text-emerald-600 dark:text-emerald-400"
            >
              {{ item.label }}
            </router-link>
          </nav>
        </div>

        <div class="flex items-center gap-2">
          <!-- 会员入口：主题切换左侧 -->
          <MembershipEntry />

          <button
            type="button"
            class="flex size-9 items-center justify-center rounded-full text-zinc-500 motion-safe:transition-colors hover:bg-zinc-100 hover:text-zinc-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-200 dark:focus-visible:ring-emerald-400/60"
            :title="theme.isDark ? '切换浅色模式' : '切换深色模式'"
            :aria-label="theme.isDark ? '切换浅色模式' : '切换深色模式'"
            @click="theme.toggle()"
          >
            <AppIcon v-if="theme.isDark" :icon="Sun" scale="md" /><AppIcon v-else :icon="MoonStar" scale="md" />
          </button>

          <NDropdown
            v-if="isLoggedIn"
            :options="userMenuOptions"
            trigger="click"
            @select="handleUserMenuSelect"
          >
            <button
              type="button"
              aria-label="用户菜单"
              class="flex items-center rounded-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:focus-visible:ring-emerald-400/60"
            >
              <img
                v-if="auth.user?.avatar"
                :src="auth.user.avatar"
                alt=""
                class="size-9 rounded-full border border-zinc-200 object-cover dark:border-zinc-700"
              />
              <span
                v-else
                class="grid size-9 place-items-center rounded-full bg-zinc-200 text-sm font-semibold text-zinc-600 dark:bg-zinc-800 dark:text-zinc-300"
              >
                {{ (auth.user?.nickname || auth.user?.username || 'U')[0]!.toUpperCase() }}
              </span>
            </button>
          </NDropdown>

          <template v-else>
            <NButton quaternary size="small" @click="authModal.show('login')">登录</NButton>
            <NButton type="primary" size="small" @click="authModal.show('register')">注册</NButton>
          </template>
        </div>
      </div>

      <!-- 移动端导航 -->
      <nav class="flex items-center gap-1 overflow-x-auto px-4 pb-2 sm:hidden">
        <router-link
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="shrink-0 rounded-full px-3 py-1.5 text-[13px] font-medium text-zinc-500 motion-safe:transition-colors hover:bg-zinc-100 hover:text-zinc-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:text-zinc-400 dark:hover:bg-zinc-800/70 dark:hover:text-zinc-100 dark:focus-visible:ring-emerald-400/60"
          active-class="text-emerald-600 dark:text-emerald-400"
        >
          {{ item.label }}
        </router-link>
      </nav>
    </header>

    <!-- 主体 -->
    <main class="mx-auto w-full max-w-7xl flex-1 px-4 sm:px-6">
      <RouterView v-slot="{ Component, route }">
        <Transition name="page-shift" mode="out-in">
          <component :is="Component" :key="route.path" />
        </Transition>
      </RouterView>
    </main>

    <!-- 登录 / 注册弹窗 -->
    <AuthModal />

    <!-- Footer：自定义页面链接左对齐，站点信息 + 备案号右对齐，共用 | 分隔符 -->
    <footer class="mt-16 border-t border-zinc-200 py-8 dark:border-zinc-800/80">
      <div class="mx-auto flex max-w-7xl flex-wrap items-center justify-between gap-4 px-4 text-[13px] text-zinc-500 sm:px-6 dark:text-zinc-400">
        <!-- 左侧：自定义页面链接 -->
        <nav v-if="footerPages.length > 0" class="flex flex-wrap items-center gap-x-4 gap-y-1.5">
          <router-link
            v-for="p in footerPages"
            :key="p.id"
            :to="`/page/${p.slug}`"
            class="motion-safe:transition-colors hover:text-zinc-600 focus-visible:underline focus-visible:outline-none dark:hover:text-zinc-300"
          >{{ p.title }}</router-link>
        </nav>

        <!-- 右侧：站点信息 + 备案号，使用 | 分隔 -->
        <div class="flex flex-wrap items-center gap-x-3 gap-y-1.5">
          <span v-if="settings.footerText">{{ settings.footerText }}</span>
          <span v-if="settings.footerText" class="text-zinc-300 dark:text-zinc-700">|</span>
          <span>{{ settings.siteName || 'XZJI' }}</span>
          <template v-if="settings.siteIcp || settings.sitePolice">
            <span class="text-zinc-300 dark:text-zinc-700">|</span>
            <a
              v-if="settings.siteIcp"
              href="https://beian.miit.gov.cn"
              target="_blank"
              rel="noopener noreferrer"
              class="inline-flex items-center gap-1 motion-safe:transition-colors hover:text-zinc-600 focus-visible:underline focus-visible:outline-none dark:hover:text-zinc-300"
            >
              <AppIcon :icon="ShieldCheck" scale="xs" class="shrink-0" />
              <span>{{ settings.siteIcp }}</span>
            </a>
            <a
              v-if="settings.sitePolice"
              href="https://beian.mps.gov.cn"
              target="_blank"
              rel="noopener noreferrer"
              class="inline-flex items-center gap-1 motion-safe:transition-colors hover:text-zinc-600 focus-visible:underline focus-visible:outline-none dark:hover:text-zinc-300"
            >
              <AppIcon :icon="BadgeCheck" scale="xs" class="shrink-0" />
              <span>{{ settings.sitePolice }}</span>
            </a>
          </template>
        </div>
      </div>
    </footer>
  </div>
</template>