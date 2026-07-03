<script setup lang="ts">
// 授权管理：接入授权升级中心（激活 / 心跳 / 状态展示 / 在线升级）
// 样式：全局 styles/license.css + admin 通用 form-card / pay-field 体系
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'
import { adminApi, type LicenseSnapshot } from '@/api/admin'
import { extractErrorMessage } from '@/types/error'
import { useLicenseStore } from '@/stores/license'
import ConfirmDialog from '@/components/ConfirmDialog.vue'

const licenseStore = useLicenseStore()

const snap = ref<LicenseSnapshot | null>(null)
const loading = ref(true)
const message = ref('')
const error = ref('')

const codeInput = ref('')
const activating = ref(false)
const beating = ref(false)

const upgradeConfirmOpen = ref(false)
const upgrading = ref(false)
const restarting = ref(false)
const upgradeNote = ref('')

onMounted(() => { fetchStatus() })

async function fetchStatus() {
  try {
    snap.value = await adminApi.license.get()
  } catch (err) {
    error.value = extractErrorMessage(err, '获取授权状态失败')
  } finally {
    loading.value = false
  }
}

// 同步刷新全局停服遮罩状态（激活/心跳改变授权有效性后，前台无需整页刷新）
function refreshGlobalLicense() {
  licenseStore.fetch()
}

// 状态徽章：文案 + 颜色档位
const badge = computed(() => {
  const s = snap.value
  if (!s) return { text: '未知', cls: 'warn' }
  if (s.valid) {
    return s.lastError
      ? { text: '已授权 · 离线宽限中', cls: 'warn' }
      : { text: '授权有效', cls: 'ok' }
  }
  const map: Record<string, string> = {
    unactivated: '未激活',
    expired: '已过期',
    revoked: '已吊销',
    domain_mismatch: '域名不匹配',
    grace_exceeded: '离线超期，已停服',
  }
  return { text: map[s.reason || ''] || `无效（${s.reason}）`, cls: s.reason === 'unactivated' ? 'warn' : 'bad' }
})

