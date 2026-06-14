<template>
  <div>
    <div class="page-head">
      <div>
        <h1>配置</h1>
        <p>{{ role === 'server' ? '服务端 frps.toml' : '客户端 frpc.toml' }} · 保存后需重启或热重载生效</p>
      </div>
      <div class="center gap-sm">
        <div class="seg">
          <button :class="{ on: mode === 'form' }" @click="mode = 'form'">表单</button>
          <button :class="{ on: mode === 'raw' }" @click="mode = 'raw'">原始 TOML</button>
        </div>
      </div>
    </div>

    <div v-if="loading" class="card"><div class="center gap-sm muted"><Icon name="loader" class="spin" /> 加载中…</div></div>

    <template v-else>
      <transition name="fade-slide" mode="out-in">
        <!-- ============ FORM ============ -->
        <div :key="mode" v-if="mode === 'form'">
          <!-- SERVER -->
          <div v-if="role === 'server'" class="grid" style="gap: 16px">
            <div class="card enter-up">
              <div class="card-title mb"><Icon name="server" style="width: 17px; height: 17px" /> 基础</div>
              <div class="row">
                <div class="field" style="margin: 0"><label>绑定地址 bindAddr</label><input class="input" v-model="srv.bindAddr" placeholder="0.0.0.0" /></div>
                <div class="field" style="margin: 0"><label>绑定端口 bindPort</label><input class="input" type="number" v-model="srv.bindPort" /></div>
              </div>
              <div class="row mt">
                <div class="field" style="margin: 0"><label>HTTP 端口 vhostHTTPPort</label><input class="input" type="number" v-model="srv.vhostHTTPPort" placeholder="0 = 关闭" /></div>
                <div class="field" style="margin: 0"><label>HTTPS 端口 vhostHTTPSPort</label><input class="input" type="number" v-model="srv.vhostHTTPSPort" placeholder="0 = 关闭" /></div>
              </div>
              <div class="field mt" style="margin: 0"><label>泛域名 subDomainHost</label><input class="input" v-model="srv.subDomainHost" placeholder="可选,如 frp.example.com" /></div>
            </div>

            <div class="card enter-up">
              <div class="card-title mb"><Icon name="shield" style="width: 17px; height: 17px" /> 认证</div>
              <div class="field" style="margin: 0">
                <label>Token</label>
                <input class="input mono" v-model="srv.authToken" placeholder="留空则不启用 token 认证" />
                <span class="hint">客户端需配置相同 token 才能连接。</span>
              </div>
            </div>

            <div class="card enter-up">
              <div class="card-title mb"><Icon name="activity" style="width: 17px; height: 17px" /> 管理面板 (webServer)</div>
              <p class="muted" style="font-size: 12px; margin: -6px 0 12px">面板依赖此服务读取流量与连接数据,建议保持开启。</p>
              <div class="row">
                <div class="field" style="margin: 0"><label>监听地址</label><input class="input" v-model="srv.webServer.addr" /></div>
                <div class="field" style="margin: 0"><label>端口</label><input class="input" type="number" v-model="srv.webServer.port" /></div>
              </div>
              <div class="row mt">
                <div class="field" style="margin: 0"><label>用户名</label><input class="input" v-model="srv.webServer.user" /></div>
                <div class="field" style="margin: 0"><label>密码</label><input class="input" v-model="srv.webServer.password" /></div>
              </div>
            </div>

            <div class="card enter-up">
              <div class="card-title mb"><Icon name="terminal" style="width: 17px; height: 17px" /> 日志</div>
              <div class="row">
                <div class="field" style="margin: 0"><label>级别</label><Select v-model="srv.logLevel" :options="logLevels" /></div>
                <div class="field" style="margin: 0"><label>保留天数</label><input class="input" type="number" v-model="srv.logMaxDays" /></div>
              </div>
            </div>
          </div>

          <!-- CLIENT -->
          <div v-else class="grid" style="gap: 16px">
            <div class="card enter-up">
              <div class="card-title mb"><Icon name="server" style="width: 17px; height: 17px" /> 连接服务端</div>
              <div class="row">
                <div class="field" style="margin: 0; flex: 2"><label>服务端地址 serverAddr</label><input class="input" v-model="cli.serverAddr" placeholder="frps 的公网 IP / 域名" /></div>
                <div class="field" style="margin: 0"><label>端口 serverPort</label><input class="input" type="number" v-model="cli.serverPort" /></div>
              </div>
              <div class="row mt">
                <div class="field" style="margin: 0"><label>传输协议</label><Select v-model="cli.protocol" :options="protocols" /></div>
                <div class="field" style="margin: 0"><label>TLS 加密</label><div style="padding-top: 4px"><label class="switch"><input type="checkbox" v-model="cli.tls" /><span class="track"></span></label></div></div>
              </div>
            </div>

            <div class="card enter-up">
              <div class="card-title mb"><Icon name="shield" style="width: 17px; height: 17px" /> 认证</div>
              <div class="field" style="margin: 0"><label>Token</label><input class="input mono" v-model="cli.authToken" placeholder="需与服务端一致" /></div>
            </div>

            <div class="card enter-up">
              <div class="card-title mb"><Icon name="activity" style="width: 17px; height: 17px" /> 本地管理 API (webServer)</div>
              <p class="muted" style="font-size: 12px; margin: -6px 0 12px">用于热重载与读取各代理状态,建议保持开启。</p>
              <div class="row">
                <div class="field" style="margin: 0"><label>监听地址</label><input class="input" v-model="cli.webServer.addr" /></div>
                <div class="field" style="margin: 0"><label>端口</label><input class="input" type="number" v-model="cli.webServer.port" /></div>
              </div>
              <div class="row mt">
                <div class="field" style="margin: 0"><label>用户名</label><input class="input" v-model="cli.webServer.user" /></div>
                <div class="field" style="margin: 0"><label>密码</label><input class="input" v-model="cli.webServer.password" /></div>
              </div>
            </div>

            <div class="card enter-up">
              <ProxyEditor v-model="cli.proxies" />
            </div>

            <div class="card enter-up">
              <div class="card-title mb"><Icon name="terminal" style="width: 17px; height: 17px" /> 日志</div>
              <div class="row">
                <div class="field" style="margin: 0"><label>级别</label><Select v-model="cli.logLevel" :options="logLevels" /></div>
                <div class="field" style="margin: 0"><label>保留天数</label><input class="input" type="number" v-model="cli.logMaxDays" /></div>
              </div>
            </div>
          </div>
        </div>

        <!-- ============ RAW ============ -->
        <div v-else key="raw" class="card enter-up">
          <p class="muted" style="font-size: 12.5px; margin: 0 0 10px">直接编辑 TOML 并保存,将覆盖配置文件(以此内容为准)。</p>
          <textarea class="textarea" v-model="raw" spellcheck="false" style="min-height: 420px"></textarea>
        </div>
      </transition>

      <!-- sticky action bar -->
      <div class="actionbar">
        <span class="muted" style="font-size: 12.5px"><Icon name="info" style="width: 14px; height: 14px; vertical-align: -2px" /> 保存后请重启 frp 生效{{ role === 'client' ? ',或使用热重载' : '' }}</span>
        <div class="center gap-sm">
          <button v-if="role === 'client'" class="btn btn-sm" :disabled="busy" @click="reloadFrp"><Icon name="refresh-cw" style="width: 14px; height: 14px" /> 热重载</button>
          <button class="btn btn-sm" :disabled="busy" @click="restartFrp"><Icon name="rotate-cw" style="width: 14px; height: 14px" /> 重启 frp</button>
          <button class="btn btn-primary" :disabled="saving" @click="save"><Icon :name="saving ? 'loader' : 'save'" :class="{ spin: saving }" /> 保存配置</button>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import Icon from '../components/Icon.vue'
