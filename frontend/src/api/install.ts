import api from '.'

export interface InstallStatus {
  installed: boolean
}

export interface InstallRequest {
  siteName: string
  siteDescription?: string
  siteURL: string
  footerText?: string
  adminUsername: string
  adminPassword: string
  adminEmail?: string
  dbHost: string
  dbPort: number
  dbUser: string
  dbPassword: string
  dbName: string
}

export interface InstallResponse {
  success: boolean
  message: string
  adminUrl: string
}

export const installApi = {
  status: () => api.get<InstallStatus>('/install/status').then((r) => r.data),
  install: (data: InstallRequest) => api.post<InstallResponse>('/install', data).then((r) => r.data),
}
