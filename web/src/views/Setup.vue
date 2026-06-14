<template>
  <div class="setup-wrap">
    <div class="setup-card card pop">
      <div class="setup-brand">
        <div class="brand-logo" style="width: 46px; height: 46px; font-size: 24px">🍦</div>
        <div>
          <h1 style="font-size: 21px">欢迎使用 frp 管理面板</h1>
          <p class="muted" style="margin: 2px 0 0; font-size: 13px">几步即可完成初始化,之后将沿用此配置</p>
        </div>
      </div>

      <!-- stepper -->
      <div class="steps">
        <div v-for="(s, i) in stepLabels" :key="i" class="step" :class="{ active: step === i + 1, done: step > i + 1 }">
          <div class="dot-n">
            <Icon v-if="step > i + 1" name="check" style="width: 14px; height: 14px" />
            <span v-else>{{ i + 1 }}</span>
          </div>
          <span class="step-label">{{ s }}</span>
        </div>
      </div>

      <!-- step 1: platform -->
      <transition name="fade-slide" mode="out-in">
        <div :key="step">
          <div v-if="step === 1">
            <h3 class="sec-title">确认运行平台</h3>
            <p class="muted sec-desc">将自动下载与该平台匹配的 frp 二进制。已为你检测到当前系统:</p>
            <div class="detected center gap-sm">
              <span class="badge badge-accent"><Icon name="cpu" style="width: 14px; height: 14px" />{{ detectedLabel }}</span>
            </div>
            <div class="field mt">
              <label>下载目标平台</label>
              <Select v-model="platformKey" :options="platformOptions" />
              <span class="hint">若面板将在另一平台运行,可在此切换。默认即可。</span>
            </div>
            <div class="actions">
              <span></span>
              <button class="btn btn-primary" @click="step = 2">下一步 <Icon name="chevron-right" /></button>
            </div>
          </div>

          <!-- step 2: role -->
          <div v-else-if="step === 2">
            <h3 class="sec-title">选择运行角色</h3>
            <p class="muted sec-desc">此选择决定面板管理 frps 还是 frpc,初始化后不可更改。</p>
            <div class="role-grid">
              <button class="role-card" :class="{ sel: role === 'server' }" @click="role = 'server'">
                <Icon name="server" class="role-ico" />
                <b>服务端 (frps)</b>
                <span>部署在公网服务器,接受客户端连接并对外转发流量。</span>
                <span class="role-check"><Icon name="check" style="width: 14px; height: 14px" /></span>
              </button>
              <button class="role-card" :class="{ sel: role === 'client' }" @click="role = 'client'">
                <Icon name="laptop" class="role-ico" />
                <b>客户端 (frpc)</b>
                <span>部署在内网机器,把本地服务穿透暴露到 frps。</span>
                <span class="role-check"><Icon name="check" style="width: 14px; height: 14px" /></span>
              </button>
            </div>
            <div class="actions">
              <button class="btn btn-ghost" @click="step = 1">返回</button>
              <button class="btn btn-primary" :disabled="!role" @click="step = 3">下一步 <Icon name="chevron-right" /></button>
            </div>
          </div>

          <!-- step 3: download -->
          <div v-else-if="step === 3">
            <h3 class="sec-title">下载 frp</h3>
            <p class="muted sec-desc">从 GitHub 获取最新版 frp 并解压到程序同级目录。</p>
            <div class="dl-target center gap-sm">
              <Icon name="download" style="width: 18px; height: 18px; color: var(--accent-ink)" />
              <span>目标:<b>frp · {{ platformLabel }}</b></span>
              <span v-if="dl.version" class="badge badge-ok" style="margin-left: auto">v{{ dl.version }}</span>
            </div>

            <div v-if="dl.started" class="dl-progress mt">
              <div class="between" style="margin-bottom: 8px">
                <span class="center gap-sm">
                  <Icon v-if="!dl.done && !dl.error" name="loader" class="spin" style="width: 16px; height: 16px; color: var(--accent)" />
                  <Icon v-else-if="dl.done" name="check" style="width: 16px; height: 16px; color: var(--ok)" />
                  <Icon v-else name="alert-triangle" style="width: 16px; height: 16px; color: var(--danger)" />
                  <span style="font-weight: 600">{{ phaseText }}</span>
                </span>
                <span class="mono muted" style="font-size: 12.5px">{{ bytesText }}</span>
              </div>
              <ProgressBar :value="dl.percent" :indeterminate="indeterminate" />
              <p v-if="dl.error" class="dl-error">{{ dl.error }}</p>
            </div>

            <div class="actions">
              <button class="btn btn-ghost" :disabled="dl.started && !dl.done && !dl.error" @click="step = 2">返回</button>
              <button v-if="!dl.started || dl.error" class="btn btn-primary" @click="startDownload">
                <Icon name="download" /> {{ dl.error ? '重试下载' : '开始下载' }}
              </button>
              <button v-else class="btn btn-primary" :disabled="!dl.done" @click="step = 4">下一步 <Icon name="chevron-right" /></button>
            </div>
          </div>

          <!-- step 4: password -->
          <div v-else>
            <h3 class="sec-title">设置面板密码</h3>
            <p class="muted sec-desc">用于登录管理面板,请妥善保管。</p>
            <div class="field">
              <label>面板密码</label>
              <div class="pw-wrap">
                <input class="input" :type="showPw ? 'text' : 'password'" v-model="password" placeholder="至少 6 位" @keyup.enter="finalize" />
                <button class="pw-eye" type="button" @click="showPw = !showPw">
                  <Icon :name="showPw ? 'eye-off' : 'eye'" style="width: 17px; height: 17px" />
                </button>
              </div>
            </div>
            <div class="field">
              <label>确认密码</label>
              <input class="input" :type="showPw ? 'text' : 'password'" v-model="password2" placeholder="再次输入" @keyup.enter="finalize" />
            </div>
            <div class="actions">
              <button class="btn btn-ghost" @click="step = 3">返回</button>
              <button class="btn btn-primary btn-lg" :disabled="busy" @click="finalize">
                <Icon v-if="busy" name="loader" class="spin" />
                <Icon v-else name="check" />
                完成初始化
              </button>
            </div>
          </div>
        </div>
      </transition>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import Icon from '../components/Icon.vue'
