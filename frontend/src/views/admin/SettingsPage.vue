<script setup lang="ts">
// 网站设置：1:1 移植自原项目 Node-XZJI/frontend/src/pages/admin/SiteSettings.jsx
// 样式来自全局 styles/admin-layout.css + admin-email.css + home.css
import { ref, computed, onMounted } from 'vue'
import api from '@/api'
import { useSiteSettingsStore } from '@/stores/siteSettings'

interface SiteSettingsData {
  siteName: string
  siteDescription: string
  footerText: string
  siteLogoType: string
  siteLogoUrl: string
  [key: string]: string
}

const siteStore = useSiteSettingsStore()
const settings = ref<SiteSettingsData>({ siteName: '', siteDescription: '', footerText: '', siteLogoType: 'text', siteLogoUrl: '' })
const loading = ref(true)
const saving = ref(false)
const message = ref('')
const uploading = ref(false)
const fileInputRef = ref<HTMLInputElement | null>(null)

onMounted(() => { fetchSettings() })

async function fetchSettings() {
  try {
    const res = await api.get('/admin/settings')
    const data = res.data
    settings.value = {
      siteName: data.siteName || '',
      siteDescription: data.siteDescription || '',
      footerText: data.footerText || '',
      siteLogoType: data.siteLogoType || 'text',
      siteLogoUrl: data.siteLogoUrl || '',
    }
  } catch (err) { console.error('获取设置失败:', err) }
  finally { loading.value = false }
}

function updateSetting(key: string, value: string) { settings.value[key] = value }

async function handleSubmit() {
  saving.value = true; message.value = ''
  try {
    await api.put('/admin/settings', settings.value)
    // 同步到全局 store，让 document.title 即时更新
    siteStore.siteName = settings.value.siteName
    siteStore.siteDescription = settings.value.siteDescription
    siteStore.footerText = settings.value.footerText
    siteStore.siteLogoType = settings.value.siteLogoType
    siteStore.siteLogoUrl = settings.value.siteLogoUrl
    siteStore.loaded = true
    message.value = '设置已保存'
  } catch (err) { console.error('保存设置失败:', err); message.value = '保存失败' }
  finally { saving.value = false }
}

async function handleLogoUpload(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  uploading.value = true; message.value = ''
  try {
    const fd = new FormData()
    fd.append('logo', file)
    const res = await api.post('/admin/upload-logo', fd)
    updateSetting('siteLogoUrl', res.data.url)
    message.value = 'Logo 上传成功'
  } catch (err: unknown) {
    const errorObj = err as { response?: { data?: { error?: string } } }
    message.value = errorObj?.response?.data?.error || '上传失败'
  } finally {
    uploading.value = false
    if (fileInputRef.value) fileInputRef.value.value = ''
  }
}

const isImageLogo = computed(() => settings.value.siteLogoType === 'image')
</script>

<template>
  <div class="admin-content">
    <div v-if="loading" class="loading-state">加载中...</div>
    <template v-else>
      <div class="content-header">
        <h2 class="content-title">网站设置</h2>
        <span class="content-sub">站点名称、Logo 及页脚信息</span>
      </div>
      <form @submit.prevent="handleSubmit">
        <div class="form-card">
          <div class="form-card-title">站点 Logo</div>
          <div class="email-toggles" style="margin-bottom: 1rem;">
            <div class="email-toggle-row">
              <div class="email-toggle-info">
                <span class="email-toggle-label">使用图片 Logo</span>
                <span class="email-toggle-desc">关闭则使用文字 Logo</span>
              </div>
              <button type="button" class="email-toggle-switch" :class="{ on: isImageLogo }" :aria-checked="isImageLogo" role="switch" @click="updateSetting('siteLogoType', isImageLogo ? 'text' : 'image')">
                <span class="email-toggle-knob" />
              </button>
            </div>
          </div>
          <div v-if="isImageLogo" class="logo-upload-area">
            <div v-if="settings.siteLogoUrl" class="logo-upload-preview">
              <img :src="settings.siteLogoUrl" alt="Logo" class="logo-upload-img" />
              <div class="logo-upload-actions">
                <input ref="fileInputRef" type="file" accept="image/*" @change="handleLogoUpload" style="display: none;" id="logo-file-input" />
                <button type="button" class="action-button secondary" :disabled="uploading" @click="fileInputRef?.click()">{{ uploading ? '上传中...' : '更换图片' }}</button>
                <button type="button" class="action-button secondary" @click="updateSetting('siteLogoUrl', '')">移除</button>
              </div>
            </div>
            <div v-else class="logo-upload-empty">
              <input ref="fileInputRef" type="file" accept="image/*" @change="handleLogoUpload" style="display: none;" id="logo-file-input" />
              <span class="logo-upload-hint">支持 JPG、PNG、GIF、WebP、SVG，不超过 2MB</span>
              <button type="button" class="action-button secondary" :disabled="uploading" @click="fileInputRef?.click()">{{ uploading ? '上传中...' : '选择图片' }}</button>
            </div>
          </div>
          <div v-else class="logo-text-preview">
            <span class="logo-text-preview-mark">{{ (settings.siteName || 'K').charAt(0).toUpperCase() }}</span>
            <span class="logo-text-preview-name">{{ settings.siteName || 'Keyword' }}</span>
          </div>
        </div>

        <div class="form-card">
          <div class="form-card-title">基本信息</div>
          <div class="form-group">
            <label class="form-label">网站名称</label>
            <input type="text" class="form-input" v-model="settings.siteName" />
          </div>
          <div class="form-group">
            <label class="form-label">网站描述</label>
            <input type="text" class="form-input" v-model="settings.siteDescription" />
          </div>
          <div class="form-group">
            <label class="form-label">页脚文字</label>
            <input type="text" class="form-input" v-model="settings.footerText" />
          </div>
        </div>

        <div v-if="message" class="form-message">{{ message }}</div>
        <div class="form-actions">
          <button type="submit" class="action-button" :disabled="saving">{{ saving ? '保存中...' : '保存设置' }}</button>
        </div>
      </form>
    </template>
  </div>
</template>
