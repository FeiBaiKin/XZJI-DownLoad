<script setup lang="ts">
// 用户管理：1:1 移植自原项目 Node-XZJI/frontend/src/pages/admin/UserManager.jsx
// 样式来自全局 styles/admin-layout.css
import { ref, onMounted, computed } from 'vue'
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
  email: ''
})
const saving = ref(false)
const message = ref('')
const confirmDeleteId = ref<number | null>(null)
const limit = 15

onMounted(() => { fetchUsers() })

async function fetchUsers() {
  loading.value = true
  try {
    const res = await api.get('/admin/users', { params: { page: page.value, limit } })
    users.value = res.data.users
    total.value = res.data.total
  } catch (err) { console.error(err) }
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
</script>

<template>
  <div class="admin-content">
    <div v-if="loading && users.length === 0" class="loading-state">加载中...</div>
    <template v-else>
      <div class="content-header">
        <h2 class="content-title">用户管理</h2>
        <span class="content-sub">共 {{ total }} 个用户</span>
      </div>
      <div v-if="message" :class="message === '保存成功' ? 'form-success' : 'form-error'">{{ message }}</div>

      <div class="data-table">
        <div class="table-header" style="grid-template-columns: 50px 100px 1fr 80px 60px 60px 80px 80px;">
          <span class="table-cell">ID</span>
          <span class="table-cell">用户名</span>
          <span class="table-cell">邮箱</span>
          <span class="table-cell">余额</span>
          <span class="table-cell">经验</span>
          <span class="table-cell">等级</span>
          <span class="table-cell">会员</span>
          <span class="table-cell">操作</span>
        </div>
        <div v-for="u in users" :key="u.id" class="table-row" style="grid-template-columns: 50px 100px 1fr 80px 60px 60px 80px 80px;">
          <span class="mono">{{ u.id }}</span>
          <span>{{ u.username }}</span>
          <span class="text-secondary">{{ u.email || '-' }}</span>
          <span class="mono">{{ parseFloat(u.balance || 0).toFixed(2) }}</span>
          <span class="mono">{{ u.exp || 0 }}</span>
          <span class="mono">Lv.{{ u.level || 1 }}</span>
          <span :class="u.membership_level ? 'membership-badge' : 'text-tertiary'">{{ u.membership_level || '无' }}</span>
          <span class="table-actions">
            <button class="action-link" @click="openEdit(u)">编辑</button>
            <button class="action-link" style="color: var(--danger);" @click="handleDelete(u.id)">删除</button>
          </span>
        </div>
      </div>

      <div v-if="totalPages > 1" class="pagination">
        <button :disabled="page <= 1" @click="setPage(page - 1)">上一页</button>
        <span>{{ page }} / {{ totalPages }}</span>
        <button :disabled="page >= totalPages" @click="setPage(page + 1)">下一页</button>
      </div>

      <!-- 编辑弹窗 -->
      <div v-if="editingUser" class="modal-overlay" @click="editingUser = null">
        <div class="modal-content" style="max-width: 480px;" @click.stop>
          <div class="modal-header">
            <h3>编辑用户: {{ editingUser.username }}</h3>
            <button class="modal-close" @click="editingUser = null">X</button>
          </div>
          <div style="padding: 1.25rem; display: flex; flex-direction: column; gap: 0.75rem;">
            <div class="form-group">
              <label class="form-label">昵称</label>
              <input class="form-input" v-model="editForm.nickname" />
            </div>
            <div class="form-group">
              <label class="form-label">邮箱</label>
              <input class="form-input" v-model="editForm.email" />
            </div>
            <div style="display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 0.75rem;">
              <div class="form-group">
                <label class="form-label">余额</label>
                <input type="number" class="form-input" step="0.01" v-model="editForm.balance" />
              </div>
              <div class="form-group">
                <label class="form-label">经验</label>
                <input type="number" class="form-input" v-model="editForm.exp" />
              </div>
              <div class="form-group">
                <label class="form-label">等级</label>
                <input type="number" class="form-input" min="1" v-model="editForm.level" />
              </div>
            </div>
            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem;">
              <div class="form-group">
                <label class="form-label">会员等级</label>
                <input class="form-input" placeholder="留空=无会员" v-model="editForm.membership_level" />
              </div>
              <div class="form-group">
                <label class="form-label">会员到期</label>
                <input type="datetime-local" class="form-input" v-model="editForm.membership_expires_at" />
              </div>
            </div>
            <div class="form-group">
              <label class="form-label">角色</label>
              <select class="form-input" v-model="editForm.role">
                <option value="user">普通用户</option>
                <option value="admin">管理员</option>
              </select>
            </div>
            <div class="form-actions">
              <button class="action-button" @click="handleSave" :disabled="saving">{{ saving ? '保存中...' : '保存' }}</button>
              <button class="action-button secondary" @click="editingUser = null">取消</button>
            </div>
          </div>
        </div>
      </div>
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
