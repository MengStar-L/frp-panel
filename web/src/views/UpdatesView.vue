<template>
  <div>
    <div class="page-head">
      <div>
        <h1>更新</h1>
        <p>检查并升级 frp 到最新版本</p>
      </div>
      <button class="btn btn-sm" :disabled="checking || dl.running" @click="check"><Icon name="refresh-cw" :class="{ spin: checking }" style="width: 14px; height: 14px" /> 重新检查</button>
    </div>

    <div class="card enter-up">
      <div v-if="checking && !info" class="center gap-sm muted"><Icon name="loader" class="spin" /> 正在检查更新…</div>
      <template v-else-if="info">
        <div class="ver-row">
          <div class="ver-box">
            <span class="muted" style="font-size: 12px">当前版本</span>
            <b class="mono">{{ info.current ? 'v' + info.current : '未知' }}</b>
          </div>
          <Icon name="chevron-right" style="width: 20px; height: 20px; color: var(--ink-faint)" />
          <div class="ver-box">
            <span class="muted" style="font-size: 12px">最新版本</span>
            <b class="mono">v{{ info.latest }}</b>
          </div>
          <div style="margin-left: auto">
            <span v-if="info.hasUpdate" class="badge badge-accent"><Icon name="download" style="width: 13px; height: 13px" /> 有新版本</span>
            <span v-else class="badge badge-ok"><Icon name="check" style="width: 13px; height: 13px" /> 已是最新</span>
          </div>
        </div>

        <div v-if="dl.running || dl.done || dl.error" class="dl-box mt">
          <div class="between" style="margin-bottom: 8px">
            <span class="center gap-sm">
              <Icon v-if="dl.running" name="loader" class="spin" style="width: 16px; height: 16px; color: var(--accent)" />
              <Icon v-else-if="dl.done" name="check" style="width: 16px; height: 16px; color: var(--ok)" />
              <Icon v-else name="alert-triangle" style="width: 16px; height: 16px; color: var(--danger)" />
              <span style="font-weight: 600">{{ phaseText }}</span>
            </span>
            <span class="mono muted" style="font-size: 12.5px">{{ bytesText }}</span>
          </div>
          <ProgressBar :value="dl.percent" :indeterminate="indeterminate" />
          <p v-if="dl.error" style="color: var(--danger); font-size: 12.5px; margin: 10px 0 0">{{ dl.error }}</p>
        </div>

        <div class="mt-lg center gap-sm">
          <button class="btn btn-primary" :disabled="dl.running" @click="doUpdate">
            <Icon :name="dl.running ? 'loader' : 'download'" :class="{ spin: dl.running }" />
            {{ info.hasUpdate ? `更新到 v${info.latest}` : `重新安装 v${info.latest}` }}
          </button>
          <a class="btn btn-ghost" :href="info.url" target="_blank" rel="noopener">查看发布说明 ↗</a>
        </div>
        <p class="hint mt">更新会先停止 frp,替换二进制后自动恢复之前的运行状态。</p>
      </template>
      <div v-else class="empty"><div class="ico">⚠️</div>{{ checkError || '无法获取版本信息' }}</div>
    </div>

    <div v-if="info?.notes" class="card enter-up mt-lg">
      <div class="card-title mb"><Icon name="info" style="width: 17px; height: 17px" /> v{{ info.latest }} 发布说明</div>
      <pre class="notes">{{ info.notes }}</pre>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import Icon from '../components/Icon.vue'
import ProgressBar from '../components/ProgressBar.vue'
import { api, sse } from '../api.js'
import { useSSE } from '../composables/useSSE.js'
import { toast } from '../composables/useToast.js'
import { formatBytes } from '../util.js'

const checking = ref(false)
const checkError = ref('')
const info = ref(null)
const dl = reactive({ running: false, phase: '', percent: 0, downloaded: 0, total: 0, error: '', done: false, version: '' })
const { open: openSSE, close: closeSSE } = useSSE()

const phaseMap = {
  resolving: '准备中…',
  downloading: '正在下载新版本',
  verifying: '正在校验…',
  extracting: '正在替换二进制…',
  done: '更新完成 🎉',
  error: '更新失败',
}
const phaseText = computed(() => phaseMap[dl.phase] || '处理中…')
const indeterminate = computed(() => dl.running && dl.phase !== 'downloading')
const bytesText = computed(() => (dl.total ? `${formatBytes(dl.downloaded)} / ${formatBytes(dl.total)} · ${dl.percent.toFixed(0)}%` : ''))

async function check() {
  checking.value = true
  checkError.value = ''
  try {
    info.value = await api.checkUpdate()
  } catch (e) {
    checkError.value = e.message
  } finally {
    checking.value = false
  }
}
onMounted(check)

function doUpdate() {
  Object.assign(dl, { running: true, phase: 'resolving', percent: 0, downloaded: 0, total: 0, error: '', done: false })
  openSSE(sse.progress, {
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
        dl.running = false
        dl.error = p.error || '更新失败'
        closeSSE()
        toast.err('更新失败', dl.error)
      } else if (p.done) {
        dl.running = false
        dl.done = true
        dl.percent = 100
        closeSSE()
        toast.ok('更新完成', `frp 已升级到 v${dl.version}`)
        setTimeout(check, 800)
      }
    },
  })
  api.performUpdate(info.value.latest).catch((e) => {
    dl.running = false
    dl.error = e.message
    closeSSE()
    toast.err('无法开始更新', e.message)
  })
}
</script>

<style scoped>
.ver-row {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}
.ver-box {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 12px 18px;
  border-radius: var(--radius-sm);
  background: var(--surface-2);
  min-width: 120px;
}
.ver-box b {
  font-size: 17px;
}
.dl-box {
  padding: 16px;
  border-radius: var(--radius-sm);
  background: var(--surface-2);
}
.notes {
  white-space: pre-wrap;
  word-break: break-word;
  font-family: var(--mono);
  font-size: 12px;
  line-height: 1.7;
  color: var(--ink-soft);
  max-height: 320px;
  overflow-y: auto;
  margin: 0;
  background: var(--surface-2);
  padding: 14px 16px;
  border-radius: var(--radius-sm);
}
</style>
