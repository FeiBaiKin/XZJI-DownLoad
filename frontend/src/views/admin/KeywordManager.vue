<script setup lang="ts">
// 关键词管理（内容块编辑器）：text/image/link 块增删排序、权限设置、网盘链接解析、分类关联
// 业务逻辑与后端契约保持不变；UI 重构为 Naive UI + Tailwind
import { ref, onMounted, h } from 'vue'
import {
  NButton, NIcon, NInput, NInputNumber, NDataTable, NTag, NCheckbox, NCheckboxGroup, NSkeleton,
  type DataTableColumns,
} from 'naive-ui'
import {
  Plus, X, ArrowUp, ArrowDown, AlignLeft, Photo, Link as LinkIcon, ExternalLink, ChevronDown,
} from '@vicons/tabler'
import { contentApi, type Category, type Keyword } from '@/api/content'
import { cloudPlatforms } from '@/constants/admin'
import ConfirmDialog from '@/components/ConfirmDialog.vue'

// 块类型定义
interface TextBlock {
  _id: string
  type: 'text'
  text: string
}

interface ImageBlock {
  _id: string
  type: 'image'
  src: string
  alt?: string
}

interface LinkBlock {
  _id: string
  type: 'link'
  url: string
  label?: string
  code?: string
  cost?: number
  access?: 'free' | 'balance' | 'membership'
}

type ContentBlock = TextBlock | ImageBlock | LinkBlock

interface ParsedLink {
  platform: string
  icon?: string
  url: string
  code: string | null
}

// 生成唯一 ID
const uid = () => Date.now().toString(36) + Math.random().toString(36).slice(2, 6)

const keywords = ref<Keyword[]>([])
const loading = ref(true)
const editingId = ref<number | null>(null)
const keyword = ref('')
const blocks = ref<ContentBlock[]>([])
const showForm = ref(false)
const linkUrl = ref('')
const parsedLink = ref<ParsedLink | null>(null)
const parserOpen = ref(false)
const confirmDeleteId = ref<number | null>(null)
const categories = ref<Category[]>([])
const selectedCategoryIds = ref<number[]>([])

onMounted(() => { fetchKeywords(); fetchCategories() })

async function fetchKeywords() {
  try {
    keywords.value = await contentApi.keywords.list()
  } catch (err) {
    console.error('获取关键词失败:', err)
  } finally {
    loading.value = false
  }
}
async function fetchCategories() {
  try { categories.value = await contentApi.categories.list() } catch { /* 忽略 */ }
}
async function fetchKeywordCategories(keywordId: number) {
  try {
    // 后端返回纯 ID 数组（[]uint）
    const ids = await contentApi.keywords.categories(keywordId)
    selectedCategoryIds.value = (ids as unknown as number[])
  } catch { selectedCategoryIds.value = [] }
}

function addBlock(type: 'text' | 'image' | 'link') {
  if (type === 'text') {
    blocks.value.push({ _id: uid(), type, text: '' })
  } else if (type === 'image') {
    blocks.value.push({ _id: uid(), type, src: '' })
  } else if (type === 'link') {
    blocks.value.push({ _id: uid(), type, url: '', cost: 0 })
  }
}
function updateBlock(idx: number, field: string, value: string | number) {
  const block = blocks.value[idx]
  blocks.value[idx] = { ...block, [field]: value } as ContentBlock
}
function removeBlock(idx: number) {
  blocks.value = blocks.value.filter((_, i) => i !== idx)
}
function moveBlock(idx: number, dir: number) {
  const newIdx = idx + dir
  if (newIdx < 0 || newIdx >= blocks.value.length) return
  const updated = [...blocks.value]
  ;[updated[idx], updated[newIdx]] = [updated[newIdx]!, updated[idx]!]
  blocks.value = updated
}

