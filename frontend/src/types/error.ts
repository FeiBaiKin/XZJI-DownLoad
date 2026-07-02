// 通用错误类型

export interface ApiError {
  response?: {
    data?: {
      error?: string
      message?: string
    }
  }
  message?: string
}

// 辅助函数：从错误对象中提取错误消息
export function extractErrorMessage(err: unknown, fallback = '操作失败'): string {
  const errorObj = err as ApiError
  return errorObj?.response?.data?.error || errorObj?.message || fallback
}
