<script setup lang="ts">
// 页面管理：创建独立页面（隐私协议、用户协议等），wangEditor 富文本编辑
// 业务逻辑保持不变；UI 重构为 Naive UI + Tailwind
import { ref, onMounted, h } from 'vue'
import { NButton, NInput, NDataTable, NAlert, NSwitch, NTag, NSkeleton, NIcon, type DataTableColumns } from 'naive-ui'
import { Plus } from '@vicons/tabler'
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

const columns: DataTableColumns<Page> = [
  { title: '标题', key: 'title', render: (row) => h('span', { class: 'font-medium' }, row.title) },
  { title: '标识', key: 'slug', width: 140, render: (row) => h('span', { class: 'font-mono text-xs text-zinc-500 dark:text-zinc-400' }, row.slug) },
  {
    title: '状态',
    key: 'is_published',
    width: 90,
    render: (row) =>
      h(NTag, { size: 'small', round: true, type: row.is_published ? 'success' : 'default', bordered: false }, { default: () => (row.is_published ? '已发布' : '草稿') }),
  },
  { title: '更新时间', key: 'updated_at', width: 120, render: (row) => h('span', { class: 'font-mono text-xs text-zinc-400' }, formatTime(row.updated_at)) },
  {
    title: '操作',
    key: 'actions',
    width: 170,
    render: (row) =>
      h('div', { class: 'flex gap-1' }, [
        h(NButton, { size: 'tiny', quaternary: true, onClick: () => window.open(`/page/${row.slug}`, '_blank') }, { default: () => '查看' }),
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
          <h2 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">页面管理</h2>
          <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">创建独立页面，如隐私协议、用户协议等</p>
        </div>
        <NButton v-if="!editing" type="primary" @click="openNew">
          <template #icon><NIcon><Plus /></NIcon></template>
          新建页面
        </NButton>
      </header>

      <!-- 编辑表单 -->
      <section v-if="editing" class="mb-6 rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
        <h3 class="mb-4 text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">
          {{ editing.id ? `编辑页面：${editing.title}` : '新建页面' }}
        </h3>
        <form class="space-y-4" @submit.prevent="handleSave">
          <div class="grid gap-3 sm:grid-cols-2">
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">页面标题</label>
              <NInput v-model:value="form.title" placeholder="页面标题" />
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">URL 标识</label>
              <NInput v-model:value="form.slug" placeholder="privacy" />
            </div>
          </div>
          <div>
            <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">内容</label>
            <RichTextEditor v-model="form.content" placeholder="开始编辑页面内容..." />
          </div>
          <div class="flex items-center gap-3">
            <label class="text-sm text-zinc-600 dark:text-zinc-400">发布状态</label>
            <NSwitch v-model:value="form.is_published" size="small" />
            <span class="text-xs text-zinc-400 dark:text-zinc-500">{{ form.is_published ? '已发布' : '未发布' }}</span>
          </div>

          <NAlert v-if="message" :type="message.includes('失败') || message.includes('不能') ? 'error' : 'success'" closable @close="message = ''">
            {{ message }}
          </NAlert>

          <div class="flex flex-col gap-2 sm:flex-row">
            <NButton attr-type="submit" type="primary" :loading="saving">保存</NButton>
            <NButton quaternary @click="resetForm">取消</NButton>
          </div>
        </form>
      </section>

      <!-- 页面列表 -->
      <NDataTable
        :columns="columns"
        :data="pages"
        :row-key="(row: Page) => row.id"
        :bordered="false"
        :scroll-x="720"
        class="overflow-hidden rounded-xl border border-zinc-200 dark:border-zinc-800"
      />
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
