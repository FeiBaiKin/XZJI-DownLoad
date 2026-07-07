<script setup lang="ts">
// 分类管理：为关键词创建分类，组织内容结构
// 业务逻辑保持不变；UI 重构为 Naive UI + Tailwind
import { ref, onMounted, h } from 'vue'
import { NButton, NInput, NInputNumber, NDataTable, NAlert, NSkeleton, type DataTableColumns } from 'naive-ui'
import { contentApi, type Category } from '@/api/content'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import { useToastStore } from '@/stores/toast'

const toast = useToastStore()

const categories = ref<Category[]>([])
const loading = ref(true)
const saving = ref(false)
const message = ref('')
const editing = ref<Category | null>(null)
const form = ref({ name: '', description: '', sort_order: 0 })
const confirmCat = ref<Category | null>(null)

onMounted(() => { fetchCategories() })

async function fetchCategories() {
  try {
    categories.value = await contentApi.categories.list()
  } catch (err) {
    console.error(err)
    toast.error('加载失败，请刷新重试')
  }
  finally { loading.value = false }
}

function resetForm() {
  editing.value = null
  form.value = { name: '', description: '', sort_order: 0 }
}

function handleEdit(cat: Category) {
  editing.value = cat
  form.value = { name: cat.name, description: cat.description || '', sort_order: cat.sort_order || 0 }
}

async function handleSave() {
  if (!form.value.name.trim()) { message.value = '分类名称不能为空'; return }
  saving.value = true
  message.value = ''
  try {
    if (editing.value) {
      await contentApi.categories.update(editing.value.id, form.value)
      message.value = '分类已更新'
    } else {
      await contentApi.categories.create(form.value)
      message.value = '分类已创建'
    }
    resetForm()
    fetchCategories()
  } catch (err: unknown) {
    const errorObj = err as { response?: { data?: { error?: string } } }
    message.value = errorObj?.response?.data?.error || '保存失败'
  } finally {
    saving.value = false
  }
}

function handleDelete(cat: Category) {
  confirmCat.value = cat
}
async function doDelete() {
  const cat = confirmCat.value
  confirmCat.value = null
  if (!cat) return
  try {
    await contentApi.categories.delete(cat.id)
    fetchCategories()
  } catch (err) {
    console.error(err)
    toast.error('删除失败，请重试')
  }
}

const columns: DataTableColumns<Category> = [
  { title: '名称', key: 'name', render: (row) => h('span', { class: 'font-medium' }, row.name) },
  {
    title: '描述',
    key: 'description',
    ellipsis: true,
    render: (row) => h('span', { class: 'text-[13px] text-zinc-500 dark:text-zinc-400' }, row.description || '-'),
  },
  { title: '排序', key: 'sort_order', width: 80, render: (row) => h('span', { class: 'font-mono text-xs' }, row.sort_order) },
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
      <NSkeleton height="160px" :sharp="false" />
      <NSkeleton height="200px" :sharp="false" />
    </div>

    <template v-else>
      <header class="mb-6">
        <h2 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">分类管理</h2>
        <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">为关键词创建分类，组织内容结构</p>
      </header>

      <!-- 新建 / 编辑表单 -->
      <section class="mb-6 rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
        <h3 class="mb-4 text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">{{ editing ? `编辑分类：${editing.name}` : '新建分类' }}</h3>
        <form class="space-y-4" @submit.prevent="handleSave">
          <div class="grid gap-3 sm:grid-cols-[minmax(0,1fr)_130px]">
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">分类名称</label>
              <NInput v-model:value="form.name" placeholder="例如：技术文档" />
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">排序</label>
              <NInputNumber
                :value="form.sort_order"
                class="w-full"
                @update:value="(v: number | null) => form.sort_order = v ?? 0"
              />
            </div>
          </div>
          <div>
            <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">描述（可选）</label>
            <NInput v-model:value="form.description" placeholder="简短描述" />
          </div>

          <NAlert v-if="message" :type="message.includes('失败') || message.includes('不能') ? 'error' : 'success'" closable @close="message = ''">
            {{ message }}
          </NAlert>

          <div class="flex flex-col gap-2 sm:flex-row">
            <NButton attr-type="submit" type="primary" :loading="saving">{{ editing ? '更新' : '创建' }}</NButton>
            <NButton v-if="editing" quaternary @click="resetForm">取消</NButton>
          </div>
        </form>
      </section>

      <!-- 分类列表 -->
      <NDataTable
        :columns="columns"
        :data="categories"
        :row-key="(row: Category) => row.id"
        :bordered="false"
        :scroll-x="620"
        class="overflow-hidden rounded-xl border border-zinc-200 dark:border-zinc-800"
      />
    </template>

    <ConfirmDialog
      :open="confirmCat !== null"
      title="删除分类"
      :message="confirmCat ? `确定删除分类「${confirmCat.name}」吗？关联的关键词不会删除，仅解除关联。` : ''"
      confirm-label="删除"
      danger
      @confirm="doDelete"
      @cancel="confirmCat = null"
    />
  </div>
</template>
