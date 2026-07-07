<script setup lang="ts">
// 会员中心：会员卡式状态面板 + 档位对比表（桌面）/ 堆叠卡（移动）
// 业务逻辑与后端契约不变：GET /user/membership + /user/profile，POST /user/buy-membership
import { ref, computed, onMounted } from 'vue'
import { NButton, NIcon, NTag, NSkeleton } from 'naive-ui'
import { Crown, Infinity as InfinityIcon, Download, Discount, Calendar, Check, Star, ChevronRight } from '@vicons/tabler'
import api from '@/api'
import { useAuthStore } from '@/stores/auth'
import { useToastStore } from '@/stores/toast'
import type { MembershipConfig, UserBrief } from '@/types/api'
import MembershipModal from '@/components/MembershipModal.vue'

const auth = useAuthStore()
const toast = useToastStore()

interface MembershipState {
  membership: { level: string; expiresAt: string } | null
  downloads: number
  dailyLimit: number
  configs?: MembershipConfig[]
}

const membership = ref<MembershipState | null>(null)
const profile = ref<UserBrief | null>(null)
const loading = ref(true)
const message = ref('')
const selectedPlan = ref<MembershipConfig | null>(null)

onMounted(() => { fetchMembership() })

async function fetchMembership() {
  try {
    const [mbRes, pfRes] = await Promise.all([
      api.get('/user/membership'),
      api.get('/user/profile'),
    ])
    membership.value = mbRes.data
    profile.value = pfRes.data
  } catch (err) {
    console.error(err)
    toast.error('加载失败，请刷新重试')
  } finally {
    loading.value = false
  }
}

const hasMembership = computed(() => !!membership.value?.membership)
const daysLeft = computed(() => {
  if (!hasMembership.value) return 0
  return Math.max(0, Math.ceil((new Date(membership.value!.membership!.expiresAt).getTime() - Date.now()) / 86400000))
})
const limit = computed(() => membership.value?.dailyLimit)
const limitText = computed(() => (limit.value === -1 || limit.value === 0 ? '不限量' : `${limit.value} 次`))
const isUnlimited = computed(() => limit.value === -1 || limit.value === 0)
const currentPlanColor = computed(() => {
  if (!hasMembership.value) return null
  return (membership.value?.configs || []).find((c) => c.level_name === membership.value!.membership!.level)?.color || null
})
const currentCfg = computed(() => {
  if (!hasMembership.value || !membership.value) return null
  return configs.value.find((c) => c.level_name === membership.value!.membership!.level) || null
})
const configs = computed(() => membership.value?.configs || [])

// 有效期横向进度（百分比），替代旧的环形进度
const expiryProgress = computed(() => {
  if (!hasMembership.value || !currentCfg.value) return 0
  if (currentCfg.value.days <= 0) return 100
  return Math.min(100, Math.max(0, Math.round((daysLeft.value / currentCfg.value.days) * 100)))
})
// 今日下载用量百分比（不限量时不渲染条）
const dlUsagePct = computed(() => {
  if (isUnlimited.value || !limit.value) return 0
  return Math.min(100, Math.max(0, Math.round((membership.value?.downloads ?? 0) / limit.value) * 100))
})

function isCurrentPlan(plan: MembershipConfig) {
  return hasMembership.value && membership.value!.membership!.level === plan.level_name
}
function handleBuyClick(plan: MembershipConfig) {
  if (isCurrentPlan(plan)) return
  selectedPlan.value = plan
}

function handleBuySuccess(data: unknown) {
  const d = data as { membershipExpiresAt: string; balance: number }
  message.value = `开通成功，会员有效期至 ${new Date(d.membershipExpiresAt).toLocaleDateString('zh-CN')}`
  if (auth.user) {
    auth.user.balance = d.balance
    auth.user.membership_level = selectedPlan.value?.level_name ?? auth.user.membership_level
    auth.user.membership_expires_at = d.membershipExpiresAt
  }
  selectedPlan.value = null
  fetchMembership()
}

