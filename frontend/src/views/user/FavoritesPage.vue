<script setup lang="ts">
// 我的收藏页：ResourceCard 列表 + 分页，取消收藏从列表移除
import { ref, computed, onMounted } from 'vue'
import { NEmpty, NSkeleton, NPagination } from 'naive-ui'
import { favoritesApi } from '@/api/favorite'
import type { PublicKeyword, PaginatedKeywords } from '@/api/content'
import { useFavoritesStore } from '@/stores/favorites'
import { useToastStore } from '@/stores/toast'
import ResourceCard from '@/components/ResourceCard.vue'

const favorites = useFavoritesStore()
const toast = useToastStore()

const PAGE_SIZE = 8
const items = ref<PublicKeyword[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(true)

const pageCount = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)))

onMounted(() => {
  favorites.fetchIds()
  fetchList()
})

async function fetchList() {
  loading.value = true
  try {
    const data: PaginatedKeywords = await favoritesApi.list(page.value, PAGE_SIZE)
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

function handlePageChange(p: number) {
  page.value = p
  fetchList()
  document.getElementById('fav-list')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
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
    <h2 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">我的收藏</h2>
    <p class="mt-2 text-sm text-zinc-500 dark:text-zinc-400">管理已收藏的关键词资源</p>

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
      <div v-else class="grid gap-3 sm:grid-cols-2 sm:gap-4">
        <ResourceCard
          v-for="kw in items"
          :key="kw.id"
          :keyword="kw"
          :favorited="true"
          @copy="handleKeywordCopy"
          @toggle-favorite="handleToggleFav(kw)"
        />
      </div>

      <!-- 分页 -->
      <div v-if="!loading && total > PAGE_SIZE" class="mt-6 flex justify-center">
        <NPagination
          :page="page"
          :page-count="pageCount"
          :item-count="total"
          show-quick-jumper
          @update:page="handlePageChange"
        />
      </div>
    </section>
  </div>
</template>
