<script setup lang="ts">
import { useToastStore } from '@/stores/toast'

// 全局 Toast 容器，渲染在 App.vue 根节点。样式来自 styles/toast.css（移植自原项目）
const toast = useToastStore()
</script>

<template>
  <div class="toast-container">
    <div v-for="t in toast.toasts" :key="t.id" class="toast" :class="`toast-${t.type}`">
      <span class="toast-icon">
        {{ t.type === 'error' ? '!' : t.type === 'success' ? '\u2713' : 'i' }}
      </span>
      <span class="toast-msg">{{ t.message }}</span>
      <button class="toast-close" @click="toast.remove(t.id)">
        <svg width="8" height="8" viewBox="0 0 10 10" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M1 1l8 8M9 1l-8 8" />
        </svg>
      </button>
    </div>
  </div>
</template>
