// 收藏状态管理：登录后 hydrate 收藏 ID 集合，供卡片组件判断已收藏/未收藏
import { ref } from 'vue'
import { defineStore } from 'pinia'
import { favoritesApi } from '@/api/favorite'
import { useAuthStore } from '@/stores/auth'
import { useToastStore } from '@/stores/toast'

export const useFavoritesStore = defineStore('favorites', () => {
  const favoritedIds = ref(new Set<number>())
  const loaded = ref(false)

  async function fetchIds() {
    const auth = useAuthStore()
    if (!auth.isLoggedIn) { favoritedIds.value = new Set(); loaded.value = true; return }
    try {
      const ids: number[] = await favoritesApi.ids()
      favoritedIds.value = new Set(ids)
    } catch { /* ignore */ }
    loaded.value = true
  }

  function has(keywordId: number): boolean {
    return favoritedIds.value.has(keywordId)
  }

  async function toggle(keywordId: number): Promise<boolean> {
    const auth = useAuthStore()
    const toast = useToastStore()
    if (!auth.isLoggedIn) {
      toast.info('请先登录后再收藏')
      return has(keywordId)
    }
    try {
      const res = await favoritesApi.toggle(keywordId)
      const nowFav = res.favorited === true
      if (nowFav) {
        favoritedIds.value.add(keywordId)
        toast.success('已收藏')
      } else {
        favoritedIds.value.delete(keywordId)
        toast.success('已取消收藏')
      }
      return nowFav
    } catch {
      toast.error('操作失败')
      return has(keywordId)
    }
  }

  function clear() {
    favoritedIds.value = new Set()
    loaded.value = false
  }

  return { favoritedIds, loaded, fetchIds, has, toggle, clear }
})
