<script setup lang="ts">
// 安全设置/修改密码：1:1 移植自原项目 Node-XZJI/frontend/src/pages/UserCenter.jsx (ChangePassword)
// 样式来自全局 styles/user.css
import { ref } from 'vue'
import api from '@/api'

const formData = ref({ currentPassword: '', newPassword: '', confirmPassword: '' })
const loading = ref(false)
const message = ref('')
const error = ref('')

async function handleSubmit() {
  error.value = ''
  message.value = ''

  if (!formData.value.currentPassword || !formData.value.newPassword) {
    error.value = '请填写当前密码和新密码'
    return
  }
  if (formData.value.newPassword.length < 6) {
    error.value = '新密码至少需要6个字符'
    return
  }
  if (formData.value.newPassword !== formData.value.confirmPassword) {
    error.value = '两次输入的新密码不一致'
    return
  }

  loading.value = true
  try {
    await api.put('/user/password', {
      currentPassword: formData.value.currentPassword,
      newPassword: formData.value.newPassword,
    })
    message.value = '密码修改成功'
    formData.value = { currentPassword: '', newPassword: '', confirmPassword: '' }
  } catch (err: any) {
    error.value = err?.response?.data?.error || '修改失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="user-content">
    <div class="content-header">
      <h2 class="content-title">安全设置</h2>
    </div>

    <div class="profile-section">
      <div class="section-header">
        <h3 class="section-title">修改密码</h3>
      </div>

      <div v-if="message" class="form-success">{{ message }}</div>
      <div v-if="error" class="form-error">{{ error }}</div>

      <form class="profile-form" @submit.prevent="handleSubmit">
        <div class="form-group">
          <label for="currentPassword" class="form-label">当前密码</label>
          <input id="currentPassword" type="password" class="form-input" v-model="formData.currentPassword" :disabled="loading" autocomplete="current-password" />
        </div>
        <div class="form-group">
          <label for="newPassword" class="form-label">新密码</label>
          <input id="newPassword" type="password" class="form-input" placeholder="至少6个字符" v-model="formData.newPassword" :disabled="loading" autocomplete="new-password" />
        </div>
        <div class="form-group">
          <label for="confirmPassword" class="form-label">确认新密码</label>
          <input id="confirmPassword" type="password" class="form-input" placeholder="再次输入新密码" v-model="formData.confirmPassword" :disabled="loading" autocomplete="new-password" />
        </div>
        <div class="form-actions">
          <button type="submit" class="action-button" :disabled="loading">{{ loading ? '修改中...' : '修改密码' }}</button>
        </div>
      </form>
    </div>
  </div>
</template>
