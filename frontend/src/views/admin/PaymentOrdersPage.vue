<script setup lang="ts">
// 支付订单：统计、筛选（状态/方式/搜索防抖）、批量删除、分页
// 业务逻辑与后端契约保持不变（后端返回 camelCase 行）；UI 重构为 Naive UI + Tailwind
import { ref, onMounted, onUnmounted, h } from 'vue'
import {
  NButton, NInput, NDataTable, NAlert, NSelect, NTag, NPagination, NSkeleton,
  type DataTableColumns,
} from 'naive-ui'
import api from '@/api'
import { formatTime } from '@/utils/format'
import ConfirmDialog from '@/components/ConfirmDialog.vue'

// 后端 adminOrderRow（handler/economy.go）返回 camelCase 字段
interface Order {
  id: number
  orderNo: string
  userId: number
  amount: number
  points: number
  status: string
  paymentMethod: string
  paymentScene: string
  tradeNo: string
  createdAt: string
  paidAt: string | null
  username: string
  avatar: string
}

interface OrderStats {
  total: number
  paidCount: number
  pendingCount: number
  totalRevenue: number
}

interface Pagination {
  page: number
  limit: number
  total: number
  totalPages: number
}

interface QueryParams {
  page: number
  limit: number
  status: string
  method: string
  search?: string
}

interface ConfirmMeta {
  title: string
  message: string
  confirmLabel: string
  danger: boolean
}

const orders = ref<Order[]>([])
const stats = ref<OrderStats>({ total: 0, paidCount: 0, pendingCount: 0, totalRevenue: 0 })
const loading = ref(true)
const pagination = ref<Pagination>({ page: 1, limit: 20, total: 0, totalPages: 0 })
const filter = ref({ status: 'all', method: 'all', search: '' })
const selectedIds = ref<number[]>([])
const deleting = ref(false)
const message = ref('')
const confirmAction = ref<null | (() => void)>(null)
const confirmMeta = ref<ConfirmMeta | null>(null)

const statusOptions = [
  { label: '全部状态', value: 'all' },
  { label: '已支付', value: 'paid' },
  { label: '待支付', value: 'pending' },
]
const methodOptions = [
  { label: '全部方式', value: 'all' },
  { label: '微信支付', value: 'wechat' },
  { label: '支付宝', value: 'alipay' },
  { label: '易支付', value: 'epay' },
]

onMounted(() => { fetchOrders() })
onUnmounted(() => { if (searchTimer) clearTimeout(searchTimer) })

let searchTimer: ReturnType<typeof setTimeout> | null = null

async function fetchOrders() {
  loading.value = true
  try {
    const params: QueryParams = { page: pagination.value.page, limit: pagination.value.limit, status: filter.value.status, method: filter.value.method }
    if (filter.value.search) params.search = filter.value.search
    const [ordersRes, statsRes] = await Promise.all([
      api.get('/admin/payment-orders', { params }),
      api.get('/admin/payment-orders/stats'),
    ])
    orders.value = ordersRes.data.list || []
    pagination.value = ordersRes.data.pagination
    stats.value = statsRes.data
    selectedIds.value = []
  } catch { /* ignore */ }
  finally { loading.value = false }
}

function handleFilterChange(key: 'status' | 'method' | 'search', value: string) {
  filter.value[key] = value
  if (key === 'search') {
    if (searchTimer) clearTimeout(searchTimer)
    searchTimer = setTimeout(() => { pagination.value.page = 1; fetchOrders() }, 300)
  } else {
    pagination.value.page = 1
    fetchOrders()
  }
}
function setPage(p: number) { pagination.value.page = p; fetchOrders() }

