import api from '.'
import type {
  Keyword, KeywordCreate, KeywordUpdate,
  Category, CategoryCreate, CategoryUpdate,
  Page, PageCreate, PageUpdate,
  Announcement, AnnouncementCreate, AnnouncementUpdate,
  PublicKeyword, PaginatedKeywords, PublicSiteSettings
} from '@/types/api'

export type { Keyword, Category, Page, Announcement, PublicKeyword, PaginatedKeywords, PublicSiteSettings }

export const contentApi = {
  // 关键词
  keywords: {
    list: () => api.get<Keyword[]>('/admin/keywords').then(r => r.data),
    get: (id: number) => api.get<Keyword>(`/admin/keywords/${id}`).then(r => r.data),
    create: (data: KeywordCreate) => api.post<Keyword>('/admin/keywords', data).then(r => r.data),
    update: (id: number, data: KeywordUpdate) => api.put<Keyword>(`/admin/keywords/${id}`, data).then(r => r.data),
    delete: (id: number) => api.delete(`/admin/keywords/${id}`).then(r => r.data),
    categories: (id: number) => api.get<number[]>(`/admin/keywords/${id}/categories`).then(r => r.data),
    setCategories: (id: number, ids: number[]) => api.put(`/admin/keywords/${id}/categories`, ids).then(r => r.data),
  },
  // 分类
  categories: {
    list: () => api.get<Category[]>('/admin/categories').then(r => r.data),
    create: (data: CategoryCreate) => api.post<Category>('/admin/categories', data).then(r => r.data),
    update: (id: number, data: CategoryUpdate) => api.put<Category>(`/admin/categories/${id}`, data).then(r => r.data),
    delete: (id: number) => api.delete(`/admin/categories/${id}`).then(r => r.data),
  },
  // 页面
  pages: {
    list: () => api.get<Page[]>('/admin/pages').then(r => r.data),
    create: (data: PageCreate) => api.post<Page>('/admin/pages', data).then(r => r.data),
    update: (id: number, data: PageUpdate) => api.put<Page>(`/admin/pages/${id}`, data).then(r => r.data),
    delete: (id: number) => api.delete(`/admin/pages/${id}`).then(r => r.data),
  },
  // 公告
  announcements: {
    list: () => api.get<Announcement[]>('/admin/announcements').then(r => r.data),
    create: (data: AnnouncementCreate) => api.post<Announcement>('/admin/announcements', data).then(r => r.data),
    update: (id: number, data: AnnouncementUpdate) => api.put<Announcement>(`/admin/announcements/${id}`, data).then(r => r.data),
    delete: (id: number) => api.delete(`/admin/announcements/${id}`).then(r => r.data),
  },
  // 公开
  public: {
    search: (keyword: string) => api.post('/search', { keyword }).then(r => r.data),
    page: (slug: string) => api.get(`/page/${slug}`).then(r => r.data),
    pages: () => api.get('/pages').then(r => r.data),
    keywords: (category?: number, page = 1, pageSize = 12) =>
      api.get<PaginatedKeywords>('/keywords', {
        params: { ...(category ? { category } : {}), page, page_size: pageSize },
      }).then(r => r.data),
    featured: (perCategory = 4, page = 1, pageSize = 12) =>
      api.get<PaginatedKeywords>('/keywords/featured', {
        params: { per_category: perCategory, page, page_size: pageSize },
      }).then(r => r.data),
    announcements: () => api.get('/announcements/active').then(r => r.data),
    categories: () => api.get<Category[]>('/categories').then(r => r.data),
    settings: () => api.get<PublicSiteSettings>('/settings').then(r => r.data),
    siteConfig: () => api.get('/site/config').then(r => r.data),
  },
}
