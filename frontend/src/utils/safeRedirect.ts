/**
 * 校验登录后 redirect 目标，防止开放重定向。
 * 仅允许站内相对路径：以 / 开头、非 //、非协议 URL。
 */
export function safeRedirectPath(raw: unknown, fallback = '/'): string {
  if (typeof raw !== 'string') return fallback
  const path = raw.trim()
  if (!path) return fallback
  if (!path.startsWith('/')) return fallback
  if (path.startsWith('//')) return fallback
  if (path.includes('://')) return fallback
  if (/^\/\\/.test(path)) return fallback
  return path
}