function periodText(days: number) {
  if (days >= 365) return `${Math.floor(days / 365)} 年`
  if (days >= 30) return `${Math.floor(days / 30)} 个月`
  return `${days} 天`
}
function annualized(plan: MembershipConfig) {
  if (plan.days <= 0) return null
  return (plan.price / plan.days) * 365
}
function discountPct(plan: MembershipConfig) {
  return plan.discount > 0 && plan.discount < 1 ? Math.round(plan.discount * 100) : null
}
function expiryShort() {
  return new Date(membership.value!.membership!.expiresAt).toLocaleDateString('zh-CN', { year: 'numeric', month: 'short', day: 'numeric' })
}
// 推荐档位：价格中间档，至少两档才有
function recommendedIdx(total: number) {
  if (total < 2) return -1
  return Math.floor(total / 2)
}
function isRecommended(i: number) {
  return i === recommendedIdx(configs.value.length)
}
// 对比表行数据
function dlCell(plan: MembershipConfig) {
  const d = plan.daily_download_limit
  return d === -1 || d === 0 ? '不限量' : `${d} 次/天`
}
function discountCell(plan: MembershipConfig) {
  const d = discountPct(plan)
  return d ? `${d} 折` : '标准价'
}
function annualCell(plan: MembershipConfig) {
  const a = annualized(plan)
  return a ? `约 ¥${a.toFixed(0)}/年` : '—'
}
function planColor(plan: MembershipConfig) {
  return plan.color || '#10b981'
}

const currentDiscountText = computed(() => {
  if (!currentCfg.value) return '标准价'
  const d = discountPct(currentCfg.value)
  return d ? `${d} 折` : '标准价'
})
</script>

