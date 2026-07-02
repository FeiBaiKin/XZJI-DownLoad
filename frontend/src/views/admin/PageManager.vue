<script setup lang="ts">
// 页面管理：1:1 移植自原项目 Node-XZJI/frontend/src/pages/admin/PageManager.jsx
// 富文本用 wangEditor（RichTextEditor）替代 react-quill
// 样式来自全局 styles/admin-layout.css + admin-orders.css + rich-text.css
import { ref, onMounted } from 'vue'
import { contentApi, type Page } from '@/api/content'
import { formatTime } from '@/utils/format'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import RichTextEditor from '@/components/RichTextEditor.vue'

interface PageForm {
  title: string
  slug: string
  content: string
  is_published: boolean
}

const pages = ref<Page[]>([])
const loading = ref(true)
const saving = ref(false)
const message = ref('')
const editing = ref<Page | null>(null)
const form = ref<PageForm>({ title: '', slug: '', content: '', is_published: true })
const confirmDelete = ref<Page | null>(null)

onMounted(() => { fetchPages() })

async function fetchPages() {
  try {
    pages.value = await contentApi.pages.list()
  } catch { /* 忽略 */ }
  finally { loading.value = false }
}

function resetForm() {
  editing.value = null
  form.value = { title: '', slug: '', content: '', is_published: true }
}

function openNew() {
  editing.value = { id: 0 } as Page
  form.value = { title: '', slug: '', content: '', is_published: true }
  message.value = ''
}

function handleEdit(page: Page) {
  editing.value = page
  form.value = {
    title: page.title, slug: page.slug,
    content: page.content || '', is_published: !!page.is_published,
  }
  message.value = ''
}

async function handleSave() {
  if (!form.value.title.trim() || !form.value.slug.trim()) { message.value = '标题和标识不能为空'; return }
  saving.value = true; message.value = ''
  try {
    if (editing.value?.id) {
      await contentApi.pages.update(editing.value.id, form.value)
      message.value = '页面已更新'
    } else {
      await contentApi.pages.create(form.value)
      message.value = '页面已创建'
    }
    resetForm()
    fetchPages()
  } catch (err: unknown) {
    const errorObj = err as { response?: { data?: { error?: string } } }
    message.value = errorObj?.response?.data?.error || '保存失败'
  } finally {
    saving.value = false
  }
}

function handleDelete(page: Page) { confirmDelete.value = page }
async function doDelete() {
  const page = confirmDelete.value
  confirmDelete.value = null
  if (!page) return
  try {
    await contentApi.pages.delete(page.id)
    fetchPages()
  } catch { /* 忽略 */ }
}
</script>

<template>
  <div class="admin-content">
    <div v-if="loading" class="loading-state">加载中...</div>
    <template v-else>
      <div class="content-header">
        <h2 class="content-title">页面管理</h2>
        <span class="content-sub">创建独立页面，如隐私协议、用户协议等</span>
      </div>

      <div v-if="!editing" style="margin-bottom: 1rem;">
        <button class="action-button" @click="openNew">新建页面</button>
      </div>

      <div v-if="editing" class="form-card" style="margin-bottom: 1.5rem;">
        <div class="form-card-title">{{ editing.id ? `编辑页面: ${editing.title}` : '新建页面' }}</div>
        <form @submit.prevent="handleSave">
          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; margin-bottom: 0.75rem;">
            <div class="form-group" style="margin: 0;">
              <label class="form-label">页面标题</label>
              <input type="text" class="form-input" v-model="form.title" />
            </div>
            <div class="form-group" style="margin: 0;">
              <label class="form-label">URL 标识</label>
              <input type="text" class="form-input" v-model="form.slug" placeholder="privacy" />
            </div>
          </div>
          <div class="form-group">
            <label class="form-label">内容</label>
            <RichTextEditor v-model="form.content" placeholder="开始编辑页面内容..." />
          </div>
          <div class="form-group" style="flex-direction: row; align-items: center; gap: 0.75rem;">
            <label class="form-label" style="margin: 0;">发布状态</label>
            <button type="button" class="email-toggle-switch" :class="{ on: form.is_published }" :aria-checked="form.is_published" role="switch" @click="form.is_published = !form.is_published">
              <span class="email-toggle-knob" />
            </button>
            <span style="font-size: 0.75rem; color: var(--text-tertiary);">{{ form.is_published ? '已发布' : '未发布' }}</span>
          </div>
          <div v-if="message" class="form-message" style="margin-bottom: 0.75rem;">{{ message }}</div>
          <div style="display: flex; gap: 0.5rem;">
            <button type="submit" class="action-button" :disabled="saving">{{ saving ? '保存中...' : '保存' }}</button>
            <button type="button" class="action-button secondary" @click="resetForm">取消</button>
          </div>
        </form>
      </div>

      <div class="order-table-wrap">
        <table class="order-table">
          <thead>
            <tr>
              <th>标题</th><th>标识</th><th>状态</th><th>更新时间</th><th style="width: 140px;">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="pages.length === 0"><td colspan="5" class="order-empty">暂无页面</td></tr>
            <tr v-for="p in pages" :key="p.id">
              <td style="font-weight: 500;">{{ p.title }}</td>
              <td class="mono-td">{{ p.slug }}</td>
              <td>{{ p.is_published ? '已发布' : '草稿' }}</td>
              <td class="mono-td dim">{{ formatTime(p.updated_at) }}</td>
              <td style="display: flex; gap: 0.75rem;">
                <a :href="`/page/${p.slug}`" target="_blank" class="action-link" style="text-decoration: none;">查看</a>
                <button class="action-link" @click="handleEdit(p)">编辑</button>
                <button class="action-link danger" @click="handleDelete(p)">删除</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>

    <ConfirmDialog
      :open="confirmDelete !== null"
      title="删除页面"
      :message="confirmDelete ? `确定删除页面「${confirmDelete.title}」吗？` : ''"
      confirm-label="删除"
      danger
      @confirm="doDelete"
      @cancel="confirmDelete = null"
    />
  </div>
</template>
