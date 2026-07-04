// 关键词收藏 API
import api from '.'

export interface FavoriteResult { favorited: boolean }

export const favoritesApi = {
  toggle: (keywordId: number) =>
    api.post<FavoriteResult>(`/user/favorites/${keywordId}`).then(r => r.data),
  list: (page = 1, pageSize = 8) =>
    api.get('/user/favorites', { params: { page, page_size: pageSize } }).then(r => r.data),
  ids: () =>
    api.get<number[]>('/user/favorites/ids').then(r => r.data),
}
