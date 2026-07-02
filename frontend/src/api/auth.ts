import api from '.'

export interface UserBrief {
  id: number
  username: string
  role: string
  email?: string
  nickname: string
  avatar: string
  balance: number
  exp: number
  level: number
  membership_level: string | null
  membership_expires_at: string | null
}

export interface LoginResult {
  message: string
  token: string
  user: UserBrief
}

export const authApi = {
  login: (username: string, password: string) =>
    api.post<LoginResult>('/auth/login', { username, password }).then((r) => r.data),

  register: (data: { username: string; password: string; email?: string; code?: string }) =>
    api.post<LoginResult>('/auth/register', data).then((r) => r.data),

  sendCode: (email: string, purpose: 'register' | 'forgot-password' | 'login') =>
    api.post<{ message: string; expiresIn: number }>('/auth/send-code', { email, purpose }).then((r) => r.data),

  verifyCode: (email: string, code: string, purpose: string) =>
    api.post<{ verified: boolean }>('/auth/verify-code', { email, code, purpose }).then((r) => r.data),

  passwordlessLogin: (email: string, code: string) =>
    api.post<LoginResult>('/auth/passwordless-login', { email, code }).then((r) => r.data),

  resetPassword: (email: string, code: string, password: string) =>
    api.post<{ message: string }>('/auth/reset-password', { email, code, password }).then((r) => r.data),

  profile: () => api.get<UserBrief>('/user/profile').then((r) => r.data),

  updateProfile: (data: { nickname?: string; email?: string }) =>
    api.put<UserBrief>('/user/profile', data).then((r) => r.data),

  changePassword: (currentPassword: string, newPassword: string) =>
    api.put<{ message: string }>('/user/password', { currentPassword, newPassword }).then((r) => r.data),

  logout: () => api.post<{ message: string }>('/auth/logout').then((r) => r.data),
}
