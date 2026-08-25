<script setup lang="ts">
// 顶栏会员入口：皇冠图标 → 速览会员套餐 → 立即开通跳会员中心
// 套餐来自 GET /user/membership（需登录），首次展开才拉取；展示口径复用 @/utils/tier，
// 与会员中心 / 后台配置页 / 购买弹窗保持一致
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NPopover, NSkeleton } from 'naive-ui'
import { Crown } from '@lucide/vue'
import AppIcon from './AppIcon.vue'
import { economyApi, type MembershipPlan } from '@/api/economy'
import { useAuthStore } from '@/stores/auth'
import { tierIcon, tierColor, readableOn, periodText, dlText, discountLabel, priceText } from '@/utils/tier'

const auth = useAuthStore()
const router = useRouter()

const show = ref(false)
const plans = ref<MembershipPlan[]>([])
const loading = ref(false)
const loaded = ref(false)

// 弹窗只做速览，最多列 3 档，完整档位在会员中心
const briefPlans = computed(() => plans.value.slice(0, 3))

async function handleShow(v: boolean) {
  show.value = v
  if (!v || loaded.value || loading.value) return
  loading.value = true
  try {
    const data = await economyApi.membership()
    plans.value = data?.configs ?? []
    loaded.value = true
  } catch {
    plans.value = []
  } finally {
    loading.value = false
  }
}

function goMembership() {
  show.value = false
  router.push('/user/membership')
}
</script>

<template>
  <NPopover
    v-if="auth.isLoggedIn"
    :show="show"
    trigger="click"
    placement="bottom-end"
    :show-arrow="false"
    raw
    @update:show="handleShow"
  >
    <template #trigger>
      <button
        type="button"
        class="flex size-9 items-center justify-center rounded-full text-amber-600 motion-safe:transition-colors hover:bg-amber-500/10 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 dark:text-amber-400 dark:hover:bg-amber-400/10 dark:focus-visible:ring-emerald-400/60"
        :aria-expanded="show"
        aria-haspopup="true"
        aria-label="会员套餐"
        title="会员套餐"
      >
        <AppIcon :icon="Crown" scale="md" />
      </button>
    </template>

    <!-- 宽度按 vw 收窄：弹窗右对齐皇冠按钮（其右侧还有主题切换与头像），
         小屏用 19rem 会顶出视口左边，vueuc 的跟随层只翻转不做水平钳制 -->
    <div class="w-[min(19rem,66vw)] overflow-hidden rounded-2xl border border-zinc-200 bg-white shadow-xl dark:border-zinc-800 dark:bg-zinc-900">
      <div class="border-b border-zinc-100 px-4 py-3 dark:border-zinc-800">
        <div class="flex items-center gap-2">
          <AppIcon :icon="Crown" scale="xs" class="text-amber-600 dark:text-amber-400" />
          <h4 class="text-[13.5px] font-semibold text-zinc-900 dark:text-zinc-100">会员套餐</h4>
        </div>
        <p class="mt-0.5 text-[11px] text-zinc-500 dark:text-zinc-400">开通后免费下载付费资源，付费链接享折扣</p>
      </div>

      <div v-if="loading" class="space-y-3 px-4 py-3">
        <div v-for="i in 2" :key="i" class="flex items-center gap-2.5">
          <NSkeleton circle width="28px" height="28px" />
          <div class="flex-1 space-y-1.5">
            <NSkeleton text width="55%" />
            <NSkeleton text width="80%" />
          </div>
        </div>
      </div>

      <p
        v-else-if="briefPlans.length === 0"
        class="px-4 py-6 text-center text-xs text-zinc-500 dark:text-zinc-400"
      >暂未开放会员档位</p>

      <ul v-else class="divide-y divide-zinc-100 dark:divide-zinc-800">
        <li
          v-for="(plan, i) in briefPlans"
          :key="plan.level_name"
          class="flex items-center gap-2.5 px-4 py-2.5"
        >
          <span
            class="grid size-7 shrink-0 place-items-center rounded-lg"
            :style="{ backgroundColor: tierColor(plan.color), color: readableOn(tierColor(plan.color)) }"
          >
            <AppIcon :icon="tierIcon(i)" scale="xs" />
          </span>
          <span class="min-w-0 flex-1">
            <span class="block truncate text-[13px] font-medium text-zinc-800 dark:text-zinc-200">{{ plan.level_name }}</span>
            <span class="mt-0.5 block truncate text-[11px] text-zinc-500 dark:text-zinc-400">
              {{ dlText(plan.daily_download_limit) }}<template v-if="discountLabel(plan.discount)"> · {{ discountLabel(plan.discount) }}</template>
            </span>
          </span>
          <span class="shrink-0 text-right">
            <span class="block font-mono text-[13px] font-bold text-zinc-900 dark:text-zinc-100">¥{{ priceText(plan.price) }}</span>
            <span class="mt-0.5 block text-[11px] text-zinc-500 dark:text-zinc-400">/ {{ periodText(plan.days) }}</span>
          </span>
        </li>
      </ul>

      <div class="border-t border-zinc-100 px-4 py-3 dark:border-zinc-800">
        <NButton type="primary" size="small" block @click="goMembership">立即开通</NButton>
        <p
          v-if="plans.length > briefPlans.length"
          class="mt-2 text-center text-[11px] text-zinc-500 dark:text-zinc-400"
        >共 {{ plans.length }} 个档位，前往会员中心查看全部</p>
      </div>
    </div>
  </NPopover>
</template>
