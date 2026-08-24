<script setup lang="ts">
// 个人中心仪表盘：余额/会员/等级卡片、每日签到、最近交易
// 业务逻辑保持不变；UI 重构为 Naive UI + Tailwind
import { ref, onMounted } from 'vue'
import { NButton, NIcon, NTag, NSkeleton, NAlert } from 'naive-ui'
import { PhWallet, PhCrown, PhStar } from '@phosphor-icons/vue'
import { economyApi } from '@/api/economy'
import { useAuthStore } from '@/stores/auth'
import { useToastStore } from '@/stores/toast'
import RechargeModal from '@/components/RechargeModal.vue'

const auth = useAuthStore()
const toast = useToastStore()

interface ProfileLike {
  balance: number
  exp: number
  level: number
}
interface CheckinState {
  checkedIn: boolean
  streak: number
  tomorrowBalance: number
  config: { dailyBalance: number; consecutiveDays: number; consecutiveBonusBalance: number; dailyExp: number }
}
interface TransactionRow {
  id: number
  type: string
  amount: number | string
  exp_earned: number
  description: string
  created_at: string
}
interface LevelInfoState {
  label: string
  progress: number
  expInCurrentLevel: number
  expNeeded: number
  nextLevel: number | null
}
interface MembershipState {
  membership: { level: string; expiresAt: string } | null
  downloads: number
  dailyLimit: number
}

const profile = ref<ProfileLike | null>(null)
const checkin = ref<CheckinState | null>(null)
const transactions = ref<TransactionRow[]>([])
const loading = ref(true)
const checkinLoading = ref(false)
const message = ref('')
const levelInfo = ref<LevelInfoState | null>(null)
const membershipStatus = ref<MembershipState | null>(null)

const rechargeModal = ref<InstanceType<typeof RechargeModal> | null>(null)

const typeLabels: Record<string, string> = {
  checkin: '签到',
  bonus: '连续奖励',
  purchase: '链接购买',
  recharge: '充值',
  membership: '购买会员',
}
const typeTagType = (t: string): 'success' | 'info' | 'warning' | 'default' => {
  if (t === 'recharge') return 'success'
  if (t === 'checkin' || t === 'bonus') return 'info'
  if (t === 'membership') return 'warning'
  return 'default'
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

    profile.value = pro as ProfileLike
    if (auth.user) {
      auth.user.balance = pro.balance
      auth.user.exp = pro.exp
      auth.user.level = pro.level
    }

    checkin.value = check
    transactions.value = trans.transactions || []
    levelInfo.value = lvl
    membershipStatus.value = mb
  } catch (err) {
    console.error('获取仪表盘数据失败:', err)
    toast.error('加载失败，请刷新重试')
  } finally {
    loading.value = false
  }
}

