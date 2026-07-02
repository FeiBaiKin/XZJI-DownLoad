<script setup lang="ts">
// 管理后台仪表盘：1:1 移植自原项目 Node-XZJI/frontend/src/pages/admin/AdminDashboard.jsx
// 样式来自全局 styles/admin-dashboard.css
import { ref, onMounted, computed } from 'vue'
import api from '@/api'
import { formatMoney } from '@/utils/format'
import { methodLabels } from '@/constants/admin'

const stats = ref<any>(null)
const loading = ref(true)

onMounted(() => { fetchStats() })

async function fetchStats() {
  try {
    const res = await api.get('/admin/stats')
    stats.value = res.data
  } catch (err) {
    console.error('获取统计失败:', err)
  } finally {
    loading.value = false
  }
}

const totalRevenue = computed(() => parseFloat(stats.value?.revenueTotal) || 0)
const methodList = computed(() => stats.value?.methodBreakdown || [])
function methodPct(revenue: number) {
  return totalRevenue.value > 0 ? Math.round((parseFloat(revenue) / totalRevenue.value) * 100) : 0
}
</script>

<template>
  <div class="admin-content">
    <div v-if="loading" class="loading-state">加载中...</div>
    <template v-else>
      <div v-if="!stats" class="empty-state">暂无数据</div>
      <template v-else>
        <div class="content-header">
          <h2 class="content-title">仪表盘</h2>
          <span class="content-sub">{{ stats.siteName || '关键词搜索' }} - 数据概览</span>
        </div>

        <div class="dash-metrics">
          <div class="dash-metric">
            <span class="dash-metric-label">注册用户</span>
            <span class="dash-metric-value">{{ stats.users || 0 }}</span>
            <span class="dash-metric-sub">本月 +{{ stats.newUsersMonth || 0 }} / 本周 +{{ stats.newUsersWeek || 0 }}</span>
          </div>
          <div class="dash-metric">
            <span class="dash-metric-label">总收入</span>
            <span class="dash-metric-value">{{ formatMoney(totalRevenue) }}</span>
            <span class="dash-metric-sub">本月 {{ formatMoney(stats.revenueMonth || 0) }}</span>
          </div>
          <div class="dash-metric">
            <span class="dash-metric-label">订单总数</span>
            <span class="dash-metric-value">{{ stats.ordersTotal || 0 }}</span>
            <span class="dash-metric-sub">已支付 {{ stats.ordersPaid || 0 }} / 待支付 {{ stats.ordersPending || 0 }}</span>
          </div>
          <div class="dash-metric">
            <span class="dash-metric-label">活跃会员</span>
            <span class="dash-metric-value">{{ stats.activeMemberships || 0 }}</span>
            <span class="dash-metric-sub" :style="stats.expiringMemberships > 0 ? { color: 'var(--danger)' } : {}">
              {{ stats.expiringMemberships > 0 ? `${stats.expiringMemberships} 人即将到期` : '全部正常' }}
            </span>
          </div>
        </div>

        <div class="dash-section">
          <div class="dash-section-header"><h3 class="dash-section-title">收入概览</h3></div>
          <div class="dash-row">
            <div class="dash-card">
              <div class="dash-card-header"><span class="dash-card-label">收入明细</span></div>
              <div class="dash-revenue-stats">
                <div class="dash-revenue-item"><span class="dash-revenue-amount">{{ formatMoney(stats.revenueToday || 0) }}</span><span class="dash-revenue-period">今日收入</span></div>
                <div class="dash-revenue-item"><span class="dash-revenue-amount">{{ formatMoney(stats.revenueMonth || 0) }}</span><span class="dash-revenue-period">本月收入</span></div>
                <div class="dash-revenue-item"><span class="dash-revenue-amount">{{ formatMoney(totalRevenue) }}</span><span class="dash-revenue-period">累计收入</span></div>
              </div>
            </div>
            <div class="dash-card">
              <div class="dash-card-header"><span class="dash-card-label">支付方式</span></div>
              <div class="dash-method-list">
                <span v-if="methodList.length === 0" class="dash-empty">暂无支付数据</span>
                <div v-for="m in methodList" :key="m.payment_method" class="dash-method-row">
                  <div class="dash-method-info">
                    <span class="dash-method-name">{{ methodLabels[m.payment_method] || m.payment_method }}</span>
                    <span class="dash-method-count">{{ m.count }} 笔</span>
                  </div>
                  <div class="dash-method-bar-track"><div class="dash-method-bar-fill" :style="{ width: methodPct(m.revenue) + '%' }" /></div>
                  <span class="dash-method-pct">{{ methodPct(m.revenue) }}%</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="dash-section">
          <div class="dash-section-header"><h3 class="dash-section-title">用户与内容</h3></div>
          <div class="dash-row dash-row-3">
            <div class="dash-card">
              <div class="dash-card-header"><span class="dash-card-label">用户增长</span></div>
              <div class="dash-stats-vert">
                <div class="dash-stat-row"><span class="dash-stat-num">{{ stats.newUsersToday || 0 }}</span><span class="dash-stat-desc">今日新增</span></div>
                <div class="dash-stat-row"><span class="dash-stat-num">{{ stats.newUsersWeek || 0 }}</span><span class="dash-stat-desc">本周新增</span></div>
                <div class="dash-stat-row"><span class="dash-stat-num">{{ stats.newUsersMonth || 0 }}</span><span class="dash-stat-desc">本月新增</span></div>
              </div>
            </div>
            <div class="dash-card">
              <div class="dash-card-header"><span class="dash-card-label">内容数据</span></div>
              <div class="dash-stats-vert">
                <div class="dash-stat-row"><span class="dash-stat-num">{{ stats.keywords || 0 }}</span><span class="dash-stat-desc">关键词总数</span></div>
                <div class="dash-stat-row"><span class="dash-stat-num">{{ stats.checkinsToday || 0 }}</span><span class="dash-stat-desc">今日签到</span></div>
              </div>
            </div>
            <div class="dash-card">
              <div class="dash-card-header"><span class="dash-card-label">会员状态</span></div>
              <div class="dash-stats-vert">
                <div class="dash-stat-row"><span class="dash-stat-num">{{ stats.activeMemberships || 0 }}</span><span class="dash-stat-desc">活跃会员</span></div>
                <div class="dash-stat-row"><span class="dash-stat-num" :style="stats.expiringMemberships > 0 ? { color: 'var(--danger)' } : {}">{{ stats.expiringMemberships || 0 }}</span><span class="dash-stat-desc">7天内到期</span></div>
              </div>
            </div>
          </div>
        </div>
      </template>
    </template>
  </div>
</template>