function parseLink() {
  if (!linkUrl.value.trim()) return
  const raw = linkUrl.value.trim()
  const codeMatch = raw.match(/(?:提取码|提取密码|访问码|密码|[?&]?pwd|code)[：:=:\s]*([a-zA-Z0-9]{4,8})/i)
  const extractCode = codeMatch ? codeMatch[1] : ''
  // URL 按可见 ASCII 截取（分享文本里的中文说明、提取码等自然截断，与原 CJK 排除逻辑等价）
  const urlMatch = raw.match(/(https?:\/\/[\x21-\x7e]+)/i)
  const cleanUrl = urlMatch ? urlMatch[1]!.replace(/[，,。;；！!）)]+$/, '') : raw

  let platform: { name: string; icon?: string; test: RegExp } | null = null
  for (const p of cloudPlatforms) { if (p.test.test(cleanUrl)) { platform = p; break } }
  // 迅雷云盘（不在 constants 中，单独检测）
  if (!platform && /pan\.xunlei\.com/i.test(cleanUrl)) platform = { name: '迅雷云盘', icon: 'XL', test: /pan\.xunlei\.com/i }

  if (!platform) parsedLink.value = { platform: '通用链接', url: cleanUrl, code: extractCode || null }
  else parsedLink.value = { platform: platform.name, icon: platform.icon, url: cleanUrl, code: extractCode || null }
}

function addParsedLinkAsBlocks() {
  if (!parsedLink.value) return
  blocks.value.push({ _id: uid(), type: 'link', url: parsedLink.value.url, label: parsedLink.value.platform, code: parsedLink.value.code || '' })
  parsedLink.value = null
  linkUrl.value = ''
}

async function handleSubmit() {
  if (!keyword.value.trim() || blocks.value.length === 0) return
  try {
    const payload = { keyword: keyword.value.trim(), response: blocks.value }
    let savedId: number | undefined | null = editingId.value
    if (editingId.value) {
      const kw = await contentApi.keywords.update(editingId.value, payload)
      savedId = kw?.id ?? editingId.value
    } else {
      const kw = await contentApi.keywords.create(payload)
      savedId = kw?.id
    }
    if (savedId) {
      // 后端 setCategories 接收纯数组 []uint
      await contentApi.keywords.setCategories(savedId, selectedCategoryIds.value)
    }
    handleCancel()
    fetchKeywords()
  } catch (err) {
    console.error('保存失败:', err)
  }
}

function handleEdit(item: Keyword) {
  editingId.value = item.id
  keyword.value = item.keyword
  // 后端返回 response 为 JSON 字符串（块数组或纯文本），前端解析
  let resp: string | ContentBlock[] = item.response as string | ContentBlock[]
  if (typeof resp === 'string') {
    try { resp = JSON.parse(resp) } catch { /* 纯文本，保持字符串 */ }
  }
  if (Array.isArray(resp)) blocks.value = resp.map((b) => ({ ...b, _id: (b as ContentBlock)._id || uid() }) as ContentBlock)
  else blocks.value = [{ _id: uid(), type: 'text', text: typeof resp === 'string' ? resp : String(resp || '') }]
  showForm.value = true
  fetchKeywordCategories(item.id)
}

function handleDelete(id: number) {
  confirmDeleteId.value = id
}
async function doDelete() {
  const id = confirmDeleteId.value
  confirmDeleteId.value = null
  if (id == null) return
  try {
    await contentApi.keywords.delete(id)
    fetchKeywords()
  } catch (err) {
    console.error('删除失败:', err)
  }
}

function handleCancel() {
  showForm.value = false
  editingId.value = null
  keyword.value = ''
  blocks.value = []
  linkUrl.value = ''
  parsedLink.value = null
  selectedCategoryIds.value = []
}

function getPreviewText(response: Keyword['response']) {
  let resp: string | ContentBlock[] = response as string | ContentBlock[]
  if (typeof resp === 'string') {
    try { resp = JSON.parse(resp) } catch { return String(resp || '').slice(0, 60) }
  }
  if (Array.isArray(resp)) {
    return resp.map((b) => {
      if (b.type === 'text') return b.text || ''
      if (b.type === 'image') return '[图片]'
      if (b.type === 'link') return `[链接: ${b.label || b.url}]`
      return ''
    }).join(' / ').slice(0, 60) || '(空)'
  }
  return String(resp || '').slice(0, 60)
}

