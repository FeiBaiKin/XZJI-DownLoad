<script setup lang="ts">
// 公告管理：前台弹窗公告，用户关闭后不再显示
// 业务逻辑保持不变；UI 重构为 Naive UI + Tailwind
import { ref, onMounted, h } from 'vue'
import {
  NButton, NInput, NDataTable, NAlert, NSwitch, NTag, NSkeleton, NIcon, NRadioGroup, NRadioButton,
  type DataTableColumns,
} from 'naive-ui'
import { Plus } from '@vicons/tabler'
import { contentApi, type Announcement } from '@/api/content'
import { formatTime } from '@/utils/format'
import { typeLabels, typeOptions } from '@/constants/admin'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import RichTextEditor from '@/components/RichTextEditor.vue'
import { useToastStore } from '@/stores/toast'

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
const toast = useToastStore()

onMounted(() => { fetchAnnouncements() })

async function fetchAnnouncements() {
  try {
    announcements.value = await contentApi.announcements.list()
  } catch (err) {
    console.error(err)
    toast.error('加载失败，请刷新重试')
  }
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
  } catch (err) {
    console.error(err)
    toast.error('删除失败，请重试')
  }
}

const typeTagType = (t: string): 'info' | 'warning' | 'success' =>
  (t === 'warning' ? 'warning' : t === 'success' ? 'success' : 'info')

const columns: DataTableColumns<Announcement> = [
  { title: '标题', key: 'title', render: (row) => h('span', { class: 'font-medium' }, row.title) },
  {
    title: '类型',
    key: 'type',
    width: 90,
    render: (row) =>
      h(NTag, { size: 'small', round: true, bordered: false, type: typeTagType(row.type) }, { default: () => typeLabels[row.type] || row.type }),
  },
  {
    title: '状态',
    key: 'is_active',
    width: 90,
    render: (row) => h('span', { class: row.is_active ? 'text-emerald-600 dark:text-emerald-400 text-[13px]' : 'text-zinc-500 dark:text-zinc-400 text-[13px]' }, row.is_active ? '启用' : '已停用'),
  },
  { title: '时间', key: 'created_at', width: 120, render: (row) => h('span', { class: 'font-mono text-xs text-zinc-500 dark:text-zinc-400' }, formatTime(row.created_at)) },
  {
    title: '操作',
    key: 'actions',
    width: 130,
    render: (row) =>
      h('div', { class: 'flex gap-1' }, [
        h(NButton, { size: 'tiny', quaternary: true, type: 'primary', onClick: () => handleEdit(row) }, { default: () => '编辑' }),
        h(NButton, { size: 'tiny', quaternary: true, type: 'error', onClick: () => handleDelete(row) }, { default: () => '删除' }),
      ]),
  },
]
</script>

<template>
  <div>
    <div v-if="loading" class="space-y-4">
      <NSkeleton text width="20%" size="large" />
      <NSkeleton height="240px" :sharp="false" />
    </div>

    <template v-else>
      <header class="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">公告管理</h2>
          <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">管理前台弹窗公告，用户关闭后不再显示</p>
        </div>
        <NButton v-if="!showForm" type="primary" @click="openNew">
          <template #icon><NIcon><Plus /></NIcon></template>
          新建公告
        </NButton>
      </header>

      <!-- 编辑表单 -->
      <section v-if="showForm" class="mb-6 rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
        <h3 class="mb-4 text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">{{ editing ? '编辑公告' : '新建公告' }}</h3>
        <form class="space-y-4" @submit.prevent="handleSave">
          <div>
            <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">标题</label>
            <NInput v-model:value="form.title" placeholder="公告标题" />
          </div>
          <div>
            <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">内容</label>
            <RichTextEditor v-model="form.content" simple placeholder="开始编辑公告内容..." />
          </div>
          <div class="flex flex-wrap items-center gap-x-8 gap-y-3">
            <div class="flex items-center gap-3">
              <label class="text-sm text-zinc-600 dark:text-zinc-400">类型</label>
              <NRadioGroup v-model:value="form.type" size="small">
                <NRadioButton v-for="t in typeOptions" :key="t" :value="t">{{ typeLabels[t] }}</NRadioButton>
              </NRadioGroup>
            </div>
            <div class="flex items-center gap-3">
              <label class="text-sm text-zinc-600 dark:text-zinc-400">启用</label>
              <NSwitch v-model:value="form.is_active" size="small" />
            </div>
          </div>

          <NAlert v-if="message" :type="message.includes('失败') || message.includes('不能') ? 'error' : 'success'" closable @close="message = ''">
            {{ message }}
          </NAlert>

          <div class="flex flex-col gap-2 sm:flex-row">
            <NButton attr-type="submit" type="primary" :loading="saving">{{ editing ? '更新' : '创建' }}</NButton>
            <NButton quaternary @click="resetForm">取消</NButton>
          </div>
        </form>
      </section>

      <!-- 公告列表 -->
      <NDataTable
        :columns="columns"
        :data="announcements"
        :row-key="(row: Announcement) => row.id"
        :bordered="false"
        :scroll-x="650"
        class="overflow-hidden rounded-xl border border-zinc-200 dark:border-zinc-800"
      />
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
