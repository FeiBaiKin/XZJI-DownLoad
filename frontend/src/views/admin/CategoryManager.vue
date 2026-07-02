<script setup lang="ts">
// 分类管理：1:1 移植自原项目 Node-XZJI/frontend/src/pages/admin/CategoryManager.jsx
// 样式来自全局 styles/admin-layout.css + admin-orders.css
import { ref, onMounted } from 'vue'
import { contentApi, type Category } from '@/api/content'
import ConfirmDialog from '@/components/ConfirmDialog.vue'

const categories = ref<Category[]>([])
const loading = ref(true)
const saving = ref(false)
const message = ref('')
const editing = ref<Category | null>(null)
const form = ref({ name: '', description: '', sort_order: 0 })
const confirmCat = ref<Category | null>(null)

onMounted(() => { fetchCategories() })

async function fetchCategories() {
  try {
    categories.value = await contentApi.categories.list()
  } catch { /* 忽略 */ }
  finally { loading.value = false }
}

function resetForm() {
  editing.value = null
  form.value = { name: '', description: '', sort_order: 0 }
}

function handleEdit(cat: Category) {
  editing.value = cat
  form.value = { name: cat.name, description: cat.description || '', sort_order: cat.sort_order || 0 }
}

async function handleSave() {
  if (!form.value.name.trim()) { message.value = '分类名称不能为空'; return }
  saving.value = true
  message.value = ''
  try {
    if (editing.value) {
      await contentApi.categories.update(editing.value.id, form.value)
      message.value = '分类已更新'
    } else {
      await contentApi.categories.create(form.value)
      message.value = '分类已创建'
    }
    resetForm()
    fetchCategories()
  } catch (err: unknown) {
    const errorObj = err as { response?: { data?: { error?: string } } }
    message.value = errorObj?.response?.data?.error || '保存失败'
  } finally {
    saving.value = false
  }
}

function handleDelete(cat: Category) {
  confirmCat.value = cat
}
async function doDelete() {
  const cat = confirmCat.value
  confirmCat.value = null
  if (!cat) return
  try {
    await contentApi.categories.delete(cat.id)
    fetchCategories()
  } catch { /* 忽略 */ }
}
</script>

<template>
  <div class="admin-content">
    <div v-if="loading" class="loading-state">加载中...</div>
    <template v-else>
      <div class="content-header">
        <h2 class="content-title">分类管理</h2>
        <span class="content-sub">为关键词创建分类，组织内容结构</span>
      </div>

      <div class="form-card" style="margin-bottom: 1.5rem;">
        <div class="form-card-title">{{ editing ? '编辑分类' : '新建分类' }}</div>
        <form @submit.prevent="handleSave">
          <div style="display: grid; grid-template-columns: 1fr 80px; gap: 0.75rem; margin-bottom: 0.75rem;">
            <div class="form-group" style="margin: 0;">
              <label class="form-label">分类名称</label>
              <input type="text" class="form-input" v-model="form.name" placeholder="例如：技术文档" />
            </div>
            <div class="form-group" style="margin: 0;">
              <label class="form-label">排序</label>
              <input type="number" class="form-input" v-model.number="form.sort_order" />
            </div>
          </div>
          <div class="form-group">
            <label class="form-label">描述（可选）</label>
            <input type="text" class="form-input" v-model="form.description" placeholder="简短描述" />
          </div>
          <div v-if="message" class="form-message" style="margin-bottom: 0.75rem;">{{ message }}</div>
          <div style="display: flex; gap: 0.5rem;">
            <button type="submit" class="action-button" :disabled="saving">{{ saving ? '保存中...' : (editing ? '更新' : '创建') }}</button>
            <button v-if="editing" type="button" class="action-button secondary" @click="resetForm">取消</button>
          </div>
        </form>
      </div>

      <div class="order-table-wrap">
        <table class="order-table">
          <thead>
            <tr>
              <th>名称</th>
              <th>描述</th>
              <th>排序</th>
              <th style="width: 120px;">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="categories.length === 0"><td colspan="4" class="order-empty">暂无分类</td></tr>
            <tr v-for="c in categories" :key="c.id">
              <td style="font-weight: 500;">{{ c.name }}</td>
              <td style="color: var(--text-secondary); font-size: 0.8rem;">{{ c.description || '-' }}</td>
              <td class="mono-td">{{ c.sort_order }}</td>
              <td style="display: flex; gap: 0.75rem;">
                <button class="action-link" @click="handleEdit(c)">编辑</button>
                <button class="action-link danger" @click="handleDelete(c)">删除</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>

    <ConfirmDialog
      :open="confirmCat !== null"
      title="删除分类"
      :message="confirmCat ? `确定删除分类「${confirmCat.name}」吗？关联的关键词不会删除，仅解除关联。` : ''"
      confirm-label="删除"
      danger
      @confirm="doDelete"
      @cancel="confirmCat = null"
    />
  </div>
</template>