async function handleCheckin() {
  checkinLoading.value = true
  message.value = ''
  try {
    const data = await economyApi.checkIn() as unknown as {
      balanceEarned: number; expEarned: number; isBonus: boolean; leveledUp: boolean; levelLabel: string
    }
    let msg = `签到成功！获得 ${data.balanceEarned} 元${data.isBonus ? ' (含连续签到奖励)' : ''}，+${data.expEarned} 经验`
    if (data.leveledUp) msg += `，升级至 ${data.levelLabel}！`
    message.value = msg
    fetchDashboard()
  } catch (err: unknown) {
    const errorObj = err as { response?: { data?: { error?: string } } }
    toast.error(errorObj?.response?.data?.error || '签到失败，请稍后重试')
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
  <div>
    <!-- 加载骨架 -->
    <div v-if="loading" class="space-y-6">
      <NSkeleton text width="20%" size="large" />
      <div class="grid gap-4 sm:grid-cols-2 md:grid-cols-3">
        <div v-for="i in 3" :key="i" class="rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <NSkeleton text width="40%" />
          <NSkeleton text width="60%" class="mt-3" size="large" />
        </div>
      </div>
      <NSkeleton text :repeat="3" />
    </div>

    <template v-else>
      <header>
        <h1 class="text-xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">个人中心</h1>
        <p class="mt-1 text-sm text-zinc-500 dark:text-zinc-400">余额、会员与签到概览</p>
      </header>

      <NAlert v-if="message" type="success" class="mt-4" closable @close="message = ''">{{ message }}</NAlert>

      <!-- 余额 & 会员 & 等级 卡片 -->
      <div class="mt-6 grid gap-4 sm:grid-cols-2 md:grid-cols-3">
        <!-- 余额 -->
        <div class="flex flex-col rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <div class="flex items-center justify-between">
            <span class="text-[13px] text-zinc-500 dark:text-zinc-400">账户余额</span>
            <NIcon :size="18" class="text-emerald-500/70"><PhWallet /></NIcon>
          </div>
          <p class="mt-2 font-mono text-3xl font-semibold tracking-tight text-zinc-900 dark:text-zinc-50">
            {{ (profile?.balance || 0).toFixed(2) }}
            <span class="text-sm font-normal text-zinc-500 dark:text-zinc-400">元</span>
          </p>
          <div class="mt-auto pt-4">
            <NButton type="primary" size="small" @click="openRecharge">立即充值</NButton>
          </div>
        </div>

        <!-- 会员状态 -->
        <div v-if="membershipStatus" class="flex flex-col rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <div class="flex items-center justify-between">
            <span class="text-[13px] text-zinc-500 dark:text-zinc-400">会员状态</span>
            <NIcon :size="18" class="text-amber-500/80"><PhCrown weight="duotone" /></NIcon>
          </div>
          <p class="mt-2 text-2xl font-semibold tracking-tight text-zinc-900 dark:text-zinc-50">
            {{ membershipStatus.membership ? membershipStatus.membership.level : '非会员' }}
          </p>
          <div class="mt-auto pt-4">
            <template v-if="membershipStatus.membership">
              <p class="text-xs text-zinc-500 dark:text-zinc-400">
                <span class="font-mono font-medium text-zinc-800 dark:text-zinc-200">{{ daysLeftOf(membershipStatus.membership.expiresAt) }}</span> 天后到期
                <span class="mx-1.5 text-zinc-300 dark:text-zinc-700">/</span>
                今日下载
                <span class="font-mono font-medium text-zinc-800 dark:text-zinc-200">
                  {{ membershipStatus.downloads }}/{{ membershipStatus.dailyLimit === -1 ? '∞' : membershipStatus.dailyLimit }}
                </span>
              </p>
            </template>
            <NButton v-else size="small" secondary type="primary" @click="$router.push('/user/membership')">开通会员</NButton>
          </div>
        </div>

        <!-- 等级 -->
        <div class="flex flex-col rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
          <div class="flex items-center justify-between">
            <span class="text-[13px] text-zinc-500 dark:text-zinc-400">用户等级</span>
            <NIcon :size="18" class="text-emerald-500/70"><PhStar /></NIcon>
          </div>
          <p class="mt-2 text-2xl font-semibold tracking-tight text-zinc-900 dark:text-zinc-50">
            {{ levelInfo?.label || `Lv.${profile?.level || 1}` }}
          </p>
          <div v-if="levelInfo && levelInfo.nextLevel" class="mt-auto pt-4">
            <div class="h-1.5 overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-800">
              <div
                class="h-full rounded-full bg-emerald-500 transition-all dark:bg-emerald-400"
                :style="{ width: Math.min(100, levelInfo.progress) + '%' }"
              />
            </div>
            <p class="mt-1.5 font-mono text-xs text-zinc-500 dark:text-zinc-400">
              {{ levelInfo.expInCurrentLevel }}/{{ levelInfo.expNeeded }} EXP
            </p>
          </div>
        </div>
      </div>

      <!-- 签到 -->
      <section v-if="checkin" class="mt-6 rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
        <div class="flex items-center justify-between">
          <h3 class="text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">每日签到</h3>
          <NTag v-if="checkin.checkedIn" type="success" size="small" round>今日已签到</NTag>
          <NButton v-else type="primary" size="small" :loading="checkinLoading" @click="handleCheckin">立即签到</NButton>
        </div>

        <div class="mt-4 flex flex-wrap items-center gap-x-6 gap-y-3">
          <div>
            <p class="text-xs text-zinc-500 dark:text-zinc-400">连续签到</p>
            <p class="font-mono text-xl font-semibold text-zinc-900 dark:text-zinc-50">{{ checkin.streak }} <span class="text-xs font-normal text-zinc-500 dark:text-zinc-400">天</span></p>
          </div>
          <!-- 连续签到进度点：最后一格为奖励日 -->
          <div class="flex max-w-full flex-wrap items-center gap-1.5">
            <span
              v-for="i in checkin.config.consecutiveDays"
              :key="i"
              class="size-2.5 rounded-full transition-colors"
              :class="i <= checkin.streak
                ? (i === checkin.config.consecutiveDays ? 'bg-amber-400' : 'bg-emerald-500 dark:bg-emerald-400')
                : (i === checkin.config.consecutiveDays ? 'bg-amber-200 dark:bg-amber-400/25' : 'bg-zinc-200 dark:bg-zinc-700')"
              :title="i === checkin.config.consecutiveDays ? `连续 ${checkin.config.consecutiveDays} 天有额外奖励` : `第 ${i} 天`"
            />
          </div>
          <p class="text-xs text-zinc-500 dark:text-zinc-400">
            <template v-if="checkin.checkedIn">明天签到可获得 <span class="font-mono text-emerald-600 dark:text-emerald-400">+{{ checkin.tomorrowBalance }}</span> 元</template>
            <template v-else>今日签到可获得 <span class="font-mono text-emerald-600 dark:text-emerald-400">+{{ checkin.config.dailyBalance }}</span> 元</template>
          </p>
        </div>
      </section>

      <!-- 最近交易 -->
      <section class="mt-6 rounded-2xl border border-zinc-200 bg-white p-5 dark:border-zinc-800 dark:bg-zinc-900">
        <h3 class="text-[15px] font-semibold text-zinc-900 dark:text-zinc-100">最近交易</h3>
        <ul v-if="transactions.length > 0" class="mt-2 divide-y divide-zinc-100 dark:divide-zinc-800">
          <li v-for="tx in transactions" :key="tx.id" class="flex flex-wrap items-center gap-x-3 gap-y-1 py-3">
            <NTag :type="typeTagType(tx.type)" size="small" round>{{ typeLabels[tx.type] || tx.type }}</NTag>
            <span class="min-w-0 flex-1 truncate text-sm text-zinc-600 dark:text-zinc-300">{{ tx.description }}</span>
            <span
              class="font-mono text-sm font-medium"
              :class="parseFloat(String(tx.amount)) >= 0 ? 'text-emerald-600 dark:text-emerald-400' : 'text-zinc-500 dark:text-zinc-400'"
            >
              {{ parseFloat(String(tx.amount)) > 0 ? '+' : '' }}{{ parseFloat(String(tx.amount)).toFixed(2) }} 元
            </span>
            <span v-if="tx.exp_earned > 0" class="font-mono text-xs text-emerald-600 dark:text-emerald-400">+{{ tx.exp_earned }} EXP</span>
            <span class="font-mono text-xs text-zinc-500 dark:text-zinc-400">{{ new Date(tx.created_at).toLocaleDateString('zh-CN') }}</span>
          </li>
        </ul>
        <p v-else class="mt-4 rounded-lg bg-zinc-50 py-8 text-center text-sm text-zinc-500 dark:bg-zinc-800/50 dark:text-zinc-400">暂无交易记录</p>
      </section>

      <!-- 充值弹窗 -->
      <RechargeModal ref="rechargeModal" @refresh="fetchDashboard" />
    </template>
  </div>
</template>
