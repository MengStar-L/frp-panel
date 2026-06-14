import { createRouter, createWebHistory } from 'vue-router'
import { reactive } from 'vue'
import { api } from './api.js'

// Shared, reactive bootstrap state. Refreshed from /api/state on navigation and
// explicitly after login / setup / logout.
export const session = reactive({
  configured: false,
  authenticated: false,
  role: '',
  frpVersion: '',
  panelVersion: '',
  loaded: false,
})

export async function refreshSession() {
  try {
    const s = await api.state()
    Object.assign(session, s, { loaded: true })
  } catch {
    session.loaded = true
  }
  return session
}

const routes = [
  { path: '/setup', name: 'setup', component: () => import('./views/Setup.vue'), meta: { public: true } },
  { path: '/login', name: 'login', component: () => import('./views/Login.vue'), meta: { public: true } },
  {
    path: '/',
    component: () => import('./components/AppShell.vue'),
    children: [
      { path: '', redirect: { name: 'dashboard' } },
      { path: 'dashboard', name: 'dashboard', component: () => import('./views/Dashboard.vue') },
      { path: 'config', name: 'config', component: () => import('./views/ConfigView.vue') },
      { path: 'monitor', name: 'monitor', component: () => import('./views/MonitorView.vue') },
      { path: 'logs', name: 'logs', component: () => import('./views/LogsView.vue') },
      { path: 'updates', name: 'updates', component: () => import('./views/UpdatesView.vue') },
      { path: 'settings', name: 'settings', component: () => import('./views/SettingsView.vue') },
    ],
  },
  { path: '/:pathMatch(.*)*', redirect: { name: 'dashboard' } },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior: () => ({ top: 0 }),
})

router.beforeEach(async (to) => {
  await refreshSession()

  if (!session.configured) {
    return to.name === 'setup' ? true : { name: 'setup' }
  }
  // Configured: setup is no longer reachable.
  if (to.name === 'setup') return { name: session.authenticated ? 'dashboard' : 'login' }

  if (!session.authenticated) {
    return to.name === 'login' ? true : { name: 'login', query: { redirect: to.fullPath } }
  }
  // Authenticated: bounce away from login.
  if (to.name === 'login') return { name: 'dashboard' }
  return true
})

export default router
