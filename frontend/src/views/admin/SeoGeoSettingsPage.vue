<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { NAlert, NButton, NInput, NSkeleton, NSwitch } from 'naive-ui'
import api from '@/api'

interface SeoGeoSettings {
  seoEnabled: string
  seoSiteUrl: string
  seoTitleTemplate: string
  seoDefaultKeywords: string
  seoDefaultImage: string
  seoRobotsIndex: string
  seoGoogleVerification: string
  seoBingVerification: string
  geoEnabled: string
  geoOrganizationName: string
  geoOrganizationDescription: string
  geoOrganizationLogo: string
  geoAllowAICrawlers: string
  geoLlmsDescription: string
  geoContactEmail: string
  geoSocialProfiles: string
  [key: string]: string
}

const defaults: SeoGeoSettings = {
  seoEnabled: 'true',
  seoSiteUrl: '',
  seoTitleTemplate: '%page% | %site%',
  seoDefaultKeywords: '',
  seoDefaultImage: '',
  seoRobotsIndex: 'true',
  seoGoogleVerification: '',
  seoBingVerification: '',
  geoEnabled: 'true',
  geoOrganizationName: '',
  geoOrganizationDescription: '',
  geoOrganizationLogo: '',
  geoAllowAICrawlers: 'true',
  geoLlmsDescription: '',
  geoContactEmail: '',
  geoSocialProfiles: '',
}

const settings = ref<SeoGeoSettings>({ ...defaults })
const loading = ref(true)
const saving = ref(false)
const message = ref('')
const error = ref('')

onMounted(() => { fetchSettings() })

async function fetchSettings() {
  try {
    const { data } = await api.get('/admin/settings')
    settings.value = { ...defaults, ...data }
  } catch (err) {
    console.error('获取 SEO / GEO 设置失败:', err)
    error.value = '加载失败，请刷新重试'
  } finally {
    loading.value = false
  }
}

function updateSwitch(key: keyof SeoGeoSettings, value: boolean) {
  settings.value[key] = String(value)
}

