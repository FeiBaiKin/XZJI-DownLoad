<script setup lang="ts">
// 公告管理：1:1 移植自原项目 Node-XZJI/frontend/src/pages/admin/AnnouncementManager.jsx
// 富文本用 wangEditor（RichTextEditor simple）替代 react-quill
import { ref, onMounted } from 'vue'
import { contentApi, type Announcement } from '@/api/content'
import { formatTime } from '@/utils/format'
import { typeLabels, typeOptions } from '@/constants/admin'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import RichTextEditor from '@/components/RichTextEditor.vue'

interface AnnouncementForm {
  title: string
  content: string
  type: string
  is_active: boolean
}

const announcements = ref<Announcement[]>([])
const loading = ref(true)
const saving = ref(false)
const message = ref('')
const showForm = ref(false)
const editing = ref<Announcement | null>(null)
const form = ref<AnnouncementForm>({ title: '', content: '', type: 'info', is_active: true })
const confirmDelete = ref<Announcement | null>(null)

onMounted(() => { fetchAnnouncements() })

async function fetchAnnouncements() {
  try {
    announcements.value = await contentApi.announcements.list()
  } catch { /* 忽略 */ }
  finally { loading.value = false }
}

function resetForm() {
  editing.value = null
  showForm.value = false
  form.value = { title: '', content: '', type: 'info', is_active: true }
}

function openNew() {
  editing.value = null
  form.value = { title: '', content: '', type: 'info', is_active: true }
  showForm.value = true
  message.value = ''
}

function handleEdit(a: Announcement) {
  editing.value = a
  form.value = { title: a.title, content: a.content || '', type: a.type || 'info', is_active: !!a.is_active }
  showForm.value = true
  message.value = ''
}

async function handleSave() {
  if (!form.value.title.trim()) { message.value = '标题不能为空'; return }
  saving.value = true; message.value = ''
  try {
    if (editing.value) {
      await contentApi.announcements.update(editing.value.id, form.value)
      message.value = '公告已更新'
    } else {
      await contentApi.announcements.create(form.value)
      message.value = '公告已创建'
    }
    resetForm()
    fetchAnnouncements()
  } catch (err: unknown) {
    const errorObj = err as { response?: { data?: { error?: string } } }
    message.value = errorObj?.response?.data?.error || '保存失败'
  } finally {
    saving.value = false
  }
}

function handleDelete(a: Announcement) { confirmDelete.value = a }
async function doDelete() {
  const a = confirmDelete.value
  confirmDelete.value = null
  if (!a) return
  try {
    await contentApi.announcements.delete(a.id)
    fetchAnnouncements()
  } catch { /* 忽略 */ }
}
</script>

<template>
  <div class="admin-content">
    <div v-if="loading" class="loading-state">加载中...</div>
    <template v-else>
      <div class="content-header">
        <h2 class="content-title">公告管理</h2>
        <span class="content-sub">管理前台弹窗公告，用户关闭后不再显示</span>
      </div>

      <div v-if="!showForm" style="margin-bottom: 1rem;">
        <button class="action-button" @click="openNew">新建公告</button>
      </div>

      <div v-if="showForm" class="form-card" style="margin-bottom: 1.5rem;">
        <div class="form-card-title">{{ editing ? '编辑公告' : '新建公告' }}</div>
        <form @submit.prevent="handleSave">
          <div class="form-group">
            <label class="form-label">标题</label>
            <input type="text" class="form-input" v-model="form.title" />
          </div>
          <div class="form-group">
            <label class="form-label">内容</label>
            <RichTextEditor v-model="form.content" simple placeholder="开始编辑公告内容..." />
          </div>
          <div style="display: flex; gap: 1.5rem; align-items: flex-end; flex-wrap: wrap; margin-bottom: 0.75rem;">
            <div class="form-group" style="margin: 0;">
              <label class="form-label">类型</label>
              <div class="pay-scene-toggle" style="display: inline-flex;">
                <button v-for="t in typeOptions" :key="t" type="button" class="pay-scene-btn" :class="{ active: form.type === t }" @click="form.type = t">{{ typeLabels[t] }}</button>
              </div>
            </div>
            <div class="form-group" style="flex-direction: row; align-items: center; gap: 0.75rem; margin: 0;">
              <label class="form-label" style="margin: 0;">启用</label>
              <button type="button" class="email-toggle-switch" :class="{ on: form.is_active }" :aria-checked="form.is_active" role="switch" @click="form.is_active = !form.is_active">
                <span class="email-toggle-knob" />
              </button>
            </div>
          </div>
          <div v-if="message" class="form-message" style="margin-bottom: 0.75rem;">{{ message }}</div>
          <div style="display: flex; gap: 0.5rem;">
            <button type="submit" class="action-button" :disabled="saving">{{ saving ? '保存中...' : (editing ? '更新' : '创建') }}</button>
            <button type="button" class="action-button secondary" @click="resetForm">取消</button>
          </div>
        </form>
      </div>

      <div class="order-table-wrap">
        <table class="order-table">
          <thead>
            <tr>
              <th>标题</th><th>类型</th><th>状态</th><th>时间</th><th style="width: 100px;">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="announcements.length === 0"><td colspan="5" class="order-empty">暂无公告</td></tr>
            <tr v-for="a in announcements" :key="a.id">
              <td style="font-weight: 500;">{{ a.title }}</td>
              <td>{{ typeLabels[a.type] || a.type }}</td>
              <td>{{ a.is_active ? '启用' : '已停用' }}</td>
              <td class="mono-td dim">{{ formatTime(a.created_at) }}</td>
              <td style="display: flex; gap: 0.75rem;">
                <button class="action-link" @click="handleEdit(a)">编辑</button>
                <button class="action-link danger" @click="handleDelete(a)">删除</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>

    <ConfirmDialog
      :open="confirmDelete !== null"
      title="删除公告"
      :message="confirmDelete ? `确定删除公告「${confirmDelete.title}」吗？` : ''"
      confirm-label="删除"
      danger
      @confirm="doDelete"
      @cancel="confirmDelete = null"
    />
  </div>
</template>