function fmtTime(unix?: number): string {
  if (!unix) return '-'
  const d = new Date(unix * 1000)
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

const expiresText = computed(() => {
  const s = snap.value
  if (!s || !s.activated) return '-'
  return s.expiresAt === 0 ? '永久有效' : fmtTime(s.expiresAt)
})

function fmtSize(bytes: number): string {
  if (!bytes) return '-'
  return (bytes / 1024 / 1024).toFixed(2) + ' MB'
}

async function handleActivate() {
  if (!codeInput.value.trim()) { error.value = '请输入授权码'; return }
  activating.value = true; message.value = ''; error.value = ''
  try {
    const res = await adminApi.license.activate(codeInput.value.trim())
    snap.value = res.snapshot
    if (res.status === 'active') {
      message.value = res.message
      codeInput.value = ''
    } else {
      error.value = res.message
    }
    refreshGlobalLicense()
  } catch (err) {
    error.value = extractErrorMessage(err, '激活失败')
  } finally {
    activating.value = false
  }
}

async function handleHeartbeat() {
  beating.value = true; message.value = ''; error.value = ''
  try {
    const res = await adminApi.license.heartbeat()
    snap.value = res.snapshot
    message.value = res.message
    refreshGlobalLicense()
  } catch (err: unknown) {
    const data = (err as { response?: { data?: { snapshot?: LicenseSnapshot } } })?.response?.data
    if (data?.snapshot) snap.value = data.snapshot
    error.value = extractErrorMessage(err, '心跳失败')
  } finally {
    beating.value = false
  }
}

async function handleUpgrade() {
  upgradeConfirmOpen.value = false
  upgrading.value = true; message.value = ''; error.value = ''; upgradeNote.value = ''
  try {
    const res = await adminApi.license.upgrade()
    if (res.restarting) {
      restarting.value = true
      pollRestart()
    } else {
      upgradeNote.value = res.message + (res.stagingPath ? `（暂存位置：${res.stagingPath}）` : '')
      await fetchStatus()
    }
  } catch (err) {
    error.value = extractErrorMessage(err, '升级失败')
  } finally {
    upgrading.value = false
  }
}

// 升级重启后轮询健康检查，恢复即整页刷新（加载新版前端资源）
function pollRestart() {
  let attempts = 0
  const timer = setInterval(async () => {
    attempts++
    try {
      await axios.get('/api/health', { timeout: 2000 })
      clearInterval(timer)
      window.location.reload()
    } catch {
      if (attempts >= 45) {
        clearInterval(timer)
        restarting.value = false
        error.value = '服务重启超时，请手动检查服务状态后刷新页面'
      }
    }
  }, 2000)
}
</script>

<template>
  <div class="admin-content">
    <div v-if="loading">
      <div class="content-header"><h2 class="content-title">授权管理</h2></div>
      <div class="loading-state">加载中...</div>
    </div>
    <template v-else>
      <div class="content-header">
        <h2 class="content-title">授权管理</h2>
        <span class="content-sub">软件授权激活、心跳与在线升级（授权中心：www.xzji.top）</span>
      </div>

      <!-- 授权状态 -->
      <div class="form-card">
        <div class="form-card-title">授权状态</div>
        <div class="license-status-head">
          <span class="license-badge" :class="badge.cls">{{ badge.text }}</span>
          <div class="license-actions">
            <button type="button" class="action-button secondary" :disabled="beating || !snap?.activated" @click="handleHeartbeat">
              {{ beating ? '正在联系授权中心...' : '立即心跳' }}
            </button>
            <button type="button" class="action-button secondary" @click="fetchStatus">刷新</button>
          </div>
        </div>
        <div class="license-info-grid" v-if="snap">
          <div class="license-info-item"><span class="license-info-label">授权码</span><span class="license-info-value">{{ snap.code || '未激活' }}</span></div>
          <div class="license-info-item"><span class="license-info-label">绑定域名</span><span class="license-info-value">{{ snap.domain }}</span></div>
          <div class="license-info-item"><span class="license-info-label">到期时间</span><span class="license-info-value">{{ expiresText }}</span></div>
          <div class="license-info-item"><span class="license-info-label">上次验证</span><span class="license-info-value">{{ fmtTime(snap.lastVerifiedAt) }}</span></div>
          <div class="license-info-item" v-if="snap.activated"><span class="license-info-label">心跳间隔</span><span class="license-info-value">{{ snap.heartbeatIntervalHours || '-' }} 小时</span></div>
          <div class="license-info-item" v-if="snap.activated"><span class="license-info-label">离线宽限期至</span><span class="license-info-value">{{ fmtTime(snap.graceDeadline) }}</span></div>
        </div>
        <div v-if="snap?.lastError" class="license-warn-note">最近一次联系授权中心失败：{{ snap.lastError }}<template v-if="snap.valid">（宽限期内凭本地缓存正常运行）</template></div>
        <div v-if="snap && !snap.valid" class="license-error-note">前台已停止对外服务，访客将看到「授权已过期」提示；激活有效授权后自动恢复。</div>
      </div>

      <!-- 激活 -->
      <div class="form-card">
        <div class="form-card-title">{{ snap?.activated ? '重新激活 / 更换授权码' : '激活站点' }}</div>
        <div class="pay-fields">
          <div class="pay-field" style="grid-column: 1 / -1;">
            <span class="pay-field-label">授权码</span>
            <input
              type="text"
              class="pay-field-input mono"
              placeholder="XXXXX-XXXXX-XXXXX-XXXXX"
              v-model="codeInput"
              @keyup.enter="handleActivate"
            />
            <span class="pay-field-hint">授权码由服务商提供；首次激活将绑定当前站点域名 {{ snap?.domain }}，同域名重装可重复激活</span>
          </div>
        </div>
        <div class="form-actions" style="margin-top: 1rem;">
          <button type="button" class="action-button" :disabled="activating" @click="handleActivate">
            {{ activating ? '激活中...' : '激活' }}
          </button>
        </div>
      </div>

      <!-- 版本与升级 -->
      <div class="form-card">
        <div class="form-card-title">版本与升级</div>
        <div class="license-info-grid">
          <div class="license-info-item"><span class="license-info-label">当前版本</span><span class="license-info-value">v{{ snap?.currentVersion }}</span></div>
          <div class="license-info-item"><span class="license-info-label">最新版本</span><span class="license-info-value">{{ snap?.latestVersion ? 'v' + snap.latestVersion : '暂无发布版本' }}</span></div>
        </div>

        <template v-if="snap?.update">
          <div class="license-upgrade-row">
            <span class="license-info-value">发现新版本 v{{ snap.update.version }}（{{ fmtSize(snap.update.file_size) }}）</span>
            <span v-if="snap.update.forced" class="license-forced-tag">强制更新</span>
            <button type="button" class="action-button" :disabled="upgrading || restarting || !snap.valid" @click="upgradeConfirmOpen = true">
              {{ upgrading ? '正在下载升级...' : '下载并升级' }}
            </button>
          </div>
          <div v-if="snap.update.changelog" class="license-changelog">{{ snap.update.changelog }}</div>
          <div v-if="!snap.valid" class="license-warn-note">授权无效时无法升级，请先激活有效授权</div>
        </template>
        <div v-else class="license-info-label" style="margin-top: 0.75rem;">当前已是最新版本</div>

        <div v-if="restarting" class="license-warn-note">升级完成，服务正在重启，页面将自动刷新...</div>
        <div v-if="upgradeNote" class="license-warn-note">{{ upgradeNote }}</div>
      </div>

      <div v-if="message" class="form-success" style="margin-bottom: 1rem;">{{ message }}</div>
      <div v-if="error" class="form-error" style="margin-bottom: 1rem;">{{ error }}</div>

      <ConfirmDialog
        :open="upgradeConfirmOpen"
        title="确认升级"
        :message="`将下载 v${snap?.update?.version} 更新包并自动完成校验、备份与替换，随后服务会自动重启（约数秒不可用）。是否继续？`"
        confirm-label="开始升级"
        :loading="upgrading"
        @confirm="handleUpgrade"
        @cancel="upgradeConfirmOpen = false"
      />
    </template>
  </div>
</template>
