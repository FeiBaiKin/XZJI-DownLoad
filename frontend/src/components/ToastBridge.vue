<script setup lang="ts">
// 将根组件树中的 Naive UI message API 注册给 Pinia toast store。
// 该组件必须放在 NMessageProvider 内部，才能安全调用 useMessage。
import { useMessage } from 'naive-ui'
import { onUnmounted } from 'vue'
import { registerToastApi, unregisterToastApi } from '@/stores/toast'

const message = useMessage()
registerToastApi(message)

onUnmounted(() => {
  unregisterToastApi(message)
})
</script>

<template>
  <!-- 仅用于建立 message API 桥接，不渲染可见内容。 -->
  <span aria-hidden="true" class="hidden" />
</template>