function handleDelete(id: number) {
  confirmMeta.value = { title: '删除订单', message: '确定删除该订单吗？', confirmLabel: '删除', danger: true }
  confirmAction.value = async () => {
    confirmAction.value = null; confirmMeta.value = null
    try {
      await api.delete(`/admin/payment-orders/${id}`)
      message.value = '订单已删除'; fetchOrders()
    } catch { message.value = '删除失败' }
  }
}
function handleBatchDelete() {
  if (selectedIds.value.length === 0) return
  const n = selectedIds.value.length
  confirmMeta.value = { title: '批量删除', message: `确定删除选中的 ${n} 条订单吗？此操作不可恢复。`, confirmLabel: `删除 ${n} 条`, danger: true }
  confirmAction.value = async () => {
    confirmAction.value = null; confirmMeta.value = null
    deleting.value = true; message.value = ''
    try {
      const res = await api.post('/admin/payment-orders/batch-delete', { ids: selectedIds.value })
      message.value = res.data.message || '批量删除成功'
      selectedIds.value = []
      fetchOrders()
    } catch { message.value = '批量删除失败' }
    finally { deleting.value = false }
  }
}

function methodLabel(m: string) {
  return m === 'wechat' ? '微信' : m === 'alipay' ? '支付宝' : (m === 'epay' || m === 'yipay') ? '易支付' : m === 'mock' ? '模拟' : (m || '-')
}

const columns: DataTableColumns<Order> = [
  { type: 'selection' },
  {
    title: '订单号',
    key: 'orderNo',
    width: 130,
    render: (row) => h('span', { class: 'font-mono text-xs text-zinc-600 dark:text-zinc-300', title: row.tradeNo || '' }, (row.orderNo || '').slice(-12)),
  },
  {
    title: '用户',
    key: 'username',
    width: 140,
    ellipsis: true,
    render: (row) =>
      h('div', { class: 'flex items-center gap-2 min-w-0' }, [
        row.avatar
          ? h('img', { src: row.avatar, alt: '', class: 'size-6 shrink-0 rounded-full object-cover' })
          : h('span', { class: 'grid size-6 shrink-0 place-items-center rounded-full bg-zinc-200 text-[10px] font-semibold text-zinc-600 dark:bg-zinc-700 dark:text-zinc-300' }, (row.username || '?')[0]?.toUpperCase()),
        h('span', { class: 'truncate text-[13px]' }, row.username),
      ]),
  },
  { title: '金额', key: 'amount', width: 90, render: (row) => h('span', { class: 'font-mono text-[13px] font-medium' }, (row.amount || 0).toFixed(2)) },
  { title: 'EXP', key: 'points', width: 70, render: (row) => h('span', { class: 'font-mono text-xs text-zinc-500 dark:text-zinc-400' }, row.points || 0) },
  { title: '方式', key: 'paymentMethod', width: 84, render: (row) => h('span', { class: 'text-[13px]' }, methodLabel(row.paymentMethod)) },
  { title: '场景', key: 'paymentScene', width: 90, render: (row) => h('span', { class: 'font-mono text-xs text-zinc-400' }, row.paymentScene || '-') },
  {
    title: '状态',
    key: 'status',
    width: 88,
    render: (row) =>
      h(
        NTag,
        { size: 'small', round: true, bordered: false, type: row.status === 'paid' ? 'success' : 'warning' },
        { default: () => (row.status === 'paid' ? '已支付' : row.status === 'pending' ? '待支付' : row.status) },
      ),
  },
  { title: '时间', key: 'createdAt', width: 110, render: (row) => h('span', { class: 'font-mono text-xs text-zinc-400' }, formatTime(row.createdAt)) },
  {
    title: '操作',
    key: 'actions',
    width: 72,
    render: (row) => h(NButton, { size: 'tiny', quaternary: true, type: 'error', onClick: () => handleDelete(row.id) }, { default: () => '删除' }),
  },
]
</script>

