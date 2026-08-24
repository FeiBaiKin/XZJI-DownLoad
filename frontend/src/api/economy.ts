// 用户经济系统 API（充值、会员、签到、交易记录等）
// 与 authApi / contentApi 同级，收拢散落在各页面的 api 直接调用
import api from '.'

// ── 类型 ──
export interface PaymentMethod { name: string; label: string }
export interface RechargeResult { type: string; scene: string; payUrl: string; qrCode: string; orderNo: string; bonusExp?: number; paymentMethod: string }
/** 订单类型：recharge 充值入余额 · membership 会员直付（付款回调直接开通会员） */
export type PaymentOrderType = 'recharge' | 'membership'
export interface RechargeStatus { orderNo: string; amount: number; status: string; points: number; paymentMethod: string; paidAt: string | null; orderType: PaymentOrderType; membershipLevel: string }
export interface PurchaseResult { message: string; alreadyPurchased?: boolean; membership?: boolean; balance?: number; balanceSpent?: number; originalCost?: number; discountApplied?: number; url?: string; code?: string }
export interface PurchaseStatus { purchased: boolean; url?: string; code?: string }
export interface MembershipBrief { level: string; expiresAt: string }
export interface MembershipStatus { membership: MembershipBrief | null; downloads: number; dailyLimit: number; configs: MembershipPlan[] }
export interface MembershipPlan { level_name: string; days: number; price: number; daily_download_limit: number; discount: number; color: string; badge?: string }
/** 余额开通的返回 */
export interface MembershipBuyResult { message: string; balance: number; membershipLevel: string | null; membershipExpiresAt: string | null }
/** 第三方支付开通的返回：结构与充值下单一致，前端复用扫码 / 跳转 / 轮询流程 */
export interface MembershipPayResult extends RechargeResult { pay: true; amount: number; membershipLevel: string }
export interface Transaction { id: number; type: string; amount: number; balance_amount: number; exp_earned: number; description: string; created_at: string }
export interface TransactionsResult { transactions: Transaction[]; total: number; page: number; limit: number }
export interface CheckInStatus { checkedIn: boolean; streak: number; todayBalance: number; todayExp: number; isBonus: boolean; config: { dailyBalance: number; consecutiveDays: number; consecutiveBonusBalance: number; dailyExp: number }; tomorrowBalance: number }
export interface CheckInResult { message: string; balanceEarned: number; expEarned: number; totalBalance: number; totalExp: number; level: number; streak: number; isBonus: boolean; leveledUp: boolean; newLevel?: number; levelLabel?: string }
export interface LevelInfo { level: number; label: string; exp: number; expInCurrentLevel: number; expNeeded: number; progress: number; nextLevel: number; nextLabel: string }

// ── API ──
export const economyApi = {
  // profile 从 authApi 透传（仪表盘等页面同时需要经济数据和用户资料）
  profile: () => api.get<any>('/user/profile').then(r => r.data),
  paymentMethods: () => api.get<PaymentMethod[]>('/user/payment-methods').then(r => r.data),
  recharge: (amount: number, method: string) =>
    api.post<RechargeResult>('/user/recharge', { amount, method }).then(r => r.data),
  rechargeStatus: (orderNo: string) =>
    api.get<RechargeStatus>(`/user/recharge-status/${orderNo}`).then(r => r.data),
  purchaseLink: (keywordId: number, blockIndex: number) =>
    api.post<PurchaseResult>('/user/purchase-link', { keywordId, blockIndex }).then(r => r.data),
  purchaseStatus: (keywordId: number, blockIndex: number) =>
    api.get<PurchaseStatus>('/user/purchase-status', { params: { keywordId, blockIndex } }).then(r => r.data),
  membership: () => api.get<MembershipStatus>('/user/membership').then(r => r.data),
  // method 省略 / 'balance' 走余额扣费；传后台已开启的支付方式则返回支付下单结果
  buyMembership: (level: string, method?: string) =>
    api.post<MembershipBuyResult | MembershipPayResult>('/user/buy-membership', { level, method }).then(r => r.data),
  transactions: (page?: number, limit?: number) =>
    api.get<TransactionsResult>('/user/transactions', { params: { page, limit } }).then(r => r.data),
  downloadsToday: () => api.get<{ downloads: number; date: string }>('/user/downloads-today').then(r => r.data),
  checkInStatus: () => api.get<CheckInStatus>('/user/checkin-status').then(r => r.data),
  checkIn: () => api.post<CheckInResult>('/user/checkin').then(r => r.data),
  levelInfo: () => api.get<LevelInfo>('/user/level-info').then(r => r.data),
  uploadAvatar: (formData: FormData) =>
    api.post<{ url: string }>('/user/avatar', formData, { headers: { 'Content-Type': 'multipart/form-data' } }).then(r => r.data),
}
