<script setup lang="ts">
// 确认弹窗：保持原有 props/emits 契约（各管理页复用），内部改为 Naive UI 实现
import { NModal, NButton } from 'naive-ui'

defineProps<{
  open: boolean
  title?: string
  message?: string
  confirmLabel?: string
  loading?: boolean
  danger?: boolean
}>()
const emit = defineEmits<{ confirm: []; cancel: [] }>()
</script>

<template>
  <NModal
    :show="open"
    :mask-closable="!loading"
    :close-on-esc="!loading"
    @update:show="(v: boolean) => { if (!v) emit('cancel') }"
  >
    <div class="w-[92vw] max-w-90 rounded-2xl border border-zinc-200 bg-white p-6 shadow-xl dark:border-zinc-800 dark:bg-zinc-900">
      <h3 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">{{ title || '确认操作' }}</h3>
      <p class="mt-2 text-sm leading-relaxed text-zinc-500 dark:text-zinc-400">{{ message }}</p>
      <div class="mt-6 flex justify-end gap-3">
        <NButton quaternary :disabled="loading" @click="emit('cancel')">取消</NButton>
        <NButton :type="danger ? 'error' : 'primary'" :loading="loading" @click="emit('confirm')">
          {{ confirmLabel || '确认' }}
        </NButton>
      </div>
    </div>
  </NModal>
</template>
