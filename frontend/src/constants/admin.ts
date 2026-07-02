// 管理后台共享常量（移植自原项目 Node-XZJI/frontend/src/constants/admin.js）

// 支付方式标签
export const methodLabels: Record<string, string> = {
  wechat: '微信',
  alipay: '支付宝',
  epay: '易支付',
}

// 公告类型
export const typeLabels: Record<string, string> = { info: '通知', warning: '重要', success: '活动' }
export const typeOptions: string[] = ['info', 'warning', 'success']

// 支付配置提供方
export const PROVIDERS = [
  { key: 'wechat', label: '微信支付', icon: 'W', brief: 'Native 扫码支付 V3' },
  { key: 'alipay', label: '支付宝', icon: 'A', brief: '跳转支付 / 扫码支付' },
  { key: 'epay', label: '易支付', icon: 'E', brief: '彩虹聚合收银台' },
]

// 网盘平台识别规则（关键词响应编辑器的链接解析）
export const cloudPlatforms: { name: string; test: RegExp; icon: string }[] = [
  { name: '百度网盘', test: /pan\.baidu\.com/i, icon: 'BD' },
  { name: '阿里云盘', test: /alipan\.com|aliyundrive\.com/i, icon: 'AL' },
  { name: '夸克网盘', test: /pan\.quark\.cn/i, icon: 'QK' },
  { name: '蓝奏云', test: /lanzou[a-z]*\.com/i, icon: 'LZ' },
  { name: '天翼云盘', test: /cloud\.189\.cn/i, icon: 'TY' },
  { name: '123云盘', test: /123(?:pan|684|865|912)\./i, icon: '12' },
  { name: '腾讯微云', test: /share\.weiyun\.com/i, icon: 'WY' },
  { name: 'UC网盘', test: /drive\.uc\.cn/i, icon: 'UC' },
]
