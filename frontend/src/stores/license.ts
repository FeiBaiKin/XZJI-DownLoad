import { defineStore } from 'pinia'
import { ref } from 'vue'
import axios from 'axios'

// 授权状态（前台停服遮罩用）：
// - 启动时查询一次 /api/license/status；
// - 业务接口返回 license_blocked 时由 axios 拦截器调用 markBlocked。
// 默认 valid=true，检查完成前不遮罩，避免正常站点闪屏。
export const useLicenseStore = defineStore('license', () => {
  const valid = ref(true)
  const checked = ref(false)
  const message = ref('')

  async function fetch() {
    try {
      const { data } = await axios.get('/api/license/status')
      valid.value = data.valid !== false
      message.value = data.message || ''
    } catch {
      // 查询失败（网络等）不遮罩，交由业务接口的 license_blocked 兜底
    } finally {
      checked.value = true
    }
  }

  function markBlocked(msg?: string) {
    valid.value = false
    if (msg) message.value = msg
    checked.value = true
  }

  return { valid, checked, message, fetch, markBlocked }
})
