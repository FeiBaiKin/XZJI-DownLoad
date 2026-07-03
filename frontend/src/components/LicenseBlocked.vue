<script setup lang="ts">
// 授权停服全页遮罩：授权无效时替代前台所有页面显示（接入文档约定文案），
// 管理后台 /admin 与 /login 不受影响，便于管理员登录处理授权。
import { useLicenseStore } from '@/stores/license'

const license = useLicenseStore()
</script>

<template>
  <div class="license-blocked-overlay">
    <div class="license-blocked-card">
      <div class="license-blocked-icon" aria-hidden="true">
        <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
          <rect x="3" y="11" width="18" height="11" rx="2" ry="2" />
          <path d="M7 11V7a5 5 0 0110 0v4" />
        </svg>
      </div>
      <h1 class="license-blocked-title">{{ license.message || '授权已过期，请联系服务商' }}</h1>
      <p class="license-blocked-desc">本站点的软件授权已失效，前台服务暂时停止。</p>
      <p class="license-blocked-footer">
        站点管理员？<router-link to="/admin/login">登录管理后台</router-link>查看授权状态并重新激活。
      </p>
    </div>
  </div>
</template>
