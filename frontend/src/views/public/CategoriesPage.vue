<script setup lang="ts">
// 分类页：分类切换 + 资源列表（使用 ResourceCard 组件）
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { NEmpty, NSkeleton } from 'naive-ui'
import { contentApi, type Category, type PublicKeyword } from '@/api/content'
import { useToastStore } from '@/stores/toast'
import { useFavoritesStore } from '@/stores/favorites'
import ResourceCard from '@/components/ResourceCard.vue'
import AppPagination from '@/components/AppPagination.vue'

const toast = useToastStore()
const router = useRouter()
const favorites = useFavoritesStore()

const pageSize = ref(8)

const categories = ref<Category[]>([])
const selectedCategory = ref<number | null>(null)
const keywords = ref<PublicKeyword[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(true)

const currentCategoryName = computed(() => {
  if (selectedCategory.value == null) return '全部分类'
  return categories.value.find((c) => c.id === selectedCategory.value)?.name || '分类'
})

onMounted(async () => {
  favorites.fetchIds()
  const catParam = router.currentRoute.value.query.category
  if (catParam) {
    const id = Number(catParam)
    if (!isNaN(id)) selectedCategory.value = id
  }
  await fetchCategories()
  await fetchKeywords()
})

async function fetchCategories() {
  try {
    categories.value = await contentApi.public.categories()
  } catch {
    categories.value = []
  }
}

async function fetchKeywords() {
  loading.value = true
  try {
    const data = await contentApi.public.keywords(
      selectedCategory.value ?? undefined,
      page.value,
      pageSize.value,
    )
    keywords.value = data?.items ?? []
    total.value = data?.total ?? 0
  } catch {
    keywords.value = []
    total.value = 0
    toast.error('加载失败，请刷新重试')
  } finally {
    loading.value = false
  }
}

function selectCategory(id: number | null) {
  if (selectedCategory.value === id) return
  selectedCategory.value = id
  page.value = 1
  router.replace({ query: id ? { category: String(id) } : {} })
  fetchKeywords()
}

function handlePageSizeChange(size: number) {
  pageSize.value = size
  page.value = 1
  fetchKeywords()
}

function handlePageChange(p: number) {
  page.value = p
  fetchKeywords()
  document.getElementById('kw-list')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

function handleKeywordCopy(kw: PublicKeyword) {
  toast.success(`已复制「${kw.keyword}」，粘贴到首页搜索框获取资源`)
}

function pillClass(active: boolean) {
  return active
    ? 'border-emerald-600 bg-emerald-600 text-white dark:border-emerald-400 dark:bg-emerald-400 dark:text-zinc-950'
    : 'border-zinc-200 text-zinc-600 hover:border-emerald-400/60 hover:text-emerald-600 dark:border-zinc-700 dark:text-zinc-400 dark:hover:border-emerald-400/40 dark:hover:text-emerald-400'
}
</script>

<template>
  <div>
    <!-- 分类切换 -->
    <section class="pt-14 sm:pt-20">
      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          class="rounded-full border px-3.5 py-1.5 text-[13px] font-medium motion-safe:transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:focus-visible:ring-emerald-400/60"
          :class="pillClass(selectedCategory === null)"
          @click="selectCategory(null)"
        >全部</button>
        <button
          v-for="c in categories"
          :key="c.id"
          type="button"
          class="rounded-full border px-3.5 py-1.5 text-[13px] font-medium motion-safe:transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:focus-visible:ring-emerald-400/60"
          :class="pillClass(selectedCategory === c.id)"
          :title="c.description || c.name"
          @click="selectCategory(c.id)"
        >{{ c.name }}</button>
      </div>
    </section>

    <!-- 关键词列表 -->
    <section id="kw-list" class="mt-6 scroll-mt-20 pb-10">
      <div class="mb-4 flex items-baseline justify-between">
        <h2 class="text-lg font-semibold tracking-tight text-zinc-900 dark:text-zinc-100">
          {{ currentCategoryName }}
        </h2>
        <span v-if="!loading" class="font-mono text-xs text-zinc-500 dark:text-zinc-400">
          共 {{ total }} 个
        </span>
      </div>

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
      <div v-else-if="keywords.length === 0" class="rounded-2xl border border-dashed border-zinc-300 py-14 dark:border-zinc-700">
        <NEmpty :description="selectedCategory != null ? '该分类下暂无关键词' : '暂无关键词'">
          <template #extra>
            <span class="text-xs text-zinc-500 dark:text-zinc-400">试试其他分类，或在后台添加关键词</span>
          </template>
        </NEmpty>
      </div>

      <!-- 资源列表（两列网格卡片） -->
      <div v-else class="grid gap-3 sm:grid-cols-2 sm:gap-4">
        <ResourceCard
          v-for="kw in keywords"
          :key="kw.id"
          :keyword="kw"
          :category-name="currentCategoryName !== '全部分类' ? currentCategoryName : undefined"
          :favorited="favorites.has(kw.id)"
          @copy="handleKeywordCopy"
          @toggle-favorite="favorites.toggle(kw.id)"
        />
      </div>

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
