// 会员档位展示辅助：图标轮换、标识色可读性、周期 / 额度 / 折扣 / 单价 / 角标文案
// 供前台会员中心、后台会员配置、购买弹窗共用，保证三处展示口径一致
import { h, type Component } from 'vue'
import { PhDiamond, PhCrown, PhMedal, PhRocket, PhStar } from '@phosphor-icons/vue'

// 档位图标统一 duotone 字重：双层渲染，强调层随 currentColor 继承档位标识色
const duotone =
  (Icon: Component): Component =>
  (props: Record<string, unknown>) =>
    h(Icon, { weight: 'duotone', ...props })

const TIER_ICONS: Component[] = [duotone(PhDiamond), duotone(PhCrown), duotone(PhMedal), duotone(PhRocket), duotone(PhStar)]

/** 按档位序号轮换图标：钻石 → 皇冠 → 勋章 → 火箭 → 星星 */
export function tierIcon(index: number): Component {
  return TIER_ICONS[((index % TIER_ICONS.length) + TIER_ICONS.length) % TIER_ICONS.length]!
}

/** 非会员 / 无档位时的兜底皇冠，与档位图标同字重 */
export const TIER_FALLBACK_ICON: Component = duotone(PhCrown)

/** 档位标识色，空值回退站点 emerald */
export function tierColor(color?: string | null): string {
  return color || '#10b981'
}

/** 新增档位时的默认标识色轮换，避免每档都是同色 */
export const NEW_TIER_COLORS = ['#10b981', '#f59e0b', '#8b5cf6', '#0ea5e9', '#f43f5e'] as const

function srgbChannel(c: number): number {
  const s = c / 255
  return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
}

/**
 * 根据档位色亮度返回可读的前景色（白 / 深墨）。
 * 白字对比度低于 3:1（大字号 AA 下限）时改用深色文字，兜住管理员选浅色的情况。
 */
export function readableOn(color: string): string {
  const hex = color.replace('#', '')
  const full = hex.length === 3 ? hex.split('').map((c) => c + c).join('') : hex
  const int = Number.parseInt(full.slice(0, 6), 16)
  if (Number.isNaN(int)) return '#ffffff'
  const r = (int >> 16) & 255
  const g = (int >> 8) & 255
  const b = int & 255
  const lum = 0.2126 * srgbChannel(r) + 0.7152 * srgbChannel(g) + 0.0722 * srgbChannel(b)
  return 1.05 / (lum + 0.05) >= 3 ? '#ffffff' : '#1c1917'
}

/** 档位色低透明度渐染，用作卡片头部底色（浅深主题通用） */
export function tierWash(color: string): string {
  const c = tierColor(color)
  return `linear-gradient(150deg, color-mix(in srgb, ${c} 16%, transparent), color-mix(in srgb, ${c} 4%, transparent) 55%, transparent)`
}

/** 天数 → 口语周期：365 → 1 年、90 → 3 个月、7 → 7 天 */
export function periodText(days: number): string {
  if (!days || days <= 0) return '-'
  if (days >= 365) return `${Math.floor(days / 365)} 年`
  if (days >= 30) return `${Math.floor(days / 30)} 个月`
  return `${days} 天`
}

/** 每日下载额度文案（-1 / 0 视为不限量） */
export function dlText(limit: number): string {
  return limit === -1 || limit === 0 ? '不限量' : `${limit} 次/天`
}

/** 折扣文案：0.8 → "80 折"（沿用站内口径）；0 / 1 → null 表示原价 */
export function discountLabel(discount: number): string | null {
  return discount > 0 && discount < 1 ? `${Math.round(discount * 100)} 折` : null
}

/** 价格文案：整数不带小数，非整数保留两位 */
export function priceText(price: number): string {
  return Number.isInteger(price) ? String(price) : price.toFixed(2)
}

/** 折合每天单价：约 ¥0.9/天；价格或天数无效时返回 null */
export function perDayText(price: number, days: number): string | null {
  if (!days || days <= 0 || price <= 0) return null
  const v = price / days
  const s = v >= 1 ? String(Math.round(v * 10) / 10) : String(Math.round(v * 100) / 100)
  return `约 ¥${s}/天`
}

/** 套餐角标文案：去空白后为空返回 null 表示不显示（管理员可自定义，如「推荐」「超值」） */
export function badgeText(badge?: string | null): string | null {
  const t = (badge ?? '').trim()
  return t || null
}
