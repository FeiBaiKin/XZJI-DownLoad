<script setup lang="ts">
// 签到配置：1:1 移植自原项目 Node-XZJI/frontend/src/pages/admin/CheckinConfig.jsx
// 样式来自全局 styles/admin-layout.css + user.css
import { ref, onMounted } from 'vue'
import api from '@/api'

const config = ref<any>({ daily_balance: 0.50, consecutive_days: 7, consecutive_bonus_balance: 2.00, daily_exp: 10 })
const loading = ref(true)
const saving = ref(false)
const message = ref('')

onMounted(() => { fetchConfig() })

async function fetchConfig() {
  try {
    const res = await api.get('/admin/checkin-config')
    config.value = res.data
  } catch (err) { console.error(err) }
  finally { loading.value = false }
}

async function handleSave() {
  saving.value = true
  message.value = ''
  try {
    await api.put('/admin/checkin-config', config.value)
    message.value = '保存成功'
  } catch { message.value = '保存失败' }
  finally { saving.value = false }
}
</script>

<template>
  <div class="admin-content">
    <div v-if="loading" class="loading-state">加载中...</div>
    <template v-else>
      <div class="content-header">
        <h2 class="content-title">签到配置</h2>
      </div>
      <div class="profile-section">
        <form @submit.prevent="handleSave">
          <div class="form-group">
            <label class="form-label">每日签到余额 (元)</label>
            <input type="number" class="form-input" min="0" step="0.01" v-model.number="config.daily_balance" />
          </div>
          <div class="form-group">
            <label class="form-label">每日签到经验</label>
            <input type="number" class="form-input" min="1" v-model.number="config.daily_exp" />
          </div>
          <div class="form-group">
            <label class="form-label">连续签到天数（触发奖励）</label>
            <input type="number" class="form-input" min="2" v-model.number="config.consecutive_days" />
          </div>
          <div class="form-group">
            <label class="form-label">连续签到额外余额 (元)</label>
            <input type="number" class="form-input" min="0" step="0.01" v-model.number="config.consecutive_bonus_balance" />
          </div>
          <div v-if="message" :class="message === '保存成功' ? 'form-success' : 'form-error'">{{ message }}</div>
          <div class="form-actions">
            <button type="submit" class="action-button" :disabled="saving">{{ saving ? '保存中...' : '保存配置' }}</button>
          </div>
        </form>
      </div>
    </template>
  </div>
</template>
