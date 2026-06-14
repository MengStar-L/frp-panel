<template>
  <div>
    <div class="page-head">
      <div>
        <h1>监控</h1>
        <p>{{ role === 'server' ? '服务端流量、连接与在线客户端' : '各代理的连接状态' }}</p>
      </div>
      <button class="btn btn-sm" :disabled="loading" @click="refresh"><Icon name="refresh-cw" :class="{ spin: loading }" style="width: 14px; height: 14px" /> 刷新</button>
    </div>

    <div v-if="error" class="card enter-up">
      <div class="empty">
        <div class="ico">📡</div>
        <p style="font-weight: 600; color: var(--ink)">{{ error }}</p>
        <p class="hint">请确认 frp 正在运行,且配置中已启用 webServer 管理端口。</p>
        <button class="btn btn-sm mt" @click="refresh"><Icon name="refresh-cw" style="width: 14px; height: 14px" /> 重试</button>
      </div>
    </div>

    <template v-else>
      <!-- ===== SERVER ===== -->
      <template v-if="role === 'server'">
        <div class="grid cols-4">
          <div class="card hover enter-up stat">
            <span class="label"><Icon name="arrow-down" style="width: 14px; height: 14px" /> 累计流入</span>
            <span class="value">{{ fmt(stat.in) }}</span>
          </div>
          <div class="card hover enter-up stat">
            <span class="label"><Icon name="arrow-up" style="width: 14px; height: 14px" /> 累计流出</span>
            <span class="value">{{ fmt(stat.out) }}</span>
          </div>
          <div class="card hover enter-up stat">
            <span class="label"><Icon name="link" style="width: 14px; height: 14px" /> 当前连接</span>
            <span class="value">{{ stat.conns ?? '—' }}</span>
          </div>
          <div class="card hover enter-up stat">
            <span class="label"><Icon name="users" style="width: 14px; height: 14px" /> 在线客户端</span>
            <span class="value">{{ stat.clients ?? '—' }}</span>
          </div>
        </div>

        <div class="card enter-up mt-lg">
          <div class="card-title mb"><Icon name="link" style="width: 17px; height: 17px" /> 代理 <span class="badge">{{ proxies.length }}</span></div>
          <div v-if="!proxies.length" class="empty"><div class="ico">🔌</div>暂无活动代理</div>
          <table v-else class="tbl">
            <thead><tr><th>名称</th><th>类型</th><th>状态</th><th>今日流入</th><th>今日流出</th><th>连接</th></tr></thead>
            <tbody>
              <tr v-for="p in proxies" :key="p.type + p.name">
                <td><b>{{ p.name }}</b></td>
                <td><span class="badge badge-accent" style="text-transform: uppercase">{{ p.type }}</span></td>
                <td><span class="badge" :class="p.status === 'online' ? 'badge-ok' : 'badge-danger'">{{ p.status === 'online' ? '在线' : (p.status || '离线') }}</span></td>
                <td class="mono">{{ fmt(p.in) }}</td>
                <td class="mono">{{ fmt(p.out) }}</td>
                <td class="mono">{{ p.conns ?? 0 }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>

      <!-- ===== CLIENT ===== -->
      <template v-else>
        <div class="card enter-up">
          <div class="card-title mb"><Icon name="link" style="width: 17px; height: 17px" /> 代理状态 <span class="badge">{{ items.length }}</span></div>
          <div v-if="!items.length" class="empty"><div class="ico">🔌</div>尚未配置代理,或 frp 未运行</div>
          <div v-else class="plist">
            <div v-for="p in items" :key="p.type + p.name" class="prow">
              <div class="center gap-sm">
                <span class="dot" :class="dotClass(p)"></span>
                <div>
                  <b>{{ p.name }}</b>
                  <span class="mono faint" style="font-size: 12px; display: block">{{ p.local }} <Icon name="chevron-right" style="width: 11px; height: 11px; vertical-align: -1px" /> {{ p.remote || '—' }}</span>
                </div>
              </div>
              <div class="center gap-sm">
                <span class="badge badge-accent" style="text-transform: uppercase">{{ p.type }}</span>
                <span class="badge" :class="statusBadge(p)">{{ p.status || '—' }}</span>
              </div>
            </div>
            <div v-for="p in items.filter((x) => x.err)" :key="'e' + p.name" class="prow-err">
              <Icon name="alert-triangle" style="width: 13px; height: 13px" /> <b>{{ p.name }}</b>: {{ p.err }}
            </div>
          </div>
        </div>
      </template>

      <p class="faint mt" style="font-size: 12px; text-align: center">每 5 秒自动刷新</p>
    </template>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import Icon from '../components/Icon.vue'
import { api } from '../api.js'
import { session } from '../router.js'
import { formatBytes } from '../util.js'

const role = ref(session.role)
const loading = ref(false)
const error = ref('')
const stat = reactive({ in: 0, out: 0, conns: null, clients: null })
const proxies = ref([])
const items = ref([])

const fmt = (n) => formatBytes(n || 0)
const pick = (o, ...keys) => {
  for (const k of keys) if (o && o[k] !== undefined && o[k] !== null) return o[k]
  return undefined
}

function parseServer(d) {
  const info = d.serverInfo || {}
  stat.in = pick(info, 'totalTrafficIn', 'total_traffic_in') || 0
  stat.out = pick(info, 'totalTrafficOut', 'total_traffic_out') || 0
  stat.conns = pick(info, 'curConns', 'cur_conns') ?? null
  stat.clients = pick(info, 'clientCounts', 'client_counts') ?? null
  const out = []
  for (const [type, block] of Object.entries(d.proxies || {})) {
    for (const p of block?.proxies || []) {
      out.push({
        type,
        name: p.name,
        status: pick(p, 'status'),
        in: pick(p, 'todayTrafficIn', 'today_traffic_in') || 0,
        out: pick(p, 'todayTrafficOut', 'today_traffic_out') || 0,
        conns: pick(p, 'curConns', 'cur_conns') ?? 0,
      })
    }
  }
  out.sort((a, b) => a.name.localeCompare(b.name))
  proxies.value = out
}

function parseClient(d) {
  const status = d.status || {}
  const out = []
  for (const [type, arr] of Object.entries(status)) {
    for (const p of arr || []) {
      out.push({
        type,
        name: p.name,
        status: p.status,
        err: p.err,
        local: pick(p, 'local_addr', 'localAddr') || '',
        remote: pick(p, 'remote_addr', 'remoteAddr') || '',
      })
    }
  }
  items.value = out
}

let timer
async function refresh() {
  loading.value = true
  try {
    const d = await api.monitor()
    error.value = ''
    role.value = d.role
    if (d.role === 'server') parseServer(d)
    else parseClient(d)
  } catch (e) {
    error.value = e.message || '无法获取监控数据'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  refresh()
  timer = setInterval(refresh, 5000)
})
onUnmounted(() => clearInterval(timer))

const dotClass = (p) => (p.status === 'running' ? 'on' : p.err ? 'off' : 'warn')
const statusBadge = (p) => (p.status === 'running' ? 'badge-ok' : p.err ? 'badge-danger' : 'badge-warn')
</script>

<style scoped>
.tbl {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.tbl th {
  text-align: left;
  font-size: 11.5px;
  color: var(--ink-faint);
  font-weight: 600;
  padding: 6px 10px;
  border-bottom: 1px solid var(--border);
}
.tbl td {
  padding: 11px 10px;
  border-bottom: 1px solid var(--border);
}
.tbl tbody tr {
  transition: background 0.15s var(--ease);
}
.tbl tbody tr:hover {
  background: var(--surface-2);
}
.plist {
  display: flex;
  flex-direction: column;
  gap: 9px;
}
.prow {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  border-radius: var(--radius-sm);
  background: var(--surface-2);
}
.prow b {
  font-size: 14px;
}
.prow-err {
  font-size: 12px;
  color: #b34433;
  padding: 6px 14px;
}
</style>