<template>
  <div>
    <div v-if="loading && orders.length === 0" class="space-y-4">
      <NSkeleton text width="20%" size="large" />
      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <NSkeleton v-for="i in 4" :key="i" height="84px" :sharp="false" />
      </div>
      <NSkeleton height="320px" :sharp="false" />
    </div>

    <template v-else>
      <header class="mb-6">
        <h2 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">支付订单</h2>
        <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">管理充值订单，支持筛选、删除</p>
      </header>

      <!-- 统计 -->
      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div class="rounded-2xl border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
          <p class="font-mono text-2xl font-semibold text-zinc-900 dark:text-zinc-50">{{ stats.total }}</p>
          <p class="mt-1 text-xs text-zinc-400 dark:text-zinc-500">总订单</p>
        </div>
        <div class="rounded-2xl border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
          <p class="font-mono text-2xl font-semibold text-emerald-600 dark:text-emerald-400">{{ stats.paidCount }}</p>
          <p class="mt-1 text-xs text-zinc-400 dark:text-zinc-500">已支付</p>
        </div>
        <div class="rounded-2xl border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
          <p class="font-mono text-2xl font-semibold text-amber-600 dark:text-amber-400">{{ stats.pendingCount }}</p>
          <p class="mt-1 text-xs text-zinc-400 dark:text-zinc-500">待支付</p>
        </div>
        <div class="rounded-2xl border border-zinc-200 bg-white p-4 dark:border-zinc-800 dark:bg-zinc-900">
          <p class="font-mono text-2xl font-semibold text-zinc-900 dark:text-zinc-50">{{ Number(stats.totalRevenue || 0).toFixed(2) }}</p>
          <p class="mt-1 text-xs text-zinc-400 dark:text-zinc-500">收入（元）</p>
        </div>
      </div>

      <!-- 筛选 -->
      <div class="mt-5 flex flex-wrap items-center gap-2.5">
        <NSelect
          class="w-full sm:w-32"
          :value="filter.status"
          :options="statusOptions"
          @update:value="(v: string) => handleFilterChange('status', v)"
        />
        <NSelect
          class="w-full sm:w-32"
          :value="filter.method"
          :options="methodOptions"
          @update:value="(v: string) => handleFilterChange('method', v)"
        />
        <NInput
          class="w-full sm:w-56"
          placeholder="搜索订单号 / 用户名"
          clearable
          :value="filter.search"
          @update:value="(v: string) => handleFilterChange('search', v)"
        />
        <NButton
          v-if="selectedIds.length > 0"
          type="error"
          class="w-full sm:w-auto"
          :loading="deleting"
          @click="handleBatchDelete"
        >
          删除选中（{{ selectedIds.length }}）
        </NButton>
      </div>

      <NAlert
        v-if="message"
        :type="message.includes('失败') ? 'error' : 'success'"
        class="mt-4"
        closable
        @close="message = ''"
      >{{ message }}</NAlert>

      <!-- 订单表 -->
      <NDataTable
        class="mt-4 overflow-hidden rounded-xl border border-zinc-200 dark:border-zinc-800"
        :columns="columns"
        :data="orders"
        :row-key="(row: Order) => row.id"
        :bordered="false"
        :loading="loading"
        :checked-row-keys="selectedIds"
        :scroll-x="920"
        @update:checked-row-keys="(keys) => selectedIds = keys as number[]"
      />

      <div v-if="pagination.totalPages > 1" class="mt-4 flex flex-wrap items-center justify-end gap-3">
        <span class="font-mono text-xs text-zinc-400 dark:text-zinc-500">共 {{ pagination.total }} 条</span>
        <NPagination :page="pagination.page" :page-count="pagination.totalPages" @update:page="setPage" />
      </div>
    </template>

    <ConfirmDialog
      :open="confirmAction !== null"
      :title="confirmMeta?.title"
      :message="confirmMeta?.message"
      :confirm-label="confirmMeta?.confirmLabel"
      :danger="confirmMeta?.danger"
      :loading="deleting"
      @confirm="confirmAction?.()"
      @cancel="confirmAction = null; confirmMeta = null"
    />
  </div>
</template>
