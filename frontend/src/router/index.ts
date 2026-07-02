import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

function requireAuth() {
  const auth = useAuthStore()
  if (!auth.isLoggedIn) return '/login?redirect=' + encodeURIComponent(location.pathname)
  return true
}

function requireAdmin() {
  const auth = useAuthStore()
  if (!auth.isLoggedIn) return '/login?redirect=' + encodeURIComponent(location.pathname)
  if (!auth.isAdmin) return '/user'
  return true
}

const routes: RouteRecordRaw[] = [
  { path: '/', name: 'home', component: () => import('@/views/public/HomePage.vue'), meta: { public: true } },
  { path: '/install', name: 'install', component: () => import('@/views/public/InstallView.vue'), meta: { public: true } },
  { path: '/login', name: 'login', component: () => import('@/views/public/LoginPage.vue'), meta: { public: true } },
  { path: '/register', name: 'register', component: () => import('@/views/public/RegisterPage.vue'), meta: { public: true } },
  { path: '/forgot-password', name: 'forgot', component: () => import('@/views/public/ForgotPasswordPage.vue'), meta: { public: true } },
  { path: '/page/:slug', name: 'page', component: () => import('@/views/public/PageView.vue'), meta: { public: true } },
  {
    path: '/user',
    component: () => import('@/layouts/UserLayout.vue'),
    beforeEnter: (_to, _from) => requireAuth(),
    children: [
      { path: '', name: 'dashboard', component: () => import('@/views/user/UserDashboard.vue') },
      { path: 'profile', name: 'profile', component: () => import('@/views/user/ProfilePage.vue') },
      { path: 'membership', name: 'membership', component: () => import('@/views/user/MembershipPage.vue') },
      { path: 'security', name: 'security', component: () => import('@/views/user/PasswordPage.vue') },
    ],
  },
  {
    path: '/admin',
    component: () => import('@/layouts/AdminLayout.vue'),
    beforeEnter: (_to, _from) => requireAdmin(),
    children: [
      { path: '', name: 'admin-dashboard', component: () => import('@/views/admin/DashboardPage.vue') },
      { path: 'keywords', name: 'admin-keywords', component: () => import('@/views/admin/KeywordManager.vue') },
      { path: 'categories', name: 'admin-categories', component: () => import('@/views/admin/CategoryManager.vue') },
      { path: 'pages', name: 'admin-pages', component: () => import('@/views/admin/PageManager.vue') },
      { path: 'announcements', name: 'admin-announcements', component: () => import('@/views/admin/AnnouncementManager.vue') },
      { path: 'settings', name: 'admin-settings', component: () => import('@/views/admin/SettingsPage.vue') },
      { path: 'email', name: 'admin-email', component: () => import('@/views/admin/EmailConfigPage.vue') },
      { path: 'users', name: 'admin-users', component: () => import('@/views/admin/UserManagerPage.vue') },
      { path: 'checkin', name: 'admin-checkin', component: () => import('@/views/admin/CheckinConfigPage.vue') },
      { path: 'levels', name: 'admin-level', component: () => import('@/views/admin/LevelConfigPage.vue') },
      { path: 'membership', name: 'admin-membership', component: () => import('@/views/admin/MembershipConfigPage.vue') },
      { path: 'payment-config', name: 'admin-payment', component: () => import('@/views/admin/PaymentConfigPage.vue') },
      { path: 'orders', name: 'admin-orders', component: () => import('@/views/admin/PaymentOrdersPage.vue') },
    ],
  },
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

const router = createRouter({ history: createWebHistory(), routes })

router.beforeEach(async (to, _from) => {
  const auth = useAuthStore()
  // 首次进入时尝试从 HttpOnly Cookie 恢复登录态（刷新后 user 为空但 cookie 仍有效）
  if (!auth.initialized) {
    await auth.fetchProfile()
  }
  if (to.meta.public) return true
  if (!auth.isLoggedIn) return '/login?redirect=' + encodeURIComponent(to.fullPath)
  return true
})

export default router