<template>
  <div>
    <!-- 加载骨架 -->
    <div v-if="loading" class="space-y-6">
      <NSkeleton text width="22%" size="large" />
      <NSkeleton height="132px" :sharp="false" />
      <NSkeleton height="220px" :sharp="false" />
    </div>

    <template v-else>
      <!-- 页眉 -->
      <header class="flex flex-col gap-1.5">
        <h1 class="text-[26px] font-bold leading-tight tracking-tight text-zinc-900 sm:text-3xl dark:text-zinc-50">会员中心</h1>
        <p class="max-w-[58ch] text-sm leading-relaxed text-zinc-500 dark:text-zinc-400">
          开通会员后可免费下载付费资源，并享受专属折扣。重复购买同档位会叠加有效期，不同档位切换不退还差价。
        </p>
      </header>

      <!-- 成功提示 -->
      <div
        v-if="message"
        class="mt-5 flex items-center justify-between gap-3 rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-2.5 text-sm text-emerald-700 dark:border-emerald-400/20 dark:bg-emerald-400/10 dark:text-emerald-400"
      >
        <span class="flex items-center gap-2"><NIcon :size="16"><Check /></NIcon>{{ message }}</span>
        <button type="button" class="shrink-0 text-emerald-600/70 hover:text-emerald-700 dark:text-emerald-400/70" @click="message = ''">关闭</button>
      </div>

      <!-- 会员卡式状态面板 -->
      <section
        class="mt-6 overflow-hidden rounded-2xl border bg-white dark:bg-zinc-900"
        :class="hasMembership ? 'border-zinc-200 dark:border-zinc-800' : 'border-dashed border-zinc-300 bg-zinc-50/60 dark:border-zinc-700 dark:bg-zinc-900/40'"
      >
        <div class="flex">
          <!-- 左侧色条（会员态） -->
          <div v-if="hasMembership" class="w-1.5 shrink-0" :style="{ backgroundColor: currentPlanColor || '#10b981' }" />

          <div class="flex-1 p-6 sm:p-7">
            <!-- 会员态 -->
            <div v-if="hasMembership" class="flex flex-col gap-7 lg:flex-row lg:items-center lg:justify-between">
              <!-- 等级标识 -->
              <div class="flex items-center gap-4">
                <span
                  class="grid size-14 shrink-0 place-items-center rounded-2xl text-white shadow-sm"
                  :style="{ backgroundColor: currentPlanColor || '#10b981' }"
                >
                  <NIcon :size="28"><Crown /></NIcon>
                </span>
                <div class="min-w-0">
                  <div class="flex items-center gap-2.5">
                    <span class="truncate text-2xl font-bold tracking-tight" :style="currentPlanColor ? { color: currentPlanColor } : {}">
                      {{ membership!.membership!.level }}
                    </span>
                    <NTag type="success" size="small" round :bordered="false">生效中</NTag>
                  </div>
                  <p class="mt-1 flex items-center gap-1.5 text-xs text-zinc-500 dark:text-zinc-400">
                    <NIcon :size="13"><Calendar /></NIcon>
                    {{ expiryShort() }} 到期 · 剩余 {{ daysLeft }} 天
                  </p>
                </div>
              </div>

              <!-- 指标三联（hairline 分隔） -->
              <div class="grid grid-cols-1 gap-5 sm:grid-cols-3 sm:gap-0 sm:divide-x sm:divide-zinc-100 lg:w-[60%] dark:sm:divide-zinc-800">
                <!-- 有效期进度 -->
                <div class="sm:px-6 sm:first:pl-0">
                  <p class="text-xs font-medium text-zinc-500 dark:text-zinc-400">有效期进度</p>
                  <p class="mt-1.5 font-mono text-lg font-bold text-zinc-900 dark:text-zinc-50">
                    {{ daysLeft }}<span class="ml-1 text-xs font-normal text-zinc-400 dark:text-zinc-500">/ {{ currentCfg?.days ?? 0 }} 天</span>
                  </p>
                  <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-800">
                    <div class="h-full rounded-full transition-all" :style="{ width: `${expiryProgress}%`, backgroundColor: currentPlanColor || '#10b981' }" />
                  </div>
                </div>
                <!-- 今日下载 -->
                <div class="sm:px-6">
                  <p class="text-xs font-medium text-zinc-500 dark:text-zinc-400">今日下载</p>
                  <p class="mt-1.5 font-mono text-lg font-bold text-zinc-900 dark:text-zinc-50">
                    {{ membership!.downloads }}<span class="ml-1 text-xs font-normal text-zinc-400 dark:text-zinc-500">/ {{ limitText }}</span>
                  </p>
                  <div v-if="!isUnlimited" class="mt-2 h-1.5 overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-800">
                    <div class="h-full rounded-full bg-emerald-500 transition-all dark:bg-emerald-400" :style="{ width: `${dlUsagePct}%` }" />
                  </div>
                  <p v-else class="mt-2 text-[11px] text-zinc-400 dark:text-zinc-500">不限量</p>
                </div>
                <!-- 折扣权益 -->
                <div class="sm:px-6">
                  <p class="text-xs font-medium text-zinc-500 dark:text-zinc-400">付费链接折扣</p>
                  <p class="mt-1.5 text-lg font-bold text-zinc-900 dark:text-zinc-50">{{ currentDiscountText }}</p>
                  <p class="mt-2 flex items-center gap-1 text-[11px] text-zinc-400 dark:text-zinc-500">
                    <NIcon :size="12"><Discount /></NIcon>购买付费链接时自动应用
                  </p>
                </div>
              </div>
            </div>

            <!-- 非会员态 -->
            <div v-else class="flex flex-col gap-5 sm:flex-row sm:items-center sm:justify-between">
              <div class="flex items-center gap-4">
                <span class="grid size-14 shrink-0 place-items-center rounded-2xl bg-zinc-100 text-zinc-400 dark:bg-zinc-800 dark:text-zinc-500">
                  <NIcon :size="28"><Crown /></NIcon>
                </span>
                <div>
                  <p class="text-xl font-bold text-zinc-700 dark:text-zinc-300">尚未开通会员</p>
                  <p class="mt-1 text-xs text-zinc-500 dark:text-zinc-400">选择下方档位开通，享受付费内容免费下载与专属折扣。</p>
                </div>
              </div>
              <a
                href="#plans"
                class="inline-flex items-center gap-1.5 self-start rounded-lg bg-zinc-900 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-zinc-800 sm:self-auto dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-white"
              >
                查看档位
                <NIcon :size="15"><ChevronRight /></NIcon>
              </a>
            </div>
          </div>
        </div>
      </section>

      <!-- 档位对比 -->
      <section id="plans" class="mt-10 scroll-mt-6">
        <div class="flex items-baseline justify-between gap-3">
          <h2 class="text-lg font-semibold tracking-tight text-zinc-900 dark:text-zinc-100">选择档位</h2>
          <p class="text-xs text-zinc-500 dark:text-zinc-400">共 {{ configs.length }} 个档位</p>
        </div>

        <!-- 空状态 -->
        <div
          v-if="configs.length === 0"
          class="mt-4 rounded-2xl border border-dashed border-zinc-300 py-16 text-center dark:border-zinc-700"
        >
          <NIcon :size="28" class="mx-auto text-zinc-300 dark:text-zinc-600"><Crown /></NIcon>
          <p class="mt-3 text-sm text-zinc-500 dark:text-zinc-400">暂未配置会员档位，请稍后再来。</p>
        </div>

        <!-- 桌面：对比表 -->
        <div v-else class="mt-4 hidden overflow-hidden rounded-2xl border border-zinc-200 dark:border-zinc-800 lg:block">
          <table class="w-full border-collapse text-sm">
            <thead>
              <tr>
                <th class="w-32 bg-zinc-50/60 px-5 py-4 text-left text-xs font-medium text-zinc-500 align-bottom dark:bg-zinc-900/60 dark:text-zinc-400">
                  对比项
                </th>
                <th
                  v-for="(plan, i) in configs"
                  :key="plan.level_name"
                  class="px-5 py-4 align-bottom"
                  :class="isRecommended(i) ? 'bg-emerald-50/40 dark:bg-emerald-400/5' : ''"
                >
                  <div class="flex items-center gap-2">
                    <span class="size-2.5 rounded-full" :style="{ backgroundColor: planColor(plan) }" />
                    <span class="text-base font-bold tracking-tight" :style="{ color: planColor(plan) }">{{ plan.level_name }}</span>
                    <NTag v-if="isCurrentPlan(plan)" size="small" round type="success" :bordered="false">当前</NTag>
                    <NTag v-else-if="isRecommended(i)" size="small" round :bordered="false" class="!bg-emerald-100 !text-emerald-700 dark:!bg-emerald-400/15 dark:!text-emerald-400">推荐</NTag>
                  </div>
                  <div class="mt-2 flex items-baseline gap-1">
                    <span class="text-xs text-zinc-400 dark:text-zinc-500">¥</span>
                    <span class="font-mono text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">{{ plan.price }}</span>
                    <span class="text-xs text-zinc-400 dark:text-zinc-500">/ {{ periodText(plan.days) }}</span>
                  </div>
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-zinc-100 dark:divide-zinc-800">
              <tr>
                <td class="bg-zinc-50/40 px-5 py-3.5 text-zinc-500 dark:bg-zinc-900/40 dark:text-zinc-400">有效期</td>
                <td
                  v-for="(plan, i) in configs"
                  :key="plan.level_name"
                  class="px-5 py-3.5 font-medium text-zinc-800 dark:text-zinc-200"
                  :class="isRecommended(i) ? 'bg-emerald-50/30 dark:bg-emerald-400/5' : ''"
                >{{ periodText(plan.days) }}</td>
              </tr>
              <tr>
                <td class="bg-zinc-50/40 px-5 py-3.5 text-zinc-500 dark:bg-zinc-900/40 dark:text-zinc-400">每日下载</td>
                <td
                  v-for="(plan, i) in configs"
                  :key="plan.level_name"
                  class="px-5 py-3.5 font-medium text-zinc-800 dark:text-zinc-200"
                  :class="isRecommended(i) ? 'bg-emerald-50/30 dark:bg-emerald-400/5' : ''"
                >{{ dlCell(plan) }}</td>
              </tr>
              <tr>
                <td class="bg-zinc-50/40 px-5 py-3.5 text-zinc-500 dark:bg-zinc-900/40 dark:text-zinc-400">付费链接折扣</td>
                <td
                  v-for="(plan, i) in configs"
                  :key="plan.level_name"
                  class="px-5 py-3.5 font-medium text-zinc-800 dark:text-zinc-200"
                  :class="isRecommended(i) ? 'bg-emerald-50/30 dark:bg-emerald-400/5' : ''"
                >{{ discountCell(plan) }}</td>
              </tr>
              <tr>
                <td class="bg-zinc-50/40 px-5 py-3.5 text-zinc-500 dark:bg-zinc-900/40 dark:text-zinc-400">年化单价</td>
                <td
                  v-for="(plan, i) in configs"
                  :key="plan.level_name"
                  class="px-5 py-3.5 text-sm text-zinc-500 dark:text-zinc-400"
                  :class="isRecommended(i) ? 'bg-emerald-50/30 dark:bg-emerald-400/5' : ''"
                >{{ annualCell(plan) }}</td>
              </tr>
              <tr>
                <td class="bg-zinc-50/40 px-5 py-4 dark:bg-zinc-900/40"></td>
                <td
                  v-for="(plan, i) in configs"
                  :key="plan.level_name"
                  class="px-5 py-4"
                  :class="isRecommended(i) ? 'bg-emerald-50/30 dark:bg-emerald-400/5' : ''"
                >
                  <NButton
                    :type="isCurrentPlan(plan) ? 'default' : 'primary'"
                    :secondary="isCurrentPlan(plan)"
                    :disabled="isCurrentPlan(plan)"
                    block
                    @click="handleBuyClick(plan)"
                  >
                    {{ isCurrentPlan(plan) ? '当前方案' : '立即开通' }}
                  </NButton>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- 移动：堆叠卡 -->
        <div class="mt-4 space-y-4 lg:hidden">
          <article
            v-for="(plan, i) in configs"
            :key="plan.level_name"
            class="relative overflow-hidden rounded-2xl border bg-white dark:bg-zinc-900"
            :class="[
              isCurrentPlan(plan)
                ? 'border-zinc-300 dark:border-zinc-700'
                : (isRecommended(i)
                    ? 'border-emerald-300 dark:border-emerald-400/25'
                    : 'border-zinc-200 dark:border-zinc-800'),
            ]"
          >
            <div class="h-1 w-full" :style="{ background: planColor(plan) }" />
            <div class="p-5">
              <div class="flex items-center justify-between gap-2">
                <span class="truncate text-lg font-bold tracking-tight" :style="{ color: planColor(plan) }">{{ plan.level_name }}</span>
                <NTag v-if="isCurrentPlan(plan)" size="small" round type="success" :bordered="false">当前</NTag>
                <NTag v-else-if="isRecommended(i)" size="small" round :bordered="false" class="!bg-emerald-100 !text-emerald-700 dark:!bg-emerald-400/15 dark:!text-emerald-400">推荐</NTag>
              </div>
              <div class="mt-3 flex items-baseline gap-1">
                <span class="text-sm font-medium text-zinc-400 dark:text-zinc-500">¥</span>
                <span class="font-mono text-3xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50">{{ plan.price }}</span>
                <span class="ml-1 text-sm text-zinc-400 dark:text-zinc-500">/ {{ periodText(plan.days) }}</span>
              </div>

              <dl class="mt-4 space-y-2 border-t border-zinc-100 pt-4 text-sm dark:border-zinc-800">
                <div class="flex justify-between"><dt class="text-zinc-500 dark:text-zinc-400">有效期</dt><dd class="font-medium text-zinc-800 dark:text-zinc-200">{{ periodText(plan.days) }}</dd></div>
                <div class="flex justify-between"><dt class="text-zinc-500 dark:text-zinc-400">每日下载</dt><dd class="font-medium text-zinc-800 dark:text-zinc-200">{{ dlCell(plan) }}</dd></div>
                <div class="flex justify-between"><dt class="text-zinc-500 dark:text-zinc-400">付费折扣</dt><dd class="font-medium text-zinc-800 dark:text-zinc-200">{{ discountCell(plan) }}</dd></div>
                <div v-if="annualized(plan)" class="flex justify-between"><dt class="text-zinc-500 dark:text-zinc-400">年化单价</dt><dd class="text-zinc-500 dark:text-zinc-400">{{ annualCell(plan) }}</dd></div>
              </dl>

              <NButton
                class="mt-4"
                :type="isCurrentPlan(plan) ? 'default' : 'primary'"
                :secondary="isCurrentPlan(plan)"
                :disabled="isCurrentPlan(plan)"
                block
                @click="handleBuyClick(plan)"
              >
                {{ isCurrentPlan(plan) ? '当前方案' : '立即开通' }}
              </NButton>
            </div>
          </article>
        </div>
      </section>

      <!-- 权益说明 -->
      <section v-if="configs.length > 0" class="mt-10 rounded-2xl border border-zinc-200 bg-zinc-50/40 p-6 dark:border-zinc-800 dark:bg-zinc-900/40">
        <h3 class="text-sm font-semibold text-zinc-800 dark:text-zinc-200">会员权益说明</h3>
        <ul class="mt-3 grid gap-2.5 text-sm text-zinc-600 sm:grid-cols-2 dark:text-zinc-400">
          <li class="flex items-start gap-2"><NIcon :size="16" class="mt-0.5 shrink-0 text-emerald-500 dark:text-emerald-400"><Download /></NIcon>每日下载额度内可免费获取付费资源，超出额度后按原价购买。</li>
          <li class="flex items-start gap-2"><NIcon :size="16" class="mt-0.5 shrink-0 text-emerald-500 dark:text-emerald-400"><Discount /></NIcon>购买付费链接时自动享受档位对应折扣，折扣在结算时应用。</li>
          <li class="flex items-start gap-2"><NIcon :size="16" class="mt-0.5 shrink-0 text-emerald-500 dark:text-emerald-400"><InfinityIcon /></NIcon>续费同档位会从当前到期日叠加延长有效期，不会损失剩余天数。</li>
          <li class="flex items-start gap-2"><NIcon :size="16" class="mt-0.5 shrink-0 text-emerald-500 dark:text-emerald-400"><Star /></NIcon>切换至不同档位不退还差价，请按需选择合适的档位。</li>
        </ul>
      </section>
    </template>

    <!-- 购买弹窗 -->
    <MembershipModal
      v-if="selectedPlan && profile"
      :plan="selectedPlan"
      :profile="profile"
      @close="selectedPlan = null"
      @success="handleBuySuccess"
    />
  </div>
</template>
