<script setup lang="ts">
import { computed } from 'vue'
import { NPagination } from 'naive-ui'

const props = withDefaults(defineProps<{
  page: number
  pageSize: number
  itemCount: number
  pageSizes?: number[]
}>(), {
  pageSizes: () => [10, 20, 50, 100],
})

const emit = defineEmits<{
  'update:page': [page: number]
  'update:page-size': [pageSize: number]
}>()

const availablePageSizes = computed(() => {
  return [...new Set([...props.pageSizes, props.pageSize])]
    .filter(size => Number.isInteger(size) && size > 0)
    .sort((a, b) => a - b)
})
</script>

<template>
  <div class="flex flex-wrap items-center justify-center gap-3 sm:justify-end">
    <span class="text-xs text-zinc-500 dark:text-zinc-400">共 {{ itemCount }} 条</span>
    <NPagination
      :page="page"
      :page-size="pageSize"
      :item-count="itemCount"
      :page-sizes="availablePageSizes"
      show-size-picker
      show-quick-jumper
      @update:page="emit('update:page', $event)"
      @update:page-size="emit('update:page-size', $event)"
    />
  </div>
</template>
