<script setup lang="ts">
// 个人中心仪表盘：1:1 移植自原项目 Node-XZJI/frontend/src/pages/UserCenter.jsx (UserDashboard)
// 样式来自全局 styles/user.css
import { ref, onMounted } from 'vue'
import { economyApi } from '@/api/economy'
import { useAuthStore } from '@/stores/auth'
import RechargeModal from '@/components/RechargeModal.vue'

const auth = useAuthStore()

const profile = ref<any>(null)
const checkin = ref<any>(null)
const transactions = ref<any[]>([])
const loading = ref(true)
const checkinLoading = ref(false)
const message = ref('')
const levelInfo = ref<any>(null)
const membershipStatus = ref<any>(null)

const rechargeModal = ref<InstanceType<typeof RechargeModal> | null>(null)

const typeLabels: Record<string, string> = {
  checkin: '签到',
  bonus: '连续奖励',
  purchase: '链接购买',
  recharge: '充值',
  membership: '购买会员',
}

onMounted(() => { fetchDashboard() })

async function fetchDashboard() {
  try {
    const [pro, check, trans, lvl, mb] = await Promise.all([
      economyApi.profile(),        // 复用 economyApi，内部调 /user/profile
      economyApi.checkInStatus(),
      economyApi.transactions(1, 5),
      economyApi.levelInfo(),
      economyApi.membership(),
    ])

    profile.value = pro
    if (auth.user) {
      auth.user.balance = pro.balance
      auth.user.exp = pro.exp
      auth.user.level = pro.level
    }

    checkin.value = check as any
    transactions.value = (trans as any).list || []
    levelInfo.value = lvl
    membershipStatus.value = mb
  } catch (err) {
    console.error('获取仪表盘数据失败:', err)
  } finally {
    loading.value = false
  }
}

async function handleCheckin() {
  checkinLoading.value = true
  message.value = ''
  try {
    const data = await economyApi.checkIn() as any
    let msg = `签到成功！获得 ${data.balanceEarned} 元${data.isBonus ? ' (含连续签到奖励)' : ''}，+${data.expEarned} 经验`
    if (data.leveledUp) msg += `，升级至 ${data.levelLabel}！`
    message.value = msg
    fetchDashboard()
  } catch (err: any) {
    message.value = err?.message || '签到失败'
  } finally {
    checkinLoading.value = false
  }
}

function openRecharge() {
  rechargeModal.value?.open()
}

function daysLeftOf(expiresAt: string) {
  return Math.max(0, Math.ceil((new Date(expiresAt).getTime() - Date.now()) / 86400000))
}
</script>

