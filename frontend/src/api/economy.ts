// 用户经济系统 API（充值、会员、签到、交易记录等）
// 与 authApi / contentApi 同级，收拢散落在各页面的 api 直接调用
import api from '.'

// ── 类型 ──
export interface PaymentMethod { name: string; label: string }
export interface RechargeResult { type: string; scene: string; payUrl: string; qrCode: string; orderNo: string; bonusExp?: number; paymentMethod: string }
export interface RechargeStatus { orderNo: string; amount: number; status: string; points: number; paymentMethod: string; paidAt: string | null }
export interface PurchaseResult { success: boolean; url?: string; code?: string; message?: string }
export interface PurchaseStatus { purchased: boolean; url?: string; code?: string }
export interface MembershipStatus { active: boolean; level?: string; expiresAt?: string; dailyDownloads?: number; dailyLimit?: number; plans: MembershipPlan[] }
export interface MembershipPlan { levelName: string; days: number; price: number; dailyDownloadLimit: number; discount: number; color: string }
export interface Transaction { id: number; type: string; amount: number; balance_amount: number; exp_earned: number; description: string; created_at: string }
export interface CheckInStatus { checkedToday: boolean; consecutive: number; nextBalance: number; nextExp: number }
export interface CheckInResult { balance: number; exp: number; consecutive: number; isBonus: boolean }
export interface LevelInfo { level: number; label: string; exp: number; nextExp: number; progress: number }

// ── API ──
export const economyApi = {
  // profile 从 authApi 透传（仪表盘等页面同时需要经济数据和用户资料）
  profile: () => api.get<any>('/user/profile').then(r => r.data),
  paymentMethods: () => api.get<PaymentMethod[]>('/user/payment-methods').then(r => r.data),
  recharge: (amount: number, method: string) =>
    api.post<RechargeResult>('/user/recharge', { amount, method }).then(r => r.data),
  rechargeStatus: (orderNo: string) =>
    api.get<RechargeStatus>(`/user/recharge-status/${orderNo}`).then(r => r.data),
  purchaseLink: (keywordId: number, blockIndex: number, balanceCost: number, access: string) =>
    api.post<PurchaseResult>('/user/purchase-link', { keywordId, blockIndex, balanceCost, access }).then(r => r.data),
  purchaseStatus: (keywordId: number, blockIndex: number) =>
    api.get<PurchaseStatus>('/user/purchase-status', { params: { keywordId, blockIndex } }).then(r => r.data),
  membership: () => api.get<MembershipStatus>('/user/membership').then(r => r.data),
  buyMembership: (level: string) =>
    api.post<{ success: boolean }>('/user/buy-membership', { level }).then(r => r.data),
  transactions: (page?: number, limit?: number) =>
    api.get<{ list: Transaction[]; pagination: any }>('/user/transactions', { params: { page, limit } }).then(r => r.data),
  downloadsToday: () => api.get<{ count: number }>('/user/downloads-today').then(r => r.data),
  checkInStatus: () => api.get<CheckInStatus>('/user/checkin-status').then(r => r.data),
  checkIn: () => api.post<CheckInResult>('/user/checkin').then(r => r.data),
  levelInfo: () => api.get<LevelInfo>('/user/level-info').then(r => r.data),
  uploadAvatar: (formData: FormData) =>
    api.post<{ url: string }>('/user/avatar', formData, { headers: { 'Content-Type': 'multipart/form-data' } }).then(r => r.data),
}
