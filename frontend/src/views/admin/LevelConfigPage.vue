<script setup lang="ts">
// 等级配置：1:1 移植自原项目 Node-XZJI/frontend/src/pages/admin/LevelConfig.jsx
// 样式来自全局 styles/admin-extra.css
import { ref, onMounted } from 'vue'
import api from '@/api'
import type { LevelConfig } from '@/types/api'

const levels = ref<LevelConfig[]>([])
const loading = ref(true)
const saving = ref(false)
const message = ref('')

onMounted(() => { fetchLevels() })

async function fetchLevels() {
  try {
    const res = await api.get('/admin/level-config')
    levels.value = res.data
  } catch (err) { console.error(err) }
  finally { loading.value = false }
}

function updateLevel(idx: number, field: string, value: string | number) {
  levels.value[idx] = { ...levels.value[idx], [field]: value }
}

function addLevel() {
  const maxLevel = levels.value.length > 0 ? Math.max(...levels.value.map((l) => l.level)) : 0
  const maxExp = levels.value.length > 0 ? Math.max(...levels.value.map((l) => l.exp_required)) : 0
  levels.value.push({ level: maxLevel + 1, exp_required: maxExp + 1000, label: '' })
}

function removeLevel(idx: number) {
  levels.value = levels.value.filter((_, i) => i !== idx)
}

async function handleSave() {
  saving.value = true
  message.value = ''
  try {
    // 后端 level-config PUT 接收纯数组（原版发送 {levels}），此处适配 GO 契约
    await api.put('/admin/level-config', levels.value)
    message.value = '保存成功'
    fetchLevels()
  } catch { message.value = '保存失败' }
  finally { saving.value = false }
}
</script>

<template>
  <div class="admin-content">
    <div v-if="loading" class="loading-state">加载中...</div>
    <template v-else>
      <div class="content-header">
        <h2 class="content-title">等级配置</h2>
        <button class="action-link" @click="addLevel">+ 添加等级</button>
      </div>
      <div class="profile-section">
        <form @submit.prevent="handleSave">
          <div class="level-config-list">
            <div v-for="(lv, idx) in levels" :key="idx" class="level-config-row">
              <div class="level-num">Lv.{{ lv.level }}</div>
              <input type="text" class="form-input level-label-input" placeholder="等级名称" :value="lv.label || ''" @input="(e) => updateLevel(idx, 'label', (e.target as HTMLInputElement).value)" />
              <input type="number" class="form-input level-exp-input" min="0" placeholder="所需经验" :value="lv.exp_required" @input="(e) => updateLevel(idx, 'exp_required', parseInt((e.target as HTMLInputElement).value) || 0)" />
              <button type="button" class="action-link" style="color: var(--danger);" @click="removeLevel(idx)">删除</button>
            </div>
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