<template>
  <div class="user-content">
    <div v-if="loading" class="loading-state">加载中...</div>
    <template v-else>
    <div class="content-header">
      <h2 class="content-title">个人中心</h2>
    </div>

    <div v-if="message" class="form-success">{{ message }}</div>

      <!-- 余额 & 等级 & 会员 卡片 -->
      <div class="dashboard-cards">
        <div class="dashboard-card card-balance">
          <div class="card-icon">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <rect x="1.5" y="3.5" width="13" height="9" rx="1.5" />
              <path d="M11 8.5a1 1 0 100-2 1 1 0 000 2z" />
              <line x1="4" y1="7" x2="7" y2="7" /><line x1="4" y1="9" x2="7" y2="9" />
            </svg>
          </div>
          <div class="card-body">
            <div class="card-label">账户余额</div>
            <div class="card-value">{{ (profile?.balance || 0).toFixed(2) }}</div>
          </div>
          <div class="card-action">
            <button class="card-btn primary" @click="openRecharge">立即充值</button>
          </div>
        </div>

        <div v-if="membershipStatus" class="dashboard-card card-membership">
          <div class="card-icon">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <path d="M8 1.5l2.5 5 5.5.5-4 4 1 5.5L8 13.5l-5 3 1-5.5-4-4 5.5-.5z" />
            </svg>
          </div>
          <div class="card-body">
            <div class="card-label">会员状态</div>
            <div class="card-value">
              {{ membershipStatus.membership ? membershipStatus.membership.level : '非会员' }}
            </div>
          </div>
          <div class="card-action">
            <template v-if="membershipStatus.membership">
              <div class="membership-expiry">
                <span class="expiry-text">{{ daysLeftOf(membershipStatus.membership.expiresAt) }} 天后到期</span>
                <span class="expiry-downloads">
                  下载 {{ membershipStatus.downloads }}/{{ membershipStatus.dailyLimit === -1 ? '无限制' : membershipStatus.dailyLimit }}
                </span>
              </div>
            </template>
            <router-link v-else to="/user/membership" class="card-btn primary" style="text-decoration: none; display: block; text-align: center;">
              开通会员
            </router-link>
          </div>
        </div>

        <div class="dashboard-card card-level">
          <div class="card-icon">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
              <polygon points="8,1.5 10,5.5 14.5,6 11.25,9.25 12,13.5 8,11.5 4,13.5 4.75,9.25 1.5,6 6,5.5" />
            </svg>
          </div>
          <div class="card-body">
            <div class="card-label">用户等级</div>
            <div class="card-value">{{ levelInfo?.label || `Lv.${profile?.level || 1}` }}</div>
          </div>
          <div v-if="levelInfo && levelInfo.nextLevel" class="card-action">
            <div class="level-progress-mini">
              <div class="level-bar-mini">
                <div class="level-fill-mini" :style="{ width: levelInfo.progress + '%' }" />
              </div>
              <span class="level-text-mini">{{ levelInfo.expInCurrentLevel }}/{{ levelInfo.expNeeded }} EXP</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 签到进度 -->
      <div v-if="checkin" class="dashboard-section">
        <div class="section-header">
          <h3 class="section-title">签到</h3>
          <div class="checkin-action-header">
            <span v-if="checkin.checkedIn" class="checkin-done-badge">今日已签到</span>
            <button v-else class="action-link" :disabled="checkinLoading" @click="handleCheckin">
              {{ checkinLoading ? '签到中...' : '立即签到' }}
            </button>
          </div>
        </div>
        <div class="checkin-streak">
          <div class="streak-info">
            <span class="streak-label">连续签到</span>
            <span class="streak-count">{{ checkin.streak }} 天</span>
          </div>
          <div class="streak-bar">
            <div
              v-for="i in checkin.config.consecutiveDays"
              :key="i"
              class="streak-dot"
              :class="{ filled: i <= checkin.streak, bonus: i === checkin.config.consecutiveDays }"
            />
          </div>
          <div class="streak-hint">
            <template v-if="checkin.checkedIn">明天签到可获得 +{{ checkin.tomorrowBalance }} 元</template>
            <template v-else>今日签到可获得 +{{ checkin.config.dailyBalance }} 元</template>
          </div>
        </div>
      </div>

      <!-- 交易记录 -->
      <div class="dashboard-section">
        <div class="section-header">
          <h3 class="section-title">最近交易</h3>
        </div>
        <div v-if="transactions.length > 0" class="transactions-list">
          <div v-for="tx in transactions" :key="tx.id" class="tx-row">
            <span class="tx-badge" :class="`tx-${tx.type}`">{{ typeLabels[tx.type] || tx.type }}</span>
            <span class="tx-desc">{{ tx.description }}</span>
            <span class="tx-amount" :class="parseFloat(tx.amount) >= 0 ? 'positive' : 'negative'">
              {{ parseFloat(tx.amount) > 0 ? '+' : '' }}{{ parseFloat(tx.amount).toFixed(2) }} 元
            </span>
            <span v-if="tx.exp_earned > 0" class="tx-exp">+{{ tx.exp_earned }} EXP</span>
            <span class="tx-time">{{ new Date(tx.created_at).toLocaleDateString('zh-CN') }}</span>
          </div>
        </div>
        <div v-else class="empty-state">暂无交易记录</div>
      </div>

      <!-- 充值弹窗（复用现有 RechargeModal） -->
      <RechargeModal ref="rechargeModal" @refresh="fetchDashboard" />
    </template>
  </div>
</template>