import ProxyEditor from '../components/ProxyEditor.vue'
import Select from '../components/Select.vue'
import { api } from '../api.js'
import { toast } from '../composables/useToast.js'

const logLevels = ['trace', 'debug', 'info', 'warn', 'error']
const protocols = ['tcp', 'kcp', 'quic', 'websocket', 'wss']

const loading = ref(true)
const mode = ref('form')
const saving = ref(false)
const busy = ref(false)
const role = ref('')
const raw = ref('')

const srv = reactive({})
const cli = reactive({})

function normServer(s = {}) {
  return {
    bindAddr: s.bindAddr || '',
    bindPort: s.bindPort || 7000,
    vhostHTTPPort: s.vhostHTTPPort || 0,
    vhostHTTPSPort: s.vhostHTTPSPort || 0,
    subDomainHost: s.subDomainHost || '',
    authToken: s.auth?.token || '',
    webServer: {
      addr: s.webServer?.addr || '0.0.0.0',
      port: s.webServer?.port || 7500,
      user: s.webServer?.user || 'admin',
      password: s.webServer?.password || '',
    },
    logLevel: s.log?.level || 'info',
    logMaxDays: s.log?.maxDays || 3,
  }
}

function normClient(c = {}) {
  return {
    serverAddr: c.serverAddr || '',
    serverPort: c.serverPort || 7000,
    authToken: c.auth?.token || '',
    webServer: {
      addr: c.webServer?.addr || '127.0.0.1',
      port: c.webServer?.port || 7400,
      user: c.webServer?.user || 'admin',
      password: c.webServer?.password || '',
    },
    protocol: c.transport?.protocol || 'tcp',
    tls: c.transport?.tls?.enable !== false,
    logLevel: c.log?.level || 'info',
    logMaxDays: c.log?.maxDays || 3,
    proxies: c.proxies || [],
  }
}

