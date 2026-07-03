<script setup lang="ts">
// 管理后台布局：1:1 移植自原项目 Node-XZJI/frontend/src/pages/AdminPage.jsx (AdminLayout)
// 样式来自全局 styles/admin-layout.css
// 裁剪：授权门禁/极验（按重构方案暂不实现）；授权管理/系统升级已接入授权升级中心（/admin/license）
import { reactive, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const theme = useThemeStore()

function isActive(path: string) {
  return route.path === path
}
function startsWith(prefix: string) {
  return route.path.startsWith(prefix)
}

// 折叠分组：按当前路由自动展开所属组
const openGroups = reactive<Record<string, boolean>>({
  content: startsWith('/admin/keywords') || startsWith('/admin/categories') || startsWith('/admin/pages') || startsWith('/admin/announcements'),
  users: startsWith('/admin/checkin') || startsWith('/admin/levels') || startsWith('/admin/membership') || startsWith('/admin/users'),
  payment: startsWith('/admin/payment') || startsWith('/admin/orders'),
  system: startsWith('/admin/settings') || startsWith('/admin/email') || startsWith('/admin/license'),
})
function toggleGroup(group: string) {
  openGroups[group] = !openGroups[group]
}

const isAdmin = computed(() => auth.isAdmin)

function handleLogout() {
  auth.logout()
  router.push('/admin/login')
}
</script>

<template>
  <!-- 非管理员拦截 -->
  <div v-if="auth.user && !isAdmin" class="forbidden-page">
    <div class="forbidden-card">
      <div class="forbidden-icon" aria-hidden="true">
        <svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
          <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
          <path d="M7 11V7a5 5 0 0110 0v4" />
        </svg>
      </div>
      <h1 class="forbidden-title">无权访问</h1>
      <p class="forbidden-desc">
        你的账户 <strong>{{ auth.user?.username || '未知' }}</strong> 没有管理员权限，无法访问管理后台。
      </p>
      <div class="forbidden-actions">
        <router-link to="/" class="action-button">返回首页</router-link>
        <button class="action-button secondary" @click="handleLogout">退出登录</button>
      </div>
    </div>
  </div>

  <div v-else class="admin-layout">
    <aside class="admin-sidebar">
      <div class="sidebar-header">
        <div class="sidebar-brand">
          <router-link to="/" class="sidebar-brand-link" title="返回首页">
            <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M1 7l6-5.5L13 7" /><path d="M3 5.5v6.5h3V8h2v4h3V5.5" /></svg>
          </router-link>
          <span class="sidebar-brand-text">管理后台</span>
        </div>
      </div>

      <nav class="sidebar-nav">
        <router-link to="/admin" class="nav-item" :class="{ active: isActive('/admin') }">仪表盘</router-link>

        <!-- 内容管理 -->
        <div class="nav-group">
          <div class="nav-group-header" :class="{ open: openGroups.content }" @click="toggleGroup('content')">
            <span class="nav-group-caret">&#9654;</span> 内容管理
          </div>
          <div class="nav-group-items" :class="{ open: openGroups.content }">
            <router-link to="/admin/keywords" class="nav-sub-item" :class="{ active: isActive('/admin/keywords') }">关键词管理</router-link>
            <router-link to="/admin/categories" class="nav-sub-item" :class="{ active: isActive('/admin/categories') }">分类管理</router-link>
            <router-link to="/admin/pages" class="nav-sub-item" :class="{ active: isActive('/admin/pages') }">页面管理</router-link>
            <router-link to="/admin/announcements" class="nav-sub-item" :class="{ active: isActive('/admin/announcements') }">公告管理</router-link>
          </div>
        </div>

        <!-- 用户系统 -->
        <div class="nav-group">
          <div class="nav-group-header" :class="{ open: openGroups.users }" @click="toggleGroup('users')">
            <span class="nav-group-caret">&#9654;</span> 用户系统
          </div>
          <div class="nav-group-items" :class="{ open: openGroups.users }">
            <router-link to="/admin/users" class="nav-sub-item" :class="{ active: isActive('/admin/users') }">用户管理</router-link>
            <router-link to="/admin/checkin" class="nav-sub-item" :class="{ active: isActive('/admin/checkin') }">签到配置</router-link>
            <router-link to="/admin/levels" class="nav-sub-item" :class="{ active: isActive('/admin/levels') }">等级配置</router-link>
            <router-link to="/admin/membership" class="nav-sub-item" :class="{ active: isActive('/admin/membership') }">会员配置</router-link>
          </div>
        </div>

        <!-- 支付系统 -->
        <div class="nav-group">
          <div class="nav-group-header" :class="{ open: openGroups.payment }" @click="toggleGroup('payment')">
            <span class="nav-group-caret">&#9654;</span> 支付系统
          </div>
          <div class="nav-group-items" :class="{ open: openGroups.payment }">
            <router-link to="/admin/payment-config" class="nav-sub-item" :class="{ active: isActive('/admin/payment-config') }">支付配置</router-link>
            <router-link to="/admin/orders" class="nav-sub-item" :class="{ active: isActive('/admin/orders') }">支付订单</router-link>
          </div>
        </div>

        <!-- 系统设置 -->
        <div class="nav-group">
          <div class="nav-group-header" :class="{ open: openGroups.system }" @click="toggleGroup('system')">
            <span class="nav-group-caret">&#9654;</span> 系统设置
          </div>
          <div class="nav-group-items" :class="{ open: openGroups.system }">
            <router-link to="/admin/settings" class="nav-sub-item" :class="{ active: isActive('/admin/settings') }">网站设置</router-link>
            <router-link to="/admin/email" class="nav-sub-item" :class="{ active: isActive('/admin/email') }">邮箱配置</router-link>
            <router-link to="/admin/license" class="nav-sub-item" :class="{ active: isActive('/admin/license') }">授权管理</router-link>
          </div>
        </div>
      </nav>

      <div class="sidebar-footer">
        <div class="user-info">
          <span class="user-name">{{ auth.user?.username || 'Admin' }}</span>
        </div>
        <button class="theme-toggle-btn" @click="theme.toggle()" :title="theme.isDark ? '切换浅色模式' : '切换深色模式'">
          <svg v-if="theme.isDark" width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="7" cy="7" r="3" /><path d="M7 1v1.5M7 11.5V13M13 7h-1.5M2.5 7H1M11.2 2.8l-1 1M3.8 10.2l-1 1M11.2 11.2l-1-1M3.8 3.8l-1-1" /></svg>
          <svg v-else width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M7 1.5A5.5 5.5 0 1012.5 7 3.5 3.5 0 017 1.5z" /></svg>
        </button>
        <button class="logout-button" @click="handleLogout">退出登录</button>
      </div>
    </aside>

    <main class="admin-main">
      <RouterView />
    </main>
  </div>
</template>
