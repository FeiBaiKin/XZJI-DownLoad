<script setup lang="ts">
// 管理后台布局：桌面端固定侧栏（NMenu 分组导航）+ 移动端抽屉
// 导航结构与路由保持不变；UI 重构为 Naive UI + Tailwind
import { h, ref, computed, type Component } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { NMenu, NIcon, NButton, NDrawer, NDrawerContent, type MenuOption } from 'naive-ui'
import {
  Dashboard, Folders, Users, CreditCard, Settings, Sun, MoonStars, Logout, Menu2, Home, ShieldLock,
} from '@vicons/tabler'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const theme = useThemeStore()

const isAdmin = computed(() => auth.isAdmin)
const mobileNavOpen = ref(false)

function renderIcon(icon: Component) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

const menuOptions: MenuOption[] = [
  { label: '仪表盘', key: '/admin', icon: renderIcon(Dashboard) },
  {
    label: '内容管理',
    key: 'group-content',
    icon: renderIcon(Folders),
    children: [
      { label: '关键词管理', key: '/admin/keywords' },
      { label: '分类管理', key: '/admin/categories' },
      { label: '页面管理', key: '/admin/pages' },
      { label: '公告管理', key: '/admin/announcements' },
    ],
  },
  {
    label: '用户系统',
    key: 'group-users',
    icon: renderIcon(Users),
    children: [
      { label: '用户管理', key: '/admin/users' },
      { label: '签到配置', key: '/admin/checkin' },
      { label: '等级配置', key: '/admin/levels' },
      { label: '会员配置', key: '/admin/membership' },
    ],
  },
  {
    label: '支付系统',
    key: 'group-payment',
    icon: renderIcon(CreditCard),
    children: [
      { label: '支付配置', key: '/admin/payment-config' },
      { label: '支付订单', key: '/admin/orders' },
    ],
  },
  {
    label: '系统设置',
    key: 'group-system',
    icon: renderIcon(Settings),
    children: [
      { label: '网站设置', key: '/admin/settings' },
      { label: '邮箱配置', key: '/admin/email' },
      { label: '授权管理', key: '/admin/license' },
    ],
  },
]

// 按当前路由展开所属分组
const defaultExpandedKeys = computed(() => {
  const keys: string[] = []
  for (const opt of menuOptions) {
    if (opt.children?.some((c) => c.key === route.path)) keys.push(opt.key as string)
  }
  return keys.length ? keys : ['group-content']
})

function handleMenuSelect(key: string) {
  if (key.startsWith('/')) {
    router.push(key)
    mobileNavOpen.value = false
  }
}

function handleLogout() {
  auth.logout()
  router.push('/admin/login')
}
</script>

