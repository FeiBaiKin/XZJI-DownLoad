// 格式化工具（移植自原项目 Node-XZJI/frontend/src/utils/format.js）

// 格式化时间戳为 M/D HH:mm
export function formatTime(ts: string | number | Date | null): string {
  if (!ts) return '-'
  const d = new Date(ts)
  return `${d.getMonth() + 1}/${d.getDate()} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

// 格式化金额：>=10000 显示为 "X.X 万"
export function formatMoney(v: number | string): string {
  const n = parseFloat(v as string)
  if (n >= 10000) return (n / 10000).toFixed(1) + ' 万'
  return n.toFixed(2)
}
