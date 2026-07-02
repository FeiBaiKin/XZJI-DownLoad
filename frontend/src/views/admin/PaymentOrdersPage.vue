<script setup lang="ts">
// 支付订单：1:1 移植自原项目 Node-XZJI/frontend/src/pages/admin/PaymentOrders.jsx
// 样式来自全局 styles/admin-orders.css
import { ref, onMounted, onUnmounted, computed } from 'vue'
import api from '@/api'
import { formatTime } from '@/utils/format'
import ConfirmDialog from '@/components/ConfirmDialog.vue'

interface Order {
  id: number
  order_no: string
  user_id: number
  amount: number
  points: number
  status: string
  payment_method: string
  trade_no: string
  created_at: string
  paid_at: string | null
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

const orders = ref<Order[]>([])
const stats = ref<OrderStats>({ total: 0, paidCount: 0, pendingCount: 0, totalRevenue: 0 })
const loading = ref(true)
const pagination = ref<Pagination>({ page: 1, limit: 20, total: 0, totalPages: 0 })
const filter = ref({ status: 'all', method: 'all', search: '' })
const selectedIds = ref<number[]>([])
const deleting = ref(false)
const message = ref('')
const confirmAction = ref<null | (() => void)>(null)
const confirmMeta = ref<{ type: string; count?: number; id?: number } | null>(null)

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

function toggleSelect(id: number) {
  selectedIds.value = selectedIds.value.includes(id)
    ? selectedIds.value.filter((x) => x !== id)
    : [...selectedIds.value, id]
}
const allSelected = computed(() => orders.value.length > 0 && selectedIds.value.length === orders.value.length)
function toggleSelectAll() {
  selectedIds.value = allSelected.value ? [] : orders.value.map((o) => o.id)
}

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

function statusLabel(s: string) { return s === 'paid' ? '已支付' : s === 'pending' ? '待支付' : s }
function methodLabel(m: string) {
  return m === 'wechat' ? '微信' : m === 'alipay' ? '支付宝' : (m === 'epay' || m === 'yipay') ? '易支付' : m === 'mock' ? '模拟' : (m || '-')
}
function statusClass(s: string) { return s === 'paid' ? 'order-status-paid' : 'order-status-pending' }
</script>

<template>
  <div class="admin-content">
    <div v-if="loading && orders.length === 0" class="loading-state">加载中...</div>
    <template v-else>
      <div class="content-header">
        <h2 class="content-title">支付订单</h2>
        <span class="content-sub">管理充值订单，支持筛选、删除</span>
      </div>

      <div class="order-stats-bar">
        <div class="order-stat-item"><span class="order-stat-value">{{ stats.total }}</span><span class="order-stat-label">总订单</span></div>
        <div class="order-stat-item"><span class="order-stat-value order-paid">{{ stats.paidCount }}</span><span class="order-stat-label">已支付</span></div>
        <div class="order-stat-item"><span class="order-stat-value order-pending">{{ stats.pendingCount }}</span><span class="order-stat-label">待支付</span></div>
        <div class="order-stat-item"><span class="order-stat-value">{{ parseFloat(stats.totalRevenue || 0).toFixed(2) }}</span><span class="order-stat-label">收入 (元)</span></div>
      </div>

      <div class="order-filters">
        <select class="form-input order-filter-select" :value="filter.status" @change="(e) => handleFilterChange('status', (e.target as HTMLSelectElement).value)">
          <option value="all">全部状态</option><option value="paid">已支付</option><option value="pending">待支付</option>
        </select>
        <select class="form-input order-filter-select" :value="filter.method" @change="(e) => handleFilterChange('method', (e.target as HTMLSelectElement).value)">
          <option value="all">全部方式</option><option value="wechat">微信支付</option><option value="alipay">支付宝</option><option value="epay">易支付</option>
        </select>
        <input type="text" class="form-input order-filter-search" placeholder="搜索订单号 / 用户名" :value="filter.search" @input="(e) => handleFilterChange('search', (e.target as HTMLInputElement).value)" />
        <button v-if="selectedIds.length > 0" class="action-button" style="background: var(--danger); color: #fff; border-color: var(--danger);" :disabled="deleting" @click="handleBatchDelete">
          {{ deleting ? '删除中...' : `删除选中 (${selectedIds.length})` }}
        </button>
      </div>

      <div v-if="message" class="form-message" style="margin-bottom: 0.75rem;">{{ message }}</div>

      <div class="order-table-wrap">
        <table class="order-table">
          <thead>
            <tr>
              <th style="width: 36px;"><input type="checkbox" :checked="allSelected" @change="toggleSelectAll" class="order-checkbox" /></th>
              <th>订单号</th><th>用户</th><th>金额</th><th>EXP</th><th>方式</th><th>场景</th><th>状态</th><th>时间</th><th style="width: 50px;">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="orders.length === 0"><td colspan="10" class="order-empty">暂无订单数据</td></tr>
            <tr v-for="o in orders" :key="o.id">
              <td><input type="checkbox" :checked="selectedIds.includes(o.id)" @change="toggleSelect(o.id)" class="order-checkbox" /></td>
              <td class="mono-td" :title="o.tradeNo || ''">{{ (o.orderNo || '').slice(-12) }}</td>
              <td class="user-cell">
                <img v-if="o.avatar" :src="o.avatar" alt="" class="order-avatar" />
                <span>{{ o.username }}</span>
              </td>
              <td class="mono-td">{{ (o.amount || 0).toFixed(2) }}</td>
              <td class="mono-td">{{ o.points || 0 }}</td>
              <td>{{ methodLabel(o.paymentMethod) }}</td>
              <td class="mono-td dim">{{ o.paymentScene || '-' }}</td>
              <td><span :class="statusClass(o.status)">{{ statusLabel(o.status) }}</span></td>
              <td class="mono-td dim">{{ formatTime(o.createdAt) }}</td>
              <td>
                <button class="order-delete-btn" title="删除" @click="handleDelete(o.id)">
                  <svg width="12" height="12" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"><path d="M2 4h10M5 4V2.5a.5.5 0 01.5-.5h3a.5.5 0 01.5.5V4M11 4v7.5a1 1 0 01-1 1H4a1 1 0 01-1-1V4" /><line x1="5.5" y1="6.5" x2="5.5" y2="10.5" /><line x1="8.5" y1="6.5" x2="8.5" y2="10.5" /></svg>
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="pagination.totalPages > 1" class="order-pagination">
        <button class="action-button secondary" :disabled="pagination.page <= 1" @click="setPage(pagination.page - 1)">上一页</button>
        <span class="pagination-info">{{ pagination.page }} / {{ pagination.totalPages }} (共 {{ pagination.total }} 条)</span>
        <button class="action-button secondary" :disabled="pagination.page >= pagination.totalPages" @click="setPage(pagination.page + 1)">下一页</button>
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
