<template>
  <div class="shell">
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-logo">🍦</div>
        <div class="brand-text">
          <b>frp 面板</b>
          <span>{{ roleLabel }}</span>
        </div>
      </div>

      <router-link
        v-for="n in nav"
        :key="n.name"
        :to="{ name: n.name }"
        class="nav"
        :class="{ active: route.name === n.name }"
      >
        <Icon :name="n.icon" />
        <span class="nav-label">{{ n.label }}</span>
      </router-link>

      <div class="sidebar-foot">
        <div class="center" style="padding: 8px 12px; gap: 8px">
          <span class="dot" :class="running ? 'on' : 'off'"></span>
          <span class="nav-label muted" style="font-size: 12.5px">{{ running ? 'frp 运行中' : 'frp 已停止' }}</span>
        </div>
        <button class="nav" style="width: 100%; border: 0; background: none" @click="logout">
          <Icon name="log-out" />
          <span class="nav-label">退出登录</span>
        </button>
      </div>
    </aside>

    <main class="main">
      <div class="main-inner">
        <router-view v-slot="{ Component }">
          <transition name="fade-slide" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Icon from './Icon.vue'
import { api } from '../api.js'
import { session, refreshSession } from '../router.js'

const route = useRoute()
const router = useRouter()
const running = ref(false)

const roleLabel = computed(() => (session.role === 'server' ? '服务端 · frps' : '客户端 · frpc'))

const nav = [
  { name: 'dashboard', label: '概览', icon: 'layout-grid' },
  { name: 'config', label: '配置', icon: 'sliders' },
  { name: 'monitor', label: '监控', icon: 'activity' },
  { name: 'logs', label: '日志', icon: 'terminal' },
  { name: 'updates', label: '更新', icon: 'refresh-cw' },
  { name: 'settings', label: '设置', icon: 'settings' },
]

let timer
async function poll() {
  try {
    const s = await api.frpStatus()
    running.value = !!s.proc?.running
  } catch {
    /* ignore transient errors */
  }
}
onMounted(() => {
  poll()
  timer = setInterval(poll, 5000)
})
onUnmounted(() => clearInterval(timer))

async function logout() {
  try {
    await api.logout()
  } catch {
    /* ignore */
  }
  await refreshSession()
  router.push({ name: 'login' })
}
</script>
