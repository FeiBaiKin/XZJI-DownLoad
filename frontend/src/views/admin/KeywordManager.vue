<script setup lang="ts">
// 关键词管理：列表全宽展示 + 右侧抽屉编辑/新增
// 点击「添加关键词」或「编辑」打开右侧抽屉，保存后列表原地刷新
// 业务逻辑与后端契约保持不变
import { ref, onMounted, computed, h } from 'vue'
import {
  NButton, NIcon, NInput, NInputNumber, NDataTable, NTag, NCheckbox, NCheckboxGroup,
  NSelect, NDrawer, NDrawerContent, NModal, NRadioGroup, NRadio, NUpload,
  type DataTableColumns, type UploadFileInfo,
} from 'naive-ui'
import {
  PhPlus, PhX, PhArrowUp, PhArrowDown, PhTextAlignLeft, PhImage, PhLink as LinkIcon, PhArrowSquareOut, PhCaretDown, PhTrash,
  PhDownloadSimple, PhUploadSimple, PhEye, PhEyeSlash,
} from '@phosphor-icons/vue'
import { contentApi, type Category, type Keyword } from '@/api/content'
import type { KeywordAttribute, KeywordStatus } from '@/types/api'
import { extractErrorMessage } from '@/types/error'
import { cloudPlatforms } from '@/constants/admin'
import { useToastStore } from '@/stores/toast'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AppPagination from '@/components/AppPagination.vue'
import IconCropperModal from '@/components/IconCropperModal.vue'

const toast = useToastStore()

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
const drawerOpen = ref(false)
const keyword = ref('')
const keywordStatus = ref<KeywordStatus>('published')
const blocks = ref<ContentBlock[]>([])
const linkUrl = ref('')
const parsedLink = ref<ParsedLink | null>(null)
const parserOpen = ref(false)
const previewOpen = ref(false)
const confirmDeleteId = ref<number | null>(null)
const categories = ref<Category[]>([])
const selectedCategoryIds = ref<number[]>([])
// 关键词图标
const keywordIcon = ref<string>('')
const iconPickerOpen = ref(false)
// 关键词属性
const attributes = ref<KeywordAttribute[]>([])
// 列表分页 + 分类/状态筛选
const page = ref(1)
const total = ref(0)
const limit = ref(10)
const filterCategoryId = ref<number | null>(null)
const filterStatus = ref<KeywordStatus | ''>('')
const checkedRowKeys = ref<Array<string | number>>([])
// 导入
const importOpen = ref(false)
const importMode = ref<'skip' | 'overwrite'>('skip')
const importBusy = ref(false)
const importResult = ref<{ created: number; updated: number; skipped: number; failed: number; errors?: string[] } | null>(null)
const exportBusy = ref(false)

const categoryNameMap = computed(() => {
  const m = new Map<number, string>()
  for (const c of categories.value) m.set(c.id, c.name)
  return m
})
const categoryFilterOptions = computed(() => [
  { label: '全部分类', value: 0 },
  ...categories.value.map((c) => ({ label: c.name, value: c.id })),
])
const statusFilterOptions = [
  { label: '全部状态', value: '' },
  { label: '已上架', value: 'published' },
  { label: '已下架', value: 'draft' },
]

onMounted(() => { fetchKeywords(); fetchCategories() })