function blockAccess(block: ContentBlock) {
  if (block.type === 'link') {
    return block.access || (block.cost && block.cost > 0 ? 'balance' : 'free')
  }
  return 'free'
}

const accessLabels: Record<string, string> = { free: '免费', balance: '付费', membership: '会员' }
const blockTypeLabel = (t: ContentBlock['type']) => (t === 'text' ? '文本' : t === 'image' ? '图片' : '链接')

// 列表表格列
const columns: DataTableColumns<Keyword> = [
  { title: 'ID', key: 'id', width: 72, render: (row) => h('span', { class: 'font-mono text-xs text-zinc-400' }, row.id) },
  { title: '关键词', key: 'keyword', width: 200, render: (row) => h('span', { class: 'font-medium' }, row.keyword) },
  {
    title: '响应内容',
    key: 'response',
    ellipsis: true,
    render: (row) => h('span', { class: 'text-zinc-500 dark:text-zinc-400 text-[13px]' }, getPreviewText(row.response)),
  },
  {
    title: '操作',
    key: 'actions',
    width: 130,
    render: (row) =>
      h('div', { class: 'flex gap-1' }, [
        h(NButton, { size: 'tiny', quaternary: true, type: 'primary', onClick: () => handleEdit(row) }, { default: () => '编辑' }),
        h(NButton, { size: 'tiny', quaternary: true, type: 'error', onClick: () => handleDelete(row.id) }, { default: () => '删除' }),
      ]),
  },
]
</script>

