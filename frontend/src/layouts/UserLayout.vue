<script setup lang="ts">
// 用户中心布局：1:1 移植自原项目 Node-XZJI/frontend/src/pages/UserCenter.jsx (UserLayout)
// 样式来自全局 styles/user.css
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

function isActive(path: string) {
  return route.path === path
}

function handleLogout() {
  auth.logout()
  router.push('/login')
}
</script>

<template>
  <div class="user-layout">
    <aside class="user-sidebar">
      <div class="sidebar-header">
        <router-link to="/" class="back-home-link">
          <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M2 6l6-4.667L14 6v7.333a1.333 1.333 0 01-1.333 1.334H3.333A1.333 1.333 0 012 13.333V6z" />
            <path d="M6 14.667V8h4v6.667" />
          </svg>
          <span>返回首页</span>
        </router-link>
      </div>

      <nav class="sidebar-nav">
        <router-link to="/user" class="nav-item" :class="{ active: isActive('/user') }">个人中心</router-link>
        <router-link to="/user/profile" class="nav-item" :class="{ active: isActive('/user/profile') }">个人资料</router-link>
        <router-link to="/user/membership" class="nav-item" :class="{ active: isActive('/user/membership') }">会员中心</router-link>
        <router-link to="/user/security" class="nav-item" :class="{ active: isActive('/user/security') }">安全设置</router-link>
      </nav>

      <div class="sidebar-footer">
        <div class="user-info">
          <span class="user-name">{{ auth.user?.nickname || auth.user?.username || '用户' }}</span>
          <span class="user-role">{{ auth.user?.role === 'admin' ? '管理员' : '普通用户' }}</span>
        </div>
        <button class="logout-button" @click="handleLogout">退出登录</button>
      </div>
    </aside>

    <main class="user-main">
      <RouterView />
    </main>
  </div>
</template>
