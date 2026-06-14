<template>
  <div>
    <div class="page-head">
      <div>
        <h1>设置</h1>
        <p>面板与 frp 运行偏好</p>
      </div>
    </div>

    <div class="grid" style="gap: 16px">
      <!-- frp info -->
      <div class="card enter-up">
        <div class="card-title mb"><Icon name="zap" style="width: 17px; height: 17px" /> frp 信息</div>
        <div class="kv">
          <div><span>角色</span><b>{{ s.role === 'server' ? '服务端 (frps)' : '客户端 (frpc)' }}</b></div>
          <div><span>版本</span><b>{{ s.frp?.version ? 'v' + s.frp.version : '—' }}</b></div>
          <div><span>平台</span><b class="mono">{{ s.frp?.os }}/{{ s.frp?.arch }}</b></div>
          <div><span>监听地址</span><b class="mono">{{ s.listenAddr }}</b></div>
        </div>
        <p class="hint mt">前往「更新」页可检查并升级 frp。修改监听地址需编辑 panel.json 并重启面板。</p>
      </div>

      <!-- autostart -->
      <div class="card enter-up">
        <div class="between">
          <div>
            <div class="card-title"><Icon name="play" style="width: 17px; height: 17px" /> 开机自启</div>
            <p class="muted" style="font-size: 12.5px; margin: 4px 0 0">面板启动时自动拉起 frp 进程</p>
          </div>
          <label class="switch"><input type="checkbox" v-model="autoStart" @change="saveAuto" /><span class="track"></span></label>
        </div>
      </div>

      <!-- change password -->
      <div class="card enter-up">
        <div class="card-title mb"><Icon name="lock" style="width: 17px; height: 17px" /> 修改面板密码</div>
        <div class="field"><label>当前密码</label><input class="input" type="password" v-model="pw.current" /></div>
        <div class="row">
          <div class="field" style="margin: 0"><label>新密码</label><input class="input" type="password" v-model="pw.next" placeholder="至少 6 位" /></div>
          <div class="field" style="margin: 0"><label>确认新密码</label><input class="input" type="password" v-model="pw.confirm" /></div>
        </div>
        <button class="btn btn-primary mt" :disabled="pwBusy" @click="changePassword"><Icon :name="pwBusy ? 'loader' : 'check'" :class="{ spin: pwBusy }" /> 更新密码</button>
      </div>

      <div class="card enter-up center" style="justify-content: space-between">
        <span class="muted" style="font-size: 12.5px">frp 管理面板 v{{ panelVersion }}</span>
        <a class="muted" href="https://github.com/fatedier/frp" target="_blank" rel="noopener" style="font-size: 12.5px">frp 项目主页 ↗</a>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import Icon from '../components/Icon.vue'
import { api } from '../api.js'
import { session } from '../router.js'
import { toast } from '../composables/useToast.js'

const s = reactive({})
const autoStart = ref(false)
const pw = reactive({ current: '', next: '', confirm: '' })
const pwBusy = ref(false)
const panelVersion = session.panelVersion || '1.0.0'

onMounted(async () => {
  try {
    const r = await api.settings()
    Object.assign(s, r)
    autoStart.value = !!r.autoStart
  } catch (e) {
    toast.err('加载设置失败', e.message)
  }
})

async function saveAuto() {
  try {
    await api.saveSettings({ autoStart: autoStart.value })
    toast.ok(autoStart.value ? '已开启自启' : '已关闭自启')
  } catch (e) {
    toast.err('保存失败', e.message)
    autoStart.value = !autoStart.value
  }
}

async function changePassword() {
  if (pw.next.length < 6) return toast.warn('密码太短', '新密码至少 6 位')
  if (pw.next !== pw.confirm) return toast.warn('两次密码不一致')
  pwBusy.value = true
  try {
    await api.changePassword(pw.current, pw.next)
    toast.ok('密码已更新')
    pw.current = pw.next = pw.confirm = ''
  } catch (e) {
    toast.err('修改失败', e.status === 401 ? '当前密码错误' : e.message)
  } finally {
    pwBusy.value = false
  }
}
</script>

<style scoped>
.kv {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px 24px;
}
.kv > div {
  display: flex;
  justify-content: space-between;
  border-bottom: 1px dashed var(--border);
  padding-bottom: 8px;
}
.kv span {
  color: var(--ink-soft);
  font-size: 13px;
}
@media (max-width: 560px) {
  .kv {
    grid-template-columns: 1fr;
  }
}
</style>
