<script setup lang="ts">
// 确认弹窗：1:1 移植自原项目 Node-XZJI/frontend/src/components/ConfirmDialog.jsx
// 复用全局 modal.css 样式
defineProps<{
  open: boolean
  title?: string
  message?: string
  confirmLabel?: string
  loading?: boolean
  danger?: boolean
}>()
const emit = defineEmits<{ confirm: []; cancel: [] }>()

function handleOverlayClick(e: MouseEvent, loading?: boolean) {
  if (e.target === e.currentTarget && !loading) emit('cancel')
}
</script>

<template>
  <div v-if="open" class="modal-overlay" @click="(e) => handleOverlayClick(e, loading)">
    <div class="modal-content" style="max-width: 360px;">
      <div class="modal-header">
        <h3>{{ title || '确认操作' }}</h3>
        <button class="modal-close" @click="emit('cancel')" :disabled="loading">X</button>
      </div>
      <div style="padding: 1.25rem;">
        <p style="margin: 0 0 1.25rem; font-size: 0.85rem; color: var(--text-secondary); line-height: 1.6;">
          {{ message }}
        </p>
        <div style="display: flex; gap: 0.75rem; justify-content: flex-end;">
          <button class="action-button secondary" @click="emit('cancel')" :disabled="loading">取消</button>
          <button
            class="action-button"
            :style="danger ? { background: 'var(--danger)', color: '#fff', borderColor: 'var(--danger)' } : {}"
            @click="emit('confirm')"
            :disabled="loading"
          >
            {{ loading ? '处理中...' : confirmLabel || '确认' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