async function load() {
  loading.value = true
  try {
    const b = await api.getConfig()
    role.value = b.role
    raw.value = b.raw || ''
    if (b.role === 'server') Object.assign(srv, normServer(b.server))
    else Object.assign(cli, normClient(b.client))
  } catch (e) {
    toast.err('加载配置失败', e.message)
  } finally {
    loading.value = false
  }
}
onMounted(load)

function buildServer() {
  return {
    bindAddr: srv.bindAddr || '',
    bindPort: Number(srv.bindPort) || 0,
    vhostHTTPPort: Number(srv.vhostHTTPPort) || 0,
    vhostHTTPSPort: Number(srv.vhostHTTPSPort) || 0,
    subDomainHost: srv.subDomainHost || '',
    auth: srv.authToken ? { method: 'token', token: srv.authToken } : null,
    webServer: {
      addr: srv.webServer.addr,
      port: Number(srv.webServer.port) || 0,
      user: srv.webServer.user,
      password: srv.webServer.password,
    },
    log: { to: 'console', level: srv.logLevel, maxDays: Number(srv.logMaxDays) || 0 },
  }
}

function buildClient() {
  return {
    serverAddr: cli.serverAddr,
    serverPort: Number(cli.serverPort) || 0,
    auth: cli.authToken ? { method: 'token', token: cli.authToken } : null,
    webServer: {
      addr: cli.webServer.addr,
      port: Number(cli.webServer.port) || 0,
      user: cli.webServer.user,
      password: cli.webServer.password,
    },
    transport: { protocol: cli.protocol || 'tcp', tls: { enable: !!cli.tls } },
    log: { to: 'console', level: cli.logLevel, maxDays: Number(cli.logMaxDays) || 0 },
    proxies: cli.proxies,
  }
}

async function save() {
  saving.value = true
  try {
    if (mode.value === 'raw') {
      await api.saveRaw(raw.value)
    } else if (role.value === 'server') {
      await api.saveConfig({ server: buildServer() })
    } else {
      await api.saveConfig({ client: buildClient() })
    }
    toast.ok('配置已保存')
    await load() // refresh both form and raw views from disk
  } catch (e) {
    toast.err('保存失败', e.message)
  } finally {
    saving.value = false
  }
}

async function restartFrp() {
  busy.value = true
  try {
    await api.frpRestart()
    toast.ok('frp 已重启')
  } catch (e) {
    toast.err('重启失败', e.message)
  } finally {
    busy.value = false
  }
}

async function reloadFrp() {
  busy.value = true
  try {
    await api.frpReload()
    toast.ok('已热重载')
  } catch (e) {
    toast.err('热重载失败', e.message)
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.seg {
  display: inline-flex;
  background: var(--surface-2);
  border-radius: var(--radius-sm);
  padding: 3px;
  gap: 2px;
}
.seg button {
  border: 0;
  background: none;
  padding: 6px 16px;
  border-radius: var(--radius-xs);
  font-size: 13px;
  font-weight: 600;
  color: var(--ink-soft);
  cursor: pointer;
  transition: all 0.2s var(--ease);
}
.seg button.on {
  background: var(--surface);
  color: var(--accent-ink);
  box-shadow: var(--shadow-sm);
}
.actionbar {
  position: sticky;
  bottom: 16px;
  margin-top: 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  flex-wrap: wrap;
  padding: 12px 18px;
  border-radius: var(--radius);
  background: rgba(255, 253, 248, 0.92);
  backdrop-filter: blur(10px);
  border: 1px solid var(--border);
  box-shadow: var(--shadow);
}
</style>
