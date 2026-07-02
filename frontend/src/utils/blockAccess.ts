// 块访问权限判定：与后端 service/content.go blockAccess 保持严格一致。
// 后端位置: backend/internal/service/content.go:450-458
// 修改任一侧时务必同步更新另一侧。
import type { KeywordBlock } from '@/types/api'

export function blockAccess(block: KeywordBlock): 'free' | 'balance' | 'membership' {
  if (block.access && block.access !== 'free') {
    return block.access
  }
  if (block.cost && block.cost > 0) {
    return 'balance'
  }
  return 'free'
}
