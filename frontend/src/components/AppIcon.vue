<script setup lang="ts">
// 全站唯一图标出口，统一 Lucide 的尺寸与线宽口径
// - 尺寸只走五档：旧代码里 10~28px 的连续取值会让奇数尺寸落在半像素上，边缘发虚且彼此差异不可辨
// - absolute-stroke-width 让线宽脱离 viewBox 缩放，14px 与 24px 图标同为 1.5px 视觉重量
// - 不提供“减细线宽”的降权手段：辅助信息一律靠颜色降权（text-zinc-500 起），细线在小尺寸下会糊成灰影
import type { Component } from 'vue'

/** xs 元数据角标 / sm 正文行内 / md 导航工具栏 / lg 区块标题 / xl 空态强调 */
type IconScale = 'xs' | 'sm' | 'md' | 'lg' | 'xl'

const SCALE_PX: Record<IconScale, number> = {
  xs: 14,
  sm: 16,
  md: 20,
  lg: 24,
  xl: 32,
}

const props = withDefaults(
  defineProps<{
    icon: Component
    scale?: IconScale
    /** 激活态 / 强调态，线宽提到 2px */
    strong?: boolean
    /** 实心填充，用于已收藏、已选中一类开关态 */
    filled?: boolean
  }>(),
  { scale: 'sm', strong: false, filled: false },
)
</script>

<template>
  <component
    :is="props.icon"
    :size="SCALE_PX[props.scale]"
    :stroke-width="props.strong ? 2 : 1.5"
    :absolute-stroke-width="true"
    :fill="props.filled ? 'currentColor' : 'none'"
    class="shrink-0"
    aria-hidden="true"
    focusable="false"
  />
</template>
