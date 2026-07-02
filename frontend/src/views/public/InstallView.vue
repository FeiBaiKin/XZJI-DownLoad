<script setup lang="ts">
import { ref } from 'vue'
import api from '@/api'

const step = ref<'form'|'loading'|'error'>('form')
const errorMsg = ref('')
const form = ref({ siteName: '', siteDescription: '', siteURL: '', footerText: '', adminUsername: '', adminPassword: '', adminEmail: '', dbHost: '127.0.0.1', dbPort: 3306, dbUser: 'admin', dbPassword: '123456', dbName: 'xzji' })

async function submit() {
  step.value = 'loading'
  try {
    const s = await api.get('/install/status'); if (s.data.installed) { errorMsg.value = '系统已安装'; step.value = 'error'; return }
    await api.post('/install', form.value)
    step.value = 'error'; errorMsg.value = '安装成功！请重启服务后访问首页。'
  } catch (e: any) { errorMsg.value = e?.response?.data?.error || '安装失败'; step.value = 'error' }
}
</script>

<template>
  <div class="install-page">
    <div class="install-container">
      <div class="install-header"><h1 class="install-title">系统安装</h1><p class="install-subtitle">配置站点信息与数据库连接</p></div>
      <form @submit.prevent="submit">
        <div class="install-card"><h2 class="install-card-title">网站信息</h2>
          <div class="install-field"><label class="install-label">网站名称</label><input class="install-input" v-model="form.siteName" placeholder="如：关键词搜索平台" /></div>
          <div class="install-field"><label class="install-label">站点地址 <span style="color: var(--danger);">*</span></label><input class="install-input" v-model="form.siteURL" placeholder="https://your-domain.com" /><p style="font-size:0.7rem;color:var(--text-tertiary);margin-top:0.3rem;">用于 CORS 安全策略，必须以 http:// 或 https:// 开头</p></div>
          <div class="install-field"><label class="install-label">网站描述</label><input class="install-input" v-model="form.siteDescription" placeholder="输入关键词，发现后端配置的响应内容" /></div>
          <div class="install-field"><label class="install-label">页脚文本</label><input class="install-input" v-model="form.footerText" placeholder="Keyword Search App" /></div>
        </div>
        <div class="install-card"><h2 class="install-card-title">管理员账号</h2>
          <div class="install-field"><label class="install-label">用户名</label><input class="install-input" v-model="form.adminUsername" placeholder="至少 3 个字符" /></div>
          <div class="install-field"><label class="install-label">密码</label><input class="install-input" type="password" v-model="form.adminPassword" placeholder="至少 6 个字符" /></div>
          <div class="install-field"><label class="install-label">邮箱</label><input class="install-input" v-model="form.adminEmail" placeholder="选填" /></div>
        </div>
        <div class="install-card"><h2 class="install-card-title">数据库配置</h2>
          <div class="install-row"><div class="install-field install-field-half"><label class="install-label">主机</label><input class="install-input" v-model="form.dbHost" /></div><div class="install-field install-field-half"><label class="install-label">端口</label><input class="install-input" type="number" v-model="form.dbPort" /></div></div>
          <div class="install-field"><label class="install-label">用户名</label><input class="install-input" v-model="form.dbUser" /></div>
          <div class="install-field"><label class="install-label">密码</label><input class="install-input" type="password" v-model="form.dbPassword" /></div>
          <div class="install-field"><label class="install-label">数据库名</label><input class="install-input" v-model="form.dbName" /></div>
        </div>
        <div v-if="errorMsg" class="form-error" style="margin-bottom:1rem;">{{ errorMsg }}</div>
        <button type="submit" class="install-submit-btn" :disabled="step === 'loading'">{{ step === 'loading' ? '正在安装...' : '开始安装' }}</button>
      </form>
    </div>
  </div>
</template>

<style scoped>
.install-page { min-height: 100vh; padding: 3rem 1.5rem; display: flex; justify-content: center; }
.install-container { max-width: 560px; width: 100%; }
.install-header { margin-bottom: 2rem; }
.install-title { font-size: 2rem; font-weight: 700; letter-spacing: -0.04em; }
.install-subtitle { font-size: 0.9rem; color: var(--text-secondary); margin-top: 0.5rem; }
.install-card { border: 1px solid var(--border-default); padding: 1.5rem; margin-bottom: 1rem; background: var(--bg-secondary); }
.install-card-title { font-size: 0.9rem; font-weight: 600; margin-bottom: 1rem; }
.install-field { margin-bottom: 0.875rem; }
.install-label { display: block; font-family: 'JetBrains Mono', monospace; font-size: 0.6875rem; color: var(--text-tertiary); text-transform: uppercase; letter-spacing: 0.08em; margin-bottom: 0.4rem; }
.install-input { width: 100%; padding: 0.75rem; font-family: 'JetBrains Mono', monospace; font-size: 0.875rem; background: var(--bg-primary); border: 1px solid var(--border-default); color: var(--text-primary); outline: none; }
.install-input:focus { border-color: var(--text-secondary); }
.install-row { display: flex; gap: 0.75rem; }
.install-field-half { flex: 1; }
.install-submit-btn { width: 100%; padding: 0.875rem; font-size: 0.875rem; font-family: 'Space Grotesk', sans-serif; font-weight: 600; background-color: var(--white); color: var(--black); border: 1px solid var(--white); cursor: pointer; transition: all 0.15s ease; }
.install-submit-btn:hover:not(:disabled) { background-color: var(--text-primary); border-color: var(--text-primary); }
.install-submit-btn:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