import ProgressBar from '../components/ProgressBar.vue'
import Select from '../components/Select.vue'
import { api, sse } from '../api.js'
import { refreshSession } from '../router.js'
import { useSSE } from '../composables/useSSE.js'
import { toast } from '../composables/useToast.js'
import { formatBytes } from '../util.js'

const router = useRouter()
const stepLabels = ['平台', '角色', '下载', '密码']
const step = ref(1)

const options = ref([])
const detected = reactive({ os: '', arch: '', label: '' })
const platformKey = ref('')
const role = ref('')

const dl = reactive({ started: false, phase: '', percent: 0, downloaded: 0, total: 0, version: '', error: '', done: false })
const { open: openSSE, close: closeSSE } = useSSE()

const password = ref('')
const password2 = ref('')
const showPw = ref(false)
const busy = ref(false)

const detectedLabel = computed(() => detected.label || `${detected.os}/${detected.arch}`)
const platformLabel = computed(() => {
  const o = options.value.find((o) => o.os + '|' + o.arch === platformKey.value)
  return o ? o.label : platformKey.value.replace('|', '/')
})
const platformOptions = computed(() =>
  options.value.map((o) => ({ value: `${o.os}|${o.arch}`, label: `${o.label} (${o.os}/${o.arch})` })),
)
const indeterminate = computed(() => dl.started && !dl.done && !dl.error && (dl.phase === 'resolving' || dl.phase === 'verifying' || dl.phase === 'extracting'))
const phaseMap = {
  resolving: '正在查询版本…',
  downloading: '正在下载',
  verifying: '正在校验完整性…',
  extracting: '正在解压…',
  done: '下载完成 🎉',
  error: '下载失败',
}
const phaseText = computed(() => phaseMap[dl.phase] || '准备中…')
const bytesText = computed(() => {
  if (!dl.total) return ''
  return `${formatBytes(dl.downloaded)} / ${formatBytes(dl.total)} · ${dl.percent.toFixed(0)}%`
})

onMounted(async () => {
  try {
    const r = await api.platforms()
    Object.assign(detected, r.detected)
    options.value = r.options
    platformKey.value = detected.os + '|' + detected.arch
  } catch (e) {
    toast.err('加载失败', e.message)
  }
})

