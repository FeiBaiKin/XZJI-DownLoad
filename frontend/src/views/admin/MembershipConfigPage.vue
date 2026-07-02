<script setup lang="ts">
// 会员配置：1:1 移植自原项目 Node-XZJI/frontend/src/pages/admin/MembershipConfig.jsx
// 样式来自全局 styles/admin-membership.css
import { ref, onMounted } from 'vue'
import api from '@/api'
import type { MembershipConfig } from '@/types/api'

const levels = ref<MembershipConfig[]>([])
const loading = ref(true)
const saving = ref(false)
const message = ref('')
const expandedIdx = ref<number | null>(null)

onMounted(() => { fetchLevels() })

async function fetchLevels() {
  try {
    const res = await api.get('/admin/membership-config')
    levels.value = res.data
  } catch (err) { console.error(err) }
  finally { loading.value = false }
}

function updateLevel(idx: number, field: string, value: string | number) {
  levels.value[idx] = { ...levels.value[idx], [field]: value }
}

function addLevel() {
  levels.value.push({ level_name: '', days: 30, price: 0, daily_download_limit: 20, discount: 0, color: '#888888' })
  expandedIdx.value = levels.value.length - 1
}

function removeLevel(idx: number) {
  levels.value = levels.value.filter((_, i) => i !== idx)
  if (expandedIdx.value === idx) expandedIdx.value = null
  else if (expandedIdx.value !== null && expandedIdx.value > idx) expandedIdx.value -= 1
}

async function handleSave() {
  saving.value = true
  message.value = ''
  try {
    // 后端 membership-config PUT 接收纯数组（原版发送 {levels}），此处适配 GO 契约
    await api.put('/admin/membership-config', levels.value)
    message.value = '保存成功'
    fetchLevels()
  } catch (err: unknown) {
    const errorObj = err as { response?: { data?: { error?: string } } }
    message.value = errorObj?.response?.data?.error || '保存失败'
  }
  finally { saving.value = false }
}
</script>

<template>
  <div class="admin-content">
    <div v-if="loading" class="loading-state">加载中...</div>
    <template v-else>
      <div class="content-header">
        <h2 class="content-title">会员配置</h2>
        <button class="action-button secondary" @click="addLevel">+ 添加等级</button>
      </div>
      <form @submit.prevent="handleSave">
        <div class="mbr-list">
          <div v-if="levels.length === 0" class="empty-state">暂无会员等级，点击上方按钮添加</div>
          <div v-for="(lv, idx) in levels" :key="idx" class="mbr-row" :class="{ expanded: expandedIdx === idx }">
            <div class="mbr-summary" :style="{ borderLeftColor: lv.color || 'var(--border-default)' }" @click="expandedIdx = expandedIdx === idx ? null : idx">
              <span class="mbr-caret">{{ expandedIdx === idx ? '-' : '+' }}</span>
              <span class="mbr-name">{{ lv.level_name || '未命名' }}</span>
              <span class="mbr-stat"><span class="mbr-stat-label">价格</span><span class="mbr-stat-value">{{ lv.price > 0 ? `${lv.price}元` : '免费' }}</span></span>
              <span class="mbr-stat"><span class="mbr-stat-label">天数</span><span class="mbr-stat-value">{{ lv.days }}天</span></span>
              <span class="mbr-stat"><span class="mbr-stat-label">折扣</span><span class="mbr-stat-value">{{ lv.discount > 0 && lv.discount < 1 ? `${Math.round(lv.discount * 100)}折` : '-' }}</span></span>
              <span class="mbr-stat"><span class="mbr-stat-label">下载/天</span><span class="mbr-stat-value">{{ lv.daily_download_limit > 0 ? lv.daily_download_limit : '无限' }}</span></span>
              <span class="mbr-color-dot" :style="{ backgroundColor: lv.color || 'var(--text-secondary)' }" :title="lv.color || '未设置'" />
              <button type="button" class="mbr-remove" title="删除" @click.stop="removeLevel(idx)">
                <svg width="10" height="10" viewBox="0 0 10 10" fill="none" stroke="currentColor" stroke-width="1.2"><path d="M1 1l8 8M9 1l-8 8" /></svg>
              </button>
            </div>
            <div v-if="expandedIdx === idx" class="mbr-detail">
              <div class="mbr-fields">
                <div class="mbr-field">
                  <label class="mbr-label">等级名称</label>
                  <input class="form-input mbr-input" placeholder="等级名称" :value="lv.level_name" @input="(e) => updateLevel(idx, 'level_name', (e.target as HTMLInputElement).value)" />
                </div>
                <div class="mbr-field">
                  <label class="mbr-label">有效天数</label>
                  <input type="number" class="form-input mbr-input" min="1" :value="lv.days" @input="(e) => updateLevel(idx, 'days', parseInt((e.target as HTMLInputElement).value) || 0)" />
                </div>
                <div class="mbr-field">
                  <label class="mbr-label">价格 (元)</label>
                  <input type="number" class="form-input mbr-input" min="0" step="0.01" :value="lv.price" @input="(e) => updateLevel(idx, 'price', parseFloat((e.target as HTMLInputElement).value) || 0)" />
                </div>
                <div class="mbr-field">
                  <label class="mbr-label">折扣率</label>
                  <div class="mbr-discount-row">
                    <input type="number" class="form-input mbr-input" min="0" max="1" step="0.01" :value="lv.discount || 0" @input="(e) => updateLevel(idx, 'discount', parseFloat((e.target as HTMLInputElement).value) || 0)" />
                    <span class="mbr-discount-badge">{{ lv.discount > 0 && lv.discount < 1 ? `${Math.round(lv.discount * 100)}折` : '原价' }}</span>
                  </div>
                </div>
                <div class="mbr-field">
                  <label class="mbr-label">每日下载</label>
                  <input type="number" class="form-input mbr-input" min="0" :value="lv.daily_download_limit" @input="(e) => updateLevel(idx, 'daily_download_limit', parseInt((e.target as HTMLInputElement).value) || 0)" />
                </div>
                <div class="mbr-field">
                  <label class="mbr-label">标识颜色</label>
                  <div class="mbr-color-row">
                    <span class="mbr-color-preview" :style="{ backgroundColor: lv.color || 'var(--text-secondary)' }" />
                    <input type="text" class="form-input mbr-input" :value="lv.color || ''" @input="(e) => updateLevel(idx, 'color', (e.target as HTMLInputElement).value)" />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
        <div v-if="message" :class="message === '保存成功' ? 'form-success' : 'form-error'" style="margin-top: 1rem;">{{ message }}</div>
        <div class="form-actions">
          <button type="submit" class="action-button" :disabled="saving">{{ saving ? '保存中...' : '保存配置' }}</button>
        </div>
      </form>
    </template>
  </div>
</template>
