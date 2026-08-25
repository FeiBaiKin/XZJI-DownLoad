<script setup lang="ts">
// 授权停服全页遮罩：授权无效时替代前台所有页面显示（接入文档约定文案），
// 管理后台 /admin 与 /login 不受影响，便于管理员登录处理授权。
import { Lock } from '@lucide/vue'
import AppIcon from './AppIcon.vue'
import { useLicenseStore } from '@/stores/license'

const license = useLicenseStore()
</script>

<template>
  <div class="flex min-h-dvh items-center justify-center px-6 py-12">
    <div class="w-full max-w-md rounded-2xl border border-zinc-200 bg-white p-6 text-center shadow-sm sm:p-10 dark:border-zinc-800 dark:bg-zinc-900">
      <div class="mx-auto flex size-14 items-center justify-center rounded-full bg-amber-100 text-amber-600 dark:bg-amber-400/10 dark:text-amber-400">
        <AppIcon :icon="Lock" scale="lg" />
      </div>
      <h1 class="mt-5 text-lg font-semibold text-zinc-900 dark:text-zinc-100">
        {{ license.message || '授权已过期，请联系服务商' }}
      </h1>
      <p class="mt-2 text-sm text-zinc-500 dark:text-zinc-400">本站点的软件授权已失效，前台服务暂时停止。</p>
      <p class="mt-8 text-xs text-zinc-500 dark:text-zinc-400">
        站点管理员？
        <router-link to="/admin/login" class="rounded font-medium text-emerald-600 hover:underline focus-visible:underline focus-visible:outline-none dark:text-emerald-400">
          登录管理后台
        </router-link>
        查看授权状态并重新激活。
      </p>
    </div>
  </div>
</template>
