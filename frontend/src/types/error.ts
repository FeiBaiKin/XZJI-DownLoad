// 通用错误类型（对齐后端 apperr：{ error, code }）

export type ApiErrorCode =
  | 'bad_request'
  | 'unauthorized'
  | 'forbidden'
  | 'not_found'
  | 'conflict'
  | 'rate_limited'
  | 'internal'
  | 'license_blocked'
  | string

export interface ApiErrorBody {
  error?: string
  message?: string
  code?: ApiErrorCode
}

export interface ApiError {
  response?: {
    status?: number
    data?: ApiErrorBody
  }
  message?: string
}

/** 从错误对象中提取用户可读消息 */
export function extractErrorMessage(err: unknown, fallback = '操作失败'): string {
  const errorObj = err as ApiError
  return errorObj?.response?.data?.error
    || errorObj?.response?.data?.message
    || errorObj?.message
    || fallback
}

/** 提取稳定错误码（若有） */
export function extractErrorCode(err: unknown): ApiErrorCode | undefined {
  return (err as ApiError)?.response?.data?.code
}
