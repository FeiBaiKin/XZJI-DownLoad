<script setup lang="ts">
// 关键词管理（块编辑器）：1:1 移植自原项目 Node-XZJI/frontend/src/pages/admin/KeywordManager.jsx
// 样式来自全局 styles/admin-keyword.css + admin-layout.css
import { ref, onMounted } from 'vue'
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
    // 后端返回纯 ID 数组（[]uint），原版返回分类对象数组并 .map(c=>c.id)
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
  ;[updated[idx], updated[newIdx]] = [updated[newIdx], updated[idx]]
  blocks.value = updated
}

function parseLink() {
  if (!linkUrl.value.trim()) return
  const raw = linkUrl.value.trim()
  const codeMatch = raw.match(/(?:提取码|提取密码|访问码|密码|[?&]?pwd|code)[：:=:\s]*([a-zA-Z0-9]{4,8})/i)
  const extractCode = codeMatch ? codeMatch[1] : ''
  const urlMatch = raw.match(/(https?:\/\/[^\s\u4e00-\u9fff]+)/i)
  const cleanUrl = urlMatch ? urlMatch[1].replace(/[，,。;；！!）)]+$/, '') : raw

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
      // 后端 setCategories 接收纯数组 []uint（原版发送 {categoryIds}）
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
  // 后端返回 response 为 JSON 字符串（块数组或纯文本），原版后端已解析；此处前端解析
  let resp: string | ContentBlock[] = item.response
  if (typeof resp === 'string') {
    try { resp = JSON.parse(resp) } catch { /* 纯文本，保持字符串 */ }
  }
  if (Array.isArray(resp)) blocks.value = resp.map((b) => ({ ...b, _id: (b as any)._id || uid() }) as ContentBlock)
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

function toggleCategory(id: number) {
  selectedCategoryIds.value = selectedCategoryIds.value.includes(id)
    ? selectedCategoryIds.value.filter((x) => x !== id)
    : [...selectedCategoryIds.value, id]
}

