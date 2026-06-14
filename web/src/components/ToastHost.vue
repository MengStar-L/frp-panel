<template>
  <div class="toast-host">
    <transition-group name="toast">
      <div v-for="t in toasts" :key="t.id" class="toast" :class="t.type" @click="dismiss(t.id)">
        <span class="ico">
          <Icon :name="iconFor(t.type)" style="width:13px;height:13px" />
        </span>
        <div>
          <b>{{ t.title }}</b>
          <div v-if="t.msg" class="msg">{{ t.msg }}</div>
        </div>
      </div>
    </transition-group>
  </div>
</template>

<script setup>
import { toasts, dismiss } from '../composables/useToast.js'
import Icon from './Icon.vue'

const icons = { ok: 'check', err: 'x', info: 'info', warn: 'alert-triangle' }
const iconFor = (t) => icons[t] || 'info'
</script>
