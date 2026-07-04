// Naive UI 全局主题定制：与 Tailwind zinc/emerald 体系对齐
// - 品牌色：翡翠绿（浅色端加深、深色端提亮，保证 WCAG 对比度）
// - 字体：系统字体栈（国内部署不依赖外网字体服务）
import type { GlobalThemeOverrides } from 'naive-ui'

export const fontSans =
  "-apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei UI', 'Microsoft YaHei', 'Noto Sans SC', 'Helvetica Neue', Arial, sans-serif"
export const fontMono =
  "ui-monospace, 'SFMono-Regular', 'Cascadia Mono', 'JetBrains Mono', Menlo, Consolas, 'Liberation Mono', monospace"

const shared: NonNullable<GlobalThemeOverrides['common']> = {
  fontFamily: fontSans,
  fontFamilyMono: fontMono,
  fontSize: '14px',
  fontWeightStrong: '600',
  borderRadius: '8px',
  borderRadiusSmall: '6px',
}

export const lightThemeOverrides: GlobalThemeOverrides = {
  common: {
    ...shared,
    bodyColor: '#f6f6f7',
    primaryColor: '#059669',
    primaryColorHover: '#10b981',
    primaryColorPressed: '#047857',
    primaryColorSuppl: '#10b981',
  },
}

export const darkThemeOverrides: GlobalThemeOverrides = {
  common: {
    ...shared,
    bodyColor: '#09090b',
    primaryColor: '#34d399',
    primaryColorHover: '#6ee7b7',
    primaryColorPressed: '#10b981',
    primaryColorSuppl: '#34d399',
  },
}