function getPreviewText(response: string | ContentBlock[]) {
  let resp: string | ContentBlock[] = response
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
</script>

<template>
  <div class="admin-content">
    <div class="content-header">
      <h2 class="content-title">关键词管理</h2>
      <button class="action-button" @click="showForm = true">添加关键词</button>
    </div>

    <div v-if="showForm" class="kw-editor">
      <div class="kw-editor-header">
        <h3 class="kw-editor-title">{{ editingId ? '编辑关键词' : '添加关键词' }}</h3>
        <button type="button" class="kw-editor-close" @click="handleCancel">X</button>
      </div>
      <div class="kw-editor-body">
        <div class="kw-editor-main">
          <div class="kw-field">
            <label class="kw-field-label">关键词</label>
            <input type="text" class="form-input" placeholder="输入关键词，例如: hello, react..." v-model="keyword" required />
          </div>

          <div class="kw-blocks">
            <div class="kw-blocks-toolbar">
              <span class="kw-field-label">响应内容块</span>
              <div class="kw-block-types">
                <button type="button" class="kw-type-btn" @click="addBlock('text')">
                  <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.2"><path d="M2 3h8M2 6h6M2 9h4" /></svg>文本
                </button>
                <button type="button" class="kw-type-btn" @click="addBlock('image')">
                  <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.2"><rect x="1" y="2" width="10" height="8" /><circle cx="4" cy="5" r="1" /><path d="M1 9l3-3 2 2 2-2 3 3" /></svg>图片
                </button>
                <button type="button" class="kw-type-btn" @click="addBlock('link')">
                  <svg width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.2"><path d="M5 7l2-2M4.5 3.5l-1 1a2.12 2.12 0 000 3l1 1a2.12 2.12 0 003 0l1-1" /></svg>链接
                </button>
              </div>
            </div>

            <div v-if="blocks.length === 0" class="kw-blocks-empty">
              <span class="kw-empty-icon">+</span>
              <span>添加文本、图片或链接块来构建响应内容</span>
            </div>
            <div v-else class="kw-blocks-list">
              <div v-for="(block, idx) in blocks" :key="block._id" class="kw-block" :class="`kw-block-${block.type}`">
                <div class="kw-block-gutter">
                  <span class="kw-block-idx">{{ String(idx + 1).padStart(2, '0') }}</span>
                  <div class="kw-block-arrows">
                    <button type="button" :disabled="idx === 0" title="上移" @click="moveBlock(idx, -1)">&#8593;</button>
                    <button type="button" :disabled="idx === blocks.length - 1" title="下移" @click="moveBlock(idx, 1)">&#8595;</button>
                  </div>
                </div>
                <div class="kw-block-body">
                  <div class="kw-block-head">
                    <span class="kw-block-type">{{ block.type === 'text' ? 'TEXT' : block.type === 'image' ? 'IMAGE' : 'LINK' }}</span>
                    <button type="button" class="kw-block-remove" title="删除" @click="removeBlock(idx)">
                      <svg width="10" height="10" viewBox="0 0 10 10" fill="none" stroke="currentColor" stroke-width="1.2"><path d="M1 1l8 8M9 1l-8 8" /></svg>
                    </button>
                  </div>

                  <textarea v-if="block.type === 'text'" class="form-input form-textarea" placeholder="输入文本内容..." :value="block.text || ''" @input="(e) => updateBlock(idx, 'text', (e.target as HTMLTextAreaElement).value)" rows="3" />
                  <div v-else-if="block.type === 'image'" class="kw-block-fields">
                    <input type="text" class="form-input" placeholder="图片 URL 地址..." :value="block.src || ''" @input="(e) => updateBlock(idx, 'src', (e.target as HTMLInputElement).value)" />
                    <input type="text" class="form-input" placeholder="图片描述 (alt, 可选)..." :value="block.alt || ''" @input="(e) => updateBlock(idx, 'alt', (e.target as HTMLInputElement).value)" />
                    <div v-if="block.src" class="kw-image-preview">
                      <img :src="block.src" :alt="block.alt || ''" @error="(e) => (e.target as HTMLImageElement).style.display = 'none'" />
                    </div>
                  </div>
                  <div v-else-if="block.type === 'link'" class="kw-block-fields">
                    <input type="text" class="form-input" placeholder="链接 URL..." :value="block.url || ''" @input="(e) => updateBlock(idx, 'url', (e.target as HTMLInputElement).value)" />
                    <div class="kw-link-row">
                      <input type="text" class="form-input" placeholder="显示名称 (可选)" :value="block.label || ''" @input="(e) => updateBlock(idx, 'label', (e.target as HTMLInputElement).value)" />
                      <input type="text" class="form-input" placeholder="提取码 (可选)" :value="block.code || ''" @input="(e) => updateBlock(idx, 'code', (e.target as HTMLInputElement).value)" />
                    </div>
                    <div class="kw-access-row">
                      <span class="kw-access-label">权限</span>
                      <div class="kw-access-tabs">
                        <button
                          v-for="mode in ['free', 'balance', 'membership']"
                          :key="mode"
                          type="button"
                          class="kw-access-tab"
                          :class="{ active: blockAccess(block) === mode }"
                          @click="updateBlock(idx, 'access', mode)"
                        >
                          {{ mode === 'free' ? '免费' : mode === 'balance' ? '付费' : '会员' }}
                        </button>
                      </div>
                      <input
                        v-if="blockAccess(block) === 'balance' || blockAccess(block) === 'free'"
                        type="number"
                        class="form-input kw-cost-input"
                        placeholder="价格"
                        min="0"
                        step="0.01"
                        :value="blockAccess(block) === 'free' ? 0 : (block.cost || 0)"
                        @input="(e) => updateBlock(idx, 'cost', parseFloat((e.target as HTMLInputElement).value) || 0)"
                        :disabled="blockAccess(block) === 'free'"
                      />
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div v-if="categories.length > 0" class="kw-category-section">
            <span class="kw-field-label">所属分类</span>
            <div class="kw-category-list">
              <label v-for="c in categories" :key="c.id" class="kw-category-check">
                <input type="checkbox" :checked="selectedCategoryIds.includes(c.id)" @change="toggleCategory(c.id)" />
                <span>{{ c.name }}</span>
              </label>
            </div>
          </div>

          <div class="kw-form-actions">
            <button type="button" class="action-button" @click="handleSubmit">{{ editingId ? '保存修改' : '添加关键词' }}</button>
            <button type="button" class="action-button secondary" @click="handleCancel">取消</button>
          </div>
        </div>

        <div class="kw-editor-side">
          <div class="kw-preview">
            <div class="kw-preview-header">
              <span class="kw-field-label">响应预览</span>
              <span class="kw-preview-badge">{{ blocks.length }} 块</span>
            </div>
            <div class="kw-preview-body">
              <div v-if="blocks.length === 0" class="kw-preview-empty">添加内容块后此处将显示预览</div>
              <div v-for="(block, idx) in blocks" :key="idx" class="kw-preview-block">
                <p v-if="block.type === 'text'" class="kw-preview-text">{{ block.text || '(空文本)' }}</p>
                <div v-else-if="block.type === 'image'" class="kw-preview-image">
                  <img v-if="block.src" :src="block.src" alt="" />
                  <span v-else class="kw-preview-placeholder">[图片]</span>
                </div>
                <div v-else-if="block.type === 'link'" class="kw-preview-link">
                  <span class="kw-preview-link-icon">&#8599;</span>
                  <span class="kw-preview-link-label">{{ block.label || block.url || '(空链接)' }}</span>
                  <span v-if="block.code" class="kw-preview-link-code">{{ block.code }}</span>
                </div>
              </div>
            </div>
          </div>

          <details class="kw-parser">
            <summary class="kw-parser-summary">
              <span class="kw-field-label">链接解析器</span>
              <span class="kw-parser-hint">从网盘分享链接自动提取</span>
            </summary>
            <div class="kw-parser-body">
              <div class="kw-parser-input-row">
                <input type="text" class="form-input" placeholder="粘贴网盘链接..." v-model="linkUrl" @input="parsedLink = null" />
                <button type="button" class="action-button secondary" @click="parseLink">解析</button>
              </div>
              <div v-if="parsedLink" class="kw-parser-result">
                <div class="kw-parser-row">
                  <span class="kw-parser-label">平台</span>
                  <span class="kw-parser-platform">
                    <span v-if="parsedLink.icon" class="kw-parser-icon">{{ parsedLink.icon }}</span>{{ parsedLink.platform }}
                  </span>
                </div>
                <div class="kw-parser-row"><span class="kw-parser-label">链接</span><span class="kw-parser-value">{{ parsedLink.url }}</span></div>
                <div v-if="parsedLink.code" class="kw-parser-row"><span class="kw-parser-label">提取码</span><span class="kw-parser-code">{{ parsedLink.code }}</span></div>
                <button type="button" class="action-button secondary" style="margin-top: 0.5rem; width: 100%;" @click="addParsedLinkAsBlocks">添加为链接块</button>
              </div>
            </div>
          </details>
        </div>
      </div>
    </div>

    <div v-if="loading" class="loading-state">加载中...</div>
    <div v-else class="data-table">
      <div class="table-header">
        <span class="table-cell">ID</span><span class="table-cell">关键词</span><span class="table-cell">响应内容</span><span class="table-cell">操作</span>
      </div>
      <div v-if="keywords.length === 0" class="empty-state">暂无关键词数据</div>
      <div v-for="item in keywords" :key="item.id" class="table-row">
        <span class="table-cell mono">{{ item.id }}</span>
        <span class="table-cell">{{ item.keyword }}</span>
        <span class="table-cell truncate">{{ getPreviewText(item.response) }}</span>
        <span class="table-cell actions">
          <button class="action-link" @click="handleEdit(item)">编辑</button>
          <button class="action-link danger" @click="handleDelete(item.id)">删除</button>
        </span>
      </div>
    </div>

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
