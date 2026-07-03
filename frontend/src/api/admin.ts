// 管理后台 API（站点设置、用户管理、配置管理、订单管理等）
// 收拢散落在各 admin 页面的 api 直接调用
import api from '.'

// ── 类型 ──
export interface DashboardStats { totalUsers: number; totalRevenue: number; totalOrders: number; activeMembers: number; revenueByMethod: any[]; userGrowth: any[]; contentStats: any }
export interface SiteSettings { siteName: string; siteDescription: string; footerText: string; siteLogoType: string; siteLogoUrl: string; [key: string]: string }
export interface EmailConfig { email_smtp_host: string; email_smtp_port: string; email_smtp_user: string; email_smtp_pass: string; email_from_name: string; email_force_verify: string; email_enable_passwordless: string; [key: string]: string }
export interface UserRecord { id: number; username: string; role: string; email: string; nickname: string; avatar: string; balance: number; exp: number; level: number; membership_level: string | null; membership_expires_at: string | null; email_verified: boolean; created_at: string }
export interface PaymentOrderRow { id: number; order_no: string; user_id: number; amount: number; points: number; status: string; payment_method: string; payment_scene: string; trade_no: string; created_at: string; paid_at: string | null; username: string; avatar: string }
export interface CheckInConfig { daily_balance: number; daily_exp: number; consecutive_days: number; consecutive_bonus_balance: number }
export interface LevelConfigItem { level: number; exp_required: number; label: string }
export interface MembershipConfigItem { level_name: string; days: number; price: number; daily_download_limit: number; discount: number; color: string }
export interface PaymentConfig { payment_site_url: string; payment_enabled_providers: string; payment_wx_appid: string; payment_wx_mchid: string; payment_wx_serial_no: string; payment_wx_api_v3_key: string; payment_wx_private_key: string; payment_wx_notify_url: string; payment_ali_appid: string; payment_ali_private_key: string; payment_ali_public_key: string; payment_ali_scene: string; payment_ali_notify_url: string; payment_ali_return_url: string; payment_epay_gateway: string; payment_epay_pid: string; payment_epay_key: string; payment_epay_notify_url: string; payment_epay_return_url: string; [key: string]: string }
export interface LicenseUpdateInfo { version: string; changelog: string; file_size: number; sha256: string; forced: boolean }
export interface LicenseSnapshot {
  activated: boolean
  valid: boolean
  reason?: string
  status?: string
  code?: string
  domain: string
  expiresAt?: number
  lastVerifiedAt?: number
  graceDeadline?: number
  heartbeatIntervalHours?: number
  gracePeriodDays?: number
  currentVersion: string
  latestVersion?: string
  update?: LicenseUpdateInfo | null
  lastError?: string
  lastCheckAt?: number
}
export interface LicenseUpgradeResult { version: string; applied: boolean; restarting: boolean; stagingPath?: string; backupDir?: string; message: string }

// ── API ──
export const adminApi = {
  // 仪表盘
  stats: () => api.get<DashboardStats>('/admin/stats').then(r => r.data),
  // 站点设置
  settings: {
    get: () => api.get<SiteSettings>('/admin/settings').then(r => r.data),
    put: (data: Partial<SiteSettings>) => api.put('/admin/settings', data).then(r => r.data),
  },
  uploadLogo: (formData: FormData) =>
    api.post<{ url: string }>('/admin/upload-logo', formData, { headers: { 'Content-Type': 'multipart/form-data' } }).then(r => r.data),
  // 邮箱配置
  email: {
    get: () => api.get<EmailConfig>('/admin/email-config').then(r => r.data),
    put: (data: Partial<EmailConfig>) => api.put('/admin/email-config', data).then(r => r.data),
    test: () => api.post<{ message: string }>('/admin/email-test').then(r => r.data),
  },
  // 签到配置
  checkIn: {
    get: () => api.get<CheckInConfig>('/admin/checkin-config').then(r => r.data),
    put: (data: Partial<CheckInConfig>) => api.put('/admin/checkin-config', data).then(r => r.data),
  },
  // 等级配置
  level: {
    get: () => api.get<LevelConfigItem[]>('/admin/level-config').then(r => r.data),
    put: (data: LevelConfigItem[]) => api.put('/admin/level-config', data).then(r => r.data),
  },
  // 会员配置
  membership: {
    get: () => api.get<MembershipConfigItem[]>('/admin/membership-config').then(r => r.data),
    put: (data: MembershipConfigItem[]) => api.put('/admin/membership-config', data).then(r => r.data),
  },
  // 支付配置
  payment: {
    get: () => api.get<PaymentConfig>('/admin/payment-config').then(r => r.data),
    put: (data: Partial<PaymentConfig>) => api.put('/admin/payment-config', data).then(r => r.data),
  },
  // 支付订单
  orders: {
    list: (params?: any) => api.get<{ list: PaymentOrderRow[]; pagination: any }>('/admin/payment-orders', { params }).then(r => r.data),
    stats: () => api.get<{ total: number; paidCount: number; pendingCount: number; totalRevenue: number }>('/admin/payment-orders/stats').then(r => r.data),
    batchDelete: (ids: number[]) => api.post('/admin/payment-orders/batch-delete', { ids }).then(r => r.data),
    delete: (id: number) => api.delete(`/admin/payment-orders/${id}`).then(r => r.data),
  },
  // 用户管理
  users: {
    list: (params?: any) => api.get<{ list: UserRecord[]; pagination: any }>('/admin/users', { params }).then(r => r.data),
    get: (id: number) => api.get<UserRecord>(`/admin/users/${id}`).then(r => r.data),
    update: (id: number, data: Partial<UserRecord>) => api.put(`/admin/users/${id}`, data).then(r => r.data),
    delete: (id: number) => api.delete(`/admin/users/${id}`).then(r => r.data),
  },
  // 会员提醒手动触发
  triggerMembershipReminders: () => api.post('/admin/trigger-membership-reminders').then(r => r.data),
  // 授权管理（授权升级中心）
  license: {
    get: () => api.get<LicenseSnapshot>('/admin/license').then(r => r.data),
    activate: (code: string) =>
      api.post<{ message: string; status: string; snapshot: LicenseSnapshot }>('/admin/license/activate', { code }).then(r => r.data),
    heartbeat: () =>
      api.post<{ message: string; snapshot: LicenseSnapshot }>('/admin/license/heartbeat').then(r => r.data),
    // 升级含下载耗时，单独放宽超时
    upgrade: () =>
      api.post<LicenseUpgradeResult>('/admin/license/upgrade', {}, { timeout: 300000 }).then(r => r.data),
  },
}
