<template>
  <div>
    <div class="page-head">
      <div>
        <h1>概览</h1>
        <p>frp {{ roleLabel }}运行状态与快捷控制</p>
      </div>
      <span class="badge badge-accent">
        <Icon :name="role === 'server' ? 'server' : 'laptop'" style="width: 14px; height: 14px" />
        {{ roleLabel }}
      </span>
    </div>

    <!-- hero status -->
    <div class="card hero enter-up">
      <div class="hero-left">
        <div class="status-ring" :class="running ? 'on' : 'off'">
          <span class="dot" :class="running ? 'on' : 'off'" style="width: 14px; height: 14px"></span>
        </div>
        <div>
          <div class="status-text">{{ running ? '运行中' : '已停止' }}</div>
          <div class="muted" style="font-size: 13px">
            <template v-if="running">PID {{ proc.pid }} · 已运行 {{ uptime }}</template>
            <template v-else-if="proc.lastError">上次异常退出</template>
            <template v-else>frp 进程未运行</template>
          </div>
        </div>
      </div>
      <div class="hero-actions">
        <button v-if="!running" class="btn btn-primary" :disabled="!!busy" @click="act('start')">
          <Icon :name="busy === 'start' ? 'loader' : 'play'" :class="{ spin: busy === 'start' }" /> 启动
        </button>
        <template v-else>
          <button class="btn btn-danger" :disabled="!!busy" @click="act('stop')">
            <Icon :name="busy === 'stop' ? 'loader' : 'stop'" :class="{ spin: busy === 'stop' }" /> 停止
          </button>
          <button class="btn" :disabled="!!busy" @click="act('restart')">
            <Icon :name="busy === 'restart' ? 'loader' : 'rotate-cw'" :class="{ spin: busy === 'restart' }" /> 重启
          </button>
          <button v-if="role === 'client'" class="btn" :disabled="!!busy" @click="act('reload')">
            <Icon :name="busy === 'reload' ? 'loader' : 'refresh-cw'" :class="{ spin: busy === 'reload' }" /> 热重载
          </button>
        </template>
      </div>
    </div>

    <div v-if="proc.lastError" class="card enter-up" style="border-color: var(--danger-soft); background: var(--danger-soft); margin-top: 18px">
      <div class="center gap-sm" style="color: #b34433; font-weight: 600">
        <Icon name="alert-triangle" style="width: 17px; height: 17px" /> 最近一次错误
      </div>
      <p class="mono" style="margin: 8px 0 0; font-size: 12.5px; color: #9a4234; word-break: break-all">{{ proc.lastError }}</p>
    </div>

    <!-- stat grid -->
    <div class="grid cols-3" style="margin-top: 18px">
      <div class="card hover enter-up stat">
        <span class="label"><Icon name="zap" style="width: 14px; height: 14px" /> frp 版本</span>
        <span class="value">{{ frpVersion ? 'v' + frpVersion : '—' }}</span>
      </div>
      <div class="card hover enter-up stat">
        <span class="label"><Icon name="clock" style="width: 14px; height: 14px" /> 运行时长</span>
        <span class="value">{{ running ? uptime : '—' }}</span>
      </div>
      <div class="card hover enter-up stat">
        <span class="label"><Icon name="play" style="width: 14px; height: 14px" /> 开机自启</span>
        <span class="value" style="font-size: 18px">
          <span class="badge" :class="autoStart ? 'badge-ok' : ''">{{ autoStart ? '已开启' : '已关闭' }}</span>
        </span>
      </div>
    </div>

    <!-- quick links -->
    <div class="grid cols-2" style="margin-top: 18px">
      <router-link :to="{ name: 'config' }" class="card hover enter-up quick">
        <Icon name="sliders" class="quick-ico" />
        <div><b>编辑配置</b><span>调整端口、认证与代理</span></div>
        <Icon name="chevron-right" class="quick-arrow" />
      </router-link>
      <router-link :to="{ name: 'monitor' }" class="card hover enter-up quick">
        <Icon name="activity" class="quick-ico" />
        <div><b>实时监控</b><span>{{ role === 'server' ? '流量 · 连接 · 客户端' : '各代理连接状态' }}</span></div>
        <Icon name="chevron-right" class="quick-arrow" />
      </router-link>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import Icon from '../components/Icon.vue'
import { api } from '../api.js'
import { toast } from '../composables/useToast.js'
import { formatDuration } from '../util.js'

const proc = reactive({ running: false, pid: 0, uptimeSec: 0, lastError: '', exitCode: null })
const role = ref('')
const frpVersion = ref('')
const autoStart = ref(false)
const busy = ref('')

const running = computed(() => proc.running)
const roleLabel = computed(() => (role.value === 'server' ? '服务端' : '客户端'))
const uptime = computed(() => formatDuration(proc.uptimeSec))

function apply(s) {
  Object.assign(proc, { running: false, pid: 0, uptimeSec: 0, lastError: '', exitCode: null }, s.proc || {})
  role.value = s.role
  frpVersion.value = s.frpVersion
  autoStart.value = s.autoStart
}

let timer
async function refresh() {
  try {
    apply(await api.frpStatus())
  } catch {
    /* ignore */
  }
}
onMounted(() => {
  refresh()
  timer = setInterval(refresh, 3000)
})
onUnmounted(() => clearInterval(timer))

const labels = { start: '启动', stop: '停止', restart: '重启', reload: '热重载' }
async function act(kind) {
  busy.value = kind
  try {
    if (kind === 'start') apply(await api.frpStart())
    else if (kind === 'stop') apply(await api.frpStop())
    else if (kind === 'restart') apply(await api.frpRestart())
    else if (kind === 'reload') {
      await api.frpReload()
      await refresh()
    }
    toast.ok(labels[kind] + '成功')
  } catch (e) {
    toast.err(labels[kind] + '失败', e.message)
  } finally {
    busy.value = ''
  }
}
</script>

<style scoped>
.hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  flex-wrap: wrap;
}
.hero-left {
  display: flex;
  align-items: center;
  gap: 18px;
}
.status-ring {
  width: 58px;
  height: 58px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--surface-2);
  transition: background 0.3s var(--ease);
}
.status-ring.on {
  background: var(--ok-soft);
}
.status-ring.off {
  background: var(--danger-soft);
}
.status-text {
  font-size: 22px;
  font-weight: 700;
  letter-spacing: -0.01em;
}
.hero-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}
.quick {
  display: flex;
  align-items: center;
  gap: 14px;
  text-decoration: none;
  color: inherit;
}
.quick-ico {
  width: 24px;
  height: 24px;
  color: var(--accent-ink);
  flex-shrink: 0;
}
.quick b {
  display: block;
  font-size: 14.5px;
}
.quick span {
  font-size: 12.5px;
  color: var(--ink-soft);
}
.quick-arrow {
  width: 18px;
  height: 18px;
  color: var(--ink-faint);
  margin-left: auto;
  transition: transform 0.2s var(--ease);
}
.quick:hover .quick-arrow {
  transform: translateX(3px);
  color: var(--accent-ink);
}
</style>
