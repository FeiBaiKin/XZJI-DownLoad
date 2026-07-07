<script setup lang="ts">
// 授权管理：接入授权升级中心（激活 / 心跳 / 状态展示 / 在线升级）
// 业务逻辑保持不变；UI 重构为 Naive UI + Tailwind
import { ref, computed, onMounted } from 'vue'
import { NButton, NInput, NAlert, NTag, NSkeleton } from 'naive-ui'
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
const badge = computed<{ text: string; type: 'success' | 'warning' | 'error' }>(() => {
  const s = snap.value
  if (!s) return { text: '未知', type: 'warning' }
  if (s.valid) {
    return s.lastError
      ? { text: '已授权 · 离线宽限中', type: 'warning' }
      : { text: '授权有效', type: 'success' }
  }
  const map: Record<string, string> = {
    unactivated: '未激活',
    expired: '已过期',
    revoked: '已吊销',
    domain_mismatch: '域名不匹配',
    grace_exceeded: '离线超期，已停服',
  }
  return { text: map[s.reason || ''] || `无效（${s.reason}）`, type: s.reason === 'unactivated' ? 'warning' : 'error' }
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
  <div>
    <div v-if="loading" class="space-y-4">
      <NSkeleton text width="20%" size="large" />
      <NSkeleton height="220px" :sharp="false" />
      <NSkeleton height="160px" :sharp="false" />
    </div>

    <template v-else>
      <header class="mb-6">
        <h2 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">授权管理</h2>
        <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">软件授权激活、心跳与在线升级（授权中心：www.xzji.top）</p>
      </header>

      <div class="space-y-5">
        <!-- 授权状态 -->
        <section class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="flex items-center gap-3">
              <h3 class="text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">授权状态</h3>
              <NTag :type="badge.type" size="small" round>{{ badge.text }}</NTag>
            </div>
            <div class="flex flex-col gap-2 sm:flex-row">
              <NButton size="small" secondary :disabled="!snap?.activated" :loading="beating" @click="handleHeartbeat">
                {{ beating ? '正在联系授权中心...' : '立即心跳' }}
              </NButton>
              <NButton size="small" quaternary @click="fetchStatus">刷新</NButton>
            </div>
          </div>

          <dl v-if="snap" class="mt-4 grid gap-x-8 gap-y-3 sm:grid-cols-2 lg:grid-cols-3">
            <div>
              <dt class="text-xs text-zinc-500 dark:text-zinc-400">授权码</dt>
              <dd class="mt-0.5 truncate font-mono text-sm text-zinc-800 dark:text-zinc-200">{{ snap.code || '未激活' }}</dd>
            </div>
            <div>
              <dt class="text-xs text-zinc-500 dark:text-zinc-400">绑定域名</dt>
              <dd class="mt-0.5 truncate font-mono text-sm text-zinc-800 dark:text-zinc-200">{{ snap.domain }}</dd>
            </div>
            <div>
              <dt class="text-xs text-zinc-500 dark:text-zinc-400">到期时间</dt>
              <dd class="mt-0.5 font-mono text-sm text-zinc-800 dark:text-zinc-200">{{ expiresText }}</dd>
            </div>
            <div>
              <dt class="text-xs text-zinc-500 dark:text-zinc-400">上次验证</dt>
              <dd class="mt-0.5 font-mono text-sm text-zinc-800 dark:text-zinc-200">{{ fmtTime(snap.lastVerifiedAt) }}</dd>
            </div>
            <div v-if="snap.activated">
              <dt class="text-xs text-zinc-500 dark:text-zinc-400">心跳间隔</dt>
              <dd class="mt-0.5 font-mono text-sm text-zinc-800 dark:text-zinc-200">{{ snap.heartbeatIntervalHours || '-' }} 小时</dd>
            </div>
            <div v-if="snap.activated">
              <dt class="text-xs text-zinc-500 dark:text-zinc-400">离线宽限期至</dt>
              <dd class="mt-0.5 font-mono text-sm text-zinc-800 dark:text-zinc-200">{{ fmtTime(snap.graceDeadline) }}</dd>
            </div>
          </dl>

          <p v-if="snap?.lastError" class="mt-4 rounded-lg bg-amber-50 px-3 py-2 text-xs leading-relaxed text-amber-700 dark:bg-amber-400/10 dark:text-amber-400">
            最近一次联系授权中心失败：{{ snap.lastError }}<template v-if="snap.valid">（宽限期内凭本地缓存正常运行）</template>
          </p>
          <p v-if="snap && !snap.valid" class="mt-4 rounded-lg bg-red-50 px-3 py-2 text-xs leading-relaxed text-red-600 dark:bg-red-500/10 dark:text-red-400">
            前台已停止对外服务，访客将看到「授权已过期」提示；激活有效授权后自动恢复。
          </p>
        </section>

        <!-- 激活 -->
        <section class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <h3 class="text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">
            {{ snap?.activated ? '重新激活 / 更换授权码' : '激活站点' }}
          </h3>
          <div class="mt-4 max-w-xl">
            <label class="mb-1.5 block text-sm text-zinc-600 dark:text-zinc-400">授权码</label>
            <NInput
              v-model:value="codeInput"
              class="font-mono"
              placeholder="XXXXX-XXXXX-XXXXX-XXXXX"
              @keyup.enter="handleActivate"
            />
            <p class="mt-1.5 text-xs text-zinc-500 dark:text-zinc-400">
              授权码由服务商提供；首次激活将绑定当前站点域名 {{ snap?.domain }}，同域名重装可重复激活
            </p>
          </div>
          <NButton type="primary" class="mt-4" :loading="activating" @click="handleActivate">
            {{ activating ? '激活中...' : '激活' }}
          </NButton>
        </section>

        <!-- 版本与升级 -->
        <section class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <h3 class="text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">版本与升级</h3>
          <dl class="mt-4 grid gap-x-8 gap-y-3 sm:grid-cols-2">
            <div>
              <dt class="text-xs text-zinc-500 dark:text-zinc-400">当前版本</dt>
              <dd class="mt-0.5 font-mono text-sm text-zinc-800 dark:text-zinc-200">v{{ snap?.currentVersion }}</dd>
            </div>
            <div>
              <dt class="text-xs text-zinc-500 dark:text-zinc-400">最新版本</dt>
              <dd class="mt-0.5 font-mono text-sm text-zinc-800 dark:text-zinc-200">{{ snap?.latestVersion ? 'v' + snap.latestVersion : '暂无发布版本' }}</dd>
            </div>
          </dl>

          <template v-if="snap?.update">
            <div class="mt-4 flex flex-wrap items-center gap-3 rounded-xl bg-emerald-50/70 px-4 py-3 dark:bg-emerald-400/8">
              <span class="text-sm text-zinc-800 dark:text-zinc-200">
                发现新版本 <span class="font-mono font-semibold">v{{ snap.update.version }}</span>（{{ fmtSize(snap.update.file_size) }}）
              </span>
              <NTag v-if="snap.update.forced" type="error" size="small" round>强制更新</NTag>
              <NButton
                type="primary"
                size="small"
                :disabled="upgrading || restarting || !snap.valid"
                :loading="upgrading"
                @click="upgradeConfirmOpen = true"
              >
                {{ upgrading ? '正在下载升级...' : '下载并升级' }}
              </NButton>
            </div>
            <pre v-if="snap.update.changelog" class="mt-3 rounded-xl bg-zinc-50 p-4 font-sans text-xs leading-relaxed whitespace-pre-wrap text-zinc-600 dark:bg-zinc-800/60 dark:text-zinc-300">{{ snap.update.changelog }}</pre>
            <p v-if="!snap.valid" class="mt-3 text-xs text-amber-600 dark:text-amber-400">授权无效时无法升级，请先激活有效授权</p>
          </template>
          <p v-else class="mt-4 text-sm text-zinc-400 dark:text-zinc-500">当前已是最新版本</p>

          <NAlert v-if="restarting" type="warning" class="mt-4">升级完成，服务正在重启，页面将自动刷新...</NAlert>
          <NAlert v-if="upgradeNote" type="info" class="mt-4">{{ upgradeNote }}</NAlert>
        </section>

        <NAlert v-if="message" type="success" closable @close="message = ''">{{ message }}</NAlert>
        <NAlert v-if="error" type="error" closable @close="error = ''">{{ error }}</NAlert>
      </div>

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
