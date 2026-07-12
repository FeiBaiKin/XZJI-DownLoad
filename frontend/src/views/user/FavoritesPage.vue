<script setup lang="ts">
// 我的收藏页：ResourceCard 列表 + 分页，取消收藏从列表移除
import { ref, onMounted } from 'vue'
import { NEmpty, NSkeleton } from 'naive-ui'
import { favoritesApi } from '@/api/favorite'
import type { PublicKeyword, PaginatedKeywords } from '@/api/content'
import { useFavoritesStore } from '@/stores/favorites'
import { useToastStore } from '@/stores/toast'
import ResourceCard from '@/components/ResourceCard.vue'
import AppPagination from '@/components/AppPagination.vue'
import { scrollToElement } from '@/utils/motion'

const favorites = useFavoritesStore()
const toast = useToastStore()

const pageSize = ref(8)
const items = ref<PublicKeyword[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(true)


onMounted(() => {
  favorites.fetchIds()
  fetchList()
})

async function fetchList() {
  loading.value = true
  try {
    const data: PaginatedKeywords = await favoritesApi.list(page.value, pageSize.value)
    items.value = data?.items ?? []
    total.value = data?.total ?? 0
  } catch {
    items.value = []
    total.value = 0
    toast.error('加载失败，请刷新重试')
  } finally {
    loading.value = false
  }
}

function handlePageSizeChange(size: number) {
  pageSize.value = size
  page.value = 1
  fetchList()
}

function handlePageChange(p: number) {
  page.value = p
  fetchList()
  scrollToElement('fav-list')
}

async function handleToggleFav(kw: PublicKeyword) {
  const nowFav = await favorites.toggle(kw.id)
  if (!nowFav) {
    // 取消收藏 -> 从列表中移除
    items.value = items.value.filter(item => item.id !== kw.id)
    total.value = Math.max(0, total.value - 1)
    // 如果当前页无数据且不是第一页，回到前一页
    if (items.value.length === 0 && page.value > 1) {
      page.value--
      fetchList()
    }
    toast.success('已取消收藏')
  }
}

function handleKeywordCopy(kw: PublicKeyword) {
  toast.success(`已复制「${kw.keyword}」`)
}
</script>

<template>
  <div>
    <h1 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">我的收藏</h1>
    <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">管理已收藏的关键词资源</p>

    <section id="fav-list" class="mt-6 scroll-mt-20">
      <!-- 加载骨架 -->
      <div v-if="loading" class="grid gap-3 sm:grid-cols-2 sm:gap-4">
        <div
          v-for="i in 8"
          :key="i"
          class="flex flex-col rounded-2xl border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900"
        >
          <div class="flex items-center gap-2.5 pr-7">
            <NSkeleton width="40" height="40" :sharp="false" />
            <div class="flex-1 space-y-2">
              <NSkeleton text width="85%" />
            </div>
          </div>
          <div class="mt-2.5 space-y-2">
            <NSkeleton text width="100%" />
            <NSkeleton text width="70%" />
          </div>
          <div class="mt-3 flex items-center justify-between">
            <NSkeleton text width="60px" height="20px" :sharp="false" />
            <NSkeleton text width="40px" height="20px" :sharp="false" />
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else-if="items.length === 0" class="rounded-2xl border border-dashed border-zinc-300 py-14 dark:border-zinc-700">
        <NEmpty description="暂无收藏">
          <template #extra>
            <span class="text-xs text-zinc-500 dark:text-zinc-400">浏览首页或分类页，点击心形图标收藏感兴趣的关键词</span>
          </template>
        </NEmpty>
      </div>

      <!-- 收藏列表 -->
      <TransitionGroup v-else name="resource-list" tag="div" class="grid gap-3 sm:grid-cols-2 sm:gap-4" appear>
        <ResourceCard
          v-for="(kw, index) in items"
          :key="kw.id"
          :keyword="kw"
          :favorited="true"
          :style="{ '--item-index': index }"
          @copy="handleKeywordCopy"
          @toggle-favorite="handleToggleFav(kw)"
        />
      </TransitionGroup>

      <!-- 分页 -->
      <div v-if="!loading && total > 0" class="mt-6">
        <AppPagination
          :page="page"
          :page-size="pageSize"
          :item-count="total"
          :page-sizes="[8, 12, 20, 50]"
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        />
      </div>
    </section>
  </div>
</template>
