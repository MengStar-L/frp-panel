<template>
  <div class="logs-page">
    <div class="page-head">
      <div>
        <h1>日志</h1>
        <p>frp 进程实时输出 (SSE)</p>
      </div>
      <div class="center gap-sm">
        <span class="center gap-sm muted" style="font-size: 12.5px">
          <span class="dot" :class="connected ? 'on' : 'off'"></span>{{ connected ? '已连接' : '未连接' }}
        </span>
        <label class="center gap-sm" style="font-size: 12.5px; cursor: pointer">
          <span class="switch"><input type="checkbox" v-model="autoscroll" /><span class="track"></span></span>
          自动滚动
        </label>
        <button class="btn btn-sm" @click="clear"><Icon name="trash-2" style="width: 14px; height: 14px" /> 清空</button>
      </div>
    </div>

    <div class="card term" ref="termEl">
      <div v-for="(l, i) in lines" :key="i" class="logline" :class="lineClass(l)">{{ l }}</div>
      <div v-if="!lines.length" class="empty"><div class="ico">📜</div>暂无日志,启动 frp 后这里会实时显示输出</div>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick, onMounted } from 'vue'
import Icon from '../components/Icon.vue'
import { api, sse } from '../api.js'
import { useSSE } from '../composables/useSSE.js'
import { toast } from '../composables/useToast.js'

const lines = ref([])
const autoscroll = ref(true)
const connected = ref(false)
const termEl = ref(null)
const { open } = useSSE()
const MAX = 3000

function scrollDown() {
  if (!autoscroll.value) return
  nextTick(() => {
    const el = termEl.value
    if (el) el.scrollTop = el.scrollHeight
  })
}

onMounted(() => {
  open(sse.logs, {
    log: (data) => {
      lines.value.push(data)
      if (lines.value.length > MAX) lines.value.splice(0, lines.value.length - MAX)
      connected.value = true
      scrollDown()
    },
    error: () => {
      connected.value = false
    },
  })
})

async function clear() {
  try {
    await api.clearLogs()
    lines.value = []
    toast.ok('日志已清空')
  } catch (e) {
    toast.err('清空失败', e.message)
  }
}

function lineClass(l) {
  if (/\b(error|fail|fatal|panic)\b/i.test(l)) return 'err'
  if (/\bwarn/i.test(l)) return 'warn'
  if (l.startsWith('[panel]')) return 'panel'
  return ''
}
</script>

<style scoped>
.logs-page {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.term {
  flex: 1;
  min-height: 360px;
  max-height: calc(100vh - 200px);
  overflow-y: auto;
  font-family: var(--mono);
  font-size: 12.5px;
  line-height: 1.65;
  background: #fffdf7;
  padding: 16px 18px;
}
.logline {
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--ink);
  padding: 1px 0;
  animation: fadeUp 0.18s var(--ease-out);
}
.logline.err {
  color: #c0492f;
}
.logline.warn {
  color: #9a7414;
}
.logline.panel {
  color: var(--accent-ink);
  font-weight: 600;
}
</style>
