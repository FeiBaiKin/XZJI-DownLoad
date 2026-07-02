<script setup lang="ts">
// 个人资料：1:1 移植自原项目 Node-XZJI/frontend/src/pages/UserCenter.jsx (UserProfile)
// 样式来自全局 styles/user.css
import { ref, onMounted } from 'vue'
import api from '@/api'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()

const profile = ref<any>(null)
const loading = ref(true)
const editing = ref(false)
const saving = ref(false)
const message = ref('')
const error = ref('')
const formData = ref({ nickname: '', email: '' })
const fileInputRef = ref<HTMLInputElement | null>(null)

onMounted(() => { fetchProfile() })

async function fetchProfile() {
  try {
    const res = await api.get('/user/profile')
    profile.value = res.data
    formData.value = { nickname: res.data.nickname || '', email: res.data.email || '' }
  } catch (err) {
    console.error('获取资料失败:', err)
  } finally {
    loading.value = false
  }
}

async function handleSaveProfile() {
  saving.value = true
  error.value = ''
  message.value = ''
  try {
    const res = await api.put('/user/profile', formData.value)
    profile.value = res.data
    editing.value = false
    message.value = '资料更新成功'
    if (auth.user) auth.user.nickname = res.data.nickname
  } catch (err: any) {
    error.value = err?.response?.data?.error || '保存失败'
  } finally {
    saving.value = false
  }
}

async function handleAvatarUpload(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return

  if (file.size > 2 * 1024 * 1024) {
    error.value = '文件大小不能超过 2MB'
    return
  }

  error.value = ''
  message.value = ''

  const fd = new FormData()
  fd.append('avatar', file)
  try {
    const res = await api.post('/user/avatar', fd)
    profile.value = { ...profile.value, avatar: res.data.avatar }
    message.value = '头像更新成功'
    if (auth.user) auth.user.avatar = res.data.avatar
  } catch (err: any) {
    error.value = err?.response?.data?.error || '上传失败'
  }
}

function startEdit() {
  formData.value = { nickname: profile.value.nickname || '', email: profile.value.email || '' }
  editing.value = true
  error.value = ''
}
function cancelEdit() {
  editing.value = false
  formData.value = { nickname: profile.value?.nickname || '', email: profile.value?.email || '' }
  error.value = ''
}
</script>

<template>
  <div class="user-content">
    <div v-if="loading" class="loading-state">加载中...</div>
    <template v-else>
      <div v-if="!profile" class="empty-state">无法加载用户信息</div>
      <template v-else>
        <div class="content-header">
          <h2 class="content-title">个人资料</h2>
        </div>

        <!-- 头像区域 -->
        <div class="profile-avatar-section">
          <div class="avatar-wrapper" @click="fileInputRef?.click()">
            <img v-if="profile.avatar" :src="profile.avatar" alt="头像" class="avatar-image" />
            <div v-else class="avatar-placeholder">
              <span>{{ (profile.nickname || profile.username || 'U')[0].toUpperCase() }}</span>
            </div>
            <div class="avatar-overlay">
              <span class="avatar-overlay-text">更换头像</span>
            </div>
          </div>
          <input
            ref="fileInputRef"
            type="file"
            accept="image/jpeg,image/png,image/gif,image/webp"
            class="avatar-input-hidden"
            @change="handleAvatarUpload"
          />
          <div class="avatar-info">
            <span class="avatar-name">{{ profile.nickname || profile.username }}</span>
            <span class="avatar-hint">支持 JPG、PNG、GIF、WebP，最大 2MB</span>
          </div>
        </div>

        <!-- 基本信息 -->
        <div class="profile-section">
          <div class="section-header">
            <h3 class="section-title">基本信息</h3>
            <button v-if="!editing" class="action-link" @click="startEdit">编辑</button>
          </div>

          <div v-if="message" class="form-success">{{ message }}</div>
          <div v-if="error" class="form-error">{{ error }}</div>

          <form v-if="editing" class="profile-form" @submit.prevent="handleSaveProfile">
            <div class="form-group">
              <label class="form-label">用户名</label>
              <input type="text" class="form-input" :value="profile.username" disabled />
              <span class="form-hint">用户名不可修改</span>
            </div>
            <div class="form-group">
              <label class="form-label">昵称</label>
              <input type="text" class="form-input" v-model="formData.nickname" />
            </div>
            <div class="form-group">
              <label class="form-label">邮箱</label>
              <input type="email" class="form-input" v-model="formData.email" placeholder="用于找回密码" />
            </div>
            <div class="form-actions">
              <button type="submit" class="action-button" :disabled="saving">{{ saving ? '保存中...' : '保存' }}</button>
              <button type="button" class="action-button secondary" @click="cancelEdit">取消</button>
            </div>
          </form>

          <div v-else class="profile-details">
            <div class="detail-row"><span class="detail-label">用户名</span><span class="detail-value">{{ profile.username }}</span></div>
            <div class="detail-divider" />
            <div class="detail-row"><span class="detail-label">昵称</span><span class="detail-value">{{ profile.nickname || '未设置' }}</span></div>
            <div class="detail-divider" />
            <div class="detail-row"><span class="detail-label">邮箱</span><span class="detail-value">{{ profile.email || '未设置' }}</span></div>
            <div class="detail-divider" />
            <div class="detail-row"><span class="detail-label">等级</span><span class="detail-value mono">Lv.{{ profile.level || 1 }}</span></div>
            <div class="detail-divider" />
            <div class="detail-row"><span class="detail-label">经验</span><span class="detail-value mono">{{ profile.exp || 0 }} EXP</span></div>
            <div class="detail-divider" />
            <div class="detail-row"><span class="detail-label">余额</span><span class="detail-value mono">{{ (profile.balance || 0).toFixed(2) }} 元</span></div>
            <div class="detail-divider" />
            <div class="detail-row">
              <span class="detail-label">会员</span>
              <span class="detail-value mono">
                {{ profile.membership_level ? `${profile.membership_level} (${profile.membership_expires_at ? new Date(profile.membership_expires_at).toLocaleDateString('zh-CN') : ''})` : '无' }}
              </span>
            </div>
            <div class="detail-divider" />
            <div class="detail-row"><span class="detail-label">角色</span><span class="detail-value detail-role">{{ profile.role === 'admin' ? '管理员' : '普通用户' }}</span></div>
            <div class="detail-divider" />
            <div class="detail-row"><span class="detail-label">注册时间</span><span class="detail-value detail-time">{{ new Date(profile.created_at).toLocaleString('zh-CN') }}</span></div>
          </div>
        </div>
      </template>
    </template>
  </div>
</template>