function startDownload() {
  const [os, arch] = platformKey.value.split('|')
  Object.assign(dl, { started: true, phase: 'resolving', percent: 0, downloaded: 0, total: 0, error: '', done: false })
  // Subscribe first so we don't miss early events.
  openSSE(sse.setupProgress, {
    progress: (data) => {
      let p
      try {
        p = JSON.parse(data)
      } catch {
        return
      }
      dl.phase = p.phase
      dl.percent = p.percent || 0
      dl.downloaded = p.downloaded || 0
      dl.total = p.total || 0
      if (p.version) dl.version = p.version
      if (p.phase === 'error') {
        dl.error = p.error || '下载失败'
        closeSSE()
      } else if (p.done) {
        dl.done = true
        dl.percent = 100
        closeSSE()
        toast.ok('下载完成', `frp v${dl.version} 已就绪`)
      }
    },
  })
  api.install({ os, arch }).catch((e) => {
    dl.error = e.message
    closeSSE()
  })
}

async function finalize() {
  if (password.value.length < 6) {
    toast.warn('密码太短', '面板密码至少 6 位')
    return
  }
  if (password.value !== password2.value) {
    toast.warn('两次密码不一致', '请重新确认')
    return
  }
  busy.value = true
  try {
    await api.finalize(role.value, password.value)
    await refreshSession()
    toast.ok('初始化完成', '欢迎使用 frp 面板')
    router.push({ name: 'dashboard' })
  } catch (e) {
    toast.err('初始化失败', e.message)
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.setup-wrap {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 28px 18px;
}
.setup-card {
  width: 100%;
  max-width: 560px;
  padding: 30px 32px 26px;
}
.setup-brand {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 24px;
}
.steps {
  display: flex;
  justify-content: space-between;
  margin-bottom: 26px;
  position: relative;
}
.steps::before {
  content: '';
  position: absolute;
  top: 15px;
  left: 24px;
  right: 24px;
  height: 2px;
  background: var(--border);
  z-index: 0;
}
.step {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  z-index: 1;
}
.dot-n {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--surface-2);
  border: 2px solid var(--border);
  display: grid;
  place-items: center;
  font-weight: 700;
  font-size: 13px;
  color: var(--ink-faint);
  transition: all 0.3s var(--ease);
}
.step.active .dot-n {
  background: var(--accent);
  border-color: var(--accent);
  color: #fff;
  transform: scale(1.1);
  box-shadow: 0 4px 12px rgba(224, 164, 88, 0.4);
}
.step.done .dot-n {
  background: var(--ok);
  border-color: var(--ok);
  color: #fff;
}
.step-label {
  font-size: 12px;
  color: var(--ink-soft);
  font-weight: 550;
}
.step.active .step-label {
  color: var(--ink);
}
.sec-title {
  font-size: 16px;
  margin-bottom: 4px;
}
.sec-desc {
  font-size: 13px;
  margin: 0 0 16px;
}
.detected {
  margin-bottom: 4px;
}
.actions {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-top: 24px;
}
.role-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}
@media (max-width: 520px) {
  .role-grid {
    grid-template-columns: 1fr;
  }
}
.role-card {
  position: relative;
  text-align: left;
  padding: 18px;
  border-radius: var(--radius);
  border: 1.5px solid var(--border);
  background: var(--surface);
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 7px;
  transition: all 0.22s var(--ease);
}
.role-card:hover {
  border-color: var(--border-strong);
  transform: translateY(-2px);
  box-shadow: var(--shadow);
}
.role-card.sel {
  border-color: var(--accent);
  background: var(--accent-soft);
}
.role-card b {
  font-size: 15px;
}
.role-card span {
  font-size: 12.5px;
  color: var(--ink-soft);
  line-height: 1.5;
}
.role-ico {
  width: 26px;
  height: 26px;
  color: var(--accent-ink);
}
.role-check {
  position: absolute;
  top: 14px;
  right: 14px;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: var(--accent);
  color: #fff;
  display: grid;
  place-items: center;
  opacity: 0;
  transform: scale(0.5);
  transition: all 0.22s var(--ease-out);
}
.role-card.sel .role-check {
  opacity: 1;
  transform: scale(1);
}
.dl-target {
  padding: 12px 14px;
  border-radius: var(--radius-sm);
  background: var(--surface-2);
  font-size: 13.5px;
}
.dl-progress {
  padding: 16px;
  border-radius: var(--radius-sm);
  background: var(--surface-2);
}
.dl-error {
  color: var(--danger);
  font-size: 12.5px;
  margin: 10px 0 0;
}
.pw-wrap {
  position: relative;
}
.pw-eye {
  position: absolute;
  right: 8px;
  top: 50%;
  transform: translateY(-50%);
  background: none;
  border: 0;
  color: var(--ink-faint);
  cursor: pointer;
  padding: 4px;
  display: grid;
  place-items: center;
}
.pw-eye:hover {
  color: var(--ink);
}
</style>