async function handleSave() {
  saving.value = true
  message.value = ''
  error.value = ''
  try {
    await api.put('/admin/settings', settings.value)
    message.value = 'SEO / GEO 设置已保存'
  } catch (err: unknown) {
    const errorObj = err as { response?: { data?: { error?: string } } }
    error.value = errorObj.response?.data?.error || '保存失败'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div>
    <div v-if="loading" class="space-y-4">
      <NSkeleton text width="20%" size="large" />
      <NSkeleton height="220px" :sharp="false" />
      <NSkeleton height="300px" :sharp="false" />
      <NSkeleton height="220px" :sharp="false" />
    </div>

    <template v-else>
      <header class="mb-6">
        <h1 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">SEO / GEO 设置</h1>
        <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">搜索引擎收录、社交分享及生成式 AI 内容理解</p>
      </header>

      <form class="space-y-5" @submit.prevent="handleSave">
        <section class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <h3 class="text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">基础开关</h3>
          <div class="mt-4 space-y-3">
            <div class="flex flex-col items-start gap-3 rounded-xl bg-zinc-50 px-4 py-3 sm:flex-row sm:items-center sm:justify-between dark:bg-zinc-800/60">
              <div>
                <p class="text-sm font-medium text-zinc-800 dark:text-zinc-200">启用 SEO</p>
                <p class="mt-0.5 text-xs text-zinc-500 dark:text-zinc-400">生成页面 Meta、Canonical、Open Graph 与站点地图</p>
              </div>
              <NSwitch :value="settings.seoEnabled === 'true'" size="small" @update:value="(v: boolean) => updateSwitch('seoEnabled', v)" />
            </div>
            <div class="flex flex-col items-start gap-3 rounded-xl bg-zinc-50 px-4 py-3 sm:flex-row sm:items-center sm:justify-between dark:bg-zinc-800/60">
              <div>
                <p class="text-sm font-medium text-zinc-800 dark:text-zinc-200">允许搜索引擎收录</p>
                <p class="mt-0.5 text-xs text-zinc-500 dark:text-zinc-400">关闭后输出 noindex，并停止提供站点地图</p>
              </div>
              <NSwitch :value="settings.seoRobotsIndex === 'true'" size="small" @update:value="(v: boolean) => updateSwitch('seoRobotsIndex', v)" />
            </div>
            <div class="flex flex-col items-start gap-3 rounded-xl bg-zinc-50 px-4 py-3 sm:flex-row sm:items-center sm:justify-between dark:bg-zinc-800/60">
              <div>
                <p class="text-sm font-medium text-zinc-800 dark:text-zinc-200">启用 GEO</p>
                <p class="mt-0.5 text-xs text-zinc-500 dark:text-zinc-400">生成实体结构化数据与 llms.txt，帮助生成式引擎理解本站</p>
              </div>
              <NSwitch :value="settings.geoEnabled === 'true'" size="small" @update:value="(v: boolean) => updateSwitch('geoEnabled', v)" />
            </div>
            <div class="flex flex-col items-start gap-3 rounded-xl bg-zinc-50 px-4 py-3 sm:flex-row sm:items-center sm:justify-between dark:bg-zinc-800/60">
              <div>
                <p class="text-sm font-medium text-zinc-800 dark:text-zinc-200">允许 AI 爬虫访问</p>
                <p class="mt-0.5 text-xs text-zinc-500 dark:text-zinc-400">控制 GPTBot、ClaudeBot、PerplexityBot 等爬虫规则</p>
              </div>
              <NSwitch :value="settings.geoAllowAICrawlers === 'true'" size="small" @update:value="(v: boolean) => updateSwitch('geoAllowAICrawlers', v)" />
            </div>
          </div>
        </section>

        <section class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <h3 class="mb-4 text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">搜索展示</h3>
          <div class="grid gap-4 sm:grid-cols-2">
            <div class="sm:col-span-2">
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">站点规范地址</label>
              <NInput v-model:value="settings.seoSiteUrl" placeholder="https://example.com" />
              <p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">用于 Canonical、站点地图和结构化数据，请填写完整 HTTPS 地址</p>
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">标题模板</label>
              <NInput v-model:value="settings.seoTitleTemplate" placeholder="%page% | %site%" />
              <p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">支持 %page% 页面名称与 %site% 站点名称</p>
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">默认关键词</label>
              <NInput v-model:value="settings.seoDefaultKeywords" placeholder="关键词之间使用英文逗号分隔" />
            </div>
            <div class="sm:col-span-2">
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">默认分享图片</label>
              <NInput v-model:value="settings.seoDefaultImage" placeholder="https://example.com/cover.jpg" />
              <p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">用于 Open Graph 分享预览，推荐 1200 × 630 像素</p>
            </div>
          </div>
        </section>

        <section class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <h3 class="mb-4 text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">品牌实体信息</h3>
          <div class="grid gap-4 sm:grid-cols-2">
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">机构 / 品牌名称</label>
              <NInput v-model:value="settings.geoOrganizationName" placeholder="用于知识图谱实体识别" />
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">公开联系邮箱</label>
              <NInput v-model:value="settings.geoContactEmail" placeholder="contact@example.com" :input-props="{ type: 'email' }" />
            </div>
            <div class="sm:col-span-2">
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">机构 Logo 地址</label>
              <NInput v-model:value="settings.geoOrganizationLogo" placeholder="https://example.com/logo.png" />
            </div>
            <div class="sm:col-span-2">
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">机构简介</label>
              <NInput v-model:value="settings.geoOrganizationDescription" type="textarea" :rows="3" placeholder="用明确、可验证的语言描述品牌主体与服务" />
            </div>
            <div class="sm:col-span-2">
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">AI 内容摘要</label>
              <NInput v-model:value="settings.geoLlmsDescription" type="textarea" :rows="3" placeholder="描述本站的内容范围、核心价值与适用人群" />
              <p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">将作为 llms.txt 的站点摘要，建议保持简洁、客观</p>
            </div>
            <div class="sm:col-span-2">
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">社交 / 权威主页</label>
              <NInput v-model:value="settings.geoSocialProfiles" type="textarea" :rows="3" placeholder="每行填写一个完整 URL" />
              <p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">用于 JSON-LD sameAs 关联品牌的官方账号与权威资料页</p>
            </div>
          </div>
        </section>

        <section class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <h3 class="mb-4 text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">站长平台验证</h3>
          <div class="grid gap-4 sm:grid-cols-2">
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">Google 验证代码</label>
              <NInput v-model:value="settings.seoGoogleVerification" placeholder="仅填写 content 值" />
            </div>
            <div>
              <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">Bing 验证代码</label>
              <NInput v-model:value="settings.seoBingVerification" placeholder="仅填写 content 值" />
            </div>
          </div>
        </section>

        <NAlert v-if="message" type="success" closable @close="message = ''">{{ message }}</NAlert>
        <NAlert v-if="error" type="error" closable @close="error = ''">{{ error }}</NAlert>

        <NButton attr-type="submit" type="primary" :loading="saving">保存配置</NButton>
      </form>
    </template>
  </div>
</template>
