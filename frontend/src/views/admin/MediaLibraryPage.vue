<script setup lang="ts">
// 媒体库管理页：复用 MediaPicker 组件（管理模式 selectable=false）
import { NIcon } from 'naive-ui'
import { Photo } from '@vicons/tabler'
import MediaPicker from '@/components/MediaPicker.vue'
</script>

<template>
  <div>
    <header class="mb-6">
      <div class="flex items-center gap-2">
        <NIcon :size="20" class="text-emerald-600 dark:text-emerald-400"><Photo /></NIcon>
        <h1 class="text-2xl font-bold tracking-tight text-zinc-900 sm:text-[28px] dark:text-zinc-50">媒体库</h1>
      </div>
      <p class="mt-1 max-w-[60ch] text-sm text-zinc-500 dark:text-zinc-400">
        管理已上传的图片资源。支持拖拽上传本地图片，或将网络图片 URL 导入到本地存储。关键词图标、内容图片等可从此库复用。
      </p>
    </header>

    <MediaPicker :selectable="false" :page-size="24" />
  </div>
</template>