<template>
  <div>
    <header class="mb-6 flex flex-wrap items-center justify-between gap-3">
      <div>
        <h2 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">关键词管理</h2>
        <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">配置关键词命中后返回的内容块</p>
      </div>
      <NButton v-if="!showForm" type="primary" @click="showForm = true">
        <template #icon><NIcon><Plus /></NIcon></template>
        添加关键词
      </NButton>
    </header>

    <!-- ── 编辑器面板 ── -->
    <section v-if="showForm" class="mb-6 overflow-hidden rounded-2xl border border-zinc-200 bg-white dark:border-zinc-800 dark:bg-zinc-900">
      <div class="flex items-center justify-between border-b border-zinc-100 px-5 py-3.5 dark:border-zinc-800">
        <h3 class="text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">{{ editingId ? '编辑关键词' : '添加关键词' }}</h3>
        <button
          type="button"
          class="flex size-8 items-center justify-center rounded-full text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-600 dark:hover:bg-zinc-800 dark:hover:text-zinc-300"
          aria-label="关闭编辑器"
          @click="handleCancel"
        >
          <NIcon :size="16"><X /></NIcon>
        </button>
      </div>

      <div class="grid gap-6 p-5 lg:grid-cols-[minmax(0,1fr)_300px]">
        <!-- 主编辑区 -->
        <div class="min-w-0 space-y-5">
          <div>
            <label class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">关键词</label>
            <NInput v-model:value="keyword" placeholder="输入关键词，例如: hello, react..." />
          </div>

          <!-- 内容块 -->
          <div>
            <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
              <span class="text-sm font-medium text-zinc-700 dark:text-zinc-300">响应内容块</span>
              <div class="flex gap-1.5">
                <NButton size="tiny" secondary @click="addBlock('text')">
                  <template #icon><NIcon><AlignLeft /></NIcon></template>
                  文本
                </NButton>
                <NButton size="tiny" secondary @click="addBlock('image')">
                  <template #icon><NIcon><Photo /></NIcon></template>
                  图片
                </NButton>
                <NButton size="tiny" secondary @click="addBlock('link')">
                  <template #icon><NIcon><LinkIcon /></NIcon></template>
                  链接
                </NButton>
              </div>
            </div>

            <div
              v-if="blocks.length === 0"
              class="rounded-xl border border-dashed border-zinc-300 py-10 text-center text-sm text-zinc-400 dark:border-zinc-700 dark:text-zinc-500"
            >
              添加文本、图片或链接块来构建响应内容
            </div>

            <div v-else class="space-y-3">
              <div
                v-for="(block, idx) in blocks"
                :key="block._id"
                class="flex gap-3 rounded-xl border border-zinc-200 p-3.5 dark:border-zinc-700/80"
              >
                <!-- 侧槽：序号 + 排序 -->
                <div class="flex shrink-0 flex-col items-center gap-1.5">
                  <span class="font-mono text-[11px] text-zinc-400 dark:text-zinc-500">{{ String(idx + 1).padStart(2, '0') }}</span>
                  <button
                    type="button"
                    class="flex size-6 items-center justify-center rounded text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700 disabled:pointer-events-none disabled:opacity-30 dark:hover:bg-zinc-800 dark:hover:text-zinc-200"
                    :disabled="idx === 0"
                    title="上移"
                    @click="moveBlock(idx, -1)"
                  >
                    <NIcon :size="14"><ArrowUp /></NIcon>
                  </button>
                  <button
                    type="button"
                    class="flex size-6 items-center justify-center rounded text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700 disabled:pointer-events-none disabled:opacity-30 dark:hover:bg-zinc-800 dark:hover:text-zinc-200"
                    :disabled="idx === blocks.length - 1"
                    title="下移"
                    @click="moveBlock(idx, 1)"
                  >
                    <NIcon :size="14"><ArrowDown /></NIcon>
                  </button>
                </div>

                <!-- 块内容 -->
                <div class="min-w-0 flex-1 space-y-2.5">
                  <div class="flex items-center justify-between">
                    <NTag size="small" :bordered="false" :type="block.type === 'link' ? 'info' : 'default'">
                      {{ blockTypeLabel(block.type) }}
                    </NTag>
                    <button
                      type="button"
                      class="flex size-6 items-center justify-center rounded text-zinc-400 transition-colors hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-500/10"
                      title="删除此块"
                      @click="removeBlock(idx)"
                    >
                      <NIcon :size="13"><X /></NIcon>
                    </button>
                  </div>

                  <!-- 文本块 -->
                  <NInput
                    v-if="block.type === 'text'"
                    type="textarea"
                    :rows="3"
                    placeholder="输入文本内容..."
                    :value="block.text || ''"
                    @update:value="(v: string) => updateBlock(idx, 'text', v)"
                  />

                  <!-- 图片块 -->
                  <template v-else-if="block.type === 'image'">
                    <NInput
                      placeholder="图片 URL 地址..."
                      :value="block.src || ''"
                      @update:value="(v: string) => updateBlock(idx, 'src', v)"
                    />
                    <NInput
                      placeholder="图片描述 (alt, 可选)..."
                      :value="block.alt || ''"
                      @update:value="(v: string) => updateBlock(idx, 'alt', v)"
                    />
                    <img
                      v-if="block.src"
                      :src="block.src"
                      :alt="block.alt || ''"
                      class="max-h-40 rounded-lg border border-zinc-200 dark:border-zinc-700"
                      @error="(e) => ((e.target as HTMLImageElement).style.display = 'none')"
                    />
                  </template>

                  <!-- 链接块 -->
                  <template v-else-if="block.type === 'link'">
                    <NInput
                      placeholder="链接 URL..."
                      :value="block.url || ''"
                      @update:value="(v: string) => updateBlock(idx, 'url', v)"
                    />
                    <div class="grid gap-2.5 sm:grid-cols-2">
                      <NInput
                        placeholder="显示名称 (可选)"
                        :value="block.label || ''"
                        @update:value="(v: string) => updateBlock(idx, 'label', v)"
                      />
                      <NInput
                        placeholder="提取码 (可选)"
                        :value="block.code || ''"
                        @update:value="(v: string) => updateBlock(idx, 'code', v)"
                      />
                    </div>
                    <div class="flex flex-wrap items-center gap-3">
                      <span class="text-xs text-zinc-400 dark:text-zinc-500">权限</span>
                      <div class="flex overflow-hidden rounded-lg border border-zinc-200 dark:border-zinc-700">
                        <button
                          v-for="mode in (['free', 'balance', 'membership'] as const)"
                          :key="mode"
                          type="button"
                          class="px-3 py-1.5 text-xs font-medium transition-colors"
                          :class="blockAccess(block) === mode
                            ? 'bg-emerald-600 text-white dark:bg-emerald-400 dark:text-zinc-950'
                            : 'text-zinc-500 hover:bg-zinc-50 hover:text-zinc-800 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-200'"
                          @click="updateBlock(idx, 'access', mode)"
                        >
                          {{ accessLabels[mode] }}
                        </button>
                      </div>
                      <NInputNumber
                        v-if="blockAccess(block) === 'balance' || blockAccess(block) === 'free'"
                        size="small"
                        class="w-32"
                        placeholder="价格"
                        :min="0"
                        :step="0.01"
                        :disabled="blockAccess(block) === 'free'"
                        :value="blockAccess(block) === 'free' ? 0 : (block.cost || 0)"
                        @update:value="(v: number | null) => updateBlock(idx, 'cost', v ?? 0)"
                      >
                        <template #prefix><span class="text-xs text-zinc-400">¥</span></template>
                      </NInputNumber>
                    </div>
                  </template>
                </div>
              </div>
            </div>
          </div>

          <!-- 分类关联 -->
          <div v-if="categories.length > 0">
            <label class="mb-2 block text-sm font-medium text-zinc-700 dark:text-zinc-300">所属分类</label>
            <NCheckboxGroup
              :value="selectedCategoryIds"
              @update:value="(v: (string | number)[]) => selectedCategoryIds = v as number[]"
            >
              <div class="flex flex-wrap gap-x-5 gap-y-2">
                <NCheckbox v-for="c in categories" :key="c.id" :value="c.id" :label="c.name" />
              </div>
            </NCheckboxGroup>
          </div>

          <div class="flex gap-2 border-t border-zinc-100 pt-4 dark:border-zinc-800">
            <NButton type="primary" @click="handleSubmit">{{ editingId ? '保存修改' : '添加关键词' }}</NButton>
            <NButton quaternary @click="handleCancel">取消</NButton>
          </div>
        </div>

        <!-- 侧栏：预览 + 解析器 -->
        <aside class="space-y-4">
          <!-- 响应预览 -->
          <div class="rounded-xl border border-zinc-200 dark:border-zinc-700/80">
            <div class="flex items-center justify-between border-b border-zinc-100 px-4 py-2.5 dark:border-zinc-800">
              <span class="text-xs font-medium text-zinc-500 dark:text-zinc-400">响应预览</span>
              <span class="font-mono text-[11px] text-zinc-400 dark:text-zinc-500">{{ blocks.length }} 块</span>
            </div>
            <div class="max-h-80 space-y-2.5 overflow-y-auto p-4">
              <p v-if="blocks.length === 0" class="text-xs text-zinc-400 dark:text-zinc-500">添加内容块后此处将显示预览</p>
              <template v-for="(block, idx) in blocks" :key="idx">
                <p v-if="block.type === 'text'" class="text-[13px] leading-relaxed whitespace-pre-wrap text-zinc-600 dark:text-zinc-300">
                  {{ block.text || '(空文本)' }}
                </p>
                <div v-else-if="block.type === 'image'">
                  <img v-if="block.src" :src="block.src" alt="" class="max-h-28 rounded-lg" />
                  <span v-else class="text-xs text-zinc-400">[图片]</span>
                </div>
                <div
                  v-else-if="block.type === 'link'"
                  class="flex items-center gap-2 rounded-lg border border-zinc-200 px-2.5 py-2 text-[13px] dark:border-zinc-700"
                >
                  <NIcon :size="13" class="shrink-0 text-emerald-500"><ExternalLink /></NIcon>
                  <span class="min-w-0 flex-1 truncate text-zinc-700 dark:text-zinc-300">{{ block.label || block.url || '(空链接)' }}</span>
                  <span v-if="block.code" class="shrink-0 font-mono text-[11px] text-zinc-400">{{ block.code }}</span>
                </div>
              </template>
            </div>
          </div>

          <!-- 链接解析器 -->
          <div class="rounded-xl border border-zinc-200 dark:border-zinc-700/80">
            <button
              type="button"
              class="flex w-full items-center justify-between px-4 py-2.5 text-left"
              @click="parserOpen = !parserOpen"
            >
              <span>
                <span class="block text-xs font-medium text-zinc-500 dark:text-zinc-400">链接解析器</span>
                <span class="mt-0.5 block text-[11px] text-zinc-400 dark:text-zinc-500">从网盘分享链接自动提取</span>
              </span>
              <NIcon :size="15" class="text-zinc-400 transition-transform" :class="parserOpen ? 'rotate-180' : ''"><ChevronDown /></NIcon>
            </button>
            <div v-if="parserOpen" class="space-y-3 border-t border-zinc-100 p-4 dark:border-zinc-800">
              <div class="flex gap-2">
                <NInput
                  v-model:value="linkUrl"
                  size="small"
                  class="flex-1"
                  placeholder="粘贴网盘链接..."
                  @input="parsedLink = null"
                />
                <NButton size="small" secondary @click="parseLink">解析</NButton>
              </div>
              <div v-if="parsedLink" class="space-y-2 rounded-lg bg-zinc-50 p-3 text-[13px] dark:bg-zinc-800/60">
                <div class="flex items-center justify-between gap-2">
                  <span class="text-zinc-400 dark:text-zinc-500">平台</span>
                  <span class="flex items-center gap-1.5 font-medium text-zinc-800 dark:text-zinc-200">
                    <span
                      v-if="parsedLink.icon"
                      class="grid size-5 place-items-center rounded bg-emerald-100 font-mono text-[10px] text-emerald-700 dark:bg-emerald-400/15 dark:text-emerald-400"
                    >{{ parsedLink.icon }}</span>
                    {{ parsedLink.platform }}
                  </span>
                </div>
                <div class="flex items-center justify-between gap-2">
                  <span class="shrink-0 text-zinc-400 dark:text-zinc-500">链接</span>
                  <span class="truncate font-mono text-xs text-zinc-600 dark:text-zinc-300">{{ parsedLink.url }}</span>
                </div>
                <div v-if="parsedLink.code" class="flex items-center justify-between gap-2">
                  <span class="text-zinc-400 dark:text-zinc-500">提取码</span>
                  <span class="font-mono text-xs font-semibold text-emerald-600 dark:text-emerald-400">{{ parsedLink.code }}</span>
                </div>
                <NButton size="small" type="primary" secondary block class="mt-1" @click="addParsedLinkAsBlocks">添加为链接块</NButton>
              </div>
            </div>
          </div>
        </aside>
      </div>
    </section>

    <!-- ── 关键词列表 ── -->
    <div v-if="loading" class="space-y-2">
      <NSkeleton v-for="i in 5" :key="i" height="44px" :sharp="false" />
    </div>
    <NDataTable
      v-else
      :columns="columns"
      :data="keywords"
      :row-key="(row: Keyword) => row.id"
      :bordered="false"
      class="overflow-hidden rounded-xl border border-zinc-200 dark:border-zinc-800"
    />

    <ConfirmDialog
      :open="confirmDeleteId !== null"
      title="删除关键词"
      message="确定删除此关键词？"
      confirm-label="删除"
      danger
      @confirm="doDelete"
      @cancel="confirmDeleteId = null"
    />
  </div>
</template>
