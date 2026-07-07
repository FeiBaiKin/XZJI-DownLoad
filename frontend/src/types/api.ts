// API 响应类型定义

// 用户相关
export interface UserBrief {
  id: number
  username: string
  role: string
  email: string
  nickname: string
  avatar: string
  balance: number
  exp: number
  level: number
  membership_level: string | null
  membership_expires_at: string | null
  created_at: string
}

export interface LoginResult {
  message: string
  token: string
  user: UserBrief
}

// 关键词相关
export interface KeywordBlock {
  _id?: string
  type: 'text' | 'image' | 'link'
  text?: string
  src?: string
  alt?: string
  url?: string
  code?: string
  label?: string
  cost?: number
  access?: 'free' | 'balance' | 'membership'
}

export interface Keyword {
  id: number
  keyword: string
  response: string | KeywordBlock[]
  icon?: string
  attributes?: string // JSON 字符串: [{label,value}]
  category_ids?: number[] // 后台列表附带的所属分类 ID
  created_at: string
  updated_at: string
}

// 后台关键词分页结果
export interface AdminPaginatedKeywords {
  items: Keyword[]
  total: number
  page: number
  limit: number
}

export interface KeywordCreate {
  keyword: string
  response: string | KeywordBlock[]
}

export interface KeywordUpdate {
  keyword?: string
  response?: string | KeywordBlock[]
  icon?: string
  attributes?: string // JSON 字符串
}

// 关键词属性 [{label:'大小', value:'56MB'}]
export interface KeywordAttribute {
  label: string
  value: string
}

// 分类相关
export interface Category {
  id: number
  name: string
  description: string
  sort_order: number
  created_at: string
  updated_at: string
}

// 公开关键词预览（不含完整 response，完整内容需走 /search）
export interface PublicKeyword {
  id: number
  keyword: string
  description: string
  icon?: string
  attributes?: string // JSON 字符串: [{label,value}]
  created_at?: string
  updated_at?: string
}

// 媒体库文件
export interface MediaFile {
  id: number
  url: string
  filename: string
  origin_name: string
  mime: string
  size: number
  width: number
  height: number
  source: 'local' | 'remote'
  uploader_id: number
  created_at: string
}

// 公开关键词分页结果
export interface PaginatedKeywords {
  items: PublicKeyword[]
  total: number
  page: number
  page_size: number
}

// 公开站点配置（首页 footer / logo 用）
export interface PublicSiteSettings {
  siteName: string
  siteDescription: string
  footerText: string
  siteLogoType: string
  siteLogoUrl: string
  siteIcp: string
  sitePolice: string
  [key: string]: string
}

export interface CategoryCreate {
  name: string
  description?: string
  sort_order?: number
}

export interface CategoryUpdate {
  name?: string
  description?: string
  sort_order?: number
}

// 页面相关
export interface Page {
  id: number
  title: string
  slug: string
  content: string
  is_published: boolean
  created_at: string
  updated_at: string
}

export interface PageCreate {
  title: string
  slug: string
  content: string
  is_published: boolean
}

export interface PageUpdate {
  title?: string
  slug?: string
  content?: string
  is_published?: boolean
}

// 公告相关
export interface Announcement {
  id: number
  title: string
  content: string
  type: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface AnnouncementCreate {
  title: string
  content: string
  type?: string
  is_active?: boolean
}

export interface AnnouncementUpdate {
  title?: string
  content?: string
  type?: string
  is_active?: boolean
}

// 签到相关
export interface CheckInConfig {
  daily_balance: number
  consecutive_days: number
  consecutive_bonus_balance: number
  daily_exp: number
}

export interface CheckInStatus {
  checkedIn: boolean
  streak: number
  todayBalance: number
  todayExp: number
  isBonus: boolean
  config: {
    dailyBalance: number
    consecutiveDays: number
    consecutiveBonusBalance: number
    dailyExp: number
  }
  tomorrowBalance: number
}

export interface CheckInResult {
  message: string
  balanceEarned: number
  expEarned: number
  totalBalance: number
  totalExp: number
  level: number
  streak: number
  isBonus: boolean
  leveledUp: boolean
  newLevel: number
  levelLabel: string
}

// 等级相关
export interface LevelConfig {
  level: number
  exp_required: number
  label: string
}

export interface LevelInfo {
  level: number
  label: string
  exp: number
  expInCurrentLevel: number
  expNeeded: number
  progress: number
  nextLevel: number
  nextLabel: string
  allLevels: LevelConfig[]
}

// 会员相关
export interface MembershipConfig {
  level_name: string
  days: number
  price: number
  daily_download_limit: number
  discount: number
  color: string
}

export interface MembershipStatus {
  membership: { level: string; expiresAt: string } | null
  downloads: number
  dailyLimit: number
  configs: MembershipConfig[]
}

// 支付相关
export interface PaymentOrder {
  id: number
  order_no: string
  user_id: number
  amount: number
  points: number
  status: 'pending' | 'paid' | 'failed'
  payment_method: string
  trade_no: string
  created_at: string
  paid_at: string | null
}

export interface RechargeResult {
  orderNo: string
  payUrl: string
  qrCode: string
}

export interface PurchaseResult {
  message: string
  alreadyPurchased?: boolean
  membership?: boolean
  balance?: number
  balanceSpent?: number
  originalCost?: number
  discountApplied?: number
  url?: string
  code?: string
}

// 仪表盘统计
export interface DashboardStats {
  keywords: number
  users: number
  newUsersToday: number
  newUsersWeek: number
  newUsersMonth: number
  revenueTotal: number
  revenueMonth: number
  revenueToday: number
  ordersTotal: number
  ordersPaid: number
  ordersPending: number
  activeMemberships: number
  expiringMemberships: number
  checkinsToday: number
  siteName: string
  methodBreakdown: PaymentMethodStat[]
}

export interface PaymentMethodStat {
  payment_method: string
  count: number
  revenue: number
}

// 邮件配置
export interface EmailConfig {
  smtpHost: string
  smtpPort: number
  smtpUser: string
  smtpPass: string
  fromName: string
  forceVerify: boolean
  enablePasswordless: boolean
}

// 支付配置
export interface PaymentConfig {
  payment_site_url: string
  payment_enabled_providers: string[]
  payment_wx_appid: string
  payment_wx_mchid: string
  payment_wx_serial_no: string
  payment_wx_api_v3_key: string
  payment_wx_private_key: string
  payment_wx_notify_url: string
  payment_ali_appid: string
  payment_ali_private_key: string
  payment_ali_public_key: string
  payment_ali_scene: string
  payment_ali_notify_url: string
  payment_ali_return_url: string
  payment_epay_gateway: string
  payment_epay_pid: string
  payment_epay_key: string
  payment_epay_notify_url: string
  payment_epay_return_url: string
}

// 站点设置
export interface SiteSettings {
  [key: string]: string
}

// 通用响应
export interface ApiResponse<T = any> {
  data?: T
  error?: string
  message?: string
}

export interface PaginatedResponse<T> {
  list: T[]
  pagination: {
    page: number
    limit: number
    total: number
    totalPages: number
  }
}