<template>
  <!-- 非管理员拦截 -->
  <div v-if="auth.user && !isAdmin" class="flex min-h-dvh items-center justify-center px-6">
    <div class="w-full max-w-md rounded-2xl border border-zinc-200 bg-white p-10 text-center shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
      <div class="mx-auto flex size-14 items-center justify-center rounded-full bg-red-100 text-red-500 dark:bg-red-500/10 dark:text-red-400">
        <NIcon :size="28"><ShieldLock /></NIcon>
      </div>
      <h1 class="mt-5 text-lg font-semibold text-zinc-900 dark:text-zinc-100">无权访问</h1>
      <p class="mt-2 text-sm leading-relaxed text-zinc-500 dark:text-zinc-400">
        你的账户 <strong class="text-zinc-800 dark:text-zinc-200">{{ auth.user?.username || '未知' }}</strong>
        没有管理员权限，无法访问管理后台。
      </p>
      <div class="mt-6 flex justify-center gap-3">
        <NButton type="primary" @click="router.push('/')">返回首页</NButton>
        <NButton quaternary @click="handleLogout">退出登录</NButton>
      </div>
    </div>
  </div>

  <div v-else class="min-h-dvh lg:flex">
    <!-- 桌面侧边栏 -->
    <aside class="sticky top-0 hidden h-dvh w-60 shrink-0 flex-col border-r border-zinc-200 bg-white lg:flex dark:border-zinc-800 dark:bg-zinc-950">
      <div class="flex items-center gap-2.5 px-5 pt-5 pb-4">
        <router-link
          to="/"
          class="grid size-8 shrink-0 place-items-center rounded-lg bg-emerald-600 text-white transition-opacity hover:opacity-85 dark:bg-emerald-400 dark:text-zinc-950"
          title="返回首页"
        >
          <NIcon :size="16"><Home /></NIcon>
        </router-link>
        <span class="text-[15px] font-semibold tracking-tight text-zinc-900 dark:text-zinc-100">管理后台</span>
      </div>

      <div class="flex-1 overflow-y-auto py-1">
        <NMenu
          :options="menuOptions"
          :value="route.path"
          :default-expanded-keys="defaultExpandedKeys"
          :indent="20"
          @update:value="handleMenuSelect"
        />
      </div>

      <div class="flex items-center justify-between gap-2 border-t border-zinc-200 p-4 dark:border-zinc-800">
        <p class="min-w-0 truncate text-sm font-medium text-zinc-900 dark:text-zinc-100">{{ auth.user?.username || 'Admin' }}</p>
        <div class="flex shrink-0 items-center gap-1">
          <button
            type="button"
            class="flex size-8 items-center justify-center rounded-lg text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-700 dark:hover:bg-zinc-900 dark:hover:text-zinc-200"
            :title="theme.isDark ? '切换浅色模式' : '切换深色模式'"
            @click="theme.toggle()"
          >
            <NIcon :size="16"><Sun v-if="theme.isDark" /><MoonStars v-else /></NIcon>
          </button>
          <button
            type="button"
            class="flex size-8 items-center justify-center rounded-lg text-zinc-400 transition-colors hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-500/10"
            title="退出登录"
            @click="handleLogout"
          >
            <NIcon :size="16"><Logout /></NIcon>
          </button>
        </div>
      </div>
    </aside>

    <!-- 移动端顶栏 + 抽屉 -->
    <div class="sticky top-0 z-20 flex h-13 items-center justify-between border-b border-zinc-200 bg-white/85 px-3 backdrop-blur-md lg:hidden dark:border-zinc-800 dark:bg-zinc-950/85">
      <button
        type="button"
        class="flex size-9 items-center justify-center rounded-lg text-zinc-500 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:bg-zinc-900"
        aria-label="打开导航"
        @click="mobileNavOpen = true"
      >
        <NIcon :size="20"><Menu2 /></NIcon>
      </button>
      <span class="text-sm font-semibold text-zinc-900 dark:text-zinc-100">管理后台</span>
      <button
        type="button"
        class="flex size-9 items-center justify-center rounded-lg text-zinc-500 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:bg-zinc-900"
        :title="theme.isDark ? '切换浅色模式' : '切换深色模式'"
        @click="theme.toggle()"
      >
        <NIcon :size="17"><Sun v-if="theme.isDark" /><MoonStars v-else /></NIcon>
      </button>
    </div>

    <NDrawer v-model:show="mobileNavOpen" :width="264" placement="left">
      <NDrawerContent title="管理后台" closable body-content-class="p-0!">
        <NMenu
          :options="menuOptions"
          :value="route.path"
          :default-expanded-keys="defaultExpandedKeys"
          :indent="20"
          @update:value="handleMenuSelect"
        />
        <div class="flex items-center justify-between border-t border-zinc-200 p-4 dark:border-zinc-800">
          <span class="text-sm font-medium text-zinc-900 dark:text-zinc-100">{{ auth.user?.username || 'Admin' }}</span>
          <NButton size="small" quaternary type="error" @click="handleLogout">退出登录</NButton>
        </div>
      </NDrawerContent>
    </NDrawer>

    <!-- 主内容区 -->
    <main class="min-w-0 flex-1 px-4 py-6 sm:px-6 lg:px-8 lg:py-8">
      <div class="mx-auto max-w-6xl">
        <RouterView />
      </div>
    </main>
  </div>
</template>
