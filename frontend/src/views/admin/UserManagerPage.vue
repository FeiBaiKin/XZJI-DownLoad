<script setup lang="ts">
// 用户管理：分页列表 + 编辑（余额/经验/等级/会员/角色）+ 删除
// 业务逻辑与接口契约保持不变；UI 重构为 Naive UI + Tailwind
import { ref, onMounted, computed, h } from 'vue'
import {
  NButton, NInput, NInputNumber, NDataTable, NAlert, NSelect, NTag, NPagination, NModal, NIcon, NSkeleton,
  type DataTableColumns,
} from 'naive-ui'
import { X } from '@vicons/tabler'
import api from '@/api'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import { useToastStore } from '@/stores/toast'
import type { UserBrief } from '@/types/api'

interface UserEditForm {
  balance: number
  exp: number
  level: number
  membership_level: string | null
  membership_expires_at: string | null
  role: string
  nickname: string
  email: string
}

const toast = useToastStore()
const users = ref<UserBrief[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(true)
const editingUser = ref<UserBrief | null>(null)
const editForm = ref<UserEditForm>({
  balance: 0,
  exp: 0,
  level: 1,
  membership_level: null,
  membership_expires_at: null,
  role: 'user',
  nickname: '',
  email: '',
})
const saving = ref(false)
const message = ref('')
const confirmDeleteId = ref<number | null>(null)
const limit = 15

const roleOptions = [
  { label: '普通用户', value: 'user' },
  { label: '管理员', value: 'admin' },
]

// 会员等级下拉选项：来自会员配置，避免手输拼错导致与档位不匹配（tag 模式仍允许自定义值）
const tierOptions = ref<{ label: string; value: string }[]>([])
async function fetchTierOptions() {
  try {
    const res = await api.get('/admin/membership-config')
    tierOptions.value = (res.data || []).map((l: { level_name: string }) => ({ label: l.level_name, value: l.level_name }))
  } catch { /* 加载失败不阻塞编辑，仍可手输 */ }
}

onMounted(() => { fetchUsers(); fetchTierOptions() })

async function fetchUsers() {
  loading.value = true
  try {
    const res = await api.get('/admin/users', { params: { page: page.value, limit } })
    users.value = res.data.users
    total.value = res.data.total
  } catch (err) { console.error(err); toast.error('加载失败，请刷新重试') }
  finally { loading.value = false }
}

function setPage(p: number) { page.value = p; fetchUsers() }

const totalPages = computed(() => Math.ceil(total.value / limit))

function openEdit(user: UserBrief) {
  editingUser.value = user
  editForm.value = {
    balance: user.balance || 0,
    exp: user.exp || 0,
    level: user.level || 1,
    membership_level: user.membership_level || '',
    membership_expires_at: user.membership_expires_at ? user.membership_expires_at.slice(0, 16) : '',
    role: user.role || 'user',
    nickname: user.nickname || '',
    email: user.email || '',
  }
}

async function handleSave() {
  saving.value = true
  message.value = ''
  try {
    const body: UserEditForm = { ...editForm.value }
    if (!body.membership_level) { body.membership_level = null; body.membership_expires_at = null }
    await api.put(`/admin/users/${editingUser.value!.id}`, body)
    message.value = '保存成功'
    editingUser.value = null
    fetchUsers()
  } catch (err: unknown) {
    const errorObj = err as { response?: { data?: { error?: string } } }
    message.value = errorObj?.response?.data?.error || '保存失败'
  } finally {
    saving.value = false
  }
}

function handleDelete(id: number) { confirmDeleteId.value = id }
async function doDelete() {
  const id = confirmDeleteId.value
  confirmDeleteId.value = null
  if (id == null) return
  try {
    const res = await api.delete(`/admin/users/${id}`)
    if (res.status >= 200 && res.status < 300) fetchUsers()
    else toast.error('删除失败')
  } catch { toast.error('删除失败') }
}

const columns: DataTableColumns<UserBrief> = [
  { title: 'ID', key: 'id', width: 64, render: (row) => h('span', { class: 'font-mono text-xs text-zinc-500 dark:text-zinc-400' }, row.id) },
  { title: '用户名', key: 'username', width: 130, ellipsis: true, render: (row) => h('span', { class: 'font-medium' }, row.username) },
  { title: '昵称', key: 'nickname', width: 130, ellipsis: true, render: (row) => h('span', { class: 'text-[13px] text-zinc-700 dark:text-zinc-200' }, row.nickname || row.username || '-') },
  { title: '邮箱', key: 'email', ellipsis: true, render: (row) => h('span', { class: 'text-[13px] text-zinc-500 dark:text-zinc-400' }, row.email || '-') },
  { title: '余额', key: 'balance', width: 90, render: (row) => h('span', { class: 'font-mono text-[13px]' }, Number(row.balance || 0).toFixed(2)) },
  { title: '经验', key: 'exp', width: 76, render: (row) => h('span', { class: 'font-mono text-[13px]' }, row.exp || 0) },
  { title: '等级', key: 'level', width: 76, render: (row) => h('span', { class: 'font-mono text-[13px]' }, `Lv.${row.level || 1}`) },
  {
    title: '会员',
    key: 'membership_level',
    width: 100,
    render: (row) =>
      row.membership_level
        ? h(NTag, { size: 'small', round: true, type: 'warning', bordered: false }, { default: () => row.membership_level })
        : h('span', { class: 'text-xs text-zinc-500 dark:text-zinc-400' }, '无'),
  },
  {
    title: '操作',
    key: 'actions',
    width: 130,
    render: (row) =>
      h('div', { class: 'flex gap-1' }, [
        h(NButton, { size: 'tiny', quaternary: true, type: 'primary', onClick: () => openEdit(row) }, { default: () => '编辑' }),
        h(NButton, { size: 'tiny', quaternary: true, type: 'error', onClick: () => handleDelete(row.id) }, { default: () => '删除' }),
      ]),
  },
]
</script>

<template>
  <div>
    <div v-if="loading && users.length === 0" class="space-y-3">
      <NSkeleton text width="20%" size="large" />
      <NSkeleton v-for="i in 6" :key="i" height="42px" :sharp="false" />
    </div>

    <template v-else>
      <header class="mb-6">
        <h1 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">用户管理</h1>
        <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">共 {{ total }} 个用户</p>
      </header>

      <NAlert
        v-if="message"
        :type="message === '保存成功' ? 'success' : 'error'"
        class="mb-4"
        closable
        @close="message = ''"
      >{{ message }}</NAlert>

      <NDataTable
        :columns="columns"
        :data="users"
        :row-key="(row: UserBrief) => row.id"
        :bordered="false"
        :loading="loading"
        :scroll-x="950"
        class="overflow-hidden rounded-xl border border-zinc-200 dark:border-zinc-800"
      />

      <div v-if="totalPages > 1" class="mt-4 flex justify-end">
        <NPagination :page="page" :page-count="totalPages" @update:page="setPage" />
      </div>

      <!-- 编辑弹窗 -->
      <NModal :show="!!editingUser" :mask-closable="!saving" @update:show="(v: boolean) => { if (!v) editingUser = null }">
        <div v-if="editingUser" class="w-[92vw] max-w-lg rounded-2xl border border-zinc-200 bg-white shadow-xl dark:border-zinc-800 dark:bg-zinc-900">
          <div class="flex items-center justify-between border-b border-zinc-100 px-6 py-4 dark:border-zinc-800">
            <h3 class="text-base font-semibold text-zinc-900 dark:text-zinc-100">编辑用户：{{ editingUser.username }}</h3>
            <button
              type="button"
              class="flex size-8 items-center justify-center rounded-full text-zinc-400 transition-colors hover:bg-zinc-100 hover:text-zinc-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:hover:bg-zinc-800 dark:hover:text-zinc-300 dark:focus-visible:ring-emerald-400/60"
              aria-label="关闭"
              @click="editingUser = null"
            >
              <NIcon :size="16"><X /></NIcon>
            </button>
          </div>

          <div class="space-y-4 px-6 py-5">
            <div class="grid gap-3 sm:grid-cols-2">
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">昵称</label>
                <NInput v-model:value="editForm.nickname" placeholder="昵称" />
              </div>
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">邮箱</label>
                <NInput v-model:value="editForm.email" placeholder="邮箱" :input-props="{ type: 'email' }" />
              </div>
            </div>

            <div class="grid gap-3 sm:grid-cols-3">
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">余额</label>
                <NInputNumber
                  :value="editForm.balance"
                  :step="0.01"
                  :show-button="false"
                  class="w-full"
                  @update:value="(v: number | null) => editForm.balance = v ?? 0"
                />
              </div>
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">经验</label>
                <NInputNumber
                  :value="editForm.exp"
                  :show-button="false"
                  class="w-full"
                  @update:value="(v: number | null) => editForm.exp = v ?? 0"
                />
              </div>
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">等级</label>
                <NInputNumber
                  :value="editForm.level"
                  :min="1"
                  :show-button="false"
                  class="w-full"
                  @update:value="(v: number | null) => editForm.level = v ?? 1"
                />
              </div>
            </div>

            <div class="grid gap-3 sm:grid-cols-2">
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">会员等级</label>
                <NSelect
                  :value="editForm.membership_level || null"
                  :options="tierOptions"
                  filterable
                  tag
                  clearable
                  placeholder="选择档位，留空 = 无会员"
                  @update:value="(v: string | null) => editForm.membership_level = v || ''"
                />
              </div>
              <div>
                <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">会员到期</label>
                <!-- 保持 datetime-local 原生控件与 'YYYY-MM-DDTHH:mm' 字符串契约 -->
                <input
                  type="datetime-local"
                  aria-label="会员到期"
                  class="h-[34px] w-full rounded-lg border border-zinc-300 bg-transparent px-3 text-sm text-zinc-900 outline-none focus:border-emerald-500 dark:border-zinc-700 dark:text-zinc-100 dark:[color-scheme:dark]"
                  :value="editForm.membership_expires_at || ''"
                  @input="(e) => editForm.membership_expires_at = (e.target as HTMLInputElement).value"
                />
              </div>
            </div>

            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">角色</label>
              <NSelect v-model:value="editForm.role" :options="roleOptions" />
            </div>

            <div class="flex flex-col gap-2 pt-1 sm:flex-row">
              <NButton type="primary" :loading="saving" @click="handleSave">保存</NButton>
              <NButton quaternary @click="editingUser = null">取消</NButton>
            </div>
          </div>
        </div>
      </NModal>
    </template>

    <ConfirmDialog
      :open="confirmDeleteId !== null"
      title="删除用户"
      message="确定要删除该用户吗？此操作不可恢复。"
      confirm-label="删除"
      danger
      @confirm="doDelete"
      @cancel="confirmDeleteId = null"
    />
  </div>
</template>
