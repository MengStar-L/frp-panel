<template>
  <div>
    <div class="between mb">
      <div class="center gap-sm">
        <Icon name="link" style="width: 17px; height: 17px; color: var(--accent-ink)" />
        <span style="font-weight: 650">代理 / 隧道</span>
        <span class="badge">{{ list.length }}</span>
      </div>
      <button class="btn btn-sm btn-primary" @click="startNew"><Icon name="plus" /> 添加代理</button>
    </div>

    <transition-group name="list" tag="div" class="proxy-list">
      <div v-for="(p, i) in list" :key="p._k" class="proxy-row">
        <template v-if="editing === i">
          <ProxyForm :draft="draft" @save="commit" @cancel="cancel" />
        </template>
        <template v-else>
          <div class="proxy-info">
            <span class="badge badge-accent" style="text-transform: uppercase">{{ p.type }}</span>
            <div>
              <b>{{ p.name }}</b>
              <span class="proxy-sum mono">{{ summary(p) }}</span>
            </div>
          </div>
          <div class="center gap-sm">
            <button class="btn btn-sm btn-ghost" @click="startEdit(i)"><Icon name="pencil" style="width: 14px; height: 14px" /></button>
            <button class="btn btn-sm btn-ghost" @click="remove(i)"><Icon name="trash-2" style="width: 14px; height: 14px; color: var(--danger)" /></button>
          </div>
        </template>
      </div>
    </transition-group>

    <div v-if="editing === 'new'" class="proxy-row pop">
      <ProxyForm :draft="draft" @save="commit" @cancel="cancel" />
    </div>

    <div v-if="!list.length && editing !== 'new'" class="empty">
      <div class="ico">🔌</div>
      还没有代理,点击「添加代理」把本地服务穿透出去
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import Icon from './Icon.vue'
import ProxyForm from './ProxyForm.vue'

const props = defineProps({ modelValue: { type: Array, default: () => [] } })
const emit = defineEmits(['update:modelValue'])

let keySeq = 0
const list = ref((props.modelValue || []).map((p) => ({ ...p, _k: ++keySeq })))
const editing = ref(null) // index | 'new' | null
const draft = ref(null)

function sync() {
  emit(
    'update:modelValue',
    list.value.map(({ _k, ...rest }) => rest),
  )
}

function blankDraft() {
  return { name: '', type: 'tcp', localIP: '127.0.0.1', localPort: null, remotePort: null, customDomains: '', subdomain: '', locations: '', secretKey: '' }
}

function toDraft(p) {
  return {
    name: p.name || '',
    type: p.type || 'tcp',
    localIP: p.localIP || '127.0.0.1',
    localPort: p.localPort || null,
    remotePort: p.remotePort || null,
    customDomains: (p.customDomains || []).join(', '),
    subdomain: p.subdomain || '',
    locations: (p.locations || []).join(', '),
    secretKey: p.secretKey || '',
  }
}

function fromDraft(d) {
  const splitList = (s) =>
    s
      .split(/[,，\s]+/)
      .map((x) => x.trim())
      .filter(Boolean)
  const p = { name: d.name.trim(), type: d.type }
  if (d.localIP) p.localIP = d.localIP
  if (d.localPort) p.localPort = Number(d.localPort)
  if (['tcp', 'udp'].includes(d.type) && d.remotePort) p.remotePort = Number(d.remotePort)
  if (['http', 'https'].includes(d.type)) {
    const cd = splitList(d.customDomains)
    if (cd.length) p.customDomains = cd
    if (d.subdomain) p.subdomain = d.subdomain.trim()
    if (d.type === 'http') {
      const loc = splitList(d.locations)
      if (loc.length) p.locations = loc
    }
  }
  if (['stcp', 'sudp', 'xtcp'].includes(d.type) && d.secretKey) p.secretKey = d.secretKey.trim()
  return p
}

function startNew() {
  draft.value = blankDraft()
  editing.value = 'new'
}
function startEdit(i) {
  draft.value = toDraft(list.value[i])
  editing.value = i
}
function cancel() {
  editing.value = null
  draft.value = null
}
function commit() {
  const p = fromDraft(draft.value)
  if (editing.value === 'new') {
    list.value.push({ ...p, _k: ++keySeq })
  } else {
    const k = list.value[editing.value]._k
    list.value[editing.value] = { ...p, _k: k }
  }
  cancel()
  sync()
}
function remove(i) {
  list.value.splice(i, 1)
  sync()
}

function summary(p) {
  if (['tcp', 'udp'].includes(p.type)) return `${p.localIP || '127.0.0.1'}:${p.localPort} → 远程 :${p.remotePort}`
  if (['http', 'https'].includes(p.type)) {
    const host = (p.customDomains || []).join(', ') || (p.subdomain ? p.subdomain + '.*' : '')
    return `:${p.localPort} → ${host}`
  }
  return `:${p.localPort}${p.secretKey ? ' · 密钥保护' : ''}`
}
</script>

<style scoped>
.proxy-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.proxy-row {
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--surface-2);
  padding: 12px 14px;
}
.proxy-info {
  display: flex;
  align-items: center;
  gap: 12px;
  justify-content: flex-start;
}
.proxy-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.proxy-info b {
  display: block;
  font-size: 14px;
}
.proxy-sum {
  font-size: 12px;
  color: var(--ink-soft);
}
</style>
