<template>
  <div class="pform">
    <div class="row">
      <div class="field" style="margin: 0">
        <label>名称</label>
        <input class="input" v-model="draft.name" placeholder="如 ssh、web" />
      </div>
      <div class="field" style="margin: 0">
        <label>类型</label>
        <Select v-model="draft.type" :options="typeOptions" />
      </div>
    </div>

    <div class="row mt">
      <div class="field" style="margin: 0">
        <label>本地地址</label>
        <input class="input" v-model="draft.localIP" placeholder="127.0.0.1" />
      </div>
      <div class="field" style="margin: 0">
        <label>本地端口</label>
        <input class="input" type="number" v-model="draft.localPort" placeholder="如 22" />
      </div>
      <div v-if="isTcpUdp" class="field" style="margin: 0">
        <label>远程端口</label>
        <input class="input" type="number" v-model="draft.remotePort" placeholder="如 6000" />
      </div>
    </div>

    <div v-if="isHttp" class="mt">
      <div class="field" style="margin: 0 0 12px">
        <label>自定义域名</label>
        <input class="input" v-model="draft.customDomains" placeholder="多个用逗号分隔,如 a.example.com, b.example.com" />
      </div>
      <div class="row">
        <div class="field" style="margin: 0">
          <label>子域名 (subdomain)</label>
          <input class="input" v-model="draft.subdomain" placeholder="可选" />
        </div>
        <div v-if="draft.type === 'http'" class="field" style="margin: 0">
          <label>路径 (locations)</label>
          <input class="input" v-model="draft.locations" placeholder="可选,如 /api, /static" />
        </div>
      </div>
    </div>

    <div v-if="isSecret" class="field mt" style="margin: 0">
      <label>密钥 (secretKey)</label>
      <input class="input" v-model="draft.secretKey" placeholder="访问者需提供相同密钥" />
    </div>

    <div class="center mt" style="justify-content: flex-end; gap: 10px">
      <button class="btn btn-sm btn-ghost" @click="$emit('cancel')">取消</button>
      <button class="btn btn-sm btn-primary" :disabled="!valid" @click="$emit('save')"><Icon name="check" style="width: 14px; height: 14px" /> 保存代理</button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import Icon from './Icon.vue'
import Select from './Select.vue'

const props = defineProps({ draft: { type: Object, required: true } })
defineEmits(['save', 'cancel'])

const types = ['tcp', 'udp', 'http', 'https', 'stcp', 'sudp', 'xtcp', 'tcpmux']
const typeOptions = types.map((t) => ({ value: t, label: t.toUpperCase() }))
const isTcpUdp = computed(() => ['tcp', 'udp'].includes(props.draft.type))
const isHttp = computed(() => ['http', 'https'].includes(props.draft.type))
const isSecret = computed(() => ['stcp', 'sudp', 'xtcp'].includes(props.draft.type))

const valid = computed(() => {
  const d = props.draft
  if (!d.name?.trim()) return false
  if (!d.localPort) return false
  if (isTcpUdp.value && !d.remotePort) return false
  if (isHttp.value && !d.customDomains?.trim() && !d.subdomain?.trim()) return false
  return true
})
</script>

<style scoped>
.pform {
  width: 100%;
}
</style>
