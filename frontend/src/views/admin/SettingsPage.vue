<script setup lang="ts">
// 网站设置：站点 Logo（图片/文字）、名称、描述、页脚
// 业务逻辑保持不变；UI 重构为 Naive UI + Tailwind
import { ref, computed, onMounted } from 'vue'
import { NButton, NInput, NAlert, NSwitch, NSkeleton } from 'naive-ui'
import api from '@/api'
import { useSiteSettingsStore } from '@/stores/siteSettings'

interface SiteSettingsData {
  siteName: string
  siteDescription: string
  footerText: string
  siteLogoType: string
  siteLogoUrl: string
  siteIcp: string
  sitePolice: string
  [key: string]: string
}

const siteStore = useSiteSettingsStore()
const settings = ref<SiteSettingsData>({ siteName: '', siteDescription: '', footerText: '', siteLogoType: 'text', siteLogoUrl: '', siteIcp: '', sitePolice: '' })
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
      siteIcp: data.siteIcp || '',
      sitePolice: data.sitePolice || '',
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
    siteStore.siteIcp = settings.value.siteIcp
    siteStore.sitePolice = settings.value.sitePolice
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
  <div>
    <div v-if="loading" class="space-y-4">
      <NSkeleton text width="20%" size="large" />
      <NSkeleton height="180px" :sharp="false" />
      <NSkeleton height="220px" :sharp="false" />
    </div>

    <template v-else>
      <header class="mb-6">
        <h2 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">网站设置</h2>
        <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">站点名称、Logo 及页脚信息</p>
      </header>

      <form class="space-y-5" @submit.prevent="handleSubmit">
        <!-- 站点 Logo -->
        <section class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <h3 class="text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">站点 Logo</h3>

          <div class="mt-4 flex items-center justify-between gap-4 rounded-xl bg-zinc-50 px-4 py-3 dark:bg-zinc-800/60">
            <div>
              <p class="text-sm font-medium text-zinc-800 dark:text-zinc-200">使用图片 Logo</p>
              <p class="mt-0.5 text-xs text-zinc-400 dark:text-zinc-500">关闭则使用文字 Logo</p>
            </div>
            <NSwitch
              :value="isImageLogo"
              size="small"
              @update:value="(v: boolean) => updateSetting('siteLogoType', v ? 'image' : 'text')"
            />
          </div>

          <!-- 图片 Logo：上传与预览 -->
          <div v-if="isImageLogo" class="mt-4">
            <input ref="fileInputRef" type="file" accept="image/*" class="hidden" @change="handleLogoUpload" />
            <div v-if="settings.siteLogoUrl" class="flex flex-wrap items-center gap-4">
              <span class="inline-flex rounded-xl border border-zinc-200 bg-white p-3 dark:border-zinc-700 dark:bg-zinc-800">
                <img :src="settings.siteLogoUrl" alt="Logo" class="h-10 w-auto" />
              </span>
              <div class="flex gap-2">
                <NButton size="small" secondary :loading="uploading" @click="fileInputRef?.click()">更换图片</NButton>
                <NButton size="small" quaternary @click="updateSetting('siteLogoUrl', '')">移除</NButton>
              </div>
            </div>
            <div v-else class="flex flex-col items-start gap-2 rounded-xl border border-dashed border-zinc-300 p-5 dark:border-zinc-700">
              <p class="text-xs text-zinc-400 dark:text-zinc-500">支持 JPG、PNG、GIF、WebP、SVG，不超过 2MB</p>
              <NButton size="small" secondary :loading="uploading" @click="fileInputRef?.click()">选择图片</NButton>
            </div>
          </div>

          <!-- 文字 Logo 预览 -->
          <div v-else class="mt-4 flex items-center gap-2.5 rounded-xl border border-zinc-200 p-4 dark:border-zinc-700">
            <span class="grid size-8 place-items-center rounded-lg bg-emerald-600 text-sm font-bold text-white dark:bg-emerald-400 dark:text-zinc-950">
              {{ (settings.siteName || 'X').charAt(0).toUpperCase() }}
            </span>
            <span class="text-[15px] font-semibold tracking-tight text-zinc-900 dark:text-zinc-100">{{ settings.siteName || '站点名称' }}</span>
          </div>
        </section>

        <!-- 基本信息 -->
        <section class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <h3 class="mb-4 text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">基本信息</h3>
          <div class="max-w-lg space-y-4">
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">网站名称</label>
              <NInput v-model:value="settings.siteName" placeholder="站点名称" />
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">网站描述</label>
              <NInput v-model:value="settings.siteDescription" placeholder="首页副标题 / SEO 描述" />
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">页脚文字</label>
              <NInput v-model:value="settings.footerText" placeholder="页脚展示文本" />
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">网站备案号</label>
              <NInput v-model:value="settings.siteIcp" placeholder="如：渝ICP备2024XXXXXX号，留空不显示" />
              <p class="mt-1 text-xs text-zinc-400 dark:text-zinc-500">留空则不在前台页脚显示，前台将链接到 beian.miit.gov.cn</p>
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">公安备案号</label>
              <NInput v-model:value="settings.sitePolice" placeholder="如：渝公网安备500XXXXXXXX号，留空不显示" />
              <p class="mt-1 text-xs text-zinc-400 dark:text-zinc-500">留空则不在前台页脚显示，前台将链接到 beian.mps.gov.cn</p>
            </div>
          </div>
        </section>

        <NAlert v-if="message" :type="message.includes('失败') ? 'error' : 'success'" closable @close="message = ''">{{ message }}</NAlert>

        <NButton attr-type="submit" type="primary" :loading="saving">保存设置</NButton>
      </form>
    </template>
  </div>
</template>