async function fetchKeywords() {
  loading.value = true
  try {
    const res = await contentApi.keywords.list({
      page: page.value,
      limit: limit.value,
      category: filterCategoryId.value ?? undefined,
      status: filterStatus.value || undefined,
    })
    keywords.value = res.items
    total.value = res.total
    checkedRowKeys.value = []
    // 当前页被删空且非首页时回退一页
    if (res.items.length === 0 && page.value > 1) {
      page.value -= 1
      return fetchKeywords()
    }
  } catch (err) {
    toast.error(extractErrorMessage(err, '获取关键词失败，请刷新重试'))
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

function setPage(p: number) { page.value = p; fetchKeywords() }
function setPageSize(size: number) { limit.value = size; page.value = 1; fetchKeywords() }
function onFilterChange(v: number) {
  filterCategoryId.value = v > 0 ? v : null
  page.value = 1
  fetchKeywords()
}
function onStatusFilterChange(v: KeywordStatus | '') {
  filterStatus.value = v || ''
  page.value = 1
  fetchKeywords()
}

function statusOf(row: Keyword): KeywordStatus {
  return row.status === 'draft' ? 'draft' : 'published'
}

async function toggleRowStatus(row: Keyword) {
  const next: KeywordStatus = statusOf(row) === 'published' ? 'draft' : 'published'
  try {
    await contentApi.keywords.update(row.id, { status: next })
    toast.success(next === 'published' ? '已上架' : '已下架')
    fetchKeywords()
  } catch (err) {
    toast.error(extractErrorMessage(err, '更新状态失败'))
  }
}

async function batchSetStatus(status: KeywordStatus) {
  const ids = checkedRowKeys.value.map(Number).filter((n) => n > 0)
  if (ids.length === 0) {
    toast.error('请先勾选关键词')
    return
  }
  try {
    const res = await contentApi.keywords.batchStatus(ids, status)
    toast.success(`已更新 ${res.updated} 条`)
    fetchKeywords()
  } catch (err) {
    toast.error(extractErrorMessage(err, '批量更新失败'))
  }
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

async function handleExport(format: 'json' | 'csv') {
  exportBusy.value = true
  try {
    const data = await contentApi.keywords.export({
      format,
      category: filterCategoryId.value ?? undefined,
      status: filterStatus.value || undefined,
    })
    if (format === 'csv') {
      downloadBlob(data as Blob, 'keywords-export.csv')
    } else {
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
      downloadBlob(blob, 'keywords-export.json')
    }
    toast.success('导出成功')
  } catch (err) {
    toast.error(extractErrorMessage(err, '导出失败'))
  } finally {
    exportBusy.value = false
  }
}

async function onImportFile({ file }: { file: UploadFileInfo }) {
  if (!file.file) return
  if (importMode.value === 'overwrite') {
    const ok = window.confirm(`确认以「覆盖已存在」模式导入文件「${file.name || file.file.name}」？\n已存在的关键词内容将被覆盖，此操作不可撤销。`)
    if (!ok) return
  }
  importBusy.value = true
  importResult.value = null
  try {
    const res = await contentApi.keywords.importFile(file.file, importMode.value)
    importResult.value = res
    toast.success(`导入完成：新建 ${res.created} / 更新 ${res.updated} / 跳过 ${res.skipped}`)
    fetchKeywords()
  } catch (err) {
    toast.error(extractErrorMessage(err, '导入失败'))
  } finally {
    importBusy.value = false
  }
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
// 切换链接权限：非付费态同步清零 cost，避免残留脏数据被提交
function setAccess(idx: number, mode: 'free' | 'balance' | 'membership') {
  const block = blocks.value[idx]
  const patch = mode === 'balance' ? { access: mode } : { access: mode, cost: 0 }
  blocks.value[idx] = { ...block, ...patch } as ContentBlock
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
    // attributes 序列化为 JSON 字符串，空数组保留空字符串避免阻塞
    const attrsStr = attributes.value.length > 0 ? JSON.stringify(attributes.value) : ''
    const payload = {
      keyword: keyword.value.trim(),
      response: blocks.value,
      icon: keywordIcon.value,
      attributes: attrsStr,
      status: keywordStatus.value,
    }
    let savedId: number | undefined | null = editingId.value
    if (editingId.value) {
      const kw = await contentApi.keywords.update(editingId.value, payload)
      savedId = kw?.id ?? editingId.value
    } else {
      const kw = await contentApi.keywords.create({
        keyword: payload.keyword,
        response: payload.response,
        status: payload.status,
      })
      savedId = kw?.id
      if (savedId && (keywordIcon.value || attrsStr)) {
        await contentApi.keywords.update(savedId, { icon: keywordIcon.value, attributes: attrsStr, status: keywordStatus.value })
      }
    }
    if (savedId) {
      await contentApi.keywords.setCategories(savedId, selectedCategoryIds.value)
    }
    toast.success(editingId.value ? '保存成功' : '添加成功')
    handleCancel()
    fetchKeywords()
  } catch (err) {
    toast.error(extractErrorMessage(err, '保存失败，请重试'))
    console.error('保存失败:', err)
  }
}

function handleEdit(item: Keyword) {
  editingId.value = item.id
  keyword.value = item.keyword
  keywordStatus.value = statusOf(item)
  // 后端返回 response 为 JSON 字符串（块数组或纯文本），前端解析
  let resp: string | ContentBlock[] = item.response as string | ContentBlock[]
  if (typeof resp === 'string') {
    try { resp = JSON.parse(resp) } catch { /* 纯文本，保持字符串 */ }
  }
  if (Array.isArray(resp)) blocks.value = resp.map((b) => ({ ...b, _id: (b as ContentBlock)._id || uid() }) as ContentBlock)
  else blocks.value = [{ _id: uid(), type: 'text', text: typeof resp === 'string' ? resp : String(resp || '') }]
  // 图标 & 属性
  keywordIcon.value = item.icon || ''
  let attrs: KeywordAttribute[] = []
  if (item.attributes) {
    try {
      const parsed = JSON.parse(item.attributes)
      if (Array.isArray(parsed)) attrs = parsed.filter((a) => a && typeof a.label === 'string' && typeof a.value === 'string')
    } catch { /* 忽略损坏的 JSON */ }
  }
  attributes.value = attrs
  // 分类：优先用列表附带的 category_ids，回退到接口查询
  if (Array.isArray(item.category_ids)) selectedCategoryIds.value = [...item.category_ids]
  else fetchKeywordCategories(item.id)
  drawerOpen.value = true
}

// 打开新增抽屉：重置表单后展开
function openCreate() {
  handleCancel()
  drawerOpen.value = true
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
    // 删除的正是正在编辑的关键词时，重置表单
    if (editingId.value === id) handleCancel()
    toast.success('删除成功')
    fetchKeywords()
  } catch (err) {
    toast.error(extractErrorMessage(err, '删除失败，请重试'))
    console.error('删除失败:', err)
  }
}

function handleCancel() {
  editingId.value = null
  keyword.value = ''
  keywordStatus.value = 'published'
  blocks.value = []
  linkUrl.value = ''
  parsedLink.value = null
  selectedCategoryIds.value = []
  keywordIcon.value = ''
  attributes.value = []
  iconPickerOpen.value = false
  drawerOpen.value = false
}

// 图标
function onIconSaved(icon: string) {
  keywordIcon.value = icon
  // 编辑态下后端已写入 keyword_responses.icon，但表单保存时仍会再次写一次以保持一致
}

// 属性增删改
function addAttribute() {
  attributes.value.push({ label: '', value: '' })
}
function updateAttribute(idx: number, field: 'label' | 'value', v: string) {
  attributes.value[idx] = { ...attributes.value[idx]!, [field]: v }
}
function removeAttribute(idx: number) {
  attributes.value = attributes.value.filter((_, i) => i !== idx)
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
  { type: 'selection', width: 40 },
  { title: 'ID', key: 'id', width: 60, render: (row) => h('span', { class: 'font-mono text-xs text-zinc-500 dark:text-zinc-400' }, row.id) },
  { title: '关键词', key: 'keyword', width: 140, render: (row) => h('span', { class: 'font-medium' }, row.keyword) },
  {
    title: '状态',
    key: 'status',
    width: 88,
    render: (row) => {
      const published = statusOf(row) === 'published'
      return h(
        NTag,
        { size: 'small', bordered: false, type: published ? 'success' : 'default' },
        { default: () => (published ? '已上架' : '已下架') },
      )
    },
  },
  {
    title: '分类',
    key: 'categories',
    width: 140,
    render: (row) => {
      const ids = row.category_ids || []
      if (ids.length === 0) return h('span', { class: 'text-xs text-zinc-400 dark:text-zinc-500' }, '-')
      return h(
        'div',
        { class: 'flex flex-wrap gap-1' },
        ids
          .map((id) => {
            const name = categoryNameMap.value.get(id)
            return name ? h(NTag, { size: 'small', bordered: false, key: id }, { default: () => name }) : null
          })
          .filter(Boolean),
      )
    },
  },
  {
    title: '响应内容',
    key: 'response',
    ellipsis: true,
    render: (row) => h('span', { class: 'text-zinc-500 dark:text-zinc-400 text-[13px]' }, getPreviewText(row.response)),
  },
  {
    title: '操作',
    key: 'actions',
    width: 168,
    render: (row) => {
      const published = statusOf(row) === 'published'
      return h('div', { class: 'flex flex-wrap gap-1' }, [
        h(NButton, { size: 'tiny', quaternary: true, type: 'primary', onClick: () => handleEdit(row) }, { default: () => '编辑' }),
        h(
          NButton,
          { size: 'tiny', quaternary: true, onClick: () => toggleRowStatus(row) },
          { default: () => (published ? '下架' : '上架') },
        ),
        h(NButton, { size: 'tiny', quaternary: true, type: 'error', onClick: () => handleDelete(row.id) }, { default: () => '删除' }),
      ])
    },
  },
]
</script>

<template>
  <div>
    <header class="mb-6 flex flex-wrap items-end justify-between gap-3">
      <div>
        <h1 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">关键词管理</h1>
        <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">资源上下架、批量导入导出与内容编辑</p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <NButton secondary :loading="exportBusy" @click="handleExport('json')">
          <template #icon><NIcon><PhDownloadSimple /></NIcon></template>
          导出 JSON
        </NButton>
        <NButton secondary :loading="exportBusy" @click="handleExport('csv')">
          <template #icon><NIcon><PhDownloadSimple /></NIcon></template>
          导出 CSV
        </NButton>
        <NButton secondary @click="importOpen = true; importResult = null">
          <template #icon><NIcon><PhUploadSimple /></NIcon></template>
          导入
        </NButton>
        <NButton type="primary" @click="openCreate">
          <template #icon><NIcon><PhPlus /></NIcon></template>
          添加关键词
        </NButton>
      </div>
    </header>

    <!-- ══ 右侧抽屉：添加 / 编辑表单 ══ -->
    <NDrawer v-model:show="drawerOpen" width="min(520px, 100vw)" placement="right" :auto-focus="false">
      <NDrawerContent :title="editingId ? '编辑关键词' : '添加关键词'" closable>
        <div class="space-y-5">
          <div>
            <label class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">关键词</label>
            <NInput v-model:value="keyword" placeholder="输入关键词，例如: hello, react..." />
          </div>

          <div>
            <label class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">资源状态</label>
            <div class="flex overflow-hidden rounded-xl border border-zinc-200 dark:border-zinc-700">
              <button
                type="button"
                class="flex flex-1 items-center justify-center gap-1.5 px-3 py-2 text-sm font-medium transition-colors"
                :class="keywordStatus === 'published'
                  ? 'bg-emerald-600 text-white dark:bg-emerald-400 dark:text-zinc-950'
                  : 'text-zinc-500 hover:bg-zinc-50 dark:text-zinc-400 dark:hover:bg-zinc-800'"
                @click="keywordStatus = 'published'"
              >
                <NIcon :size="15"><PhEye /></NIcon>
                已上架
              </button>
              <button
                type="button"
                class="flex flex-1 items-center justify-center gap-1.5 px-3 py-2 text-sm font-medium transition-colors"
                :class="keywordStatus === 'draft'
                  ? 'bg-zinc-800 text-white dark:bg-zinc-200 dark:text-zinc-900'
                  : 'text-zinc-500 hover:bg-zinc-50 dark:text-zinc-400 dark:hover:bg-zinc-800'"
                @click="keywordStatus = 'draft'"
              >
                <NIcon :size="15"><PhEyeSlash /></NIcon>
                已下架
              </button>
            </div>
            <p class="mt-1.5 text-[11px] text-zinc-500 dark:text-zinc-400">
              下架后前台搜索与列表不可见，后台仍可编辑
            </p>
          </div>

          <!-- 图标 + 属性：抽屉窄宽下垂直堆叠，避免属性值输入框被挤压 -->
          <div class="grid gap-5">
            <!-- 图标选择 -->
            <div>
              <label class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">关键词图标</label>
              <div class="flex items-center gap-3">
                <div
                  class="grid size-16 shrink-0 place-items-center overflow-hidden rounded-2xl border border-dashed border-zinc-300 bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-800/60"
                >
                  <img v-if="keywordIcon" :src="keywordIcon" alt="图标" class="size-full object-cover" />
                  <NIcon v-else :size="22" class="text-zinc-300 dark:text-zinc-600"><PhImage /></NIcon>
                </div>
                <div class="flex flex-col gap-1.5">
                  <NButton
                    size="small"
                    secondary
                    :disabled="!editingId"
                    :title="!editingId ? '请先保存关键词后再上传图标' : '从媒体库选图并裁剪为图标'"
                    @click="iconPickerOpen = true"
                  >
                    <template #icon><NIcon><PhImage /></NIcon></template>
                    {{ keywordIcon ? '更换图标' : '从媒体库选择' }}
                  </NButton>
                  <NButton v-if="keywordIcon" size="small" quaternary type="error" @click="keywordIcon = ''">
                    清除
                  </NButton>
                  <p class="text-[11px] text-zinc-500 dark:text-zinc-400">
                    {{ editingId ? '从媒体库选图后裁成 256×256 方图' : '保存关键词后可上传图标' }}
                  </p>
                </div>
              </div>
            </div>

            <!-- 属性编辑 -->
            <div>
              <div class="mb-1.5 flex items-center justify-between">
                <label class="block text-sm font-medium text-zinc-700 dark:text-zinc-300">属性</label>
                <NButton size="tiny" quaternary @click="addAttribute">
                  <template #icon><NIcon><PhPlus /></NIcon></template>
                  添加
                </NButton>
              </div>
              <p v-if="attributes.length === 0" class="rounded-xl border border-dashed border-zinc-300 px-3 py-2.5 text-xs text-zinc-500 dark:border-zinc-700 dark:text-zinc-400">
                例如：大小=56MB / 平台=安卓
              </p>
              <div v-else class="space-y-2">
                <div
                  v-for="(attr, idx) in attributes"
                  :key="idx"
                  class="flex flex-col gap-2 sm:flex-row sm:items-center"
                >
                  <NInput
                    size="small"
                    class="!w-full sm:!w-28 sm:shrink-0"
                    placeholder="属性名"
                    :value="attr.label"
                    @update:value="(v: string) => updateAttribute(idx, 'label', v)"
                  />
                  <span class="text-zinc-400 dark:text-zinc-500">=</span>
                  <NInput
                    size="small"
                    class="flex-1"
                    placeholder="属性值"
                    :value="attr.value"
                    @update:value="(v: string) => updateAttribute(idx, 'value', v)"
                  />
                  <button
                    type="button"
                    class="flex size-7 shrink-0 items-center justify-center rounded-lg text-zinc-400 transition-colors hover:bg-red-50 hover:text-red-500 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-emerald-500 dark:hover:bg-red-500/10"
                    title="删除"
                    aria-label="删除属性"
                    @click="removeAttribute(idx)"
                  >
                    <NIcon :size="14"><PhTrash /></NIcon>
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- 内容块 -->
          <div>
            <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
              <span class="text-sm font-medium text-zinc-700 dark:text-zinc-300">响应内容块</span>
              <div class="flex gap-1.5">
                <NButton size="tiny" secondary @click="addBlock('text')">
                  <template #icon><NIcon><PhTextAlignLeft /></NIcon></template>
                  文本
                </NButton>
                <NButton size="tiny" secondary @click="addBlock('image')">
                  <template #icon><NIcon><PhImage /></NIcon></template>
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
                class="flex flex-col gap-3 rounded-xl border border-zinc-200 p-3.5 sm:flex-row dark:border-zinc-700/80"
              >
                <!-- 侧槽：序号 + 排序 -->
                <div class="flex shrink-0 flex-row items-center gap-1.5 sm:flex-col">
                  <span class="font-mono text-[11px] text-zinc-400 dark:text-zinc-500">{{ String(idx + 1).padStart(2, '0') }}</span>
                  <button
                    type="button"
                    class="flex size-6 items-center justify-center rounded text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700 focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-emerald-500 disabled:pointer-events-none disabled:opacity-30 dark:hover:bg-zinc-800 dark:hover:text-zinc-200"
                    :disabled="idx === 0"
                    title="上移"
                    aria-label="上移此块"
                    @click="moveBlock(idx, -1)"
                  >
                    <NIcon :size="14"><PhArrowUp /></NIcon>
                  </button>
                  <button
                    type="button"
                    class="flex size-6 items-center justify-center rounded text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700 focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-emerald-500 disabled:pointer-events-none disabled:opacity-30 dark:hover:bg-zinc-800 dark:hover:text-zinc-200"
                    :disabled="idx === blocks.length - 1"
                    title="下移"
                    aria-label="下移此块"
                    @click="moveBlock(idx, 1)"
                  >
                    <NIcon :size="14"><PhArrowDown /></NIcon>
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
                      class="flex size-6 items-center justify-center rounded text-zinc-400 transition-colors hover:bg-red-50 hover:text-red-500 focus-visible:outline-2 focus-visible:outline-offset-1 focus-visible:outline-emerald-500 dark:hover:bg-red-500/10"
                      title="删除此块"
                      aria-label="删除此块"
                      @click="removeBlock(idx)"
                    >
                      <NIcon :size="13"><PhX /></NIcon>
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
                      <span class="text-xs text-zinc-500 dark:text-zinc-400">权限</span>
                      <div class="flex overflow-hidden rounded-lg border border-zinc-200 dark:border-zinc-700">
                        <button
                          v-for="mode in (['free', 'balance', 'membership'] as const)"
                          :key="mode"
                          type="button"
                          class="px-3 py-1.5 text-xs font-medium transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-emerald-500"
                          :class="blockAccess(block) === mode
                            ? 'bg-emerald-600 text-white dark:bg-emerald-400 dark:text-zinc-950'
                            : 'text-zinc-500 hover:bg-zinc-50 hover:text-zinc-800 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-200'"
                          @click="setAccess(idx, mode)"
                        >
                          {{ accessLabels[mode] }}
                        </button>
                      </div>
                      <NInputNumber
                        v-if="blockAccess(block) === 'balance'"
                        size="small"
                        class="w-32"
                        placeholder="价格"
                        :min="0"
                        :step="0.01"
                        :value="block.cost || 0"
                        @update:value="(v: number | null) => updateBlock(idx, 'cost', v ?? 0)"
                      >
                        <template #prefix><span class="text-xs text-zinc-400 dark:text-zinc-500">¥</span></template>
                      </NInputNumber>
                    </div>
                  </template>
                </div>
              </div>
            </div>
          </div>

          <!-- 链接解析器 -->
          <div class="rounded-xl border border-zinc-200 dark:border-zinc-700/80">
            <button
              type="button"
              class="flex w-full items-center justify-between rounded-xl px-4 py-2.5 text-left focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-emerald-500"
              @click="parserOpen = !parserOpen"
            >
              <span>
                <span class="block text-xs font-medium text-zinc-500 dark:text-zinc-400">链接解析器</span>
                <span class="mt-0.5 block text-[11px] text-zinc-400 dark:text-zinc-500">从网盘分享链接自动提取</span>
              </span>
              <NIcon :size="15" class="text-zinc-400 transition-transform" :class="parserOpen ? 'rotate-180' : ''"><PhCaretDown /></NIcon>
            </button>
            <div v-if="parserOpen" class="space-y-3 border-t border-zinc-100 p-4 dark:border-zinc-800">
              <div class="flex flex-col gap-2 sm:flex-row">
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

          <!-- 响应预览（折叠） -->
          <div class="rounded-xl border border-zinc-200 dark:border-zinc-700/80">
            <button
              type="button"
              class="flex w-full items-center justify-between rounded-xl px-4 py-2.5 text-left focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-emerald-500"
              @click="previewOpen = !previewOpen"
            >
              <span class="flex items-center gap-2">
                <span class="text-xs font-medium text-zinc-500 dark:text-zinc-400">响应预览</span>
                <span class="font-mono text-[11px] text-zinc-400 dark:text-zinc-500">{{ blocks.length }} 块</span>
              </span>
              <NIcon :size="15" class="text-zinc-400 transition-transform" :class="previewOpen ? 'rotate-180' : ''"><PhCaretDown /></NIcon>
            </button>
            <div v-if="previewOpen" class="max-h-80 space-y-2.5 overflow-y-auto border-t border-zinc-100 p-4 dark:border-zinc-800">
              <p v-if="blocks.length === 0" class="text-xs text-zinc-400 dark:text-zinc-500">添加内容块后此处将显示预览</p>
              <template v-for="(block, idx) in blocks" :key="idx">
                <p v-if="block.type === 'text'" class="text-[13px] leading-relaxed whitespace-pre-wrap text-zinc-600 dark:text-zinc-300">
                  {{ block.text || '(空文本)' }}
                </p>
                <div v-else-if="block.type === 'image'">
                  <img v-if="block.src" :src="block.src" alt="" class="max-h-28 rounded-lg" />
                  <span v-else class="text-xs text-zinc-400 dark:text-zinc-500">[图片]</span>
                </div>
                <div
                  v-else-if="block.type === 'link'"
                  class="flex items-center gap-2 rounded-lg border border-zinc-200 px-2.5 py-2 text-[13px] dark:border-zinc-700"
                >
                  <NIcon :size="13" class="shrink-0 text-emerald-500"><PhArrowSquareOut /></NIcon>
                  <span class="min-w-0 flex-1 truncate text-zinc-700 dark:text-zinc-300">{{ block.label || block.url || '(空链接)' }}</span>
                  <span v-if="block.code" class="shrink-0 font-mono text-[11px] text-zinc-400 dark:text-zinc-500">{{ block.code }}</span>
                </div>
              </template>
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
        </div>

        <template #footer>
          <div class="flex flex-col gap-2 sm:flex-row">
            <NButton type="primary" :disabled="!keyword.trim() || blocks.length === 0" @click="handleSubmit">{{ editingId ? '保存修改' : '添加关键词' }}</NButton>
            <NButton quaternary @click="handleCancel">取消</NButton>
          </div>
        </template>
      </NDrawerContent>
    </NDrawer>

    <!-- ══ 列表主区（全宽） ══ -->
    <section class="space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex flex-wrap items-center gap-2">
          <span class="text-sm text-zinc-500 dark:text-zinc-400">共 {{ total }} 个关键词</span>
          <template v-if="checkedRowKeys.length > 0">
            <span class="font-mono text-xs text-emerald-600 dark:text-emerald-400">已选 {{ checkedRowKeys.length }}</span>
            <NButton size="tiny" secondary @click="batchSetStatus('published')">批量上架</NButton>
            <NButton size="tiny" secondary @click="batchSetStatus('draft')">批量下架</NButton>
          </template>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <NSelect
            :value="filterStatus"
            :options="statusFilterOptions"
            size="small"
            class="w-32"
            aria-label="按状态筛选"
            @update:value="onStatusFilterChange"
          />
          <NSelect
            :value="filterCategoryId ?? 0"
            :options="categoryFilterOptions"
            size="small"
            class="w-40"
            aria-label="按分类筛选"
            @update:value="onFilterChange"
          />
        </div>
      </div>

      <NDataTable
        :columns="columns"
        :data="keywords"
        :row-key="(row: Keyword) => row.id"
        v-model:checked-row-keys="checkedRowKeys"
        :bordered="false"
        :loading="loading"
        :scroll-x="600"
        class="overflow-hidden rounded-xl border border-zinc-200 dark:border-zinc-800"
      />

      <div v-if="total > 0">
        <AppPagination
          :page="page"
          :page-size="limit"
          :item-count="total"
          @update:page="setPage"
          @update:page-size="setPageSize"
        />
      </div>
    </section>

    <ConfirmDialog
      :open="confirmDeleteId !== null"
      title="删除关键词"
      message="确定删除此关键词？"
      confirm-label="删除"
      danger
      @confirm="doDelete"
      @cancel="confirmDeleteId = null"
    />

    <!-- 图标选择 + 裁剪弹窗 -->
    <IconCropperModal
      :show="iconPickerOpen"
      :keyword="editingId ? (keywords.find((k: Keyword) => k.id === editingId) || null) : null"
      :current-icon="keywordIcon"
      @close="iconPickerOpen = false"
      @saved="onIconSaved"
    />

    <!-- 批量导入 -->
    <NModal
      v-model:show="importOpen"
      preset="card"
      title="批量导入关键词"
      class="w-[92vw] max-w-lg"
      :bordered="false"
      :segmented="{ content: true, footer: 'soft' }"
    >
      <div class="space-y-4">
        <p class="text-sm leading-relaxed text-zinc-500 dark:text-zinc-400">
          支持导出包 JSON，或 CSV（列：keyword, status, icon, attributes, categories, response）。
          分类名不存在时会自动创建。
        </p>
        <div>
          <label class="mb-1.5 block text-sm font-medium text-zinc-700 dark:text-zinc-300">冲突处理</label>
          <NRadioGroup v-model:value="importMode" name="import-mode">
            <div class="flex flex-wrap gap-4">
              <NRadio value="skip">跳过已存在</NRadio>
              <NRadio value="overwrite">覆盖已存在</NRadio>
            </div>
          </NRadioGroup>
        </div>
        <NUpload
          :default-upload="false"
          :show-file-list="false"
          accept=".json,.csv,application/json,text/csv"
          :disabled="importBusy"
          @change="({ file }) => onImportFile({ file })"
        >
          <NButton type="primary" :loading="importBusy" secondary block>
            <template #icon><NIcon><PhUploadSimple /></NIcon></template>
            选择文件并导入
          </NButton>
        </NUpload>
        <div
          v-if="importResult"
          class="rounded-xl border border-zinc-200 bg-zinc-50 p-4 text-sm dark:border-zinc-800 dark:bg-zinc-900/60"
        >
          <p class="font-medium text-zinc-800 dark:text-zinc-100">导入结果</p>
          <dl class="mt-2 grid grid-cols-2 gap-2 font-mono text-xs text-zinc-600 dark:text-zinc-300 sm:grid-cols-4">
            <div><dt class="text-zinc-400">新建</dt><dd class="text-base text-emerald-600 dark:text-emerald-400">{{ importResult.created }}</dd></div>
            <div><dt class="text-zinc-400">更新</dt><dd class="text-base">{{ importResult.updated }}</dd></div>
            <div><dt class="text-zinc-400">跳过</dt><dd class="text-base">{{ importResult.skipped }}</dd></div>
            <div><dt class="text-zinc-400">失败</dt><dd class="text-base" :class="importResult.failed ? 'text-red-500' : ''">{{ importResult.failed }}</dd></div>
          </dl>
          <ul v-if="importResult.errors?.length" class="mt-3 max-h-32 space-y-1 overflow-y-auto text-xs text-red-500 dark:text-red-400">
            <li v-for="(e, i) in importResult.errors" :key="i">{{ e }}</li>
          </ul>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end">
          <NButton quaternary @click="importOpen = false">关闭</NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>
